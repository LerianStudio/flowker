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
	"github.com/google/uuid"
)

// CloneWorkflowCommand handles workflow cloning.
type CloneWorkflowCommand struct {
	repo  WorkflowRepository
	clock clock.Clock
}

// NewCloneWorkflowCommand creates a new CloneWorkflowCommand.
// Returns error if required dependencies are nil.
func NewCloneWorkflowCommand(
	repo WorkflowRepository,
	clk clock.Clock,
) (*CloneWorkflowCommand, error) {
	if repo == nil {
		return nil, ErrCloneWorkflowNilRepo
	}

	if clk == nil {
		clk = clock.New()
	}

	return &CloneWorkflowCommand{
		repo:  repo,
		clock: clk,
	}, nil
}

// Execute creates a copy of an existing workflow with a new name.
// The cloned workflow is always in draft status.
func (c *CloneWorkflowCommand) Execute(ctx context.Context, id uuid.UUID, input *model.CloneWorkflowInput) (*model.Workflow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if input == nil {
		return nil, ErrCloneWorkflowNilInput
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.workflow.clone")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Cloning workflow", libLog.Any("operation", "command.workflow.clone"), libLog.Any("workflow.id", id), libLog.Any("workflow.name", input.Name))

	original, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrWorkflowNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Workflow not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find original workflow", err)

		return nil, fmt.Errorf("failed to find original workflow: %w", err)
	}

	cloned, err := original.Clone(input.Name)
	if err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "failed to clone workflow", err)
		return nil, err
	}

	if err := c.repo.Create(ctx, cloned); err != nil {
		libOtel.HandleSpanError(span, "failed to persist cloned workflow", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Workflow cloned successfully", libLog.Any("operation", "command.workflow.clone"), libLog.Any("workflow.source.id", id), libLog.Any("workflow.id", cloned.ID()))

	return cloned, nil
}
