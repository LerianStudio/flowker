// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package model contains domain entities and DTOs for Flowker.
package model

import (
	"time"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/google/uuid"
)

// ProviderConfigurationStatus represents the status of a provider configuration.
type ProviderConfigurationStatus string

const (
	// ProviderConfigStatusActive indicates a provider configuration that is ready for use.
	ProviderConfigStatusActive ProviderConfigurationStatus = "active"
	// ProviderConfigStatusDisabled indicates a provider configuration that is temporarily disabled.
	ProviderConfigStatusDisabled ProviderConfigurationStatus = "disabled"
)

// ProviderConfiguration validation errors.
var (
	ErrProviderConfigNameRequired = pkg.ValidationError{
		Code:    constant.ErrProviderConfigNameRequired.Error(),
		Message: "name is required",
	}
	ErrProviderConfigNameTooLong = pkg.ValidationError{
		Code:    constant.ErrProviderConfigNameTooLong.Error(),
		Message: "name cannot exceed 100 characters",
	}
	ErrProviderConfigProviderIDRequired = pkg.ValidationError{
		Code:    constant.ErrProviderConfigProviderIDRequired.Error(),
		Message: "provider_id is required",
	}
	ErrProviderConfigConfigRequired = pkg.ValidationError{
		Code:    constant.ErrProviderConfigConfigRequired.Error(),
		Message: "config is required and cannot be empty",
	}
	ErrProviderConfigCannotDisable = pkg.ValidationError{
		Code:    constant.ErrProviderConfigCannotModify.Error(),
		Message: "only active provider configurations can be disabled",
	}
	ErrProviderConfigCannotEnable = pkg.ValidationError{
		Code:    constant.ErrProviderConfigCannotModify.Error(),
		Message: "only disabled provider configurations can be enabled",
	}
	ErrProviderConfigDescriptionTooLong = pkg.ValidationError{
		Code:    constant.ErrProviderConfigDescriptionTooLong.Error(),
		Message: "description cannot exceed 500 characters",
	}
)

const (
	maxProviderConfigNameLength        = 100
	maxProviderConfigDescriptionLength = 500
)

// ProviderConfiguration represents a configured provider instance (Rich Domain Model).
// Fields are private with validation in constructor per PROJECT_RULES.md.
type ProviderConfiguration struct {
	id          uuid.UUID
	name        string
	description *string
	providerID  string
	config      map[string]any
	status      ProviderConfigurationStatus
	metadata    map[string]any
	createdAt   time.Time
	updatedAt   time.Time
}

// validateProviderConfigData validates name, providerID, and config.
func validateProviderConfigData(name, providerID string, description *string, config map[string]any) error {
	if name == "" {
		return ErrProviderConfigNameRequired
	}

	if len(name) > maxProviderConfigNameLength {
		return ErrProviderConfigNameTooLong
	}

	if description != nil && len(*description) > maxProviderConfigDescriptionLength {
		return ErrProviderConfigDescriptionTooLong
	}

	if providerID == "" {
		return ErrProviderConfigProviderIDRequired
	}

	if len(config) == 0 {
		return ErrProviderConfigConfigRequired
	}

	return nil
}

