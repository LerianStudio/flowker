// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package model contains domain entities and DTOs for Flowker.
package model

import (
	"net/url"
	"strings"
	"time"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/google/uuid"
)

// ExecutorConfigurationStatus represents the status of a executor configuration.
type ExecutorConfigurationStatus string

const (
	// ExecutorConfigurationStatusUnconfigured indicates an executor configuration that is registered but not fully configured.
	ExecutorConfigurationStatusUnconfigured ExecutorConfigurationStatus = "unconfigured"
	// ExecutorConfigurationStatusConfigured indicates an executor configuration that is configured but not tested.
	ExecutorConfigurationStatusConfigured ExecutorConfigurationStatus = "configured"
	// ExecutorConfigurationStatusTested indicates an executor configuration that has passed connectivity test.
	ExecutorConfigurationStatusTested ExecutorConfigurationStatus = "tested"
	// ExecutorConfigurationStatusActive indicates an executor configuration that is ready for use in workflows.
	ExecutorConfigurationStatusActive ExecutorConfigurationStatus = "active"
	// ExecutorConfigurationStatusDisabled indicates an executor configuration that is temporarily disabled.
	ExecutorConfigurationStatusDisabled ExecutorConfigurationStatus = "disabled"
)

// ExecutorConfiguration validation errors.
var (
	ErrExecutorConfigNameRequired = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigNameRequired.Error(),
		Message: "name is required",
	}
	ErrExecutorConfigNameTooLong = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigNameTooLong.Error(),
		Message: "name cannot exceed 100 characters",
	}
	ErrExecutorConfigBaseURLRequired = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigBaseURLRequired.Error(),
		Message: "base_url is required",
	}
	ErrExecutorConfigBaseURLInvalid = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigBaseURLInvalid.Error(),
		Message: "base_url must be a valid HTTP or HTTPS URL",
	}
	ErrExecutorConfigBaseURLTooLong = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigBaseURLTooLong.Error(),
		Message: "base_url cannot exceed 500 characters",
	}
	ErrExecutorConfigEndpointsRequired = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigEndpointsRequired.Error(),
		Message: "at least one endpoint is required",
	}
	ErrExecutorConfigAuthRequired = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigAuthRequired.Error(),
		Message: "authentication configuration is required",
	}
	ErrExecutorConfigCannotActivate = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigCannotModify.Error(),
		Message: "only tested executor configurations can be activated",
	}
	ErrExecutorConfigCannotDisable = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigCannotModify.Error(),
		Message: "only active executor configurations can be disabled",
	}
	ErrExecutorConfigCannotUpdate = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigCannotModify.Error(),
		Message: "can only update unconfigured or configured executor configurations",
	}
	ErrExecutorConfigCannotMarkTested = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigCannotModify.Error(),
		Message: "only configured executor configurations can be marked as tested",
	}
	ErrExecutorConfigDescriptionTooLong = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigDescriptionTooLong.Error(),
		Message: "description cannot exceed 500 characters",
	}
	ErrExecutorConfigCannotMarkConfigured = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigCannotModify.Error(),
		Message: "only unconfigured executor configurations can be marked as configured",
	}
	ErrExecutorConfigCannotEnable = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigCannotModify.Error(),
		Message: "only disabled executor configurations can be enabled",
	}
	ErrExecutorConfigEndpointNameRequired = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigEndpointNameRequired.Error(),
		Message: "endpoint name is required",
	}
	ErrExecutorConfigEndpointPathRequired = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigEndpointPathRequired.Error(),
		Message: "endpoint path is required",
	}
	ErrExecutorConfigEndpointMethodRequired = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigEndpointMethodRequired.Error(),
		Message: "endpoint method is required",
	}
	ErrExecutorConfigEndpointMethodInvalid = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigEndpointMethodInvalid.Error(),
		Message: "endpoint method must be a valid HTTP method",
	}
	ErrExecutorConfigAuthTypeRequired = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigAuthTypeRequired.Error(),
		Message: "authentication type is required",
	}
	ErrExecutorConfigAuthTypeInvalid = pkg.ValidationError{
		Code:    constant.ErrExecutorConfigAuthTypeInvalid.Error(),
		Message: "authentication type must be one of: none, api_key, bearer, basic, oidc_client_credentials, oidc_user",
	}
)

