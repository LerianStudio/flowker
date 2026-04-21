// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/LerianStudio/flowker/pkg/circuitbreaker"
	"github.com/LerianStudio/flowker/pkg/condition"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/transformation"
	"github.com/google/uuid"
	"github.com/sony/gobreaker/v2"
	"go.opentelemetry.io/otel/trace"
)

const (
	executionTimeout = 5 * time.Minute
	maxRetries       = 5
)

// ProviderConfigReadRepository defines the read-only interface needed by
// ExecuteWorkflowCommand to load provider configurations at execution time.
// This is a minimal subset to avoid importing the query package (which would
// create an import cycle since query imports command for type aliases).
type ProviderConfigReadRepository interface {
	// FindByID retrieves a provider configuration by its ID.
	// Returns ErrNotFound if the provider configuration doesn't exist.
	FindByID(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error)
}

// ExecuteWorkflowCommand handles workflow execution using a state machine.
type ExecuteWorkflowCommand struct {
	executionRepo      ExecutionRepository
	workflowRepo       WorkflowRepository
	providerConfigRepo ProviderConfigReadRepository
	catalog            executor.Catalog
	cbManager          *circuitbreaker.Manager
	condEvaluator      *condition.Evaluator
	transformSvc       *transformation.Service
	auditWriter        AuditWriter
}

// NewExecuteWorkflowCommand creates a new ExecuteWorkflowCommand.
func NewExecuteWorkflowCommand(
	executionRepo ExecutionRepository,
	workflowRepo WorkflowRepository,
	providerConfigRepo ProviderConfigReadRepository,
	catalog executor.Catalog,
	cbManager *circuitbreaker.Manager,
	condEvaluator *condition.Evaluator,
	transformSvc *transformation.Service,
	auditWriter AuditWriter,
) (*ExecuteWorkflowCommand, error) {
	if executionRepo == nil {
		return nil, ErrExecuteWorkflowNilRepo
	}

	if workflowRepo == nil {
		return nil, ErrExecuteWorkflowNilWorkflowRepo
	}

	if providerConfigRepo == nil {
		return nil, ErrExecuteWorkflowNilProviderConfigRepo
	}

	if catalog == nil {
		return nil, ErrExecuteWorkflowNilCatalog
	}

	if cbManager == nil {
		return nil, ErrExecuteWorkflowNilCircuitBreaker
	}

	if condEvaluator == nil {
		return nil, ErrExecuteWorkflowNilCondEvaluator
	}

	if transformSvc == nil {
		return nil, ErrExecuteWorkflowNilTransformSvc
	}

	if auditWriter == nil {
		return nil, ErrExecuteWorkflowNilAuditWriter
	}

	return &ExecuteWorkflowCommand{
		executionRepo:      executionRepo,
		workflowRepo:       workflowRepo,
		providerConfigRepo: providerConfigRepo,
		catalog:            catalog,
		cbManager:          cbManager,
		condEvaluator:      condEvaluator,
		transformSvc:       transformSvc,
		auditWriter:        auditWriter,
	}, nil
}

