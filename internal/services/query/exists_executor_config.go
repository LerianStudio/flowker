// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/google/uuid"
)

// ExistsExecutorConfigQuery handles checking if an executor configuration exists.
type ExistsExecutorConfigQuery struct {
	repo ExecutorConfigRepository
}

// NewExistsExecutorConfigQuery creates a new ExistsExecutorConfigQuery.
// Returns error if required dependencies are nil.
func NewExistsExecutorConfigQuery(repo ExecutorConfigRepository) (*ExistsExecutorConfigQuery, error) {
	if repo == nil {
		return nil, ErrExistsExecutorConfigNilRepo
	}

	return &ExistsExecutorConfigQuery{
		repo: repo,
	}, nil
}

// Execute checks if an executor configuration with the given ID exists.
func (q *ExistsExecutorConfigQuery) Execute(ctx context.Context, id uuid.UUID) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.executor_config.exists")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Checking if executor configuration exists", libLog.Any("operation", "query.executor_config.exists"), libLog.Any("executor_config.id", id))

	_, err := q.repo.FindByID(ctx, id)
	if err != nil {
		libOtel.HandleSpanError(span, "executor configuration not found", err)
		return false, nil
	}

	return true, nil
}

// ExecuteByName checks if an executor configuration with the given name exists.
func (q *ExistsExecutorConfigQuery) ExecuteByName(ctx context.Context, name string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.executor_config.exists_by_name")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Checking if executor configuration exists by name", libLog.Any("operation", "query.executor_config.exists_by_name"), libLog.Any("executor_config.name", name))

	exists, err := q.repo.ExistsByName(ctx, name)
	if err != nil {
		libOtel.HandleSpanError(span, "failed to check executor configuration existence", err)
		return false, err
	}

	return exists, nil
}