const (
	maxExecutorConfigNameLength        = 100
	maxExecutorConfigDescriptionLength = 500
	maxExecutorConfigBaseURLLength     = 500
)

// ExecutorEndpoint represents a single endpoint in a executor configuration.
type ExecutorEndpoint struct {
	name    string
	path    string
	method  string
	timeout int
}

// NewExecutorEndpoint creates a new ExecutorEndpoint with validation.
func NewExecutorEndpoint(name, path, method string, timeout int) (*ExecutorEndpoint, error) {
	if name == "" {
		return nil, ErrExecutorConfigEndpointNameRequired
	}

	if path == "" {
		return nil, ErrExecutorConfigEndpointPathRequired
	}

	if method == "" {
		return nil, ErrExecutorConfigEndpointMethodRequired
	}

	// Validate HTTP method
	validMethods := map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true,
	}
	if !validMethods[strings.ToUpper(method)] {
		return nil, ErrExecutorConfigEndpointMethodInvalid
	}

	if timeout <= 0 {
		timeout = 30 // default timeout
	}

	return &ExecutorEndpoint{
		name:    name,
		path:    path,
		method:  strings.ToUpper(method),
		timeout: timeout,
	}, nil
}

// NewExecutorEndpointFromDB reconstructs a ExecutorEndpoint from database values.
func NewExecutorEndpointFromDB(name, path, method string, timeout int) ExecutorEndpoint {
	return ExecutorEndpoint{
		name:    name,
		path:    path,
		method:  method,
		timeout: timeout,
	}
}

// Name returns the endpoint's name.
func (e ExecutorEndpoint) Name() string { return e.name }

// Path returns the endpoint's path.
func (e ExecutorEndpoint) Path() string { return e.path }

// Method returns the endpoint's HTTP method.
func (e ExecutorEndpoint) Method() string { return e.method }

// Timeout returns the endpoint's timeout in seconds.
func (e ExecutorEndpoint) Timeout() int { return e.timeout }

// ExecutorAuthentication represents the authentication configuration for an executor.
type ExecutorAuthentication struct {
	authType string
	config   map[string]any
}

// NewExecutorAuthentication creates a new ExecutorAuthentication with validation.
func NewExecutorAuthentication(authType string, config map[string]any) (*ExecutorAuthentication, error) {
	if authType == "" {
		return nil, ErrExecutorConfigAuthTypeRequired
	}

	// Validate auth type
	validTypes := map[string]bool{
		"none": true, "api_key": true, "bearer": true, "basic": true,
		"oidc_client_credentials": true, "oidc_user": true,
	}
	if !validTypes[authType] {
		return nil, ErrExecutorConfigAuthTypeInvalid
	}

	return &ExecutorAuthentication{
		authType: authType,
		config:   cloneAuthConfig(config),
	}, nil
}

// NewExecutorAuthenticationFromDB reconstructs a ExecutorAuthentication from database values.
func NewExecutorAuthenticationFromDB(authType string, config map[string]any) ExecutorAuthentication {
	return ExecutorAuthentication{
		authType: authType,
		config:   cloneAuthConfig(config),
	}
}

// Type returns the authentication type.
func (a ExecutorAuthentication) Type() string { return a.authType }

// Config returns a copy of the authentication config.
func (a ExecutorAuthentication) Config() map[string]any { return cloneAuthConfig(a.config) }

// cloneAuthConfig creates a defensive copy of an auth config map.
func cloneAuthConfig(config map[string]any) map[string]any {
	if config == nil {
		return nil
	}

	result := make(map[string]any, len(config))
	for k, v := range config {
		result[k] = v
	}

	return result
}

// ExecutorConfiguration represents a configured external executor (Rich Domain Model).
// Fields are private with validation in constructor per PROJECT_RULES.md.
type ExecutorConfiguration struct {
	id             uuid.UUID
	name           string
	description    *string
	baseURL        string
	endpoints      []ExecutorEndpoint
	authentication ExecutorAuthentication
	status         ExecutorConfigurationStatus
	metadata       map[string]any
	createdAt      time.Time
	updatedAt      time.Time
	lastTestedAt   *time.Time
}

