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

// DisableExecutorConfigCommand handles executor configuration disabling.
type DisableExecutorConfigCommand struct {
	repo  ExecutorConfigRepository
	clock clock.Clock
}

// NewDisableExecutorConfigCommand creates a new DisableExecutorConfigCommand.
// Returns error if required dependencies are nil.
func NewDisableExecutorConfigCommand(
	repo ExecutorConfigRepository,
	clk clock.Clock,
) (*DisableExecutorConfigCommand, error) {
	if repo == nil {
		return nil, ErrDisableExecutorConfigNilRepo
	}

	if clk == nil {
		clk = clock.New()
	}

	return &DisableExecutorConfigCommand{
		repo:  repo,
		clock: clk,
	}, nil
}

// Execute transitions an executor configuration from active to disabled status.
func (c *DisableExecutorConfigCommand) Execute(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.executor_config.disable")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Disabling executor configuration", libLog.Any("operation", "command.executor_config.disable"), libLog.Any("executor_config.id", id))

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

	if err := executorConfig.Disable(); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "invalid status transition", err)
		return nil, err
	}

	if err := c.repo.Update(ctx, executorConfig, previousStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "failed to persist executor configuration", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Executor configuration disabled successfully", libLog.Any("operation", "command.executor_config.disable"), libLog.Any("executor_config.id", executorConfig.ID()))

	return executorConfig, nil
}
