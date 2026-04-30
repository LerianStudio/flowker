// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package readyz_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/internal/adapters/http/in/readyz"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockHealthChecker implements readyz.HealthChecker for testing
type mockHealthChecker struct {
	name       string
	pingErr    error
	tlsEnabled bool
}

func (m *mockHealthChecker) Name() string {
	return m.name
}

func (m *mockHealthChecker) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockHealthChecker) IsTLSEnabled() bool {
	return m.tlsEnabled
}

func TestReadyzHandler_AllDepsUp_Returns200WithCorrectShape(t *testing.T) {
	// Arrange: All dependencies are healthy
	mongoChecker := &mockHealthChecker{
		name:       "mongodb",
		pingErr:    nil,
		tlsEnabled: true,
	}
	postgresChecker := &mockHealthChecker{
		name:       "postgresql",
		pingErr:    nil,
		tlsEnabled: false,
	}

	handler := readyz.NewHandler(
		readyz.WithChecker(mongoChecker),
		readyz.WithChecker(postgresChecker),
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

	// Assert: Status code is 200
	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "Should return 200 when all deps are up")

	// Assert: Response shape matches canonical contract
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response readyz.Response
	err = json.Unmarshal(body, &response)
	require.NoError(t, err, "Response should be valid JSON")

	// Verify canonical contract fields
	assert.Equal(t, "healthy", response.Status, "Top-level status should be 'healthy'")
	assert.Equal(t, "1.2.3", response.Version, "Version should be set")
	assert.Equal(t, "local", response.DeploymentMode, "Deployment mode should be set")
	assert.NotNil(t, response.Checks, "Checks map should not be nil")

	// Verify MongoDB check
	mongoCheck, ok := response.Checks["mongodb"]
	require.True(t, ok, "Should have mongodb check")
	assert.Equal(t, "up", mongoCheck.Status, "MongoDB status should be 'up'")
	assert.True(t, mongoCheck.TLS, "MongoDB TLS should be true")
	assert.GreaterOrEqual(t, mongoCheck.LatencyMs, int64(0), "LatencyMs should be set when up")

	// Verify PostgreSQL check
	pgCheck, ok := response.Checks["postgresql"]
	require.True(t, ok, "Should have postgresql check")
	assert.Equal(t, "up", pgCheck.Status, "PostgreSQL status should be 'up'")
	assert.False(t, pgCheck.TLS, "PostgreSQL TLS should be false")
}

func TestReadyzHandler_OneDepDown_Returns503(t *testing.T) {
	// Arrange: MongoDB is up, PostgreSQL is down
	mongoChecker := &mockHealthChecker{
		name:       "mongodb",
		pingErr:    nil,
		tlsEnabled: true,
	}
	postgresChecker := &mockHealthChecker{
		name:       "postgresql",
		pingErr:    errors.New("connection refused"),
		tlsEnabled: false,
	}

	handler := readyz.NewHandler(
		readyz.WithChecker(mongoChecker),
		readyz.WithChecker(postgresChecker),
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

	// Assert: Status code is 503
	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode, "Should return 503 when any dep is down")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var response readyz.Response
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)

	// Top-level status should be unhealthy
	assert.Equal(t, "unhealthy", response.Status, "Top-level status should be 'unhealthy'")

	// MongoDB should be up
	mongoCheck, ok := response.Checks["mongodb"]
	require.True(t, ok, "Should have mongodb check")
	assert.Equal(t, "up", mongoCheck.Status)

	// PostgreSQL should be down with error
	pgCheck, ok := response.Checks["postgresql"]
	require.True(t, ok, "Should have postgresql check")
	assert.Equal(t, "down", pgCheck.Status, "PostgreSQL status should be 'down'")
	assert.Contains(t, pgCheck.Error, "connection refused", "Should include error message")
}

