// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"
	"errors"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
)

// ListProviderConfigsQuery handles listing provider configurations with filtering and pagination.
type ListProviderConfigsQuery struct {
	repo ProviderConfigRepository
}

// NewListProviderConfigsQuery creates a new ListProviderConfigsQuery.
// Returns error if required dependencies are nil.
func NewListProviderConfigsQuery(repo ProviderConfigRepository) (*ListProviderConfigsQuery, error) {
	if repo == nil {
		return nil, ErrListProviderConfigsNilRepo
	}

	return &ListProviderConfigsQuery{
		repo: repo,
	}, nil
}

// Execute retrieves provider configurations with optional filtering and pagination.
func (q *ListProviderConfigsQuery) Execute(ctx context.Context, filter ProviderConfigListFilter) (*ProviderConfigListResult, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.provider_config.list")
	defer span.End()

	// Apply defaults if not specified
	if filter.Limit <= 0 {
		filter.Limit = 10
	}

	if filter.SortBy == "" {
		filter.SortBy = "createdAt"
	}

	if filter.SortOrder == "" {
		filter.SortOrder = "DESC"
	}

	// Validate filter after defaults are applied
	if err := filter.Validate(); err != nil {
		libOtel.HandleSpanBusinessErrorEvent(span, "Invalid filter parameters", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Listing provider configurations", libLog.Any("operation", "query.provider_config.list"), libLog.Any("filter.limit", filter.Limit), libLog.Any("filter.sort_by", filter.SortBy), libLog.Any("filter.sort_order", filter.SortOrder))

	result, err := q.repo.List(ctx, filter)
	if err != nil {
		libOtel.HandleSpanError(span, "failed to list provider configurations", err)
		return nil, err
	}

	return result, nil
}