// Execute starts a workflow execution.
// Returns immediately with the execution in pending status.
// The state machine runs asynchronously in a background goroutine.
func (c *ExecuteWorkflowCommand) Execute(
	ctx context.Context,
	workflowID uuid.UUID,
	input *model.ExecuteWorkflowInput,
	idempotencyKey *string,
) (*model.WorkflowExecution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if input == nil {
		return nil, ErrExecuteWorkflowNilInput
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.execution.execute")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Starting workflow execution", libLog.Any("operation", "command.execution.execute"), libLog.Any("workflow.id", workflowID))

	// Fast-path: if idempotency key already exists, return the existing execution.
	existing, fastPathErr := c.checkIdempotencyFastPath(ctx, idempotencyKey)
	if fastPathErr != nil {
		libOtel.HandleSpanError(span, "failed to check idempotency key", fastPathErr)
		return nil, fastPathErr
	}

	if existing != nil {
		return existing, nil
	}

	// Load workflow
	workflow, err := c.workflowRepo.FindByID(ctx, workflowID)
	if err != nil {
		if errors.Is(err, constant.ErrWorkflowNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "workflow not found", err)
			return nil, constant.ErrWorkflowNotFound
		}

		libOtel.HandleSpanError(span, "failed to find workflow", err)

		return nil, fmt.Errorf("failed to find workflow: %w", err)
	}

	// Workflow must be active
	if !workflow.IsActive() {
		libOtel.HandleSpanBusinessErrorEvent(span, "workflow not active", constant.ErrExecutionNotActive)
		return nil, constant.ErrExecutionNotActive
	}

	// Count executable nodes (exclude trigger nodes)
	totalSteps := countExecutableNodes(workflow.Nodes())

	// Create execution entity
	execution := model.NewWorkflowExecution(workflowID, input.InputData, idempotencyKey, totalSteps)

	// Persist to DB (insert-first pattern for idempotency).
	// If a concurrent request already inserted the same idempotency key,
	// the unique index will reject the insert. We then fetch and return
	// the existing execution, avoiding the check-then-act race condition.
	if err := c.executionRepo.Create(ctx, execution); err != nil {
		return c.handleCreateError(ctx, span, err, idempotencyKey)
	}

	// Transition to running synchronously before returning to the caller.
	// This ensures the client never sees a "pending" execution that silently
	// fails to start if the background goroutine cannot update the status.
	previousExecStatus := execution.Status()

	if err := execution.MarkRunning(); err != nil {
		libOtel.HandleSpanError(span, "failed to transition execution to running", err)
		return nil, fmt.Errorf("failed to start execution: %w", err)
	}

	if err := c.executionRepo.Update(ctx, execution, previousExecStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "execution state conflict on start, aborting stale worker", err)
			return nil, constant.ErrConflictStateChanged
		}

		libOtel.HandleSpanError(span, "failed to mark execution as running", err)

		if markErr := execution.MarkFailed(fmt.Sprintf("failed to start execution: %v", err)); markErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		c.persistBestEffort(ctx, execution)

		return nil, fmt.Errorf("failed to start execution: %w", err)
	}

	c.auditWriter.RecordExecutionEvent(ctx, model.AuditEventExecutionStarted, model.AuditActionExecute, model.AuditResultSuccess, execution.ExecutionID(), map[string]any{
		"workflow.id": workflowID.String(),
	})

	logger.Log(ctx, libLog.LevelInfo, "Execution created and running, starting state machine", libLog.Any("operation", "command.execution.execute"), libLog.Any("execution.id", execution.ExecutionID()), libLog.Any("workflow.id", workflowID))

	// Capture an immutable snapshot before launching the goroutine.
	// The goroutine will mutate execution, so returning the original
	// would cause a data race with the caller serializing the response.
	snapshot := execution.Snapshot()

	// Run state machine in background goroutine
	go c.runStateMachine(ctx, execution, workflow)

	return snapshot, nil
}

