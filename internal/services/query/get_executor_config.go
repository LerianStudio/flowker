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

// GetExecutorConfigQuery handles retrieving an executor configuration by ID.
type GetExecutorConfigQuery struct {
	repo ExecutorConfigRepository
}

// NewGetExecutorConfigQuery creates a new GetExecutorConfigQuery.
// Returns error if required dependencies are nil.
func NewGetExecutorConfigQuery(repo ExecutorConfigRepository) (*GetExecutorConfigQuery, error) {
	if repo == nil {
		return nil, ErrGetExecutorConfigNilRepo
	}

	return &GetExecutorConfigQuery{
		repo: repo,
	}, nil
}

// Execute retrieves an executor configuration by its ID.
func (q *GetExecutorConfigQuery) Execute(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.executor_config.get")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Getting executor configuration by ID", libLog.Any("operation", "query.executor_config.get"), libLog.Any("executor_config.id", id))

	executorConfig, err := q.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrExecutorConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Executor configuration not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find executor configuration", err)

		return nil, fmt.Errorf("failed to find executor configuration: %w", err)
	}

	return executorConfig, nil
}
