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

// DeleteProviderConfigCommand handles provider configuration deletion.
type DeleteProviderConfigCommand struct {
	repo        ProviderConfigRepository
	auditWriter AuditWriter
}

// NewDeleteProviderConfigCommand creates a new DeleteProviderConfigCommand.
// Returns error if required dependencies are nil.
func NewDeleteProviderConfigCommand(
	repo ProviderConfigRepository,
	auditWriter AuditWriter,
) (*DeleteProviderConfigCommand, error) {
	if repo == nil {
		return nil, ErrDeleteProviderConfigNilRepo
	}

	if auditWriter == nil {
		return nil, ErrDeleteProviderConfigNilAuditWriter
	}

	return &DeleteProviderConfigCommand{
		repo:        repo,
		auditWriter: auditWriter,
	}, nil
}

// Execute removes a provider configuration.
func (c *DeleteProviderConfigCommand) Execute(ctx context.Context, id uuid.UUID) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.provider_config.delete")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Deleting provider configuration", libLog.Any("operation", "command.provider_config.delete"), libLog.Any("provider_config.id", id))

	providerConfig, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrProviderConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Provider configuration not found", err)
			return err
		}

		libOtel.HandleSpanError(span, "Failed to find provider configuration", err)

		return fmt.Errorf("failed to find provider configuration: %w", err)
	}

	previousStatus := providerConfig.Status()

	if err := c.repo.Delete(ctx, id, previousStatus); err != nil {
		if errors.Is(err, constant.ErrConflictStateChanged) {
			libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
			return err
		}

		libOtel.HandleSpanError(span, "failed to delete provider configuration", err)
		return err
	}

	logger.Log(ctx, libLog.LevelInfo, "Provider configuration deleted successfully", libLog.Any("operation", "command.provider_config.delete"), libLog.Any("provider_config.id", id))

	c.auditWriter.RecordProviderConfigEvent(ctx, model.AuditEventProviderConfigDeleted, model.AuditActionDelete, model.AuditResultSuccess, id, map[string]any{
		"provider_config.id": id.String(),
	})

	return nil
}
