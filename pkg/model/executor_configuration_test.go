// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package model_test

import (
	"testing"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers
func createValidEndpoint(t *testing.T) model.ExecutorEndpoint {
	t.Helper()

	endpoint, err := model.NewExecutorEndpoint("validate", "/validate", "POST", 30)
	require.NoError(t, err)

	return *endpoint
}

func createValidAuth(t *testing.T) model.ExecutorAuthentication {
	t.Helper()

	auth, err := model.NewExecutorAuthentication("api_key", map[string]any{"key": "test-key"})
	require.NoError(t, err)

	return *auth
}

// ExecutorConfiguration tests
func TestNewExecutorConfiguration_Success(t *testing.T) {
	description := "Test executor configuration"
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)

	config, err := model.NewExecutorConfiguration(
		"Test Provider",
		&description,
		"https://api.example.com",
		endpoints,
		auth,
	)

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.NotEqual(t, uuid.Nil, config.ID())
	assert.Equal(t, "Test Provider", config.Name())
	assert.Equal(t, &description, config.Description())
	assert.Equal(t, "https://api.example.com", config.BaseURL())
	assert.Len(t, config.Endpoints(), 1)
	assert.Equal(t, auth, config.Authentication())
	assert.Equal(t, model.ExecutorConfigurationStatusUnconfigured, config.Status())
	assert.False(t, config.CreatedAt().IsZero())
	assert.False(t, config.UpdatedAt().IsZero())
	assert.Nil(t, config.LastTestedAt())
}

func TestNewExecutorConfiguration_WithoutDescription(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)

	config, err := model.NewExecutorConfiguration(
		"Test Provider",
		nil,
		"https://api.example.com",
		endpoints,
		auth,
	)

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Nil(t, config.Description())
}

func TestNewExecutorConfiguration_EmptyName(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)

	config, err := model.NewExecutorConfiguration(
		"",
		nil,
		"https://api.example.com",
		endpoints,
		auth,
	)

	require.ErrorIs(t, err, model.ErrExecutorConfigNameRequired)
	assert.Nil(t, config)
}

func TestNewExecutorConfiguration_NameTooLong(t *testing.T) {
	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)

	config, err := model.NewExecutorConfiguration(
		string(longName),
		nil,
		"https://api.example.com",
		endpoints,
		auth,
	)

	require.ErrorIs(t, err, model.ErrExecutorConfigNameTooLong)
	assert.Nil(t, config)
}

func TestNewExecutorConfiguration_EmptyBaseURL(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)

	config, err := model.NewExecutorConfiguration(
		"Test Provider",
		nil,
		"",
		endpoints,
		auth,
	)

	require.ErrorIs(t, err, model.ErrExecutorConfigBaseURLRequired)
	assert.Nil(t, config)
}

func TestNewExecutorConfiguration_InvalidBaseURL(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)

	config, err := model.NewExecutorConfiguration(
		"Test Provider",
		nil,
		"not-a-valid-url",
		endpoints,
		auth,
	)

	require.ErrorIs(t, err, model.ErrExecutorConfigBaseURLInvalid)
	assert.Nil(t, config)
}

func TestNewExecutorConfiguration_HTTPBaseURL(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)

	// HTTP URLs are now allowed for development/testing with mock servers
	config, err := model.NewExecutorConfiguration(
		"Test Provider",
		nil,
		"http://api.example.com",
		endpoints,
		auth,
	)

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "http://api.example.com", config.BaseURL())
}

func TestNewExecutorConfiguration_NoEndpoints(t *testing.T) {
	auth := createValidAuth(t)

	config, err := model.NewExecutorConfiguration(
		"Test Provider",
		nil,
		"https://api.example.com",
		nil,
		auth,
	)

	require.ErrorIs(t, err, model.ErrExecutorConfigEndpointsRequired)
	assert.Nil(t, config)
}

func TestNewExecutorConfiguration_EmptyEndpoints(t *testing.T) {
	auth := createValidAuth(t)

	config, err := model.NewExecutorConfiguration(
		"Test Provider",
		nil,
		"https://api.example.com",
		[]model.ExecutorEndpoint{},
		auth,
	)

	require.ErrorIs(t, err, model.ErrExecutorConfigEndpointsRequired)
	assert.Nil(t, config)
}

