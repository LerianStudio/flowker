// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import "github.com/google/uuid"

// CreateProviderConfigurationInput is the input DTO for creating a provider configuration.
// Uses validator tags per PROJECT_RULES.md.
type CreateProviderConfigurationInput struct {
	Name        string         `json:"name" validate:"required,min=1,max=100"`
	Description *string        `json:"description,omitempty" validate:"omitempty,max=500"`
	ProviderID  string         `json:"providerId" validate:"required"`
	Config      map[string]any `json:"config" validate:"required"`
	Metadata    map[string]any `json:"metadata,omitempty" validate:"omitempty"`
}

// UpdateProviderConfigurationInput is the input DTO for updating a provider configuration.
type UpdateProviderConfigurationInput struct {
	Name        *string        `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string        `json:"description,omitempty" validate:"omitempty,max=500"`
	Config      map[string]any `json:"config,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty" validate:"omitempty"`
}

// ToDomain converts CreateProviderConfigurationInput to domain entity.
func (i *CreateProviderConfigurationInput) ToDomain() (*ProviderConfiguration, error) {
	pc, err := NewProviderConfiguration(
		i.Name,
		i.Description,
		i.ProviderID,
		i.Config,
	)
	if err != nil {
		return nil, err
	}

	// Set metadata if provided
	if i.Metadata != nil {
		for k, v := range i.Metadata {
			pc.SetMetadata(k, v)
		}
	}

	return pc, nil
}

// ProviderConfigurationFilterInput is the input DTO for listing provider configurations with filters.
type ProviderConfigurationFilterInput struct {
	Status     *string `query:"status" validate:"omitempty,oneof=active disabled"`
	ProviderID *string `query:"providerId"`
	Limit      int     `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor     string  `query:"cursor"`
	SortBy     string  `query:"sortBy" validate:"omitempty,oneof=createdAt updatedAt name"`
	SortOrder  string  `query:"sortOrder" validate:"omitempty,oneof=ASC DESC"`
}

// GetProviderConfigurationInput is the input DTO for getting a provider configuration by ID.
type GetProviderConfigurationInput struct {
	ID uuid.UUID `params:"id" validate:"required"`
}

// ProviderConfigurationStatusInput is the input DTO for changing provider configuration status.
type ProviderConfigurationStatusInput struct {
	ID uuid.UUID `params:"id" validate:"required"`
}
