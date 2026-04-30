// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"

	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"

	"github.com/LerianStudio/flowker/pkg/clock"
	"github.com/LerianStudio/flowker/pkg/model"
)

// CreateExecutorConfigCommand handles executor configuration creation.
type CreateExecutorConfigCommand struct {
	repo  ExecutorConfigRepository
	clock clock.Clock
}

// NewCreateExecutorConfigCommand creates a new CreateExecutorConfigCommand.
// Returns error if required dependencies are nil.
func NewCreateExecutorConfigCommand(
	repo ExecutorConfigRepository,
	clk clock.Clock,
) (*CreateExecutorConfigCommand, error) {
	if repo == nil {
		return nil, ErrCreateExecutorConfigNilRepo
	}

	if clk == nil {
		clk = clock.New()
	}

	return &CreateExecutorConfigCommand{
		repo:  repo,
		clock: clk,
	}, nil
}

// Execute creates a new executor configuration.
func (c *CreateExecutorConfigCommand) Execute(ctx context.Context, input *model.CreateExecutorConfigurationInput) (*model.ExecutorConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if input == nil {
		return nil, ErrCreateExecutorConfigNilInput
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.executor_config.create")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Creating executor configuration", libLog.Any("operation", "command.executor_config.create"), libLog.Any("executor_config.name", input.Name))

	executorConfig, err := input.ToDomain()
	if err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "failed to create executor configuration domain", err)
		return nil, err
	}

	if err := c.repo.Create(ctx, executorConfig); err != nil {
		libOtel.HandleSpanError(span, "failed to persist executor configuration", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Executor configuration created successfully", libLog.Any("operation", "command.executor_config.create"), libLog.Any("executor_config.id", executorConfig.ID()))

	return executorConfig, nil
}