// runStateMachine executes the workflow graph traversal.
// This runs in a background goroutine with its own timeout context.
// parentCtx is used via context.WithoutCancel to preserve request-scoped
// values (tracing, logging) while detaching from the caller's cancellation.
func (c *ExecuteWorkflowCommand) runStateMachine(
	parentCtx context.Context,
	execution *model.WorkflowExecution,
	workflow *model.Workflow,
) {
	baseCtx := context.WithoutCancel(parentCtx)

	ctx, cancel := context.WithTimeout(baseCtx, executionTimeout)
	defer cancel()

	smLogger := libCommons.NewLoggerFromContext(ctx)

	// Execution is already in "running" state (set by Execute before goroutine launch).

	// Build adjacency graph: source -> []edges
	edges := workflow.Edges()
	graph := buildGraph(edges)

	// Build node map
	nodes := workflow.Nodes()
	nodeMap := buildNodeMap(nodes)

	// Find trigger node
	triggerNodeID := findTriggerNode(nodes)
	if triggerNodeID == "" {
		if markErr := execution.MarkFailed("no trigger node found in workflow"); markErr != nil {
			smLogger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		c.persistBestEffort(ctx, execution)

		return
	}

	// Initialize workflow context with input data
	wfCtx := map[string]any{
		"workflow": execution.InputData(),
	}

	// Traverse graph from trigger node
	stepNumber := 0
	inPath := make(map[string]bool)

	if err := c.traverseNode(ctx, triggerNodeID, execution, nodeMap, graph, wfCtx, &stepNumber, inPath); err != nil {
		// If the error is a state conflict, another worker already owns this
		// execution's state. Bail out immediately — do NOT call persistBestEffort
		// which would overwrite the winner's state with empty expectedStatus.
		if errors.Is(err, constant.ErrConflictStateChanged) {
			conflictLogger := libCommons.NewLoggerFromContext(ctx)
			conflictLogger.Log(ctx, libLog.LevelWarn, "Execution state conflict in state machine, aborting stale worker",
				libLog.Any("execution.id", execution.ExecutionID()),
				libLog.Any("workflow.id", execution.WorkflowID()))
			return
		}

		if markErr := execution.MarkFailed(err.Error()); markErr != nil {
			smLogger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		// Always persist the final state — inner handlers may have called
		// MarkFailed without a subsequent Update, so the DB could still
		// show "running" even though the in-memory status is "failed".
		c.persistBestEffort(ctx, execution)

		c.auditWriter.RecordExecutionEvent(ctx, model.AuditEventExecutionFailed, model.AuditActionExecute, model.AuditResultFailed, execution.ExecutionID(), map[string]any{
			"workflow.id": execution.WorkflowID().String(),
			"error":       err.Error(),
		})

		return
	}

	// If we haven't failed, mark as completed
	if !execution.IsTerminal() {
		// Extract final output from workflow context
		var finalOutput map[string]any

		if output, ok := wfCtx["output"]; ok {
			if m, ok := output.(map[string]any); ok {
				finalOutput = m
			}
		}

		if finalOutput == nil {
			finalOutput = wfCtx
		}

		if markErr := execution.MarkCompleted(finalOutput); markErr != nil {
			smLogger.Log(ctx, libLog.LevelError, "Failed to mark execution as completed",
				libLog.Any("error.message", markErr.Error()))
		}

		c.persistBestEffort(ctx, execution)

		c.auditWriter.RecordExecutionEvent(ctx, model.AuditEventExecutionCompleted, model.AuditActionExecute, model.AuditResultSuccess, execution.ExecutionID(), map[string]any{
			"workflow.id": execution.WorkflowID().String(),
		})
	}
}

// traverseNode processes a node and follows edges to the next nodes.
// inPath tracks nodes on the current recursion stack to guard against cycles,
// while still allowing valid DAG merges (diamond patterns).
func (c *ExecuteWorkflowCommand) traverseNode(
	ctx context.Context,
	nodeID string,
	execution *model.WorkflowExecution,
	nodeMap map[string]model.WorkflowNode,
	graph map[string][]model.WorkflowEdge,
	wfCtx map[string]any,
	stepNumber *int,
	inPath map[string]bool,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("execution timeout: %w", err)
	}

	// Guard against cycles: only flag nodes on the current recursion stack
	if inPath[nodeID] {
		return fmt.Errorf("cycle detected at node %q: %w", nodeID, constant.ErrExecutionCycleDetected)
	}

	inPath[nodeID] = true
	defer delete(inPath, nodeID)

	node, ok := nodeMap[nodeID]
	if !ok {
		return fmt.Errorf("node %q not found in workflow: %w", nodeID, constant.ErrExecutionNodeFailed)
	}

	// Process node based on type
	switch node.Type() {
	case model.NodeTypeTrigger:
		// Trigger nodes are entry points - just follow edges
	case model.NodeTypeExecutor:
		*stepNumber++

		if err := c.executeExecutorNode(ctx, node, execution, wfCtx, *stepNumber); err != nil {
			return err
		}
	case model.NodeTypeConditional:
		*stepNumber++

		nextNodeID, err := c.evaluateConditionalNode(ctx, node, execution, wfCtx, graph, *stepNumber)
		if err != nil {
			return err
		}

		if nextNodeID != "" {
			return c.traverseNode(ctx, nextNodeID, execution, nodeMap, graph, wfCtx, stepNumber, inPath)
		}

		return nil
	case model.NodeTypeAction:
		*stepNumber++
		c.executeActionNode(node, execution, wfCtx, *stepNumber)
	}

	// Follow outgoing edges (non-conditional nodes)
	if node.Type() != model.NodeTypeConditional {
		outEdges := graph[nodeID]

		for _, edge := range outEdges {
			if err := c.traverseNode(ctx, edge.Target(), execution, nodeMap, graph, wfCtx, stepNumber, inPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// executeExecutorNode executes an executor node with retry and circuit breaker.
// Requires providerConfigId to be set in the node data.
func (c *ExecuteWorkflowCommand) executeExecutorNode(
	ctx context.Context,
	node model.WorkflowNode,
	execution *model.WorkflowExecution,
	wfCtx map[string]any,
	stepNumber int,
) error {
	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.execution.execute_executor_node")
	defer span.End()

	nodeName := nodeDisplayName(node)
	step := model.NewExecutionStep(stepNumber, nodeName, node.ID(), wfCtx)

	providerConfigID := node.ProviderConfigID()
	if providerConfigID == "" {
		logger.Log(ctx, libLog.LevelError, "Executor node missing providerConfigId", libLog.Any("node.id", node.ID()), libLog.Any("execution.id", execution.ExecutionID()))

		_ = step.MarkFailed("executor node missing providerConfigId in data")
		execution.AddStep(step)
		c.persistBestEffort(ctx, execution)

		if markErr := execution.MarkFailed(fmt.Sprintf("node %q: executor node missing providerConfigId", node.ID())); markErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		libOtel.HandleSpanBusinessErrorEvent(span, "node missing providerConfigId", constant.ErrExecutionNodeFailed)

		return constant.ErrExecutionNodeFailed
	}

	return c.executeWithProviderConfig(ctx, node, execution, wfCtx, &step, providerConfigID)
}

// executeWithProviderConfig executes a node using provider configuration (new model).
// Loads the provider configuration from the database and builds runner input from its config map.
func (c *ExecuteWorkflowCommand) executeWithProviderConfig(
	ctx context.Context,
	node model.WorkflowNode,
	execution *model.WorkflowExecution,
	wfCtx map[string]any,
	step *model.ExecutionStep,
	providerConfigID string,
) error {
	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.execution.execute_with_provider_config")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Executing node with provider configuration", libLog.Any("node.id", node.ID()), libLog.Any("execution.id", execution.ExecutionID()), libLog.Any("executor.id", node.ExecutorID()), libLog.Any("providerConfig.id", providerConfigID))

	// Parse and load provider configuration
	configID, err := uuid.Parse(providerConfigID)
	if err != nil {
		_ = step.MarkFailed(fmt.Sprintf("invalid providerConfigId: %v", err))
		execution.AddStep(*step)
		c.persistBestEffort(ctx, execution)

		if markErr := execution.MarkFailed(fmt.Sprintf("node %q: invalid providerConfigId", node.ID())); markErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		libOtel.HandleSpanBusinessErrorEvent(span, "invalid providerConfigId", err)

		return constant.ErrExecutionNodeFailed
	}

	providerConfig, err := c.providerConfigRepo.FindByID(ctx, configID)
	if err != nil {
		_ = step.MarkFailed(fmt.Sprintf("provider config not found: %v", err))
		execution.AddStep(*step)
		c.persistBestEffort(ctx, execution)

		if markErr := execution.MarkFailed(fmt.Sprintf("node %q: provider config not found", node.ID())); markErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		libOtel.HandleSpanError(span, "failed to load provider config", err)

		return constant.ErrExecutionNodeFailed
	}

	if !providerConfig.IsActive() {
		_ = step.MarkFailed("provider configuration is not active")
		execution.AddStep(*step)
		c.persistBestEffort(ctx, execution)

		if markErr := execution.MarkFailed(fmt.Sprintf("node %q: provider configuration is not active", node.ID())); markErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		libOtel.HandleSpanBusinessErrorEvent(span, "provider config not active", constant.ErrExecutionNodeFailed)

		return constant.ErrExecutionNodeFailed
	}

	// Apply input transformations
	requestBody, err := c.applyInputTransformations(ctx, node, wfCtx)
	if err != nil {
		_ = step.MarkFailed(fmt.Sprintf("input transformation failed: %v", err))
		execution.AddStep(*step)
		c.persistBestEffort(ctx, execution)

		if markErr := execution.MarkFailed(fmt.Sprintf("node %q: input transformation failed", node.ID())); markErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		libOtel.HandleSpanError(span, "input transformation failed", err)

		return constant.ErrExecutionNodeFailed
	}

	// Build runner input from provider configuration
	executorID := node.ExecutorID()

	input, err := c.buildRunnerInput(providerConfig, node, requestBody)
	if err != nil {
		_ = step.MarkFailed(err.Error())
		execution.AddStep(*step)
		c.persistBestEffort(ctx, execution)

		if markErr := execution.MarkFailed(fmt.Sprintf("node %q: failed to build runner input: %v", node.ID(), err)); markErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		libOtel.HandleSpanError(span, "failed to build runner input", err)

		return constant.ErrExecutionNodeFailed
	}

	// Execute with retry and circuit breaker
	result, retryErr := c.executeRunnerWithRetry(ctx, executorID, input, providerConfigID, step)
	if retryErr != nil {
		_ = step.MarkFailed(retryErr.Error())
		execution.AddStep(*step)
		c.persistBestEffort(ctx, execution)

		if markErr := execution.MarkFailed(fmt.Sprintf("node %q: executor call failed after %d attempts: %v", node.ID(), step.AttemptNumber(), retryErr)); markErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		libOtel.HandleSpanError(span, "executor call failed", retryErr)

		return constant.ErrExecutionNodeFailed
	}

	// Apply output transformations
	outputData, err := c.applyOutputTransformations(ctx, node, result)
	if err != nil {
		_ = step.MarkFailed(fmt.Sprintf("output transformation failed: %v", err))
		execution.AddStep(*step)
		c.persistBestEffort(ctx, execution)

		if markErr := execution.MarkFailed(fmt.Sprintf("node %q: output transformation failed", node.ID())); markErr != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		libOtel.HandleSpanError(span, "output transformation failed", err)

		return constant.ErrExecutionNodeFailed
	}

	// Update workflow context with node output
	wfCtx[node.ID()] = outputData

	_ = step.MarkCompleted(outputData)
	execution.AddStep(*step)

	// Persist after each step (execution stays in running status)
	if err := c.executionRepo.Update(ctx, execution, model.ExecutionStatusRunning); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			logger.Log(ctx, libLog.LevelWarn, "Execution state conflict detected, aborting stale worker",
				libLog.Any("execution.id", execution.ExecutionID()),
				libLog.Any("node.id", node.ID()))
			return constant.ErrConflictStateChanged
		}

		return fmt.Errorf("failed to persist step: %w", err)
	}

	return nil
}

// extractResponseBody returns the parsed response body from the Runner result.
// extractResponseBody returns the full runner response envelope.
// Runners wrap responses in {status, url, headers, body}. The full envelope
// is preserved so downstream mappings and conditions can access both HTTP
// metadata (status, headers) and the body consistently, regardless of
// response payload type.
func extractResponseBody(data map[string]any) map[string]any {
	return data
}

// buildRunnerInput checks if the provider implements InputBuilder for custom
// execution input construction. Falls back to the generic builder if not.
func (c *ExecuteWorkflowCommand) buildRunnerInput(
	providerConfig *model.ProviderConfiguration,
	node model.WorkflowNode,
	requestBody []byte,
) (executor.ExecutionInput, error) {
	executorID := executor.ID(node.ExecutorID())

	catalogExec, err := c.catalog.GetExecutor(executorID)
	if err == nil {
		providerObj, provErr := c.catalog.GetProvider(catalogExec.ProviderID())
		if provErr == nil {
			if builder, ok := providerObj.(executor.InputBuilder); ok {
				return builder.BuildInput(providerConfig.Config(), executorID, node.Data(), requestBody)
			}
		}
	}

	return buildProviderConfigRunnerInput(providerConfig, node, requestBody)
}

// buildProviderConfigRunnerInput constructs the executor.ExecutionInput from a ProviderConfiguration.
// The provider config's config map contains connection details (base_url, headers, api_key, etc.)
// which are merged with the node's endpoint information.
func buildProviderConfigRunnerInput(
	providerConfig *model.ProviderConfiguration,
	node model.WorkflowNode,
	requestBody []byte,
) (executor.ExecutionInput, error) {
	pcConfig := providerConfig.Config()

	// Build the config map for the runner from provider configuration
	config := make(map[string]any)

	// Extract base_url from provider config and combine with endpoint path
	if baseURL, ok := pcConfig["base_url"].(string); ok {
		endpointPath := node.EndpointName()

		// If the node has a specific path in its data, use it
		if path, ok := node.Data()["path"].(string); ok {
			endpointPath = path
		}

		if endpointPath != "" {
			fullURL, err := url.JoinPath(baseURL, endpointPath)
			if err != nil {
				return executor.ExecutionInput{}, fmt.Errorf("failed to build URL from base %q and path %q: %w", baseURL, endpointPath, err)
			}

			config["url"] = fullURL
		} else {
			config["url"] = baseURL
		}
	}

	// Extract method from node data (default to POST)
	if method, ok := node.Data()["method"].(string); ok {
		config["method"] = method
	} else {
		config["method"] = "POST"
	}

	// Copy timeout from node data or provider config
	if timeout, ok := node.Data()["timeout"].(float64); ok {
		config["timeout_seconds"] = int(timeout)
	} else if timeout, ok := pcConfig["timeout_seconds"]; ok {
		config["timeout_seconds"] = timeout
	}

	// Set request body
	if len(requestBody) > 0 {
		config["body"] = requestBody
	}

	// Copy headers from provider config
	if headers, ok := pcConfig["headers"].(map[string]any); ok {
		config["headers"] = headers
	}

	// Build authentication from provider config
	if auth := buildAuthConfig(pcConfig); auth != nil {
		config["auth"] = auth
	}

	return executor.ExecutionInput{
		Config: config,
	}, nil
}

// buildAuthConfig extracts authentication configuration from a provider config map.
// Returns nil if no authentication is configured (auth_type is empty or "none").
func buildAuthConfig(pcConfig map[string]any) map[string]any {
	authType, ok := pcConfig["auth_type"].(string)
	if !ok || authType == "" || authType == "none" {
		return nil
	}

	authConfig := make(map[string]any)

	// Copy auth-related fields from provider config.
	// Provider schemas use "api_key" (user-friendly), but the auth factory (APIKeyConfig)
	// expects "key" (Go struct field). This translates between the two naming conventions.
	if apiKey, ok := pcConfig["api_key"].(string); ok {
		authConfig["key"] = apiKey
	}

	if token, ok := pcConfig["bearer_token"].(string); ok {
		authConfig["token"] = token
	}

	if username, ok := pcConfig["username"].(string); ok {
		authConfig["username"] = username
	}

	if password, ok := pcConfig["password"].(string); ok {
		authConfig["password"] = password
	}

	// Copy OIDC fields
	if tokenURL, ok := pcConfig["token_url"].(string); ok {
		authConfig["token_url"] = tokenURL
	}

	if clientID, ok := pcConfig["client_id"].(string); ok {
		authConfig["client_id"] = clientID
	}

	if clientSecret, ok := pcConfig["client_secret"].(string); ok {
		authConfig["client_secret"] = clientSecret
	}

	return map[string]any{
		"type":   authType,
		"config": authConfig,
	}
}

// executeRunnerWithRetry executes a runner call with exponential backoff retry using provider config.
// This is similar to executeWithRetry but works with a pre-built ExecutionInput.
func (c *ExecuteWorkflowCommand) executeRunnerWithRetry(
	ctx context.Context,
	executorID string,
	input executor.ExecutionInput,
	configID string,
	step *model.ExecutionStep,
) (map[string]any, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		step.SetAttemptNumber(attempt)

		result, err := c.callRunnerWithInput(ctx, executorID, input, configID, step)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry on non-transient errors: circuit breaker open,
		// context cancellation, or errors that indicate permanent failures
		// (missing runner, validation errors) rather than transient network issues.
		if errors.Is(err, constant.ErrExecutionCircuitOpen) ||
			errors.Is(err, context.Canceled) ||
			errors.Is(err, context.DeadlineExceeded) ||
			isNonRetryableError(err) {
			return nil, lastErr
		}

		if attempt < maxRetries {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	return nil, lastErr
}

// isNonRetryableError returns true for errors that indicate permanent failures
// which should not be retried (e.g., missing runner, configuration errors).
func isNonRetryableError(err error) bool {
	msg := err.Error()

	return strings.Contains(msg, "no runner registered") ||
		strings.Contains(msg, "failed to build URL")
}

// callRunnerWithInput delegates execution to the registered Runner using a pre-built ExecutionInput,
// wrapping the call in a circuit breaker. Used by the provider config execution path.
func (c *ExecuteWorkflowCommand) callRunnerWithInput(
	ctx context.Context,
	executorID string,
	input executor.ExecutionInput,
	configID string,
	step *model.ExecutionStep,
) (map[string]any, error) {
	runner, err := c.catalog.GetRunner(executor.ID(executorID))
	if err != nil {
		return nil, fmt.Errorf("no runner registered for executor %q: %w", executorID, err)
	}

	start := time.Now()

	cbResult, cbErr := c.cbManager.Execute(configID, func() (any, error) {
		result, execErr := runner.Execute(ctx, input)
		if execErr != nil {
			return nil, execErr
		}

		// Convert executor-level errors to Go errors so the circuit breaker
		// counts them as failures and can trip accordingly.
		if result.Status == executor.ExecutionStatusError {
			return result, fmt.Errorf("executor error: %s", result.Error)
		}

		return result, nil
	})

	callDuration := time.Since(start).Milliseconds()

	// Record call details regardless of success/failure
	endpointURL := ""
	if u, ok := input.Config["url"].(string); ok {
		endpointURL = u
	}

	method := ""
	if m, ok := input.Config["method"].(string); ok {
		method = m
	}

	statusCode := 0

	if cbResult != nil {
		if execResult, ok := cbResult.(executor.ExecutionResult); ok {
			if sc, ok := execResult.Data["status"].(int); ok {
				statusCode = sc
			}
		}
	}

	callDetails := model.NewExecutorCallDetails(
		configID,
		"default",
		method,
		endpointURL,
		statusCode,
		callDuration,
	)
	step.SetExecutorCallDetails(callDetails)

	if cbErr != nil {
		if errors.Is(cbErr, gobreaker.ErrOpenState) || errors.Is(cbErr, gobreaker.ErrTooManyRequests) {
			return nil, constant.ErrExecutionCircuitOpen
		}

		return nil, cbErr
	}

	if execResult, ok := cbResult.(executor.ExecutionResult); ok {
		return extractResponseBody(execResult.Data), nil
	}

	return nil, nil
}

// evaluateConditionalNode evaluates a conditional node and returns the next node ID.
func (c *ExecuteWorkflowCommand) evaluateConditionalNode(
	ctx context.Context,
	node model.WorkflowNode,
	execution *model.WorkflowExecution,
	wfCtx map[string]any,
	graph map[string][]model.WorkflowEdge,
	stepNumber int,
) (string, error) {
	nodeName := nodeDisplayName(node)
	step := model.NewExecutionStep(stepNumber, nodeName, node.ID(), wfCtx)

	expression := node.Condition()
	if expression == "" {
		_ = step.MarkFailed("conditional node missing condition expression")
		execution.AddStep(step)
		c.persistBestEffort(ctx, execution)

		if markErr := execution.MarkFailed(fmt.Sprintf("node %q: missing condition expression", node.ID())); markErr != nil {
			condLogger := libCommons.NewLoggerFromContext(ctx)
			condLogger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		return "", constant.ErrExecutionNodeFailed
	}

	result, err := c.condEvaluator.Evaluate(expression, wfCtx)
	if err != nil {
		_ = step.MarkFailed(fmt.Sprintf("condition evaluation failed: %v", err))
		execution.AddStep(step)
		c.persistBestEffort(ctx, execution)

		if markErr := execution.MarkFailed(fmt.Sprintf("node %q: condition evaluation failed: %v", node.ID(), err)); markErr != nil {
			condLogger := libCommons.NewLoggerFromContext(ctx)
			condLogger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed (already terminal)",
				libLog.Any("error.message", markErr.Error()))
		}

		return "", constant.ErrExecutionNodeFailed
	}

	// Determine which handle to follow
	handle := "false"
	if result {
		handle = "true"
	}

	_ = step.MarkCompleted(map[string]any{
		"condition":   expression,
		"result":      result,
		"branchTaken": handle,
	})
	execution.AddStep(step)

	if err := c.executionRepo.Update(ctx, execution, model.ExecutionStatusRunning); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			return "", constant.ErrConflictStateChanged
		}

		return "", fmt.Errorf("failed to persist step: %w", err)
	}

	// Find the edge matching the handle
	outEdges := graph[node.ID()]
	for _, edge := range outEdges {
		if edge.SourceHandle() != nil && *edge.SourceHandle() == handle {
			return edge.Target(), nil
		}
	}

	// No matching edge - end of this branch
	return "", nil
}

// executeActionNode executes an action node (e.g., set_output).
func (c *ExecuteWorkflowCommand) executeActionNode(
	node model.WorkflowNode,
	execution *model.WorkflowExecution,
	wfCtx map[string]any,
	stepNumber int,
) {
	nodeName := nodeDisplayName(node)
	step := model.NewExecutionStep(stepNumber, nodeName, node.ID(), nil)

	data := node.Data()

	// Handle set_output action
	if actionType, ok := data["actionType"].(string); ok && actionType == "set_output" {
		if outputConfig, ok := data["output"].(map[string]any); ok {
			wfCtx["output"] = outputConfig
		}
	}

	_ = step.MarkCompleted(data)
	execution.AddStep(step)
}

// applyInputTransformations transforms workflow context data to executor input.
func (c *ExecuteWorkflowCommand) applyInputTransformations(
	ctx context.Context,
	node model.WorkflowNode,
	wfCtx map[string]any,
) ([]byte, error) {
	// Use field mappings if available
	if mappings := node.InputMapping(); len(mappings) > 0 {
		inputJSON, err := json.Marshal(wfCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal workflow context: %w", err)
		}

		result, err := c.transformSvc.TransformWithMappings(ctx, inputJSON, mappings)
		if err != nil {
			return nil, fmt.Errorf("input mapping transformation failed: %w", err)
		}

		return result, nil
	}

	// Use Kazaam transforms if available
	if transforms := node.Transforms(); len(transforms) > 0 {
		inputJSON, err := json.Marshal(wfCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal workflow context: %w", err)
		}

		result, err := c.transformSvc.TransformWithOperations(ctx, inputJSON, transforms)
		if err != nil {
			return nil, fmt.Errorf("transform operations failed: %w", err)
		}

		return result, nil
	}

	// Default: marshal entire workflow context
	data, err := json.Marshal(wfCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default input: %w", err)
	}

	return data, nil
}

// applyOutputTransformations transforms executor response to workflow context format.
func (c *ExecuteWorkflowCommand) applyOutputTransformations(
	ctx context.Context,
	node model.WorkflowNode,
	result map[string]any,
) (map[string]any, error) {
	// Use output mappings if available
	if mappings := node.OutputMapping(); len(mappings) > 0 {
		resultJSON, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}

		transformed, err := c.transformSvc.TransformWithMappings(ctx, resultJSON, mappings)
		if err != nil {
			return nil, fmt.Errorf("output mapping transformation failed: %w", err)
		}

		var output map[string]any
		if err := json.Unmarshal(transformed, &output); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transformed output: %w", err)
		}

		return output, nil
	}

	// Default: return raw result
	return result, nil
}

