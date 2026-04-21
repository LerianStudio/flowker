// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
)

// ListWorkflowsQuery handles listing workflows with filtering and pagination.
type ListWorkflowsQuery struct {
	repo WorkflowRepository
}

// NewListWorkflowsQuery creates a new ListWorkflowsQuery.
// Returns error if required dependencies are nil.
func NewListWorkflowsQuery(repo WorkflowRepository) (*ListWorkflowsQuery, error) {
	if repo == nil {
		return nil, ErrListWorkflowsNilRepo
	}

	return &ListWorkflowsQuery{
		repo: repo,
	}, nil
}

// Execute retrieves workflows with optional filtering and pagination.
func (q *ListWorkflowsQuery) Execute(ctx context.Context, filter WorkflowListFilter) (*WorkflowListResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.workflow.list")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Listing workflows", libLog.Any("operation", "query.workflow.list"), libLog.Any("filter.limit", filter.Limit), libLog.Any("filter.sort_by", filter.SortBy), libLog.Any("filter.sort_order", filter.SortOrder))

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
		libOtel.HandleSpanError(span, "failed to list workflows", err)
		return nil, err
	}

	return result, nil
}
