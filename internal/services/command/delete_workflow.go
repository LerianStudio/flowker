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

	"github.com/LerianStudio/flowker/pkg/clock"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/webhook"
	"github.com/google/uuid"
)

// DeleteWorkflowCommand handles workflow deletion.
type DeleteWorkflowCommand struct {
	repo            WorkflowRepository
	clock           clock.Clock
	auditWriter     AuditWriter
	webhookRegistry *webhook.Registry
}

// NewDeleteWorkflowCommand creates a new DeleteWorkflowCommand.
// Returns error if required dependencies are nil.
// webhookRegistry is optional; when non-nil, webhook routes are unregistered
// on deletion (defensive cleanup for inactive workflows).
func NewDeleteWorkflowCommand(
	repo WorkflowRepository,
	clk clock.Clock,
	auditWriter AuditWriter,
	webhookRegistry ...*webhook.Registry,
) (*DeleteWorkflowCommand, error) {
	if repo == nil {
		return nil, ErrDeleteWorkflowNilRepo
	}

	if auditWriter == nil {
		return nil, ErrDeleteWorkflowNilAuditWriter
	}

	if clk == nil {
		clk = clock.New()
	}

	var registry *webhook.Registry
	if len(webhookRegistry) > 0 {
		registry = webhookRegistry[0]
	}

	return &DeleteWorkflowCommand{
		repo:            repo,
		clock:           clk,
		auditWriter:     auditWriter,
		webhookRegistry: registry,
	}, nil
}

// Execute removes a workflow.
// Only draft and inactive workflows can be deleted.
func (c *DeleteWorkflowCommand) Execute(ctx context.Context, id uuid.UUID) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.workflow.delete")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Deleting workflow", libLog.Any("operation", "command.workflow.delete"), libLog.Any("workflow.id", id))

	workflow, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrWorkflowNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Workflow not found", err)
			return err
		}

		libOtel.HandleSpanError(span, "Failed to find workflow", err)

		return fmt.Errorf("failed to find workflow: %w", err)
	}

	// Active workflows cannot be deleted
	if workflow.IsActive() {
		libOtel.HandleSpanBusinessErrorEvent(span, "workflow cannot be deleted", constant.ErrWorkflowCannotModify)
		return constant.ErrWorkflowCannotModify
	}

	previousStatus := workflow.Status()

	if err := c.repo.Delete(ctx, id, previousStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return err
		}

		libOtel.HandleSpanError(span, "failed to delete workflow", err)
		return err
	}

	// Defensive cleanup: unregister any webhook routes for this workflow.
	// In normal flow, deactivation already removes routes, but this guards
	// against edge cases.
	if c.webhookRegistry != nil {
		c.webhookRegistry.Unregister(id)
	}

	logger.Log(ctx, libLog.LevelInfo, "Workflow deleted successfully", libLog.Any("operation", "command.workflow.delete"), libLog.Any("workflow.id", id))

	c.auditWriter.RecordWorkflowEvent(ctx, model.AuditEventWorkflowDeleted, model.AuditActionDelete, model.AuditResultSuccess, id, map[string]any{
		"workflow.id": id.String(),
	})

	return nil
}
