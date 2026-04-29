// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package health_test

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/internal/adapters/http/in/health"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestHealthHandler_Health_SelfProbeGating_WhenFalse tests that /health returns 503
// when selfProbeOK is false (startup self-probe did not pass).
func TestHealthHandler_Health_SelfProbeGating_WhenFalse(t *testing.T) {
	// Arrange: Set self-probe function to return false
	originalFunc := health.SelfProbeOKFunc
	health.SelfProbeOKFunc = func() bool { return false }
	defer func() { health.SelfProbeOKFunc = originalFunc }()

	ctrl := gomock.NewController(t)
	mock := NewMockDatabaseChecker(ctrl)
	// Note: Database check methods should NOT be called when self-probe is false
	// The handler should short-circuit before checking dependencies

	handler, err := health.NewHealthHandler(mock)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health", handler.Health)

	// Act
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert: Should return 503 because self-probe failed
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode,
		"Health should return 503 when selfProbeOK is false")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "unhealthy", "Response should contain unhealthy status")
	assert.Contains(t, string(body), "self_probe", "Response should indicate self_probe check failed")
}

// TestHealthHandler_Health_SelfProbeGating_WhenTrue tests that /health proceeds with
// normal dependency checks when selfProbeOK is true.
func TestHealthHandler_Health_SelfProbeGating_WhenTrue(t *testing.T) {
	// Arrange: Set self-probe function to return true
	originalFunc := health.SelfProbeOKFunc
	health.SelfProbeOKFunc = func() bool { return true }
	defer func() { health.SelfProbeOKFunc = originalFunc }()

	ctrl := gomock.NewController(t)
	mock := NewMockDatabaseChecker(ctrl)
	// When self-probe passed, database check should be called
	mock.EXPECT().IsConnected().Return(true)
	mock.EXPECT().Ping(gomock.Any()).Return(nil)

	handler, err := health.NewHealthHandler(mock)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health", handler.Health)

	// Act
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert: Should return 200 because both self-probe and deps are healthy
	assert.Equal(t, fiber.StatusOK, resp.StatusCode,
		"Health should return 200 when selfProbeOK is true and deps are up")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "healthy", "Response should contain healthy status")
}

// TestHealthHandler_Health_SelfProbeGating_WhenTrueAndDepsDown tests that /health returns 503
// when self-probe passed but dependencies are down.
func TestHealthHandler_Health_SelfProbeGating_WhenTrueAndDepsDown(t *testing.T) {
	// Arrange: Set self-probe function to return true but deps are down
	originalFunc := health.SelfProbeOKFunc
	health.SelfProbeOKFunc = func() bool { return true }
	defer func() { health.SelfProbeOKFunc = originalFunc }()

	ctrl := gomock.NewController(t)
	mock := NewMockDatabaseChecker(ctrl)
	// Database is not connected
	mock.EXPECT().IsConnected().Return(false)

	handler, err := health.NewHealthHandler(mock)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health", handler.Health)

	// Act
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert: Should return 503 because deps are down
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode,
		"Health should return 503 when deps are down even if selfProbeOK is true")
}
