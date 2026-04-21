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

// ProviderConfigRepository defines the read-only interface for provider configuration query operations.
// This is intentionally a subset of the command.ProviderConfigRepository interface,
// exposing only read methods to enforce CQRS separation.
type ProviderConfigRepository interface {
	// FindByID retrieves a provider configuration by its ID.
	// Returns ErrNotFound if the provider configuration doesn't exist.
	FindByID(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error)

	// FindByName retrieves a provider configuration by its name.
	// Returns ErrNotFound if the provider configuration doesn't exist.
	FindByName(ctx context.Context, name string) (*model.ProviderConfiguration, error)

	// List retrieves provider configurations with pagination and optional filtering.
	List(ctx context.Context, filter ProviderConfigListFilter) (*ProviderConfigListResult, error)

	// ExistsByName checks if a provider configuration with the given name exists.
	ExistsByName(ctx context.Context, name string) (bool, error)
}

// ProviderConfigListFilter is an alias to command.ProviderConfigListFilter.
type ProviderConfigListFilter = command.ProviderConfigListFilter

// ProviderConfigListResult is an alias to command.ProviderConfigListResult.
type ProviderConfigListResult = command.ProviderConfigListResult
