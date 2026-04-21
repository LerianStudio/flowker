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
	"github.com/google/uuid"
)

// DeleteExecutorConfigCommand handles executor configuration deletion.
type DeleteExecutorConfigCommand struct {
	repo  ExecutorConfigRepository
	clock clock.Clock
}

// NewDeleteExecutorConfigCommand creates a new DeleteExecutorConfigCommand.
// Returns error if required dependencies are nil.
func NewDeleteExecutorConfigCommand(
	repo ExecutorConfigRepository,
	clk clock.Clock,
) (*DeleteExecutorConfigCommand, error) {
	if repo == nil {
		return nil, ErrDeleteExecutorConfigNilRepo
	}

	if clk == nil {
		clk = clock.New()
	}

	return &DeleteExecutorConfigCommand{
		repo:  repo,
		clock: clk,
	}, nil
}

// Execute removes an executor configuration.
// Only unconfigured, configured, or disabled executor configurations can be deleted.
func (c *DeleteExecutorConfigCommand) Execute(ctx context.Context, id uuid.UUID) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.executor_config.delete")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Deleting executor configuration", libLog.Any("operation", "command.executor_config.delete"), libLog.Any("executor_config.id", id))

	executorConfig, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrExecutorConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Executor configuration not found", err)
			return err
		}

		libOtel.HandleSpanError(span, "Failed to find executor configuration", err)

		return fmt.Errorf("failed to find executor configuration: %w", err)
	}

	// Active or tested executor configurations cannot be deleted
	if executorConfig.IsActive() || executorConfig.IsTested() {
		libOtel.HandleSpanBusinessErrorEvent(span, "executor configuration cannot be deleted", constant.ErrExecutorConfigCannotModify)
		return constant.ErrExecutorConfigCannotModify
	}

	previousStatus := executorConfig.Status()

	if err := c.repo.Delete(ctx, id, previousStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return err
		}

		libOtel.HandleSpanError(span, "failed to delete executor configuration", err)
		return err
	}

	logger.Log(ctx, libLog.LevelInfo, "Executor configuration deleted successfully", libLog.Any("operation", "command.executor_config.delete"), libLog.Any("executor_config.id", id))

	return nil
}