// validateExecutorConfigData validates name, baseURL, and endpoints.
func validateExecutorConfigData(name, baseURL string, description *string, endpoints []ExecutorEndpoint) error {
	if name == "" {
		return ErrExecutorConfigNameRequired
	}

	if len(name) > maxExecutorConfigNameLength {
		return ErrExecutorConfigNameTooLong
	}

	if description != nil && len(*description) > maxExecutorConfigDescriptionLength {
		return ErrExecutorConfigDescriptionTooLong
	}

	if baseURL == "" {
		return ErrExecutorConfigBaseURLRequired
	}

	if len(baseURL) > maxExecutorConfigBaseURLLength {
		return ErrExecutorConfigBaseURLTooLong
	}

	// Validate URL format - allow both HTTP and HTTPS
	// HTTP is useful for development/testing with mock servers
	parsedURL, err := url.Parse(baseURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return ErrExecutorConfigBaseURLInvalid
	}

	if len(endpoints) == 0 {
		return ErrExecutorConfigEndpointsRequired
	}

	return nil
}

// NewExecutorConfiguration creates a new ExecutorConfiguration with validation.
func NewExecutorConfiguration(
	name string,
	description *string,
	baseURL string,
	endpoints []ExecutorEndpoint,
	authentication ExecutorAuthentication,
) (*ExecutorConfiguration, error) {
	if err := validateExecutorConfigData(name, baseURL, description, endpoints); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &ExecutorConfiguration{
		id:             uuid.New(),
		name:           name,
		description:    description,
		baseURL:        strings.TrimSuffix(baseURL, "/"),
		endpoints:      cloneEndpoints(endpoints),
		authentication: authentication,
		status:         ExecutorConfigurationStatusUnconfigured,
		metadata:       make(map[string]any),
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// NewExecutorConfigurationFromDB reconstructs a ExecutorConfiguration from database values.
func NewExecutorConfigurationFromDB(
	id uuid.UUID,
	name string,
	description *string,
	baseURL string,
	endpoints []ExecutorEndpoint,
	authentication ExecutorAuthentication,
	status ExecutorConfigurationStatus,
	metadata map[string]any,
	createdAt, updatedAt time.Time,
	lastTestedAt *time.Time,
) *ExecutorConfiguration {
	return &ExecutorConfiguration{
		id:             id,
		name:           name,
		description:    description,
		baseURL:        baseURL,
		endpoints:      cloneEndpoints(endpoints),
		authentication: authentication,
		status:         status,
		metadata:       cloneMetadata(metadata),
		createdAt:      createdAt,
		updatedAt:      updatedAt,
		lastTestedAt:   lastTestedAt,
	}
}

// ID returns the executor configuration's unique identifier.
func (p *ExecutorConfiguration) ID() uuid.UUID { return p.id }

// Name returns the executor configuration's name.
func (p *ExecutorConfiguration) Name() string { return p.name }

// Description returns the executor configuration's description.
func (p *ExecutorConfiguration) Description() *string { return p.description }

// BaseURL returns the executor configuration's base URL.
func (p *ExecutorConfiguration) BaseURL() string { return p.baseURL }

// Endpoints returns a copy of the executor configuration's endpoints.
func (p *ExecutorConfiguration) Endpoints() []ExecutorEndpoint {
	return cloneEndpoints(p.endpoints)
}

// Authentication returns the executor configuration's authentication configuration.
func (p *ExecutorConfiguration) Authentication() ExecutorAuthentication {
	return p.authentication
}

// Status returns the executor configuration's current status.
func (p *ExecutorConfiguration) Status() ExecutorConfigurationStatus { return p.status }

// Metadata returns a copy of the executor configuration's metadata.
func (p *ExecutorConfiguration) Metadata() map[string]any {
	return cloneMetadata(p.metadata)
}

// CreatedAt returns when the executor configuration was created.
func (p *ExecutorConfiguration) CreatedAt() time.Time { return p.createdAt }

// UpdatedAt returns when the executor configuration was last updated.
func (p *ExecutorConfiguration) UpdatedAt() time.Time { return p.updatedAt }

// LastTestedAt returns when the executor configuration was last tested.
func (p *ExecutorConfiguration) LastTestedAt() *time.Time { return p.lastTestedAt }

// IsActive returns true if the executor configuration status is active.
func (p *ExecutorConfiguration) IsActive() bool {
	return p.status == ExecutorConfigurationStatusActive
}

// IsConfigured returns true if the executor configuration status is configured.
func (p *ExecutorConfiguration) IsConfigured() bool {
	return p.status == ExecutorConfigurationStatusConfigured
}

// IsTested returns true if the executor configuration status is tested.
func (p *ExecutorConfiguration) IsTested() bool {
	return p.status == ExecutorConfigurationStatusTested
}

// IsDisabled returns true if the executor configuration status is disabled.
func (p *ExecutorConfiguration) IsDisabled() bool {
	return p.status == ExecutorConfigurationStatusDisabled
}

// MarkConfigured transitions the executor configuration from unconfigured to configured status.
func (p *ExecutorConfiguration) MarkConfigured() error {
	if p.status != ExecutorConfigurationStatusUnconfigured {
		return ErrExecutorConfigCannotMarkConfigured
	}

	p.status = ExecutorConfigurationStatusConfigured
	p.updatedAt = time.Now().UTC()

	return nil
}

// MarkTested transitions the executor configuration from configured to tested status.
func (p *ExecutorConfiguration) MarkTested() error {
	if p.status != ExecutorConfigurationStatusConfigured {
		return ErrExecutorConfigCannotMarkTested
	}

	now := time.Now().UTC()
	p.status = ExecutorConfigurationStatusTested
	p.updatedAt = now
	p.lastTestedAt = &now

	return nil
}

// Activate transitions the executor configuration from tested to active status.
func (p *ExecutorConfiguration) Activate() error {
	if p.status != ExecutorConfigurationStatusTested {
		return ErrExecutorConfigCannotActivate
	}

	p.status = ExecutorConfigurationStatusActive
	p.updatedAt = time.Now().UTC()

	return nil
}

// Disable transitions the executor configuration from active to disabled status.
func (p *ExecutorConfiguration) Disable() error {
	if p.status != ExecutorConfigurationStatusActive {
		return ErrExecutorConfigCannotDisable
	}

	p.status = ExecutorConfigurationStatusDisabled
	p.updatedAt = time.Now().UTC()

	return nil
}

// Enable transitions the executor configuration from disabled back to active status.
func (p *ExecutorConfiguration) Enable() error {
	if p.status != ExecutorConfigurationStatusDisabled {
		return ErrExecutorConfigCannotEnable
	}

	p.status = ExecutorConfigurationStatusActive
	p.updatedAt = time.Now().UTC()

	return nil
}

// Update modifies the executor configuration's configuration.
// Only unconfigured or configured executor configurations can be updated.
func (p *ExecutorConfiguration) Update(
	name string,
	description *string,
	baseURL string,
	endpoints []ExecutorEndpoint,
	authentication ExecutorAuthentication,
) error {
	if p.status != ExecutorConfigurationStatusUnconfigured && p.status != ExecutorConfigurationStatusConfigured {
		return ErrExecutorConfigCannotUpdate
	}

	if err := validateExecutorConfigData(name, baseURL, description, endpoints); err != nil {
		return err
	}

	p.name = name
	p.description = description
	p.baseURL = strings.TrimSuffix(baseURL, "/")
	p.endpoints = cloneEndpoints(endpoints)
	p.authentication = authentication
	p.updatedAt = time.Now().UTC()

	return nil
}

// SetMetadata sets a metadata key-value pair.
func (p *ExecutorConfiguration) SetMetadata(key string, value any) {
	if p.metadata == nil {
		p.metadata = make(map[string]any)
	}

	p.metadata[key] = value
	p.updatedAt = time.Now().UTC()
}

// GetEndpointByName returns an endpoint by name, or nil if not found.
func (p *ExecutorConfiguration) GetEndpointByName(name string) *ExecutorEndpoint {
	for _, e := range p.endpoints {
		if e.name == name {
			endpoint := e
			return &endpoint
		}
	}

	return nil
}

// cloneEndpoints creates a defensive copy of an endpoints slice.
func cloneEndpoints(endpoints []ExecutorEndpoint) []ExecutorEndpoint {
	if endpoints == nil {
		return nil
	}

	result := make([]ExecutorEndpoint, len(endpoints))
	copy(result, endpoints)

	return result
}