// RecoverIncompleteExecutions resumes executions that were interrupted.
func (c *ExecuteWorkflowCommand) RecoverIncompleteExecutions(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.execution.recover")
	defer span.End()

	incomplete, err := c.executionRepo.FindIncomplete(ctx)
	if err != nil {
		libOtel.HandleSpanError(span, "failed to find incomplete executions", err)
		return fmt.Errorf("failed to find incomplete executions: %w", err)
	}

	if len(incomplete) == 0 {
		logger.Log(ctx, libLog.LevelInfo, "No incomplete executions to recover")
		return nil
	}

	logger.Log(ctx, libLog.LevelInfo, "Recovering incomplete executions", libLog.Any("operation", "command.execution.recover"), libLog.Any("count", len(incomplete)))

	for _, execution := range incomplete {
		workflow, err := c.workflowRepo.FindByID(ctx, execution.WorkflowID())
		if err != nil {
			logger.Log(ctx, libLog.LevelWarn, "Failed to load workflow for recovery, marking execution as failed", libLog.Any("execution.id", execution.ExecutionID()), libLog.Any("workflow.id", execution.WorkflowID()), libLog.Any("error.message", err.Error()))

			if markErr := execution.MarkFailed("workflow not found during recovery"); markErr != nil {
				logger.Log(ctx, libLog.LevelWarn, "Failed to mark execution as failed during recovery (already terminal)",
					libLog.Any("error.message", markErr.Error()))
			}

			c.persistBestEffort(ctx, execution)

			continue
		}

		logger.Log(ctx, libLog.LevelInfo, "Resuming execution", libLog.Any("execution.id", execution.ExecutionID()), libLog.Any("workflow.id", execution.WorkflowID()), libLog.Any("lastStep", execution.LastCompletedStepNumber()))

		go c.runStateMachine(ctx, execution, workflow)
	}

	return nil
}