func TestNewExecutorConfiguration_BaseURLTrailingSlashRemoved(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)

	config, err := model.NewExecutorConfiguration(
		"Test Provider",
		nil,
		"https://api.example.com/",
		endpoints,
		auth,
	)

	require.NoError(t, err)
	require.NotNil(t, config)
	assert.Equal(t, "https://api.example.com", config.BaseURL())
}

// Status transition tests
func TestExecutorConfiguration_MarkConfigured_FromUnconfigured(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)

	err := config.MarkConfigured()

	require.NoError(t, err)
	assert.Equal(t, model.ExecutorConfigurationStatusConfigured, config.Status())
}

func TestExecutorConfiguration_MarkConfigured_FromConfigured(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)
	_ = config.MarkConfigured()

	err := config.MarkConfigured()

	require.ErrorIs(t, err, model.ErrExecutorConfigCannotMarkConfigured)
}

func TestExecutorConfiguration_MarkTested_FromConfigured(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)
	_ = config.MarkConfigured()

	err := config.MarkTested()

	require.NoError(t, err)
	assert.Equal(t, model.ExecutorConfigurationStatusTested, config.Status())
	assert.NotNil(t, config.LastTestedAt())
}

func TestExecutorConfiguration_MarkTested_FromUnconfigured(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)

	err := config.MarkTested()

	require.ErrorIs(t, err, model.ErrExecutorConfigCannotMarkTested)
}

func TestExecutorConfiguration_Activate_FromTested(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)
	_ = config.MarkConfigured()
	_ = config.MarkTested()

	err := config.Activate()

	require.NoError(t, err)
	assert.Equal(t, model.ExecutorConfigurationStatusActive, config.Status())
}

func TestExecutorConfiguration_Activate_FromUnconfigured(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)

	err := config.Activate()

	require.ErrorIs(t, err, model.ErrExecutorConfigCannotActivate)
}

func TestExecutorConfiguration_Disable_FromActive(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)
	_ = config.MarkConfigured()
	_ = config.MarkTested()
	_ = config.Activate()

	err := config.Disable()

	require.NoError(t, err)
	assert.Equal(t, model.ExecutorConfigurationStatusDisabled, config.Status())
}

func TestExecutorConfiguration_Disable_FromUnconfigured(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)

	err := config.Disable()

	require.ErrorIs(t, err, model.ErrExecutorConfigCannotDisable)
}

func TestExecutorConfiguration_Enable_FromDisabled(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)
	_ = config.MarkConfigured()
	_ = config.MarkTested()
	_ = config.Activate()
	_ = config.Disable()

	err := config.Enable()

	require.NoError(t, err)
	assert.Equal(t, model.ExecutorConfigurationStatusActive, config.Status())
}

func TestExecutorConfiguration_Enable_FromActive(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)
	_ = config.MarkConfigured()
	_ = config.MarkTested()
	_ = config.Activate()

	err := config.Enable()

	require.ErrorIs(t, err, model.ErrExecutorConfigCannotEnable)
}

// Update tests
func TestExecutorConfiguration_Update_Unconfigured(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)
	originalUpdatedAt := config.UpdatedAt()
	time.Sleep(1 * time.Millisecond)

	newDesc := "Updated description"
	newEndpoint, _ := model.NewExecutorEndpoint("new-endpoint", "/new", "GET", 60)
	newAuth, _ := model.NewExecutorAuthentication("bearer", map[string]any{"token": "new-token"})

	err := config.Update("Updated Name", &newDesc, "https://new-api.example.com", []model.ExecutorEndpoint{*newEndpoint}, *newAuth)

	require.NoError(t, err)
	assert.Equal(t, "Updated Name", config.Name())
	assert.Equal(t, &newDesc, config.Description())
	assert.Equal(t, "https://new-api.example.com", config.BaseURL())
	require.Len(t, config.Endpoints(), 1)
	assert.Equal(t, "new-endpoint", config.Endpoints()[0].Name())
	assert.True(t, config.UpdatedAt().After(originalUpdatedAt))
}

func TestExecutorConfiguration_Update_Configured(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)
	_ = config.MarkConfigured()

	err := config.Update("Updated Name", nil, "https://new-api.example.com", endpoints, auth)

	require.NoError(t, err)
	assert.Equal(t, "Updated Name", config.Name())
}

