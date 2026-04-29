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

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// UpdateProviderConfigCommand handles provider configuration updates.
type UpdateProviderConfigCommand struct {
	repo        ProviderConfigRepository
	catalog     executor.Catalog
	auditWriter AuditWriter
}

// NewUpdateProviderConfigCommand creates a new UpdateProviderConfigCommand.
// Returns error if required dependencies are nil.
func NewUpdateProviderConfigCommand(
	repo ProviderConfigRepository,
	catalog executor.Catalog,
	auditWriter AuditWriter,
) (*UpdateProviderConfigCommand, error) {
	if repo == nil {
		return nil, ErrUpdateProviderConfigNilRepo
	}

	if catalog == nil {
		return nil, ErrUpdateProviderConfigNilCatalog
	}

	if auditWriter == nil {
		return nil, ErrUpdateProviderConfigNilAuditWriter
	}

	return &UpdateProviderConfigCommand{
		repo:        repo,
		catalog:     catalog,
		auditWriter: auditWriter,
	}, nil
}

// Execute updates an existing provider configuration.
func (c *UpdateProviderConfigCommand) Execute(ctx context.Context, id uuid.UUID, input *model.UpdateProviderConfigurationInput) (*model.ProviderConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if input == nil {
		return nil, ErrUpdateProviderConfigNilInput
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.provider_config.update")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Updating provider configuration", libLog.Any("operation", "command.provider_config.update"), libLog.Any("provider_config.id", id))

	providerConfig, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrProviderConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Provider configuration not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find provider configuration", err)

		return nil, fmt.Errorf("failed to find provider configuration: %w", err)
	}

	// If config is being changed, re-validate against provider JSON Schema
	if input.Config != nil {
		provider, err := c.catalog.GetProvider(executor.ProviderID(providerConfig.ProviderID()))
		if err != nil {
			libOtel.HandleSpanBusinessErrorEvent(span, "provider not found in catalog", err)
			return nil, constant.ErrProviderNotFoundInCatalog
		}

		if err := validateConfigAgainstSchema(input.Config, provider.ConfigSchema()); err != nil {
			if errors.Is(err, ErrInvalidProviderSchema) {
				libOtel.HandleSpanError(span, "provider schema is malformed", err)
				return nil, fmt.Errorf("provider schema is malformed: %w", err)
			}

			libOtel.HandleSpanBusinessErrorEvent(span, "config validation failed against provider schema", err)

			return nil, pkg.ValidationError{
				Code:    constant.ErrProviderConfigInvalidSchema.Error(),
				Message: fmt.Sprintf("config validation failed against provider schema: %s", err),
			}
		}
	}

	previousStatus := providerConfig.Status()

	if err := providerConfig.Update(input.Name, input.Description, input.Config); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "failed to update provider configuration", err)
		return nil, err
	}

	// Apply metadata updates if provided
	if input.Metadata != nil {
		for k, v := range input.Metadata {
			providerConfig.SetMetadata(k, v)
		}
	}

	if err := c.repo.Update(ctx, providerConfig, previousStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "failed to persist provider configuration", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Provider configuration updated successfully", libLog.Any("operation", "command.provider_config.update"), libLog.Any("provider_config.id", providerConfig.ID()))

	c.auditWriter.RecordProviderConfigEvent(ctx, model.AuditEventProviderConfigUpdated, model.AuditActionUpdate, model.AuditResultSuccess, providerConfig.ID(), map[string]any{
		"provider_config.name": providerConfig.Name(),
	})

	return providerConfig, nil
}
