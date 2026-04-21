// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import "github.com/google/uuid"

// CreateExecutorConfigurationInput is the input DTO for creating an executor configuration.
// Uses validator tags per PROJECT_RULES.md.
type CreateExecutorConfigurationInput struct {
	Name           string                      `json:"name" validate:"required,min=1,max=100"`
	Description    *string                     `json:"description,omitempty" validate:"omitempty,max=500"`
	BaseURL        string                      `json:"baseUrl" validate:"required,url,max=500"`
	Endpoints      []ExecutorEndpointInput     `json:"endpoints" validate:"required,min=1,dive"`
	Authentication ExecutorAuthenticationInput `json:"authentication" validate:"required"`
	Metadata       map[string]any              `json:"metadata,omitempty" validate:"omitempty"`
}

// UpdateExecutorConfigurationInput is the input DTO for updating an executor configuration.
type UpdateExecutorConfigurationInput struct {
	Name           string                      `json:"name" validate:"required,min=1,max=100"`
	Description    *string                     `json:"description,omitempty" validate:"omitempty,max=500"`
	BaseURL        string                      `json:"baseUrl" validate:"required,url,max=500"`
	Endpoints      []ExecutorEndpointInput     `json:"endpoints" validate:"required,min=1,dive"`
	Authentication ExecutorAuthenticationInput `json:"authentication" validate:"required"`
	Metadata       map[string]any              `json:"metadata,omitempty" validate:"omitempty"`
}

// ExecutorEndpointInput is the input DTO for an executor endpoint.
type ExecutorEndpointInput struct {
	Name    string `json:"name" validate:"required,min=1,max=50"`
	Path    string `json:"path" validate:"required,min=1,max=200"`
	Method  string `json:"method" validate:"required,oneof=GET POST PUT PATCH DELETE HEAD OPTIONS"`
	Timeout int    `json:"timeout,omitempty" validate:"omitempty,min=1,max=300"`
}

// ExecutorAuthenticationInput is the input DTO for executor authentication configuration.
type ExecutorAuthenticationInput struct {
	Type   string         `json:"type" validate:"required,oneof=none api_key bearer basic oidc_client_credentials oidc_user"`
	Config map[string]any `json:"config,omitempty"`
}

// ToDomain converts CreateExecutorConfigurationInput to domain entity.
func (i *CreateExecutorConfigurationInput) ToDomain() (*ExecutorConfiguration, error) {
	endpoints := make([]ExecutorEndpoint, len(i.Endpoints))
	for idx, epInput := range i.Endpoints {
		ep, err := epInput.ToDomain()
		if err != nil {
			return nil, err
		}

		endpoints[idx] = *ep
	}

	auth, err := i.Authentication.ToDomain()
	if err != nil {
		return nil, err
	}

	executorConfig, err := NewExecutorConfiguration(
		i.Name,
		i.Description,
		i.BaseURL,
		endpoints,
		*auth,
	)
	if err != nil {
		return nil, err
	}

	// Set metadata if provided
	if i.Metadata != nil {
		for k, v := range i.Metadata {
			executorConfig.SetMetadata(k, v)
		}
	}

	return executorConfig, nil
}

// ToDomain converts ExecutorEndpointInput to domain entity.
func (i *ExecutorEndpointInput) ToDomain() (*ExecutorEndpoint, error) {
	return NewExecutorEndpoint(i.Name, i.Path, i.Method, i.Timeout)
}

// ToDomain converts ExecutorAuthenticationInput to domain entity.
func (i *ExecutorAuthenticationInput) ToDomain() (*ExecutorAuthentication, error) {
	return NewExecutorAuthentication(i.Type, i.Config)
}

// ToEndpoints converts UpdateExecutorConfigurationInput endpoints to domain entities.
func (i *UpdateExecutorConfigurationInput) ToEndpoints() ([]ExecutorEndpoint, error) {
	endpoints := make([]ExecutorEndpoint, len(i.Endpoints))
	for idx, epInput := range i.Endpoints {
		ep, err := epInput.ToDomain()
		if err != nil {
			return nil, err
		}

		endpoints[idx] = *ep
	}

	return endpoints, nil
}

// ToAuthentication converts UpdateExecutorConfigurationInput authentication to domain entity.
func (i *UpdateExecutorConfigurationInput) ToAuthentication() (*ExecutorAuthentication, error) {
	return i.Authentication.ToDomain()
}

// ExecutorConfigurationFilterInput is the input DTO for listing executor configurations with filters.
type ExecutorConfigurationFilterInput struct {
	Status    *string `query:"status" validate:"omitempty,oneof=unconfigured configured tested active disabled"`
	Limit     int     `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor    string  `query:"cursor"`
	SortBy    string  `query:"sortBy" validate:"omitempty,oneof=createdAt updatedAt name"`
	SortOrder string  `query:"sortOrder" validate:"omitempty,oneof=ASC DESC"`
}

// GetExecutorConfigurationInput is the input DTO for getting an executor configuration by ID.
type GetExecutorConfigurationInput struct {
	ID uuid.UUID `params:"id" validate:"required"`
}

// ExecutorConfigurationStatusInput is the input DTO for changing executor configuration status.
type ExecutorConfigurationStatusInput struct {
	ID uuid.UUID `params:"id" validate:"required"`
}