func TestReadyzHandler_ResponseIncludesVersionAndDeploymentMode(t *testing.T) {
	// Arrange
	handler := readyz.NewHandler(
		readyz.WithVersion("2.0.0-beta"),
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

	// Assert
	assert.Equal(t, "2.0.0-beta", response.Version, "Version should match")
	assert.Equal(t, "saas", response.DeploymentMode, "Deployment mode should match")
}

func TestReadyzHandler_NoAuthRequired_NeverReturnsUnauthorized(t *testing.T) {
	// This test verifies that /readyz is a public endpoint (mounted BEFORE auth middleware)
	// by checking that it never returns 401/403, regardless of authentication state.
	testCases := []struct {
		name       string
		authHeader string
		depUp      bool
	}{
		{"no auth header, deps up", "", true},
		{"no auth header, deps down", "", false},
		{"invalid auth header, deps up", "Bearer invalid-token", true},
		{"invalid auth header, deps down", "Bearer invalid-token", false},
		{"malformed auth header, deps up", "malformed", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checker := &mockHealthChecker{
				name: "mongodb",
			}
			if !tc.depUp {
				checker.pingErr = errors.New("down")
			}

			handler := readyz.NewHandler(
				readyz.WithChecker(checker),
			)

			app := fiber.New()
			app.Get("/readyz", handler.Readyz)

			req := httptest.NewRequest("GET", "/readyz", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer resp.Body.Close()

			// Should NEVER return 401 or 403
			assert.NotEqual(t, fiber.StatusUnauthorized, resp.StatusCode, "Should never return 401")
			assert.NotEqual(t, fiber.StatusForbidden, resp.StatusCode, "Should never return 403")

			// Should only return 200 or 503
			assert.True(t,
				resp.StatusCode == fiber.StatusOK || resp.StatusCode == fiber.StatusServiceUnavailable,
				"Should return 200 or 503, got %d", resp.StatusCode)
		})
	}
}

func TestReadyzHandler_ReportsCorrectTLSPosture(t *testing.T) {
	// Arrange: Mix of TLS enabled/disabled dependencies
	tlsChecker := &mockHealthChecker{
		name:       "secure-db",
		pingErr:    nil,
		tlsEnabled: true,
	}
	noTLSChecker := &mockHealthChecker{
		name:       "insecure-db",
		pingErr:    nil,
		tlsEnabled: false,
	}

	handler := readyz.NewHandler(
		readyz.WithChecker(tlsChecker),
		readyz.WithChecker(noTLSChecker),
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

	// Assert TLS posture is correctly reported
	secureCheck := response.Checks["secure-db"]
	assert.True(t, secureCheck.TLS, "Secure DB should report TLS=true")

	insecureCheck := response.Checks["insecure-db"]
	assert.False(t, insecureCheck.TLS, "Insecure DB should report TLS=false")
}

func TestReadyzHandler_StatusVocabulary(t *testing.T) {
	// Test the five valid status values: up, down, degraded, skipped, n/a
	testCases := []struct {
		name           string
		checkStatus    string
		expectedStatus string
		expectedHTTP   int
	}{
		{"up status", "up", "healthy", fiber.StatusOK},
		{"skipped status", "skipped", "healthy", fiber.StatusOK},
		{"n/a status", "n/a", "healthy", fiber.StatusOK},
		{"down status", "down", "unhealthy", fiber.StatusServiceUnavailable},
		{"degraded status", "degraded", "unhealthy", fiber.StatusServiceUnavailable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var checker readyz.HealthChecker

			switch tc.checkStatus {
			case "up":
				checker = &mockHealthChecker{name: "test-dep", pingErr: nil}
			case "down":
				checker = &mockHealthChecker{name: "test-dep", pingErr: errors.New("connection failed")}
			case "skipped":
				checker = &mockSkippedChecker{name: "test-dep", reason: "FEATURE_DISABLED=true"}
			case "n/a":
				checker = &mockNAChecker{name: "test-dep", reason: "multi-tenant: see /readyz/tenant/:id"}
			case "degraded":
				checker = &mockDegradedChecker{name: "test-dep", breakerState: "half-open"}
			}

			handler := readyz.NewHandler(readyz.WithChecker(checker))

			app := fiber.New()
			app.Get("/readyz", handler.Readyz)

			req := httptest.NewRequest("GET", "/readyz", nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.expectedHTTP, resp.StatusCode)

			body, _ := io.ReadAll(resp.Body)
			var response readyz.Response
			_ = json.Unmarshal(body, &response)

			assert.Equal(t, tc.expectedStatus, response.Status)
		})
	}
}

func TestReadyzHandler_SkippedRequiresReason(t *testing.T) {
	checker := &mockSkippedChecker{
		name:   "optional-cache",
		reason: "REDIS_ENABLED=false",
	}

	handler := readyz.NewHandler(readyz.WithChecker(checker))

	app := fiber.New()
	app.Get("/readyz", handler.Readyz)

	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var response readyz.Response
	_ = json.Unmarshal(body, &response)

	check := response.Checks["optional-cache"]
	assert.Equal(t, "skipped", check.Status)
	assert.Equal(t, "REDIS_ENABLED=false", check.Reason, "Skipped status MUST include reason")
}

func TestReadyzHandler_NARequiresReason(t *testing.T) {
	checker := &mockNAChecker{
		name:   "tenant-db",
		reason: "multi-tenant: see /readyz/tenant/:id",
	}

	handler := readyz.NewHandler(readyz.WithChecker(checker))

	app := fiber.New()
	app.Get("/readyz", handler.Readyz)

	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var response readyz.Response
	_ = json.Unmarshal(body, &response)

	check := response.Checks["tenant-db"]
	assert.Equal(t, "n/a", check.Status)
	assert.Equal(t, "multi-tenant: see /readyz/tenant/:id", check.Reason, "N/A status MUST include reason")
}

func TestReadyzHandler_LatencyMsOmittedForSkippedAndNA(t *testing.T) {
	skippedChecker := &mockSkippedChecker{name: "skipped-dep", reason: "disabled"}
	naChecker := &mockNAChecker{name: "na-dep", reason: "not applicable"}

	handler := readyz.NewHandler(
		readyz.WithChecker(skippedChecker),
		readyz.WithChecker(naChecker),
	)

	app := fiber.New()
	app.Get("/readyz", handler.Readyz)

	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Parse as raw JSON to check omitted fields
	var raw map[string]interface{}
	_ = json.Unmarshal(body, &raw)

	checks := raw["checks"].(map[string]interface{})

	// Skipped check should NOT have latency_ms
	skippedCheck := checks["skipped-dep"].(map[string]interface{})
	_, hasLatency := skippedCheck["latency_ms"]
	assert.False(t, hasLatency, "Skipped status should NOT have latency_ms")

	// N/A check should NOT have latency_ms
	naCheck := checks["na-dep"].(map[string]interface{})
	_, hasLatency = naCheck["latency_ms"]
	assert.False(t, hasLatency, "N/A status should NOT have latency_ms")
}

func TestReadyzHandler_NoDepsReturnsHealthy(t *testing.T) {
	// Edge case: no checkers configured
	handler := readyz.NewHandler(
		readyz.WithVersion("1.0.0"),
		readyz.WithDeploymentMode("local"),
	)

	app := fiber.New()
	app.Get("/readyz", handler.Readyz)

	req := httptest.NewRequest("GET", "/readyz", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode, "No deps should return 200")

	body, _ := io.ReadAll(resp.Body)
	var response readyz.Response
	_ = json.Unmarshal(body, &response)

	assert.Equal(t, "healthy", response.Status)
	assert.Empty(t, response.Checks)
}

// --- Mock implementations for special status types ---

// mockSkippedChecker implements HealthChecker that always returns "skipped" status
type mockSkippedChecker struct {
	name   string
	reason string
}

func (m *mockSkippedChecker) Name() string { return m.name }
func (m *mockSkippedChecker) Ping(ctx context.Context) error {
	return &readyz.SkippedError{Reason: m.reason}
}
func (m *mockSkippedChecker) IsTLSEnabled() bool { return false }

// mockNAChecker implements HealthChecker that always returns "n/a" status
type mockNAChecker struct {
	name   string
	reason string
}

func (m *mockNAChecker) Name() string { return m.name }
func (m *mockNAChecker) Ping(ctx context.Context) error {
	return &readyz.NotApplicableError{Reason: m.reason}
}
func (m *mockNAChecker) IsTLSEnabled() bool { return false }

// mockDegradedChecker implements HealthChecker that returns "degraded" status
type mockDegradedChecker struct {
	name         string
	breakerState string
}

func (m *mockDegradedChecker) Name() string { return m.name }
func (m *mockDegradedChecker) Ping(ctx context.Context) error {
	return &readyz.DegradedError{BreakerState: m.breakerState}
}
func (m *mockDegradedChecker) IsTLSEnabled() bool { return false }