// executeWithRetry executes a runner call with exponential backoff retry.
// Returns the result on success or the last error after all attempts are exhausted.
// checkIdempotencyFastPath checks whether an execution with the given
// idempotency key already exists. Returns the existing execution if found,
// nil if not found, or an error for unexpected failures.
func (c *ExecuteWorkflowCommand) checkIdempotencyFastPath(
	ctx context.Context,
	idempotencyKey *string,
) (*model.WorkflowExecution, error) {
	if idempotencyKey == nil {
		return nil, nil
	}

	key := strings.TrimSpace(*idempotencyKey)
	if key == "" {
		return nil, nil
	}

	existing, err := c.executionRepo.FindByIdempotencyKey(ctx, key)
	if err == nil {
		logger, _, _, _ := libCommons.NewTrackingFromContext(ctx)

		logger.Log(ctx, libLog.LevelInfo, "Returning existing execution for idempotency key", libLog.Any("operation", "command.execution.execute"), libLog.Any("execution.id", existing.ExecutionID()), libLog.Any("idempotencyKey", key))

		return existing, nil
	}

	if !errors.Is(err, constant.ErrExecutionNotFound) {
		return nil, fmt.Errorf("failed to check idempotency key: %w", err)
	}

	return nil, nil
}

// handleCreateError handles errors from executionRepo.Create. When the error
// is a duplicate key conflict, it attempts to resolve the conflict by fetching
// the existing execution (insert-first idempotency pattern).
func (c *ExecuteWorkflowCommand) handleCreateError(
	ctx context.Context,
	span trace.Span,
	err error,
	idempotencyKey *string,
) (*model.WorkflowExecution, error) {
	if errors.Is(err, constant.ErrExecutionDuplicate) {
		if idempotencyKey != nil {
			key := strings.TrimSpace(*idempotencyKey)

			if key != "" {
				existing, findErr := c.executionRepo.FindByIdempotencyKey(ctx, key)
				if findErr == nil {
					logger, _, _, _ := libCommons.NewTrackingFromContext(ctx)

					logger.Log(ctx, libLog.LevelInfo, "Returning existing execution (concurrent insert resolved)", libLog.Any("operation", "command.execution.execute"), libLog.Any("execution.id", existing.ExecutionID()), libLog.Any("idempotencyKey", key))

					return existing, nil
				}
			}
		}

		libOtel.HandleSpanBusinessErrorEvent(span, "duplicate idempotency key", err)

		return nil, constant.ErrExecutionDuplicate
	}

	libOtel.HandleSpanError(span, "failed to persist execution", err)

	return nil, fmt.Errorf("failed to persist execution: %w", err)
}

