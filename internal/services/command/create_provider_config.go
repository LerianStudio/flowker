// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"
	"errors"
	"fmt"
	"strings"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// CreateProviderConfigCommand handles provider configuration creation.
type CreateProviderConfigCommand struct {
	repo        ProviderConfigRepository
	catalog     executor.Catalog
	auditWriter AuditWriter
}

// NewCreateProviderConfigCommand creates a new CreateProviderConfigCommand.
// Returns error if required dependencies are nil.
func NewCreateProviderConfigCommand(
	repo ProviderConfigRepository,
	catalog executor.Catalog,
	auditWriter AuditWriter,
) (*CreateProviderConfigCommand, error) {
	if repo == nil {
		return nil, ErrCreateProviderConfigNilRepo
	}

	if catalog == nil {
		return nil, ErrCreateProviderConfigNilCatalog
	}

	if auditWriter == nil {
		return nil, ErrCreateProviderConfigNilAuditWriter
	}

	return &CreateProviderConfigCommand{
		repo:        repo,
		catalog:     catalog,
		auditWriter: auditWriter,
	}, nil
}

// Execute creates a new provider configuration.
func (c *CreateProviderConfigCommand) Execute(ctx context.Context, input *model.CreateProviderConfigurationInput) (*model.ProviderConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if input == nil {
		return nil, ErrCreateProviderConfigNilInput
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.provider_config.create")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Creating provider configuration", libLog.Any("operation", "command.provider_config.create"), libLog.Any("provider_config.name", input.Name), libLog.Any("provider_config.provider_id", input.ProviderID))

	// Verify provider exists in catalog
	provider, err := c.catalog.GetProvider(executor.ProviderID(input.ProviderID))
	if err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "provider not found in catalog", err)
		return nil, constant.ErrProviderNotFoundInCatalog
	}

	// Validate config against provider's JSON Schema
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

	// Create domain entity
	providerConfig, err := input.ToDomain()
	if err != nil {
		libOtel.HandleSpanError(span, "failed to create provider configuration domain", err)
		return nil, err
	}

	if err := c.repo.Create(ctx, providerConfig); err != nil {
		libOtel.HandleSpanError(span, "failed to persist provider configuration", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Provider configuration created successfully", libLog.Any("operation", "command.provider_config.create"), libLog.Any("provider_config.id", providerConfig.ID()))

	c.auditWriter.RecordProviderConfigEvent(ctx, model.AuditEventProviderConfigCreated, model.AuditActionCreate, model.AuditResultSuccess, providerConfig.ID(), map[string]any{
		"provider_config.name":        providerConfig.Name(),
		"provider_config.provider_id": providerConfig.ProviderID(),
	})

	return providerConfig, nil
}

// ErrInvalidProviderSchema indicates the provider's schema definition itself is malformed.
var ErrInvalidProviderSchema = errors.New("invalid provider schema definition")

// validateConfigAgainstSchema validates config against a provider's JSON Schema.
// Returns ErrInvalidProviderSchema for schema compilation errors (internal/catalog issue),
// or a regular error for config validation failures (client input issue).
func validateConfigAgainstSchema(config map[string]any, schemaStr string) error {
	if schemaStr == "" {
		// No schema to validate against — accept any config
		return nil
	}

	compiler := jsonschema.NewCompiler()

	schema, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaStr))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidProviderSchema, err)
	}

	if err := compiler.AddResource("schema.json", schema); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidProviderSchema, err)
	}

	compiled, err := compiler.Compile("schema.json")
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidProviderSchema, err)
	}

	if err := compiled.Validate(config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}
