// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"
	"errors"
	"fmt"

	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/clock"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// UpdateWorkflowCommand handles workflow updates.
type UpdateWorkflowCommand struct {
	repo                    WorkflowRepository
	providerConfigRepo      ProviderConfigReadRepository
	catalog                 executor.Catalog
	transformationValidator model.TransformationValidator
	clock                   clock.Clock
	auditWriter             AuditWriter
}

// NewUpdateWorkflowCommand creates a new UpdateWorkflowCommand.
// Returns error if required dependencies are nil.
func NewUpdateWorkflowCommand(
	repo WorkflowRepository,
	providerConfigRepo ProviderConfigReadRepository,
	catalog executor.Catalog,
	transformationValidator model.TransformationValidator,
	clk clock.Clock,
	auditWriter AuditWriter,
) (*UpdateWorkflowCommand, error) {
	if repo == nil {
		return nil, ErrUpdateWorkflowNilRepo
	}

	if auditWriter == nil {
		return nil, ErrUpdateWorkflowNilAuditWriter
	}

	if clk == nil {
		clk = clock.New()
	}

	return &UpdateWorkflowCommand{
		repo:                    repo,
		providerConfigRepo:      providerConfigRepo,
		catalog:                 catalog,
		transformationValidator: transformationValidator,
		clock:                   clk,
		auditWriter:             auditWriter,
	}, nil
}

// Execute updates an existing workflow.
// Only draft workflows can be updated.
func (c *UpdateWorkflowCommand) Execute(ctx context.Context, id uuid.UUID, input *model.UpdateWorkflowInput) (*model.Workflow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if input == nil {
		return nil, ErrUpdateWorkflowNilInput
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.workflow.update")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Updating workflow", libLog.Any("operation", "command.workflow.update"), libLog.Any("workflow.id", id))

	workflow, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrWorkflowNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Workflow not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find workflow", err)

		return nil, fmt.Errorf("failed to find workflow: %w", err)
	}

	if !workflow.IsDraft() {
		libOtel.HandleSpanBusinessErrorEvent(span, "workflow cannot be modified", constant.ErrWorkflowCannotModify)
		return nil, constant.ErrWorkflowCannotModify
	}

	nodes := input.ToNodes()
	edges := input.ToEdges()

	// Node validations are only performed when nodes are present.
	// Draft workflows can be saved without nodes; full validation happens at activation.
	if len(nodes) > 0 {
		if err := model.ValidateNodesWithCatalog(nodes, c.catalog); err != nil {
			var notFoundErr pkg.EntityNotFoundError
			if errors.As(err, &notFoundErr) {
				libOtel.HandleSpanBusinessErrorEvent(span, "executor not found", err)
				return nil, constant.ErrWorkflowExecutorNotFound
			}

			libOtel.HandleSpanBusinessErrorEvent(span, "workflow validation failed", err)

			return nil, err
		}

		if err := validateProviderConfigs(ctx, nodes, c.providerConfigRepo, c.catalog); err != nil {
			libOtel.HandleSpanBusinessErrorEvent(span, "provider config validation failed", err)
			return nil, err
		}

		if err := model.ValidateNodeTransformations(nodes, c.transformationValidator); err != nil {
			libOtel.HandleSpanBusinessErrorEvent(span, "transformation validation failed", err)
			return nil, err
		}
	}

	previousStatus := workflow.Status()

	if err := workflow.Update(input.Name, input.Description, nodes, edges); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "failed to update workflow", err)
		return nil, err
	}

	// Apply metadata updates if provided
	if input.Metadata != nil {
		for k, v := range input.Metadata {
			workflow.SetMetadata(k, v)
		}
	}

	if err := c.repo.Update(ctx, workflow, previousStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "failed to persist workflow", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Workflow updated successfully", libLog.Any("operation", "command.workflow.update"), libLog.Any("workflow.id", workflow.ID()))

	c.auditWriter.RecordWorkflowEvent(ctx, model.AuditEventWorkflowUpdated, model.AuditActionUpdate, model.AuditResultSuccess, workflow.ID(), map[string]any{
		"workflow.name": workflow.Name(),
	})

	return workflow, nil
}
