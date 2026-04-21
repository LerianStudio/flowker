// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"
	"errors"
	"fmt"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/clock"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/webhook"
	"github.com/google/uuid"
)

// ActivateWorkflowCommand handles workflow activation.
type ActivateWorkflowCommand struct {
	repo                    WorkflowRepository
	providerConfigRepo      ProviderConfigReadRepository
	catalog                 executor.Catalog
	transformationValidator model.TransformationValidator
	clock                   clock.Clock
	auditWriter             AuditWriter
	webhookRegistry         *webhook.Registry
}

// NewActivateWorkflowCommand creates a new ActivateWorkflowCommand.
// Returns error if required dependencies are nil.
// webhookRegistry is optional; when non-nil, webhook trigger routes are
// registered/validated on activation.
func NewActivateWorkflowCommand(
	repo WorkflowRepository,
	providerConfigRepo ProviderConfigReadRepository,
	catalog executor.Catalog,
	transformationValidator model.TransformationValidator,
	clk clock.Clock,
	auditWriter AuditWriter,
	webhookRegistry ...*webhook.Registry,
) (*ActivateWorkflowCommand, error) {
	if repo == nil {
		return nil, ErrActivateWorkflowNilRepo
	}

	if auditWriter == nil {
		return nil, ErrActivateWorkflowNilAuditWriter
	}

	if clk == nil {
		clk = clock.New()
	}

	var registry *webhook.Registry
	if len(webhookRegistry) > 0 {
		registry = webhookRegistry[0]
	}

	return &ActivateWorkflowCommand{
		repo:                    repo,
		providerConfigRepo:      providerConfigRepo,
		catalog:                 catalog,
		transformationValidator: transformationValidator,
		clock:                   clk,
		auditWriter:             auditWriter,
		webhookRegistry:         registry,
	}, nil
}

// Execute transitions a workflow from draft to active status.
// Validates that all referenced provider configurations exist and are active.
func (c *ActivateWorkflowCommand) Execute(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.workflow.activate")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Activating workflow", libLog.Any("operation", "command.workflow.activate"), libLog.Any("workflow.id", id))

	workflow, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrWorkflowNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Workflow not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find workflow", err)

		return nil, fmt.Errorf("failed to find workflow: %w", err)
	}

	// Full structural validation required for activation
	if err := model.ValidateWorkflowStructure(workflow.Name(), workflow.Nodes(), workflow.Edges()); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "Workflow structure validation failed", err)
		return nil, err
	}

	if err := model.ValidateNodesWithCatalog(workflow.Nodes(), c.catalog); err != nil {
		var notFoundErr pkg.EntityNotFoundError
		if errors.As(err, &notFoundErr) {
			libOtel.HandleSpanBusinessErrorEvent(span, "executor not found", err)
			return nil, constant.ErrWorkflowExecutorNotFound
		}

		libOtel.HandleSpanBusinessErrorEvent(span, "Catalog validation failed", err)

		return nil, err
	}

	if err := model.ValidateNodeTransformations(workflow.Nodes(), c.transformationValidator); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "Transformation validation failed", err)
		return nil, err
	}

	// Validate provider configs are still active before activation
	if err := validateProviderConfigsForActivation(ctx, workflow.Nodes(), c.providerConfigRepo); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "provider config validation failed at activation", err)
		return nil, err
	}

	previousStatus := workflow.Status()

	if err := workflow.Activate(); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "invalid status transition", err)
		return nil, constant.ErrWorkflowInvalidStatus
	}

	// Register webhook routes before persisting. If the webhook path is already
	// taken by another active workflow, fail the activation with a clear error.
	if c.webhookRegistry != nil {
		if regErr := c.registerWebhookRoutes(ctx, id, workflow.Nodes()); regErr != nil {
			// Revert the in-memory status change since we did not persist yet.
			libOtel.HandleSpanBusinessErrorEvent(span, "webhook registration failed", regErr)

			return nil, pkg.ValidationError{
				Code:    constant.ErrWebhookPathAlreadyRegistered.Error(),
				Message: regErr.Error(),
			}
		}
	}

	if err := c.repo.Update(ctx, workflow, previousStatus); err != nil {
		// Roll back webhook registration on persistence failure.
		if c.webhookRegistry != nil {
			c.webhookRegistry.Unregister(id)
		}

		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "failed to persist workflow", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Workflow activated successfully", libLog.Any("operation", "command.workflow.activate"), libLog.Any("workflow.id", workflow.ID()))

	c.auditWriter.RecordWorkflowEvent(ctx, model.AuditEventWorkflowActivated, model.AuditActionActivate, model.AuditResultSuccess, workflow.ID(), map[string]any{
		"workflow.name": workflow.Name(),
	})

	return workflow, nil
}

// registerWebhookRoutes inspects workflow nodes for webhook triggers and
// registers them in the webhook registry. Returns an error if any path
// is already registered by another workflow.
func (c *ActivateWorkflowCommand) registerWebhookRoutes(
	ctx context.Context,
	workflowID uuid.UUID,
	nodes []model.WorkflowNode,
) error {
	logger := libCommons.NewLoggerFromContext(ctx)

	for _, node := range nodes {
		if node.Type() != model.NodeTypeTrigger {
			continue
		}

		if node.TriggerType() != "webhook" {
			continue
		}

		data := node.Data()

		path, _ := data["path"].(string)
		method, _ := data["method"].(string)
		verifyToken, _ := data["verify_token"].(string)

		if path == "" || method == "" {
			continue
		}

		route := webhook.Route{
			WorkflowID:  workflowID,
			Path:        path,
			Method:      method,
			VerifyToken: verifyToken,
		}

		if err := c.webhookRegistry.Register(route); err != nil {
			// Unregister any routes we just registered for this workflow
			// so we leave no partial state.
			c.webhookRegistry.Unregister(workflowID)
			return err
		}

		logger.Log(ctx, libLog.LevelInfo, "Registered webhook route",
			libLog.Any("method", method),
			libLog.Any("path", path),
			libLog.Any("workflow.id", workflowID))
	}

	return nil
}

// validateProviderConfigsForActivation verifies that all provider configs
// referenced by executor nodes exist and are active at activation time.
func validateProviderConfigsForActivation(
	ctx context.Context,
	nodes []model.WorkflowNode,
	repo ProviderConfigReadRepository,
) error {
	if repo == nil {
		return nil
	}

	for _, node := range nodes {
		if node.Type() != model.NodeTypeExecutor {
			continue
		}

		providerConfigID := node.ProviderConfigID()
		if providerConfigID == "" {
			continue
		}

		configUUID, err := uuid.Parse(providerConfigID)
		if err != nil {
			return pkg.ValidationError{
				Code:    constant.ErrWorkflowInvalidProviderConfig.Error(),
				Message: fmt.Sprintf("node %s: invalid providerConfigId: %s", node.ID(), providerConfigID),
			}
		}

		providerConfig, err := repo.FindByID(ctx, configUUID)
		if err != nil {
			return pkg.ValidationError{
				Code:    constant.ErrWorkflowInvalidProviderConfig.Error(),
				Message: fmt.Sprintf("node %s: provider configuration %s not found", node.ID(), providerConfigID),
			}
		}

		if !providerConfig.IsActive() {
			return pkg.ValidationError{
				Code:    constant.ErrWorkflowInvalidProviderConfig.Error(),
				Message: fmt.Sprintf("node %s: provider configuration %s is not active (status: %s)", node.ID(), providerConfigID, providerConfig.Status()),
			}
		}
	}

	return nil
}
