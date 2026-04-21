// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"time"

	"github.com/google/uuid"
)

// ProviderConfigurationOutput is the output DTO for a provider configuration.
type ProviderConfigurationOutput struct {
	ID          uuid.UUID      `json:"id" swaggertype:"string" format:"uuid"`
	Name        string         `json:"name"`
	Description *string        `json:"description,omitempty"`
	ProviderID  string         `json:"providerId"`
	Config      map[string]any `json:"config"`
	Status      string         `json:"status"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// ProviderConfigurationCreateOutput is the minimal output for provider configuration creation.
type ProviderConfigurationCreateOutput struct {
	ID        uuid.UUID `json:"id" swaggertype:"string" format:"uuid"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// ProviderConfigurationListOutput is the output DTO for listing provider configurations.
type ProviderConfigurationListOutput struct {
	Items      []ProviderConfigurationOutput `json:"items"`
	NextCursor string                        `json:"nextCursor"`
	HasMore    bool                          `json:"hasMore"`
}

// ProviderConfigurationOutputFromDomain converts a ProviderConfiguration domain entity to output DTO.
func ProviderConfigurationOutputFromDomain(p *ProviderConfiguration) ProviderConfigurationOutput {
	return ProviderConfigurationOutput{
		ID:          p.ID(),
		Name:        p.Name(),
		Description: p.Description(),
		ProviderID:  p.ProviderID(),
		Config:      maskSensitiveConfig(p.Config()),
		Status:      string(p.Status()),
		Metadata:    p.Metadata(),
		CreatedAt:   p.CreatedAt(),
		UpdatedAt:   p.UpdatedAt(),
	}
}

// ProviderConfigurationCreateOutputFromDomain creates a minimal creation response.
func ProviderConfigurationCreateOutputFromDomain(p *ProviderConfiguration) ProviderConfigurationCreateOutput {
	return ProviderConfigurationCreateOutput{
		ID:        p.ID(),
		Name:      p.Name(),
		Status:    string(p.Status()),
		CreatedAt: p.CreatedAt(),
	}
}

// ProviderConfigurationListOutputFromDomain converts a list of provider configurations to list output.
func ProviderConfigurationListOutputFromDomain(
	configs []*ProviderConfiguration,
	nextCursor string,
	hasMore bool,
) ProviderConfigurationListOutput {
	items := make([]ProviderConfigurationOutput, len(configs))
	for i, p := range configs {
		items[i] = ProviderConfigurationOutputFromDomain(p)
	}

	return ProviderConfigurationListOutput{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}
