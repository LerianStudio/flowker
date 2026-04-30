// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"

	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
)

// ListExecutorConfigsQuery handles listing executor configurations with filtering and pagination.
type ListExecutorConfigsQuery struct {
	repo ExecutorConfigRepository
}

// NewListExecutorConfigsQuery creates a new ListExecutorConfigsQuery.
// Returns error if required dependencies are nil.
func NewListExecutorConfigsQuery(repo ExecutorConfigRepository) (*ListExecutorConfigsQuery, error) {
	if repo == nil {
		return nil, ErrListExecutorConfigsNilRepo
	}

	return &ListExecutorConfigsQuery{
		repo: repo,
	}, nil
}

// Execute retrieves executor configurations with optional filtering and pagination.
func (q *ListExecutorConfigsQuery) Execute(ctx context.Context, filter ExecutorConfigListFilter) (*ExecutorConfigListResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.executor_config.list")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Listing executor configurations", libLog.Any("operation", "query.executor_config.list"), libLog.Any("filter.limit", filter.Limit), libLog.Any("filter.sort_by", filter.SortBy), libLog.Any("filter.sort_order", filter.SortOrder))

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

	result, err := q.repo.List(ctx, filter)
	if err != nil {
		libOtel.HandleSpanError(span, "failed to list executor configurations", err)
		return nil, err
	}

	return result, nil
}
