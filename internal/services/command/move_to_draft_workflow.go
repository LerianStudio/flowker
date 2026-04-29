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
	"github.com/google/uuid"
)

// MoveToDraftWorkflowCommand handles workflow transition from inactive to draft.
type MoveToDraftWorkflowCommand struct {
	repo        WorkflowRepository
	clock       clock.Clock
	auditWriter AuditWriter
}

// NewMoveToDraftWorkflowCommand creates a new MoveToDraftWorkflowCommand.
// Returns error if required dependencies are nil.
func NewMoveToDraftWorkflowCommand(
	repo WorkflowRepository,
	clk clock.Clock,
	auditWriter AuditWriter,
) (*MoveToDraftWorkflowCommand, error) {
	if repo == nil {
		return nil, ErrMoveToDraftWorkflowNilRepo
	}

	if auditWriter == nil {
		return nil, ErrMoveToDraftWorkflowNilAuditWriter
	}

	if clk == nil {
		clk = clock.New()
	}

	return &MoveToDraftWorkflowCommand{
		repo:        repo,
		clock:       clk,
		auditWriter: auditWriter,
	}, nil
}

// Execute transitions a workflow from inactive to draft status.
func (c *MoveToDraftWorkflowCommand) Execute(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.workflow.move_to_draft")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Moving workflow to draft", libLog.Any("operation", "command.workflow.move_to_draft"), libLog.Any("workflow.id", id))

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

	if err := workflow.MoveToDraft(); err != nil {
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

	logger.Log(ctx, libLog.LevelInfo, "Workflow moved to draft successfully", libLog.Any("operation", "command.workflow.move_to_draft"), libLog.Any("workflow.id", workflow.ID()))

	c.auditWriter.RecordWorkflowEvent(ctx, model.AuditEventWorkflowDrafted, model.AuditActionDraft, model.AuditResultSuccess, workflow.ID(), map[string]any{
		"workflow.name": workflow.Name(),
	})

	return workflow, nil
}
