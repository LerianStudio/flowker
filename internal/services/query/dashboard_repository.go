// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
)

// DashboardRepository provides aggregation-only read operations for dashboard queries.
type DashboardRepository interface {
	WorkflowSummary(ctx context.Context) (*model.WorkflowSummaryOutput, error)
	ExecutionSummary(ctx context.Context, filter ExecutionSummaryFilter) (*model.ExecutionSummaryOutput, error)
}

// ExecutionSummaryFilter holds optional parameters for execution summary queries.
type ExecutionSummaryFilter struct {
	StartTime *time.Time
	EndTime   *time.Time
	Status    *string
}
