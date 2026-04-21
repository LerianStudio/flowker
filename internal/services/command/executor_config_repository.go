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

// ExecutorConfigRepository defines the interface for executor configuration data persistence.
type ExecutorConfigRepository interface {
	// Create persists a new executor configuration to the database.
	// Returns ErrDuplicateName if an executor configuration with the same name already exists.
	Create(ctx context.Context, executorConfig *model.ExecutorConfiguration) error

	// FindByID retrieves an executor configuration by its ID.
	// Returns ErrNotFound if the executor configuration doesn't exist.
	FindByID(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error)

	// FindByName retrieves an executor configuration by its name.
	// Returns ErrNotFound if the executor configuration doesn't exist.
	FindByName(ctx context.Context, name string) (*model.ExecutorConfiguration, error)

	// List retrieves executor configurations with pagination and optional filtering.
	List(ctx context.Context, filter ExecutorConfigListFilter) (*ExecutorConfigListResult, error)

	// Update persists changes to an existing executor configuration.
	// When expectedStatus is non-empty, the update is atomic (check-and-set):
	// the repository verifies the document's current status matches expectedStatus
	// before applying the write. Returns ErrConflictStateChanged if the status
	// differs (i.e., another request modified the resource concurrently).
	// When expectedStatus is empty, the update is unconditional (best-effort fallback).
	// Returns ErrNotFound if the executor configuration doesn't exist.
	Update(ctx context.Context, executorConfig *model.ExecutorConfiguration, expectedStatus model.ExecutorConfigurationStatus) error

	// Delete removes an executor configuration by its ID.
	// When expectedStatus is non-empty, the delete is atomic (check-and-set):
	// the repository verifies the document's current status matches expectedStatus
	// before removing it. Returns ErrConflictStateChanged if the status differs
	// (i.e., another request modified the resource concurrently).
	// When expectedStatus is empty, the delete is unconditional.
	// Returns ErrNotFound if the executor configuration doesn't exist.
	Delete(ctx context.Context, id uuid.UUID, expectedStatus model.ExecutorConfigurationStatus) error

	// ExistsByName checks if an executor configuration with the given name exists.
	ExistsByName(ctx context.Context, name string) (bool, error)
}

// ExecutorConfigListFilter contains parameters for listing executor configurations.
type ExecutorConfigListFilter struct {
	Status    *model.ExecutorConfigurationStatus // Filter by status (optional)
	Limit     int                                // Max items per page (1-100, default: 10)
	Cursor    string                             // Cursor from previous response
	SortBy    string                             // Field to sort by (default: "createdAt")
	SortOrder string                             // Sort direction: "ASC" or "DESC" (default: "DESC")
}

// ExecutorConfigListResult contains the paginated result of listing executor configurations.
type ExecutorConfigListResult struct {
	Items      []*model.ExecutorConfiguration
	NextCursor string
	HasMore    bool
}

// DefaultExecutorConfigListFilter returns an ExecutorConfigListFilter with default values.
func DefaultExecutorConfigListFilter() ExecutorConfigListFilter {
	return ExecutorConfigListFilter{
		Limit:     10,
		SortBy:    "createdAt",
		SortOrder: "DESC",
	}
}
