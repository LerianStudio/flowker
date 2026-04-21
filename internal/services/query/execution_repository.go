// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"

	"github.com/LerianStudio/flowker/internal/services/command"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// Aliases from command package for CQRS convenience.
type (
	ExecutionListFilter = command.ExecutionListFilter
	ExecutionListResult = command.ExecutionListResult
)

// ExecutionRepository defines the read-only interface for execution query operations.
// This is intentionally a subset of the command.ExecutionRepository interface,
// exposing only read methods to enforce CQRS separation.
type ExecutionRepository interface {
	// FindByID retrieves a workflow execution by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error)

	// List retrieves executions with filtering and cursor pagination.
	List(ctx context.Context, filter ExecutionListFilter) (*ExecutionListResult, error)
}
