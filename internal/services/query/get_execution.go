// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"
	"errors"
	"fmt"

	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"

	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// GetExecutionQuery handles retrieving an execution by ID.
type GetExecutionQuery struct {
	repo ExecutionRepository
}

// NewGetExecutionQuery creates a new GetExecutionQuery.
func NewGetExecutionQuery(repo ExecutionRepository) (*GetExecutionQuery, error) {
	if repo == nil {
		return nil, ErrGetExecutionNilRepo
	}

	return &GetExecutionQuery{repo: repo}, nil
}

// Execute retrieves an execution by its ID.
func (q *GetExecutionQuery) Execute(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.execution.get")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Getting execution by ID", libLog.Any("operation", "query.execution.get"), libLog.Any("execution.id", id))

	execution, err := q.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrExecutionNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Execution not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find execution", err)

		return nil, fmt.Errorf("failed to find execution: %w", err)
	}

	return execution, nil
}
