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
)

// GetWorkflowByNameQuery handles retrieving a workflow by name.
type GetWorkflowByNameQuery struct {
	repo WorkflowRepository
}

// NewGetWorkflowByNameQuery creates a new GetWorkflowByNameQuery.
// Returns error if required dependencies are nil.
func NewGetWorkflowByNameQuery(repo WorkflowRepository) (*GetWorkflowByNameQuery, error) {
	if repo == nil {
		return nil, ErrGetWorkflowByNameNilRepo
	}

	return &GetWorkflowByNameQuery{
		repo: repo,
	}, nil
}

// Execute retrieves a workflow by its name.
func (q *GetWorkflowByNameQuery) Execute(ctx context.Context, name string) (*model.Workflow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.workflow.get_by_name")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Getting workflow by name", libLog.Any("operation", "query.workflow.get_by_name"), libLog.Any("workflow.name", name))

	workflow, err := q.repo.FindByName(ctx, name)
	if err != nil {
		if errors.Is(err, constant.ErrWorkflowNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Workflow not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find workflow by name", err)

		return nil, fmt.Errorf("failed to find workflow by name: %w", err)
	}

	return workflow, nil
}
