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

	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// EnableProviderConfigCommand handles enabling a disabled provider configuration.
type EnableProviderConfigCommand struct {
	repo ProviderConfigRepository
}

// NewEnableProviderConfigCommand creates a new EnableProviderConfigCommand.
// Returns error if required dependencies are nil.
func NewEnableProviderConfigCommand(
	repo ProviderConfigRepository,
) (*EnableProviderConfigCommand, error) {
	if repo == nil {
		return nil, ErrEnableProviderConfigNilRepo
	}

	return &EnableProviderConfigCommand{
		repo: repo,
	}, nil
}

// Execute enables a disabled provider configuration.
func (c *EnableProviderConfigCommand) Execute(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.provider_config.enable")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Enabling provider configuration", libLog.Any("operation", "command.provider_config.enable"), libLog.Any("provider_config.id", id))

	providerConfig, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrProviderConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Provider configuration not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find provider configuration", err)

		return nil, fmt.Errorf("failed to find provider configuration: %w", err)
	}

	previousStatus := providerConfig.Status()

	if err := providerConfig.Enable(); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "failed to enable provider configuration", err)
		return nil, err
	}

	if err := c.repo.Update(ctx, providerConfig, previousStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "failed to persist provider configuration", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Provider configuration enabled successfully", libLog.Any("operation", "command.provider_config.enable"), libLog.Any("provider_config.id", providerConfig.ID()))

	return providerConfig, nil
}
