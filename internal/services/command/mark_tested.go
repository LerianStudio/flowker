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

// MarkTestedCommand handles executor configuration status transition to tested.
type MarkTestedCommand struct {
	repo  ExecutorConfigRepository
	clock clock.Clock
}

// NewMarkTestedCommand creates a new MarkTestedCommand.
// Returns error if required dependencies are nil.
func NewMarkTestedCommand(
	repo ExecutorConfigRepository,
	clk clock.Clock,
) (*MarkTestedCommand, error) {
	if repo == nil {
		return nil, ErrMarkTestedNilRepo
	}

	if clk == nil {
		clk = clock.New()
	}

	return &MarkTestedCommand{
		repo:  repo,
		clock: clk,
	}, nil
}

// Execute transitions an executor from configured to tested status.
func (c *MarkTestedCommand) Execute(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.executor_config.mark_tested")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Marking executor configuration as tested", libLog.Any("operation", "command.executor_config.mark_tested"), libLog.Any("executor_config.id", id))

	executorConfig, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrExecutorConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Executor configuration not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find executor configuration", err)

		return nil, fmt.Errorf("failed to find executor configuration: %w", err)
	}

	previousStatus := executorConfig.Status()

	if err := executorConfig.MarkTested(); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "invalid status transition", err)
		return nil, constant.ErrExecutorConfigCannotModify
	}

	if err := c.repo.Update(ctx, executorConfig, previousStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "failed to persist executor configuration", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Executor configuration marked as tested successfully", libLog.Any("operation", "command.executor_config.mark_tested"), libLog.Any("executor_config.id", executorConfig.ID()))

	return executorConfig, nil
}
