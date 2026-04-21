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

// WorkflowRepository defines the read-only interface for workflow query operations.
// This is intentionally a subset of the command.WorkflowRepository interface,
// exposing only read methods to enforce CQRS separation.
type WorkflowRepository interface {
	// FindByID retrieves a workflow by its ID.
	// Returns ErrNotFound if the workflow doesn't exist.
	FindByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error)

	// FindByName retrieves a workflow by its name.
	// Returns ErrNotFound if the workflow doesn't exist.
	FindByName(ctx context.Context, name string) (*model.Workflow, error)

	// List retrieves workflows with pagination and optional filtering.
	List(ctx context.Context, filter WorkflowListFilter) (*WorkflowListResult, error)
}

// WorkflowListFilter is an alias to command.WorkflowListFilter.
type WorkflowListFilter = command.WorkflowListFilter

// WorkflowListResult is an alias to command.WorkflowListResult.
type WorkflowListResult = command.WorkflowListResult
