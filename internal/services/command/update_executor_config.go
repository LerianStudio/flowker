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

// UpdateExecutorConfigCommand handles executor configuration updates.
type UpdateExecutorConfigCommand struct {
	repo  ExecutorConfigRepository
	clock clock.Clock
}

// NewUpdateExecutorConfigCommand creates a new UpdateExecutorConfigCommand.
// Returns error if required dependencies are nil.
func NewUpdateExecutorConfigCommand(
	repo ExecutorConfigRepository,
	clk clock.Clock,
) (*UpdateExecutorConfigCommand, error) {
	if repo == nil {
		return nil, ErrUpdateExecutorConfigNilRepo
	}

	if clk == nil {
		clk = clock.New()
	}

	return &UpdateExecutorConfigCommand{
		repo:  repo,
		clock: clk,
	}, nil
}

// Execute updates an existing executor configuration.
// Only unconfigured or configured executor configurations can be updated.
func (c *UpdateExecutorConfigCommand) Execute(ctx context.Context, id uuid.UUID, input *model.UpdateExecutorConfigurationInput) (*model.ExecutorConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if input == nil {
		return nil, ErrUpdateExecutorConfigNilInput
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.executor_config.update")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Updating executor configuration", libLog.Any("operation", "command.executor_config.update"), libLog.Any("executor_config.id", id))

	executorConfig, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrExecutorConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Executor configuration not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find executor configuration", err)

		return nil, fmt.Errorf("failed to find executor configuration: %w", err)
	}

	endpoints, err := input.ToEndpoints()
	if err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "failed to convert endpoints", err)
		return nil, err
	}

	auth, err := input.ToAuthentication()
	if err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "failed to convert authentication", err)
		return nil, err
	}

	previousStatus := executorConfig.Status()

	if err := executorConfig.Update(input.Name, input.Description, input.BaseURL, endpoints, *auth); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "failed to update executor configuration", err)
		return nil, err
	}

	// Apply metadata updates if provided
	if input.Metadata != nil {
		for k, v := range input.Metadata {
			executorConfig.SetMetadata(k, v)
		}
	}

	if err := c.repo.Update(ctx, executorConfig, previousStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "failed to persist executor configuration", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Executor configuration updated successfully", libLog.Any("operation", "command.executor_config.update"), libLog.Any("executor_config.id", executorConfig.ID()))

	return executorConfig, nil
}
