// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package health_test

import (
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/internal/adapters/http/in/health"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//go:generate go run go.uber.org/mock/mockgen@latest -source=handler.go -destination=mock_test.go -package=health_test

func TestNewHealthHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockDatabaseChecker(ctrl)

	handler, err := health.NewHealthHandler(mock)

	require.NoError(t, err)
	assert.NotNil(t, handler, "HealthHandler should not be nil")
}

func TestNewHealthHandler_WithNilDbChecker(t *testing.T) {
	handler, err := health.NewHealthHandler(nil)

	require.NoError(t, err)
	assert.NotNil(t, handler, "HealthHandler should not be nil even with nil dbChecker")
}

func TestHealthHandler_Liveness(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockDatabaseChecker(ctrl)

	handler, err := health.NewHealthHandler(mock)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health/live", handler.Liveness)

	req := httptest.NewRequest("GET", "/health/live", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Liveness should return 200 OK")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "status", "Response should contain status field")
	assert.Contains(t, string(body), "alive", "Response should contain alive status")
}

func TestHealthHandler_Liveness_WithNilDbChecker(t *testing.T) {
	handler, err := health.NewHealthHandler(nil)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health/live", handler.Liveness)

	req := httptest.NewRequest("GET", "/health/live", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	// Liveness should always return 200 regardless of database state
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Liveness should return 200 OK even with nil dbChecker")
}

func TestHealthHandler_Readiness_DatabaseConnected(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockDatabaseChecker(ctrl)

	mock.EXPECT().IsConnected().Return(true)
	mock.EXPECT().Ping(gomock.Any()).Return(nil)

	handler, err := health.NewHealthHandler(mock)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health/ready", handler.Readiness)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Readiness should return 200 when database connected")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "status", "Response should contain status field")
	assert.Contains(t, string(body), "ready", "Response should contain ready status")
}

func TestHealthHandler_Readiness_DatabaseNotConnected(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockDatabaseChecker(ctrl)

	mock.EXPECT().IsConnected().Return(false)

	handler, err := health.NewHealthHandler(mock)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health/ready", handler.Readiness)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode, "Readiness should return 503 when database disconnected")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "status", "Response should contain status field")
	assert.Contains(t, string(body), "not_ready", "Response should contain not_ready status")
}

func TestHealthHandler_Readiness_DatabasePingFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockDatabaseChecker(ctrl)

	mock.EXPECT().IsConnected().Return(true)
	mock.EXPECT().Ping(gomock.Any()).Return(errors.New("connection refused"))

	handler, err := health.NewHealthHandler(mock)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health/ready", handler.Readiness)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode, "Readiness should return 503 when database ping fails")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "not_ready", "Response should contain not_ready status")
	assert.Contains(t, string(body), "connection refused", "Response should contain error message")
}

func TestHealthHandler_Readiness_WithNilDbChecker(t *testing.T) {
	handler, err := health.NewHealthHandler(nil)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health/ready", handler.Readiness)

	req := httptest.NewRequest("GET", "/health/ready", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode, "Readiness should return 503 when dbChecker is nil")
}

func TestHealthHandler_Health_DatabaseConnected(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockDatabaseChecker(ctrl)

	mock.EXPECT().IsConnected().Return(true)
	mock.EXPECT().Ping(gomock.Any()).Return(nil)

	handler, err := health.NewHealthHandler(mock)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health", handler.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Health should return 200 when database connected")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "status", "Response should contain status")
	assert.Contains(t, string(body), "healthy", "Response should contain healthy status")
	assert.Contains(t, string(body), "version", "Response should contain version")
	assert.Contains(t, string(body), "uptime", "Response should contain uptime")
	assert.Contains(t, string(body), "checks", "Response should contain checks")
}

func TestHealthHandler_Health_DatabaseNotConnected(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := NewMockDatabaseChecker(ctrl)

	mock.EXPECT().IsConnected().Return(false)

	handler, err := health.NewHealthHandler(mock)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health", handler.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode, "Health should return 503 when database not connected")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "status", "Response should contain status")
	assert.Contains(t, string(body), "version", "Response should contain version")
	assert.Contains(t, string(body), "uptime", "Response should contain uptime")
	assert.Contains(t, string(body), "checks", "Response should contain checks")
}

func TestHealthHandler_Health_WithNilDbChecker(t *testing.T) {
	handler, err := health.NewHealthHandler(nil)
	require.NoError(t, err)

	app := fiber.New()
	app.Get("/health", handler.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode, "Health should return 503 when dbChecker is nil")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "status", "Response should contain status")
	assert.Contains(t, string(body), "unhealthy", "Response should contain unhealthy status")
}

func TestHealthResponse_Fields(t *testing.T) {
	response := health.HealthResponse{
		Status:  "healthy",
		Version: "1.0.0",
		Uptime:  "1h30m",
		Checks: map[string]health.CheckResult{
			"database": {
				Status:  "healthy",
				Message: "connection ok",
			},
		},
	}

	assert.Equal(t, "healthy", response.Status)
	assert.Equal(t, "1.0.0", response.Version)
	assert.Equal(t, "1h30m", response.Uptime)
	assert.NotNil(t, response.Checks)
	assert.Equal(t, "healthy", response.Checks["database"].Status)
	assert.Equal(t, "connection ok", response.Checks["database"].Message)
}

func TestCheckResult_Fields(t *testing.T) {
	result := health.CheckResult{
		Status:  "unhealthy",
		Message: "database ping failed",
	}

	assert.Equal(t, "unhealthy", result.Status)
	assert.Equal(t, "database ping failed", result.Message)
}
