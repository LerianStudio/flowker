// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package command contains command services for write operations.
package command

import (
	"context"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// WorkflowRepository defines the interface for workflow data persistence.
type WorkflowRepository interface {
	// Create persists a new workflow to the database.
	// Returns ErrDuplicateName if a workflow with the same name already exists.
	Create(ctx context.Context, workflow *model.Workflow) error

	// FindByID retrieves a workflow by its ID.
	// Returns ErrNotFound if the workflow doesn't exist.
	FindByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error)

	// FindByName retrieves a workflow by its name.
	// Returns ErrNotFound if the workflow doesn't exist.
	FindByName(ctx context.Context, name string) (*model.Workflow, error)

	// List retrieves workflows with pagination and optional filtering.
	List(ctx context.Context, filter WorkflowListFilter) (*WorkflowListResult, error)

	// Update persists changes to an existing workflow.
	// When expectedStatus is non-empty, the update is atomic (check-and-set):
	// the repository verifies the document's current status matches expectedStatus
	// before applying the write. Returns ErrConflictStateChanged if the status
	// differs (i.e., another request modified the resource concurrently).
	// When expectedStatus is empty, the update is unconditional (best-effort fallback).
	// Returns ErrNotFound if the workflow doesn't exist.
	Update(ctx context.Context, workflow *model.Workflow, expectedStatus model.WorkflowStatus) error

	// Delete removes a workflow by its ID.
	// When expectedStatus is non-empty, the delete is atomic (check-and-set):
	// the repository verifies the document's current status matches expectedStatus
	// before removing it. Returns ErrConflictStateChanged if the status differs
	// (i.e., another request modified the resource concurrently).
	// When expectedStatus is empty, the delete is unconditional.
	// Returns ErrNotFound if the workflow doesn't exist.
	Delete(ctx context.Context, id uuid.UUID, expectedStatus model.WorkflowStatus) error

	// ExistsByName checks if a workflow with the given name exists.
	ExistsByName(ctx context.Context, name string) (bool, error)
}

// WorkflowListFilter contains parameters for listing workflows.
type WorkflowListFilter struct {
	Status    *model.WorkflowStatus // Filter by status (optional)
	Limit     int                   // Max items per page (1-100, default: 10)
	Cursor    string                // Cursor from previous response
	SortBy    string                // Field to sort by (default: "createdAt")
	SortOrder string                // Sort direction: "ASC" or "DESC" (default: "DESC")
}

// WorkflowListResult contains the paginated result of listing workflows.
type WorkflowListResult struct {
	Items      []*model.Workflow
	NextCursor string
	HasMore    bool
}

// DefaultWorkflowListFilter returns a WorkflowListFilter with default values.
func DefaultWorkflowListFilter() WorkflowListFilter {
	return WorkflowListFilter{
		Limit:     10,
		SortBy:    "createdAt",
		SortOrder: "DESC",
	}
}
