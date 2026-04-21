// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// ProviderConfigRepository defines the interface for provider configuration data persistence.
type ProviderConfigRepository interface {
	// Create persists a new provider configuration to the database.
	// Returns ErrDuplicateName if a provider configuration with the same name already exists.
	Create(ctx context.Context, providerConfig *model.ProviderConfiguration) error

	// FindByID retrieves a provider configuration by its ID.
	// Returns ErrNotFound if the provider configuration doesn't exist.
	FindByID(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error)

	// FindByName retrieves a provider configuration by its name.
	// Returns ErrNotFound if the provider configuration doesn't exist.
	FindByName(ctx context.Context, name string) (*model.ProviderConfiguration, error)

	// List retrieves provider configurations with pagination and optional filtering.
	List(ctx context.Context, filter ProviderConfigListFilter) (*ProviderConfigListResult, error)

	// Update persists changes to an existing provider configuration.
	// When expectedStatus is non-empty, the update is atomic (check-and-set):
	// the repository verifies the document's current status matches expectedStatus
	// before applying the write. Returns ErrConflictStateChanged if the status
	// differs (i.e., another request modified the resource concurrently).
	// When expectedStatus is empty, the update is unconditional (best-effort fallback).
	// Returns ErrNotFound if the provider configuration doesn't exist.
	Update(ctx context.Context, providerConfig *model.ProviderConfiguration, expectedStatus model.ProviderConfigurationStatus) error

	// Delete removes a provider configuration by its ID.
	// When expectedStatus is non-empty, the delete is atomic (check-and-set):
	// the repository verifies the document's current status matches expectedStatus
	// before removing it. Returns ErrConflictStateChanged if the status differs
	// (i.e., another request modified the resource concurrently).
	// When expectedStatus is empty, the delete is unconditional.
	// Returns ErrNotFound if the provider configuration doesn't exist.
	Delete(ctx context.Context, id uuid.UUID, expectedStatus model.ProviderConfigurationStatus) error

	// ExistsByName checks if a provider configuration with the given name exists.
	ExistsByName(ctx context.Context, name string) (bool, error)
}

// ProviderConfigListFilter contains parameters for listing provider configurations.
type ProviderConfigListFilter struct {
	Status     *model.ProviderConfigurationStatus // Filter by status (optional)
	ProviderID *string                            // Filter by provider ID (optional)
	Limit      int                                // Max items per page (1-100, default: 10)
	Cursor     string                             // Cursor from previous response
	SortBy     string                             // Field to sort by (default: "createdAt")
	SortOrder  string                             // Sort direction: "ASC" or "DESC" (default: "DESC")
}

// ProviderConfigListResult contains the paginated result of listing provider configurations.
type ProviderConfigListResult struct {
	Items      []*model.ProviderConfiguration
	NextCursor string
	HasMore    bool
}

// DefaultProviderConfigListFilter returns a ProviderConfigListFilter with default values.
func DefaultProviderConfigListFilter() ProviderConfigListFilter {
	return ProviderConfigListFilter{
		Limit:     10,
		SortBy:    "createdAt",
		SortOrder: "DESC",
	}
}

// Validate checks that filter values are within acceptable ranges.
// Returns an error if any value is invalid (following Midaz/Tracer convention of rejecting invalid input).
func (f *ProviderConfigListFilter) Validate() error {
	if f.Limit <= 0 || f.Limit > 100 {
		return pkg.ValidationError{
			Code:    "INVALID_LIMIT",
			Message: "limit must be between 1 and 100",
		}
	}

	if f.SortOrder != "" && f.SortOrder != "ASC" && f.SortOrder != "DESC" {
		return pkg.ValidationError{
			Code:    "INVALID_SORT_ORDER",
			Message: "sort_order must be ASC or DESC",
		}
	}

	return nil
}
