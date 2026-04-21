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

	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// GetExecutionResultsQuery handles retrieving execution results by ID.
// Returns an error if the execution is still in progress.
type GetExecutionResultsQuery struct {
	repo ExecutionRepository
}

// NewGetExecutionResultsQuery creates a new GetExecutionResultsQuery.
func NewGetExecutionResultsQuery(repo ExecutionRepository) (*GetExecutionResultsQuery, error) {
	if repo == nil {
		return nil, ErrGetExecutionResultsNilRepo
	}

	return &GetExecutionResultsQuery{repo: repo}, nil
}

// Execute retrieves execution results by execution ID.
// Returns ErrExecutionInProgress if the execution is still running.
func (q *GetExecutionResultsQuery) Execute(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.execution.get_results")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Getting execution results", libLog.Any("operation", "query.execution.get_results"), libLog.Any("execution.id", id))

	execution, err := q.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrExecutionNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Execution not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find execution", err)

		return nil, fmt.Errorf("failed to find execution: %w", err)
	}

	// Results are only available for terminal executions
	if !execution.IsTerminal() {
		libOtel.HandleSpanBusinessErrorEvent(span, "Execution still in progress", constant.ErrExecutionInProgress)
		return nil, constant.ErrExecutionInProgress
	}

	return execution, nil
}
