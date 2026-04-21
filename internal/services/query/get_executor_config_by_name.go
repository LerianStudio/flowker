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
)

// GetExecutorConfigByNameQuery handles retrieving an executor configuration by name.
type GetExecutorConfigByNameQuery struct {
	repo ExecutorConfigRepository
}

// NewGetExecutorConfigByNameQuery creates a new GetExecutorConfigByNameQuery.
// Returns error if required dependencies are nil.
func NewGetExecutorConfigByNameQuery(repo ExecutorConfigRepository) (*GetExecutorConfigByNameQuery, error) {
	if repo == nil {
		return nil, ErrGetExecutorConfigByNameNilRepo
	}

	return &GetExecutorConfigByNameQuery{
		repo: repo,
	}, nil
}

// Execute retrieves an executor configuration by its name.
func (q *GetExecutorConfigByNameQuery) Execute(ctx context.Context, name string) (*model.ExecutorConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.executor_config.get_by_name")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Getting executor configuration by name", libLog.Any("operation", "query.executor_config.get_by_name"), libLog.Any("executor_config.name", name))

	executorConfig, err := q.repo.FindByName(ctx, name)
	if err != nil {
		if errors.Is(err, constant.ErrExecutorConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Executor configuration not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find executor configuration by name", err)

		return nil, fmt.Errorf("failed to find executor configuration by name: %w", err)
	}

	return executorConfig, nil
}
