// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package services

import (
	"context"
	"time"

	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/model"
)

// DashboardService is a facade that combines dashboard queries.
type DashboardService struct {
	workflowSummaryQuery  *query.GetWorkflowSummaryQuery
	executionSummaryQuery *query.GetExecutionSummaryQuery
}

// NewDashboardService creates a new DashboardService facade.
func NewDashboardService(
	wfSummary *query.GetWorkflowSummaryQuery,
	execSummary *query.GetExecutionSummaryQuery,
) (*DashboardService, error) {
	if wfSummary == nil || execSummary == nil {
		return nil, ErrDashboardServiceNilDependency
	}

	return &DashboardService{
		workflowSummaryQuery:  wfSummary,
		executionSummaryQuery: execSummary,
	}, nil
}

// WorkflowSummary retrieves the workflow dashboard summary.
func (s *DashboardService) WorkflowSummary(ctx context.Context) (*model.WorkflowSummaryOutput, error) {
	return s.workflowSummaryQuery.Execute(ctx)
}

// ExecutionSummary retrieves the execution dashboard summary with optional filters.
func (s *DashboardService) ExecutionSummary(ctx context.Context, startTime, endTime *time.Time, status *string) (*model.ExecutionSummaryOutput, error) {
	return s.executionSummaryQuery.Execute(ctx, query.ExecutionSummaryFilter{
		StartTime: startTime,
		EndTime:   endTime,
		Status:    status,
	})
}