func TestExecutorConfiguration_Update_Active(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)
	_ = config.MarkConfigured()
	_ = config.MarkTested()
	_ = config.Activate()

	err := config.Update("Updated Name", nil, "https://new-api.example.com", endpoints, auth)

	require.ErrorIs(t, err, model.ErrExecutorConfigCannotUpdate)
}

// State check methods
func TestExecutorConfiguration_IsActive(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)

	assert.False(t, config.IsActive())

	_ = config.MarkConfigured()
	_ = config.MarkTested()
	_ = config.Activate()

	assert.True(t, config.IsActive())
}

func TestExecutorConfiguration_IsConfigured(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)

	assert.False(t, config.IsConfigured())

	_ = config.MarkConfigured()

	assert.True(t, config.IsConfigured())
}

func TestExecutorConfiguration_IsTested(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)

	assert.False(t, config.IsTested())

	_ = config.MarkConfigured()
	_ = config.MarkTested()

	assert.True(t, config.IsTested())
}

func TestExecutorConfiguration_IsDisabled(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)

	assert.False(t, config.IsDisabled())

	_ = config.MarkConfigured()
	_ = config.MarkTested()
	_ = config.Activate()
	_ = config.Disable()

	assert.True(t, config.IsDisabled())
}

// Metadata tests
func TestExecutorConfiguration_SetMetadata(t *testing.T) {
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", endpoints, auth)

	config.SetMetadata("key", "value")
	config.SetMetadata("number", 42)

	metadata := config.Metadata()
	assert.Equal(t, "value", metadata["key"])
	assert.Equal(t, 42, metadata["number"])
}

// GetEndpointByName tests
func TestExecutorConfiguration_GetEndpointByName(t *testing.T) {
	endpoint1, _ := model.NewExecutorEndpoint("validate", "/validate", "POST", 30)
	endpoint2, _ := model.NewExecutorEndpoint("check", "/check", "GET", 15)
	auth := createValidAuth(t)
	config, _ := model.NewExecutorConfiguration("Test", nil, "https://api.example.com", []model.ExecutorEndpoint{*endpoint1, *endpoint2}, auth)

	found := config.GetEndpointByName("validate")
	require.NotNil(t, found)
	assert.Equal(t, "validate", found.Name())
	assert.Equal(t, "/validate", found.Path())

	notFound := config.GetEndpointByName("nonexistent")
	assert.Nil(t, notFound)
}

// ExecutorEndpoint tests
func TestNewExecutorEndpoint_Success(t *testing.T) {
	endpoint, err := model.NewExecutorEndpoint("validate", "/validate", "POST", 30)

	require.NoError(t, err)
	assert.Equal(t, "validate", endpoint.Name())
	assert.Equal(t, "/validate", endpoint.Path())
	assert.Equal(t, "POST", endpoint.Method())
	assert.Equal(t, 30, endpoint.Timeout())
}

func TestNewExecutorEndpoint_DefaultTimeout(t *testing.T) {
	endpoint, err := model.NewExecutorEndpoint("validate", "/validate", "POST", 0)

	require.NoError(t, err)
	assert.Equal(t, 30, endpoint.Timeout()) // Default timeout
}

func TestNewExecutorEndpoint_EmptyName(t *testing.T) {
	_, err := model.NewExecutorEndpoint("", "/validate", "POST", 30)

	require.ErrorIs(t, err, model.ErrExecutorConfigEndpointNameRequired)
}

func TestNewExecutorEndpoint_EmptyPath(t *testing.T) {
	_, err := model.NewExecutorEndpoint("validate", "", "POST", 30)

	require.ErrorIs(t, err, model.ErrExecutorConfigEndpointPathRequired)
}

func TestNewExecutorEndpoint_EmptyMethod(t *testing.T) {
	_, err := model.NewExecutorEndpoint("validate", "/validate", "", 30)

	require.ErrorIs(t, err, model.ErrExecutorConfigEndpointMethodRequired)
}

func TestNewExecutorEndpoint_InvalidMethod(t *testing.T) {
	_, err := model.NewExecutorEndpoint("validate", "/validate", "INVALID", 30)

	require.ErrorIs(t, err, model.ErrExecutorConfigEndpointMethodInvalid)
}

func TestNewExecutorEndpoint_ValidMethods(t *testing.T) {
	validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

	for _, method := range validMethods {
		endpoint, err := model.NewExecutorEndpoint("test", "/test", method, 30)
		require.NoError(t, err, "Method %s should be valid", method)
		assert.Equal(t, method, endpoint.Method())
	}
}

