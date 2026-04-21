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
	"github.com/google/uuid"
)

// CreateWorkflowCommand handles workflow creation.
type CreateWorkflowCommand struct {
	repo                    WorkflowRepository
	providerConfigRepo      ProviderConfigReadRepository
	catalog                 executor.Catalog
	transformationValidator model.TransformationValidator
	clock                   clock.Clock
	auditWriter             AuditWriter
}

// NewCreateWorkflowCommand creates a new CreateWorkflowCommand.
// Returns error if required dependencies are nil.
func NewCreateWorkflowCommand(
	repo WorkflowRepository,
	providerConfigRepo ProviderConfigReadRepository,
	catalog executor.Catalog,
	transformationValidator model.TransformationValidator,
	clk clock.Clock,
	auditWriter AuditWriter,
) (*CreateWorkflowCommand, error) {
	if repo == nil {
		return nil, ErrCreateWorkflowNilRepo
	}

	if auditWriter == nil {
		return nil, ErrCreateWorkflowNilAuditWriter
	}

	if clk == nil {
		clk = clock.New()
	}

	return &CreateWorkflowCommand{
		repo:                    repo,
		providerConfigRepo:      providerConfigRepo,
		catalog:                 catalog,
		transformationValidator: transformationValidator,
		clock:                   clk,
		auditWriter:             auditWriter,
	}, nil
}

// Execute creates a new workflow.
func (c *CreateWorkflowCommand) Execute(ctx context.Context, input *model.CreateWorkflowInput) (*model.Workflow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if input == nil {
		return nil, ErrCreateWorkflowNilInput
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.workflow.create")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Creating workflow", libLog.Any("operation", "command.workflow.create"), libLog.Any("workflow.name", input.Name))

	workflow, err := input.ToDomain()
	if err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "failed to create workflow domain", err)
		return nil, err
	}

	// Node validations are only performed when nodes are present.
	// Draft workflows can be saved without nodes; full validation happens at activation.
	if len(workflow.Nodes()) > 0 {
		if err := model.ValidateNodesWithCatalog(workflow.Nodes(), c.catalog); err != nil {
			var notFoundErr pkg.EntityNotFoundError
			if errors.As(err, &notFoundErr) {
				libOtel.HandleSpanBusinessErrorEvent(span, "executor not found", err)
				return nil, constant.ErrWorkflowExecutorNotFound
			}

			libOtel.HandleSpanBusinessErrorEvent(span, "workflow validation failed", err)

			return nil, err
		}

		if err := validateProviderConfigs(ctx, workflow.Nodes(), c.providerConfigRepo, c.catalog); err != nil {
			libOtel.HandleSpanBusinessErrorEvent(span, "provider config validation failed", err)
			return nil, err
		}

		if err := model.ValidateNodeTransformations(workflow.Nodes(), c.transformationValidator); err != nil {
			libOtel.HandleSpanBusinessErrorEvent(span, "transformation validation failed", err)
			return nil, err
		}
	}

	if err := c.repo.Create(ctx, workflow); err != nil {
		libOtel.HandleSpanError(span, "failed to persist workflow", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Workflow created successfully", libLog.Any("operation", "command.workflow.create"), libLog.Any("workflow.id", workflow.ID()))

	c.auditWriter.RecordWorkflowEvent(ctx, model.AuditEventWorkflowCreated, model.AuditActionCreate, model.AuditResultSuccess, workflow.ID(), map[string]any{
		"workflow.name": workflow.Name(),
	})

	return workflow, nil
}

// validateProviderConfigs validates that all providerConfigIds in executor nodes
// reference existing, active provider configurations, and that the provider matches the executor.
func validateProviderConfigs(
	ctx context.Context,
	nodes []model.WorkflowNode,
	repo ProviderConfigReadRepository,
	catalog executor.Catalog,
) error {
	if repo == nil || catalog == nil {
		return nil
	}

	for _, node := range nodes {
		if node.Type() != model.NodeTypeExecutor {
			continue
		}

		providerConfigID := node.ProviderConfigID()
		if providerConfigID == "" {
			continue // Already validated by ValidateNodesWithCatalog
		}

		configUUID, err := uuid.Parse(providerConfigID)
		if err != nil {
			continue // Already validated by ValidateNodesWithCatalog
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
				Message: fmt.Sprintf("node %s: provider configuration %s is not active", node.ID(), providerConfigID),
			}
		}

		// Cross-validate: provider config's providerID must match the executor's provider
		executorID := executor.ID(node.ExecutorID())

		e, err := catalog.GetExecutor(executorID)
		if err != nil {
			continue // Already validated by ValidateNodesWithCatalog
		}

		executorProviderID := e.ProviderID()
		configProviderID := providerConfig.ProviderID()

		if executorProviderID != "" && configProviderID != string(executorProviderID) {
			return pkg.ValidationError{
				Code: constant.ErrWorkflowProviderConfigMismatch.Error(),
				Message: fmt.Sprintf(
					"node %s: provider configuration %s belongs to provider %q but executor %s belongs to provider %q",
					node.ID(), providerConfigID, configProviderID, executorID, executorProviderID,
				),
			}
		}
	}

	return nil
}