// NewProviderConfiguration creates a new ProviderConfiguration with validation.
func NewProviderConfiguration(
	name string,
	description *string,
	providerID string,
	config map[string]any,
) (*ProviderConfiguration, error) {
	if err := validateProviderConfigData(name, providerID, description, config); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &ProviderConfiguration{
		id:          uuid.New(),
		name:        name,
		description: copyStringPtr(description),
		providerID:  providerID,
		config:      cloneConfig(config),
		status:      ProviderConfigStatusActive,
		metadata:    make(map[string]any),
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// ReconstructProviderConfiguration reconstructs a ProviderConfiguration from database values.
func ReconstructProviderConfiguration(
	id uuid.UUID,
	name string,
	description *string,
	providerID string,
	config map[string]any,
	status ProviderConfigurationStatus,
	metadata map[string]any,
	createdAt, updatedAt time.Time,
) *ProviderConfiguration {
	return &ProviderConfiguration{
		id:          id,
		name:        name,
		description: copyStringPtr(description),
		providerID:  providerID,
		config:      cloneConfig(config),
		status:      status,
		metadata:    cloneMetadata(metadata),
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// ID returns the provider configuration's unique identifier.
func (p *ProviderConfiguration) ID() uuid.UUID { return p.id }

// Name returns the provider configuration's name.
func (p *ProviderConfiguration) Name() string { return p.name }

// Description returns the provider configuration's description.
func (p *ProviderConfiguration) Description() *string {
	if p.description == nil {
		return nil
	}

	d := *p.description

	return &d
}

// ProviderID returns the provider configuration's provider ID.
func (p *ProviderConfiguration) ProviderID() string { return p.providerID }

// Config returns a copy of the provider configuration's config.
func (p *ProviderConfiguration) Config() map[string]any { return cloneConfig(p.config) }

// Status returns the provider configuration's current status.
func (p *ProviderConfiguration) Status() ProviderConfigurationStatus { return p.status }

// Metadata returns a copy of the provider configuration's metadata.
func (p *ProviderConfiguration) Metadata() map[string]any {
	return cloneMetadata(p.metadata)
}

// CreatedAt returns when the provider configuration was created.
func (p *ProviderConfiguration) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns when the provider configuration was last updated.
func (p *ProviderConfiguration) UpdatedAt() time.Time { return p.updatedAt }

// IsActive returns true if the provider configuration status is active.
func (p *ProviderConfiguration) IsActive() bool {
	return p.status == ProviderConfigStatusActive
}

// IsDisabled returns true if the provider configuration status is disabled.
func (p *ProviderConfiguration) IsDisabled() bool {
	return p.status == ProviderConfigStatusDisabled
}

// Disable transitions the provider configuration from active to disabled status.
func (p *ProviderConfiguration) Disable() error {
	if p.status != ProviderConfigStatusActive {
		return ErrProviderConfigCannotDisable
	}

	p.status = ProviderConfigStatusDisabled
	p.updatedAt = time.Now().UTC()

	return nil
}

// Enable transitions the provider configuration from disabled to active status.
func (p *ProviderConfiguration) Enable() error {
	if p.status != ProviderConfigStatusDisabled {
		return ErrProviderConfigCannotEnable
	}

	p.status = ProviderConfigStatusActive
	p.updatedAt = time.Now().UTC()

	return nil
}

// Update modifies the provider configuration's mutable fields.
func (p *ProviderConfiguration) Update(
	name *string,
	description *string,
	config map[string]any,
) error {
	if name != nil {
		if *name == "" {
			return ErrProviderConfigNameRequired
		}

		if len(*name) > maxProviderConfigNameLength {
			return ErrProviderConfigNameTooLong
		}

		p.name = *name
	}

	if description != nil {
		if len(*description) > maxProviderConfigDescriptionLength {
			return ErrProviderConfigDescriptionTooLong
		}

		p.description = copyStringPtr(description)
	}

	if config != nil {
		if len(config) == 0 {
			return ErrProviderConfigConfigRequired
		}

		p.config = cloneConfig(config)
	}

	p.updatedAt = time.Now().UTC()

	return nil
}

// SetMetadata sets a metadata key-value pair.
func (p *ProviderConfiguration) SetMetadata(key string, value any) {
	if p.metadata == nil {
		p.metadata = make(map[string]any)
	}

	p.metadata[key] = value
	p.updatedAt = time.Now().UTC()
}

// copyStringPtr creates a defensive copy of a string pointer.
func copyStringPtr(s *string) *string {
	if s == nil {
		return nil
	}

	v := *s

	return &v
}

// cloneConfig creates a deep copy of a config map.
func cloneConfig(config map[string]any) map[string]any {
	if config == nil {
		return nil
	}

	result := make(map[string]any, len(config))
	for k, v := range config {
		result[k] = cloneConfigValue(v)
	}

	return result
}

func cloneConfigValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		return cloneConfig(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = cloneConfigValue(item)
		}

		return out
	default:
		return v
	}
}
