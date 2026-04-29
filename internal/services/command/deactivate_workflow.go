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

	"github.com/LerianStudio/flowker/pkg/clock"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/webhook"
	"github.com/google/uuid"
)

// DeactivateWorkflowCommand handles workflow deactivation.
type DeactivateWorkflowCommand struct {
	repo            WorkflowRepository
	clock           clock.Clock
	auditWriter     AuditWriter
	webhookRegistry *webhook.Registry
}

// NewDeactivateWorkflowCommand creates a new DeactivateWorkflowCommand.
// Returns error if required dependencies are nil.
// webhookRegistry is optional; when non-nil, webhook routes are unregistered
// on deactivation.
func NewDeactivateWorkflowCommand(
	repo WorkflowRepository,
	clk clock.Clock,
	auditWriter AuditWriter,
	webhookRegistry ...*webhook.Registry,
) (*DeactivateWorkflowCommand, error) {
	if repo == nil {
		return nil, ErrDeactivateWorkflowNilRepo
	}

	if auditWriter == nil {
		return nil, ErrDeactivateWorkflowNilAuditWriter
	}

	if clk == nil {
		clk = clock.New()
	}

	var registry *webhook.Registry
	if len(webhookRegistry) > 0 {
		registry = webhookRegistry[0]
	}

	return &DeactivateWorkflowCommand{
		repo:            repo,
		clock:           clk,
		auditWriter:     auditWriter,
		webhookRegistry: registry,
	}, nil
}

// Execute transitions a workflow from active to inactive status.
func (c *DeactivateWorkflowCommand) Execute(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.workflow.deactivate")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Deactivating workflow", libLog.Any("operation", "command.workflow.deactivate"), libLog.Any("workflow.id", id))

	workflow, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrWorkflowNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Workflow not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find workflow", err)

		return nil, fmt.Errorf("failed to find workflow: %w", err)
	}

	previousStatus := workflow.Status()

	if err := workflow.Deactivate(); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "invalid status transition", err)
		return nil, constant.ErrWorkflowInvalidStatus
	}

	if err := c.repo.Update(ctx, workflow, previousStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "failed to persist workflow", err)
		return nil, err
	}

	// Unregister webhook routes after successful deactivation
	if c.webhookRegistry != nil {
		c.webhookRegistry.Unregister(id)
		logger.Log(ctx, libLog.LevelInfo, "Unregistered webhook routes for workflow",
			libLog.Any("operation", "command.workflow.deactivate"),
			libLog.Any("workflow.id", id))
	}

	logger.Log(ctx, libLog.LevelInfo, "Workflow deactivated successfully", libLog.Any("operation", "command.workflow.deactivate"), libLog.Any("workflow.id", workflow.ID()))

	c.auditWriter.RecordWorkflowEvent(ctx, model.AuditEventWorkflowDeactivated, model.AuditActionDeactivate, model.AuditResultSuccess, workflow.ID(), map[string]any{
		"workflow.name": workflow.Name(),
	})

	return workflow, nil
}
