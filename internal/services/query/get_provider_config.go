// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"
	"errors"
	"fmt"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// GetProviderConfigByIDQuery handles retrieving a provider configuration by ID.
type GetProviderConfigByIDQuery struct {
	repo ProviderConfigRepository
}

// NewGetProviderConfigByIDQuery creates a new GetProviderConfigByIDQuery.
// Returns error if required dependencies are nil.
func NewGetProviderConfigByIDQuery(repo ProviderConfigRepository) (*GetProviderConfigByIDQuery, error) {
	if repo == nil {
		return nil, ErrGetProviderConfigNilRepo
	}

	return &GetProviderConfigByIDQuery{
		repo: repo,
	}, nil
}

// Execute retrieves a provider configuration by its ID.
func (q *GetProviderConfigByIDQuery) Execute(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.provider_config.get")
	defer span.End()

	if id == uuid.Nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "Invalid provider configuration ID", nil)

		return nil, pkg.ValidationError{
			Code:    "INVALID_ID",
			Message: "provider configuration ID cannot be empty",
		}
	}

	logger.Log(ctx, libLog.LevelInfo, "Getting provider configuration by ID", libLog.Any("operation", "query.provider_config.get"), libLog.Any("provider_config.id", id))

	providerConfig, err := q.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrProviderConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Provider configuration not found", err)

			return nil, pkg.EntityNotFoundError{
				EntityType: "ProviderConfiguration",
				Code:       constant.ErrProviderConfigNotFound.Error(),
				Err:        err,
			}
		}

		libOtel.HandleSpanError(span, "Failed to find provider configuration", err)

		return nil, fmt.Errorf("failed to find provider configuration: %w", err)
	}

	return providerConfig, nil
}
