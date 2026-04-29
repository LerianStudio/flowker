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

// GetWorkflowQuery handles retrieving a workflow by ID.
type GetWorkflowQuery struct {
	repo WorkflowRepository
}

// NewGetWorkflowQuery creates a new GetWorkflowQuery.
// Returns error if required dependencies are nil.
func NewGetWorkflowQuery(repo WorkflowRepository) (*GetWorkflowQuery, error) {
	if repo == nil {
		return nil, ErrGetWorkflowNilRepo
	}

	return &GetWorkflowQuery{
		repo: repo,
	}, nil
}

// Execute retrieves a workflow by its ID.
func (q *GetWorkflowQuery) Execute(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.workflow.get")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Getting workflow by ID", libLog.Any("operation", "query.workflow.get"), libLog.Any("workflow.id", id))

	workflow, err := q.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrWorkflowNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Workflow not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find workflow", err)

		return nil, fmt.Errorf("failed to find workflow: %w", err)
	}

	return workflow, nil
}