func TestNewExecutorEndpoint_LowercaseMethodConverted(t *testing.T) {
	endpoint, err := model.NewExecutorEndpoint("test", "/test", "get", 30)

	require.NoError(t, err)
	assert.Equal(t, "GET", endpoint.Method())
}

// ExecutorAuthentication tests
func TestNewExecutorAuthentication_Success(t *testing.T) {
	authConfig := map[string]any{"key": "test-api-key"}

	auth, err := model.NewExecutorAuthentication("api_key", authConfig)

	require.NoError(t, err)
	assert.Equal(t, "api_key", auth.Type())
	assert.Equal(t, authConfig, auth.Config())
}

func TestNewExecutorAuthentication_EmptyType(t *testing.T) {
	_, err := model.NewExecutorAuthentication("", nil)

	require.ErrorIs(t, err, model.ErrExecutorConfigAuthTypeRequired)
}

func TestNewExecutorAuthentication_InvalidType(t *testing.T) {
	_, err := model.NewExecutorAuthentication("invalid_type", nil)

	require.ErrorIs(t, err, model.ErrExecutorConfigAuthTypeInvalid)
}

func TestNewExecutorAuthentication_ValidTypes(t *testing.T) {
	validTypes := []string{"none", "api_key", "bearer", "basic", "oidc_client_credentials", "oidc_user"}

	for _, authType := range validTypes {
		auth, err := model.NewExecutorAuthentication(authType, nil)
		require.NoError(t, err, "Auth type %s should be valid", authType)
		assert.Equal(t, authType, auth.Type())
	}
}

func TestNewExecutorAuthentication_NilConfig(t *testing.T) {
	auth, err := model.NewExecutorAuthentication("none", nil)

	require.NoError(t, err)
	assert.Nil(t, auth.Config())
}

// Status string tests
func TestExecutorConfigurationStatus_String(t *testing.T) {
	assert.Equal(t, "unconfigured", string(model.ExecutorConfigurationStatusUnconfigured))
	assert.Equal(t, "configured", string(model.ExecutorConfigurationStatusConfigured))
	assert.Equal(t, "tested", string(model.ExecutorConfigurationStatusTested))
	assert.Equal(t, "active", string(model.ExecutorConfigurationStatusActive))
	assert.Equal(t, "disabled", string(model.ExecutorConfigurationStatusDisabled))
}

// Reconstitution tests
func TestNewExecutorConfigurationFromDB(t *testing.T) {
	id := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testedAt := now.Add(-time.Hour)
	description := "Test description"
	endpoints := []model.ExecutorEndpoint{createValidEndpoint(t)}
	auth := createValidAuth(t)
	metadata := map[string]any{"key": "value"}

	config := model.NewExecutorConfigurationFromDB(
		id,
		"Test Provider",
		&description,
		"https://api.example.com",
		endpoints,
		auth,
		model.ExecutorConfigurationStatusActive,
		metadata,
		now,
		now,
		&testedAt,
	)

	assert.Equal(t, id, config.ID())
	assert.Equal(t, "Test Provider", config.Name())
	assert.Equal(t, &description, config.Description())
	assert.Equal(t, "https://api.example.com", config.BaseURL())
	assert.Len(t, config.Endpoints(), 1)
	assert.Equal(t, model.ExecutorConfigurationStatusActive, config.Status())
	assert.Equal(t, metadata, config.Metadata())
	assert.Equal(t, now, config.CreatedAt())
	assert.Equal(t, now, config.UpdatedAt())
	assert.Equal(t, &testedAt, config.LastTestedAt())
}

func TestNewExecutorEndpointFromDB(t *testing.T) {
	endpoint := model.NewExecutorEndpointFromDB("validate", "/validate", "POST", 30)

	assert.Equal(t, "validate", endpoint.Name())
	assert.Equal(t, "/validate", endpoint.Path())
	assert.Equal(t, "POST", endpoint.Method())
	assert.Equal(t, 30, endpoint.Timeout())
}

func TestNewExecutorAuthenticationFromDB(t *testing.T) {
	config := map[string]any{"key": "test-key"}
	auth := model.NewExecutorAuthenticationFromDB("api_key", config)

	assert.Equal(t, "api_key", auth.Type())
	assert.Equal(t, config, auth.Config())
}