// persistBestEffort attempts to persist execution state and logs failures.
// Used in error paths where the execution is already being marked as failed
// and there is no meaningful way to propagate the persist error.
// Uses empty expectedStatus to skip atomic check (best-effort fallback).
func (c *ExecuteWorkflowCommand) persistBestEffort(ctx context.Context, execution *model.WorkflowExecution) {
	if err := c.executionRepo.Update(ctx, execution, ""); err != nil {
		logger, _, _, _ := libCommons.NewTrackingFromContext(ctx)

		logger.Log(ctx, libLog.LevelError, "Failed to persist execution state (best-effort)", libLog.Any("execution.id", execution.ExecutionID()), libLog.Any("execution.status", string(execution.Status())), libLog.Any("error.message", err.Error()))
	}
}

// Helper functions

// countExecutableNodes counts nodes excluding trigger nodes.
func countExecutableNodes(nodes []model.WorkflowNode) int {
	count := 0

	for _, n := range nodes {
		if n.Type() != model.NodeTypeTrigger {
			count++
		}
	}

	return count
}

// buildGraph creates an adjacency list from edges.
func buildGraph(edges []model.WorkflowEdge) map[string][]model.WorkflowEdge {
	graph := make(map[string][]model.WorkflowEdge)
	for _, edge := range edges {
		graph[edge.Source()] = append(graph[edge.Source()], edge)
	}

	return graph
}

// buildNodeMap creates a map of node ID -> node.
func buildNodeMap(nodes []model.WorkflowNode) map[string]model.WorkflowNode {
	m := make(map[string]model.WorkflowNode, len(nodes))
	for _, node := range nodes {
		m[node.ID()] = node
	}

	return m
}

// findTriggerNode returns the ID of the first trigger node.
func findTriggerNode(nodes []model.WorkflowNode) string {
	for _, node := range nodes {
		if node.Type() == model.NodeTypeTrigger {
			return node.ID()
		}
	}

	return ""
}

// nodeDisplayName returns a display name for a node.
func nodeDisplayName(node model.WorkflowNode) string {
	if node.Name() != nil && *node.Name() != "" {
		return *node.Name()
	}

	return fmt.Sprintf("%s_%s", node.Type(), node.ID())
}
