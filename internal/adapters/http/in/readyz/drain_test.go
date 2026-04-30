// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package readyz_test

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/internal/adapters/http/in/readyz"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadyzHandler_DrainState_WhenDraining tests that /readyz returns 503
// immediately when draining state is true (graceful shutdown in progress).
func TestReadyzHandler_DrainState_WhenDraining(t *testing.T) {
	// Arrange: Set draining function to return true
	originalFunc := readyz.IsDrainingFunc
	readyz.IsDrainingFunc = func() bool { return true }
	defer func() { readyz.IsDrainingFunc = originalFunc }()

	// Even with healthy dependencies, should return 503 when draining
	mongoChecker := &mockHealthChecker{
		name:       "mongodb",
		pingErr:    nil,
		tlsEnabled: true,
	}

	handler := readyz.NewHandler(
		readyz.WithChecker(mongoChecker),
		readyz.WithVersion("1.2.3"),
		readyz.WithDeploymentMode("local"),
	)

	app := fiber.New()
	app.Get("/readyz", handler.Readyz)

	// Act
	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert: Status code should be 503 due to draining state
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode,
		"Should return 503 when draining state is true")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response readyz.Response
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	// Status should be "unhealthy" when draining
	assert.Equal(t, "unhealthy", response.Status,
		"Top-level status should be 'unhealthy' when draining")
}

// TestReadyzHandler_DrainState_WhenNotDraining tests that /readyz proceeds with
// normal dependency checks when draining state is false.
func TestReadyzHandler_DrainState_WhenNotDraining(t *testing.T) {
	// Arrange: Set draining function to return false
	originalFunc := readyz.IsDrainingFunc
	readyz.IsDrainingFunc = func() bool { return false }
	defer func() { readyz.IsDrainingFunc = originalFunc }()

	mongoChecker := &mockHealthChecker{
		name:       "mongodb",
		pingErr:    nil,
		tlsEnabled: true,
	}

	handler := readyz.NewHandler(
		readyz.WithChecker(mongoChecker),
		readyz.WithVersion("1.2.3"),
		readyz.WithDeploymentMode("local"),
	)

	app := fiber.New()
	app.Get("/readyz", handler.Readyz)

	// Act
	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert: Should return 200 when not draining and deps are up
	assert.Equal(t, fiber.StatusOK, resp.StatusCode,
		"Should return 200 when draining state is false and deps are up")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response readyz.Response
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response.Status,
		"Top-level status should be 'healthy' when not draining and deps are up")
}

// TestReadyzHandler_DrainState_ShortCircuits tests that drain check happens
// BEFORE dependency checks (no deps should be pinged when draining).
func TestReadyzHandler_DrainState_ShortCircuits(t *testing.T) {
	// Arrange: Set draining function to return true
	originalFunc := readyz.IsDrainingFunc
	readyz.IsDrainingFunc = func() bool { return true }
	defer func() { readyz.IsDrainingFunc = originalFunc }()

	// Use a checker that would fail - but it should never be called
	checker := &mockHealthChecker{
		name:       "should-not-be-checked",
		pingErr:    nil, // Would return healthy if called
		tlsEnabled: false,
	}

	handler := readyz.NewHandler(
		readyz.WithChecker(checker),
		readyz.WithVersion("1.2.3"),
		readyz.WithDeploymentMode("local"),
	)

	app := fiber.New()
	app.Get("/readyz", handler.Readyz)

	// Act
	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert: Should return 503 immediately
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode,
		"Should return 503 immediately when draining")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response readyz.Response
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	// Checks map should be empty because drain check short-circuits
	assert.Empty(t, response.Checks,
		"Checks map should be empty when drain short-circuits")
}

// TestReadyzHandler_DrainState_PreservesVersionAndDeploymentMode tests that
// even when draining, version and deployment_mode are included in response.
func TestReadyzHandler_DrainState_PreservesVersionAndDeploymentMode(t *testing.T) {
	// Arrange: Set draining function to return true
	originalFunc := readyz.IsDrainingFunc
	readyz.IsDrainingFunc = func() bool { return true }
	defer func() { readyz.IsDrainingFunc = originalFunc }()

	handler := readyz.NewHandler(
		readyz.WithVersion("2.0.0-drain-test"),
		readyz.WithDeploymentMode("saas"),
	)

	app := fiber.New()
	app.Get("/readyz", handler.Readyz)

	// Act
	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response readyz.Response
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	// Assert: Version and deployment mode should still be present
	assert.Equal(t, "2.0.0-drain-test", response.Version,
		"Version should be preserved when draining")
	assert.Equal(t, "saas", response.DeploymentMode,
		"DeploymentMode should be preserved when draining")
}
