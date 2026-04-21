// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"
	"fmt"
	"strings"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// DefaultExecutionSortField is the default sort column for execution queries.
const DefaultExecutionSortField = "startedAt"

// validExecutionSortFields defines the whitelist of valid sort fields for executions.
var validExecutionSortFields = map[string]bool{
	"startedAt":   true,
	"completedAt": true,
}

// IsValidExecutionSortField checks if the given field is a valid sort field for executions.
func IsValidExecutionSortField(field string) bool {
	return validExecutionSortFields[field]
}

// ExecutionListFilter defines filter and pagination options for listing executions.
type ExecutionListFilter struct {
	WorkflowID *uuid.UUID
	Status     *model.ExecutionStatus
	Limit      int
	Cursor     string
	SortBy     string
	SortOrder  string
}

// ApplyDefaults sets default values for Limit, SortBy, and SortOrder fields.
func (f *ExecutionListFilter) ApplyDefaults() {
	if f.Limit <= 0 {
		f.Limit = constant.DefaultPaginationLimit
	} else if f.Limit > constant.MaxPaginationLimit {
		f.Limit = constant.MaxPaginationLimit
	}

	// Only apply sort defaults when not using cursor pagination,
	// because cursor already contains sort configuration.
	if f.Cursor == "" {
		if f.SortBy == "" {
			f.SortBy = DefaultExecutionSortField
		}

		if f.SortOrder == "" {
			f.SortOrder = "DESC"
		} else {
			f.SortOrder = strings.ToUpper(f.SortOrder)
		}
	}
}

// Validate ensures ExecutionListFilter has valid values.
// Call ApplyDefaults() before Validate() if you want defaults applied.
func (f *ExecutionListFilter) Validate() error {
	if f.Limit < constant.MinPaginationLimit {
		return pkg.ValidationError{
			Code:    constant.ErrPaginationLimitExceeded.Error(),
			Title:   "Invalid Limit",
			Message: fmt.Sprintf("limit must be at least %d", constant.MinPaginationLimit),
		}
	}

	if f.Limit > constant.MaxPaginationLimit {
		return pkg.ValidationError{
			Code:    constant.ErrPaginationLimitExceeded.Error(),
			Title:   "Pagination Limit Exceeded",
			Message: fmt.Sprintf("limit must not exceed %d", constant.MaxPaginationLimit),
		}
	}

	if f.Status != nil && !f.Status.IsValid() {
		return pkg.ValidationError{
			Code:    constant.ErrInvalidQueryParameter.Error(),
			Title:   "Invalid Status Filter",
			Message: "status must be one of [pending, running, completed, failed]",
		}
	}

	if f.SortBy != "" && !IsValidExecutionSortField(f.SortBy) {
		return pkg.ValidationError{
			Code:    constant.ErrInvalidQueryParameter.Error(),
			Title:   "Invalid Sort Field",
			Message: "sortBy must be one of [startedAt, completedAt]",
		}
	}

	if f.SortOrder != "" {
		upper := strings.ToUpper(f.SortOrder)
		if upper != "ASC" && upper != "DESC" {
			return pkg.ValidationError{
				Code:    constant.ErrInvalidSortOrder.Error(),
				Title:   "Invalid Sort Order",
				Message: "sortOrder must be ASC or DESC",
			}
		}
	}

	// Cursor + sort consistency: reject sortBy/sortOrder when cursor is present
	if f.Cursor != "" && (f.SortBy != "" || f.SortOrder != "") {
		return pkg.ValidationError{
			Code:    constant.ErrInvalidQueryParameter.Error(),
			Title:   "Invalid Pagination Parameters",
			Message: "sortBy and sortOrder cannot be used with cursor; cursor already contains sort configuration",
		}
	}

	return nil
}

// ExecutionListResult contains the paginated list of executions.
type ExecutionListResult struct {
	Items      []*model.WorkflowExecution
	NextCursor string
	HasMore    bool
}

// ExecutionRepository defines the interface for workflow execution data persistence.
type ExecutionRepository interface {
	// Create persists a new workflow execution to the database.
	Create(ctx context.Context, execution *model.WorkflowExecution) error

	// FindByID retrieves a workflow execution by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error)

	// FindByIdempotencyKey retrieves a workflow execution by its idempotency key.
	FindByIdempotencyKey(ctx context.Context, key string) (*model.WorkflowExecution, error)

	// Update persists changes to an existing workflow execution.
	// When expectedStatus is non-empty, the update is atomic (check-and-set):
	// the repository verifies the document's current status matches expectedStatus
	// before applying the write. Returns ErrConflictStateChanged if the status
	// differs (i.e., another goroutine/request modified the execution concurrently).
	// When expectedStatus is empty, the update is unconditional (best-effort fallback).
	Update(ctx context.Context, execution *model.WorkflowExecution, expectedStatus model.ExecutionStatus) error

	// FindIncomplete retrieves all executions with status pending or running (for recovery).
	FindIncomplete(ctx context.Context) ([]*model.WorkflowExecution, error)

	// List retrieves executions with filtering and cursor pagination.
	List(ctx context.Context, filter ExecutionListFilter) (*ExecutionListResult, error)
}
