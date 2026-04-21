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

// ExecutorConfigRepository defines the read-only interface for executor configuration query operations.
// This is intentionally a subset of the command.ExecutorConfigRepository interface,
// exposing only read methods to enforce CQRS separation.
type ExecutorConfigRepository interface {
	// FindByID retrieves an executor configuration by its ID.
	// Returns ErrNotFound if the executor configuration doesn't exist.
	FindByID(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error)

	// FindByName retrieves an executor configuration by its name.
	// Returns ErrNotFound if the executor configuration doesn't exist.
	FindByName(ctx context.Context, name string) (*model.ExecutorConfiguration, error)

	// List retrieves executor configurations with pagination and optional filtering.
	List(ctx context.Context, filter ExecutorConfigListFilter) (*ExecutorConfigListResult, error)

	// ExistsByName checks if an executor configuration with the given name exists.
	ExistsByName(ctx context.Context, name string) (bool, error)
}

// ExecutorConfigListFilter is an alias to command.ExecutorConfigListFilter.
type ExecutorConfigListFilter = command.ExecutorConfigListFilter

// ExecutorConfigListResult is an alias to command.ExecutorConfigListResult.
type ExecutorConfigListResult = command.ExecutorConfigListResult
