// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"
	"errors"

	"github.com/LerianStudio/flowker/pkg/model"
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
)

// GetExecutionSummaryQuery handles retrieving the execution dashboard summary.
type GetExecutionSummaryQuery struct {
	repo DashboardRepository
}

// NewGetExecutionSummaryQuery creates a new GetExecutionSummaryQuery.
// Returns error if required dependencies are nil.
func NewGetExecutionSummaryQuery(repo DashboardRepository) (*GetExecutionSummaryQuery, error) {
	if repo == nil {
		return nil, ErrDashboardNilRepo
	}

	return &GetExecutionSummaryQuery{
		repo: repo,
	}, nil
}

// Execute retrieves the execution summary aggregation with optional filters.
func (q *GetExecutionSummaryQuery) Execute(ctx context.Context, filter ExecutionSummaryFilter) (*model.ExecutionSummaryOutput, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.dashboard.execution_summary")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Retrieving execution summary", libLog.Any("operation", "query.dashboard.execution_summary"))

	result, err := q.repo.ExecutionSummary(ctx, filter)
	if err != nil {
		libOtel.HandleSpanError(span, "failed to get execution summary", err)
		return nil, err
	}

	return result, nil
}
