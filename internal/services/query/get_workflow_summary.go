// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"
	"errors"

	"github.com/LerianStudio/flowker/pkg/model"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
)

// GetWorkflowSummaryQuery handles retrieving the workflow dashboard summary.
type GetWorkflowSummaryQuery struct {
	repo DashboardRepository
}

// NewGetWorkflowSummaryQuery creates a new GetWorkflowSummaryQuery.
// Returns error if required dependencies are nil.
func NewGetWorkflowSummaryQuery(repo DashboardRepository) (*GetWorkflowSummaryQuery, error) {
	if repo == nil {
		return nil, ErrDashboardNilRepo
	}

	return &GetWorkflowSummaryQuery{
		repo: repo,
	}, nil
}

// Execute retrieves the workflow summary aggregation.
func (q *GetWorkflowSummaryQuery) Execute(ctx context.Context) (*model.WorkflowSummaryOutput, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.dashboard.workflow_summary")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Retrieving workflow summary", libLog.Any("operation", "query.dashboard.workflow_summary"))

	result, err := q.repo.WorkflowSummary(ctx)
	if err != nil {
		libOtel.HandleSpanError(span, "failed to get workflow summary", err)
		return nil, err
	}

	return result, nil
}
