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

// ListExecutionsQuery handles listing executions with filtering and pagination.
type ListExecutionsQuery struct {
	repo ExecutionRepository
}

// NewListExecutionsQuery creates a new ListExecutionsQuery.
// Returns error if required dependencies are nil.
func NewListExecutionsQuery(repo ExecutionRepository) (*ListExecutionsQuery, error) {
	if repo == nil {
		return nil, ErrListExecutionsNilRepo
	}

	return &ListExecutionsQuery{
		repo: repo,
	}, nil
}

// Execute retrieves executions with optional filtering and pagination.
func (q *ListExecutionsQuery) Execute(ctx context.Context, filter ExecutionListFilter) (*ExecutionListResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.execution.list")
	defer span.End()

	// Apply defaults first to normalize values
	filter.ApplyDefaults()

	// Validate filter values after defaults are applied
	if err := filter.Validate(); err != nil {
		libOtel.HandleSpanError(span, "invalid filter", err)
		return nil, err
	}

	logger.Log(ctx, libLog.LevelInfo, "Listing executions", libLog.Any("operation", "query.execution.list"), libLog.Any("filter.limit", filter.Limit), libLog.Any("filter.sort_by", filter.SortBy), libLog.Any("filter.sort_order", filter.SortOrder))

	result, err := q.repo.List(ctx, filter)
	if err != nil {
		libOtel.HandleSpanError(span, "failed to list executions", err)
		return nil, err
	}

	return result, nil
}
