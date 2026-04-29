//go:build unit

// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	authMiddleware "github.com/LerianStudio/lib-auth/v2/auth/middleware"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	"github.com/gofiber/fiber/v2"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// testLogger is a minimal log.Logger implementation for tests that discards all output.
type testLogger struct{}

func (l *testLogger) Log(_ context.Context, _ libLog.Level, _ string, _ ...libLog.Field) {}
func (l *testLogger) With(_ ...libLog.Field) libLog.Logger                               { return l }
func (l *testLogger) WithGroup(_ string) libLog.Logger                                   { return l }
func (l *testLogger) Enabled(_ libLog.Level) bool                                        { return false }
func (l *testLogger) Sync(_ context.Context) error                                       { return nil }

// createTestJWT builds a signed JWT string for testing.
// lib-auth's checkAuthorization uses ParseUnverified, so the signing key is irrelevant.
func createTestJWT(claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		panic("failed to sign test JWT: " + err.Error())
	}

	return signed
}

// newTestAuthGuard creates an AuthGuard backed by a real AuthClient pointing
// at fakeAuthServerURL (when plugin auth is enabled).
func newTestAuthGuard(t *testing.T, cfg AuthGuardConfig, fakeAuthServerURL string) *AuthGuard {
	t.Helper()

	address := ""
	if cfg.PluginAuthEnabled && fakeAuthServerURL != "" {
		address = fakeAuthServerURL
	}

	var logger libLog.Logger = &testLogger{}
	authClient := authMiddleware.NewAuthClient(address, cfg.PluginAuthEnabled, &logger)

	return NewAuthGuard(cfg, authClient)
}

// newTestApp creates a Fiber app with a single GET /test route protected by the given handler.
func newTestApp(authHandler fiber.Handler) *fiber.App {
	app := fiber.New()
	app.Get("/test", authHandler, func(c *fiber.Ctx) error {
		return c.SendString("success")
	})

	return app
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestNewAuthGuard_Disabled verifies that when PluginAuthEnabled=false and
// APIKeyEnabled=false, requests pass through without any authentication.
func TestNewAuthGuard_Disabled(t *testing.T) {
	t.Parallel()

	guard := newTestAuthGuard(t, AuthGuardConfig{
		APIKey:            "",
		APIKeyEnabled:     false,
		PluginAuthEnabled: false,
		AppName:           "flowker",
	}, "")

	require.NotNil(t, guard)

	app := newTestApp(guard.Protect("workflows", "manage"))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Authorization header, no API key.

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"with all auth disabled, request should pass through")
	assert.Equal(t, "success", string(body))
}

// TestNewAuthGuard_Enabled_MissingToken verifies that when PluginAuthEnabled=true,
// a request without a Bearer token is rejected with 401 "Missing Token".
func TestNewAuthGuard_Enabled_MissingToken(t *testing.T) {
	t.Parallel()

	// Mock auth server that returns 401 for any request.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("healthy"))
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockServer.Close()

	guard := newTestAuthGuard(t, AuthGuardConfig{
		APIKey:            "",
		APIKeyEnabled:     false,
		PluginAuthEnabled: true,
		AppName:           "flowker",
	}, mockServer.URL)

	require.NotNil(t, guard)

	app := newTestApp(guard.Protect("workflows", "manage"))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Authorization header.

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"missing token should return 401")
	assert.Equal(t, "Missing Token", string(body),
		"response body should be 'Missing Token'")
}

// TestNewAuthGuard_Enabled_ValidToken verifies that when PluginAuthEnabled=true
// and the Access Manager returns authorized=true, the request passes through.
func TestNewAuthGuard_Enabled_ValidToken(t *testing.T) {
	t.Parallel()

	// Mock auth server that returns authorized=true.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("healthy"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := authMiddleware.AuthResponse{Authorized: true}

		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	guard := newTestAuthGuard(t, AuthGuardConfig{
		APIKey:            "",
		APIKeyEnabled:     false,
		PluginAuthEnabled: true,
		AppName:           "flowker",
	}, mockServer.URL)

	require.NotNil(t, guard)

	app := newTestApp(guard.Protect("workflows", "manage"))

	// Create a valid JWT with the claims lib-auth expects.
	token := createTestJWT(jwt.MapClaims{
		"type":  "application",
		"sub":   "flowker-service",
		"owner": "lerian",
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"valid token with authorized=true should pass through")
	assert.Equal(t, "success", string(body))
}

// TestNewAuthGuard_Enabled_UnauthorizedToken verifies that when PluginAuthEnabled=true
// and the Access Manager returns authorized=false, the request is rejected with 403.
func TestNewAuthGuard_Enabled_UnauthorizedToken(t *testing.T) {
	t.Parallel()

	// Mock auth server that returns authorized=false.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("healthy"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := authMiddleware.AuthResponse{Authorized: false}

		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	guard := newTestAuthGuard(t, AuthGuardConfig{
		APIKey:            "",
		APIKeyEnabled:     false,
		PluginAuthEnabled: true,
		AppName:           "flowker",
	}, mockServer.URL)

	require.NotNil(t, guard)

	app := newTestApp(guard.Protect("workflows", "manage"))

	// Create a valid JWT.
	token := createTestJWT(jwt.MapClaims{
		"type":  "application",
		"sub":   "flowker-service",
		"owner": "lerian",
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusForbidden, resp.StatusCode,
		"unauthorized token should return 403")
	assert.Equal(t, "Forbidden", string(body),
		"response body should be 'Forbidden'")
}

// TestNewAuthGuard_ReturnsNilWhenPluginEnabledAndClientNil verifies that
// NewAuthGuard returns nil when PluginAuthEnabled=true but authClient is nil.
func TestNewAuthGuard_ReturnsNilWhenPluginEnabledAndClientNil(t *testing.T) {
	t.Parallel()

	guard := NewAuthGuard(AuthGuardConfig{
		APIKey:            "test-key",
		APIKeyEnabled:     true,
		PluginAuthEnabled: true,
		AppName:           "flowker",
	}, nil)

	assert.Nil(t, guard,
		"Expected nil guard when PluginAuthEnabled=true and authClient is nil")
}

// TestAuthGuard_PluginAuthPriority verifies that plugin auth takes priority
// over API key auth. When both are enabled, Protect() uses plugin auth.
func TestAuthGuard_PluginAuthPriority(t *testing.T) {
	t.Parallel()

	// A mock server that returns 403 (Forbidden) for all non-health requests.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("healthy"))
			return
		}

		w.WriteHeader(http.StatusForbidden)
	}))
	defer mockServer.Close()

	guard := newTestAuthGuard(t, AuthGuardConfig{
		APIKey:            "valid-key",
		APIKeyEnabled:     true,
		PluginAuthEnabled: true,
		AppName:           "flowker",
	}, mockServer.URL)

	require.NotNil(t, guard)

	app := newTestApp(guard.Protect("workflows", "manage"))

	// Send only an API key (no Bearer token). If plugin auth is used, it will
	// respond with "Missing Token" rather than accepting the API key.
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(HeaderAPIKey, "valid-key")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"plugin auth should take priority and reject the request without Bearer token")
	assert.Equal(t, "Missing Token", string(body),
		"should get plugin auth's Missing Token, not API key auth pass-through")
}

// ---------------------------------------------------------------------------
// AuthGuard.With() tests — forceAPIKeyAuth behavior
// ---------------------------------------------------------------------------

// TestAuthGuard_With_ForceAPIKey_ValidKey verifies that With(forceAPIKeyAuth=true)
// bypasses plugin auth and accepts a valid API key.
func TestAuthGuard_With_ForceAPIKey_ValidKey(t *testing.T) {
	t.Parallel()

	guard := newTestAuthGuard(t, AuthGuardConfig{
		APIKey:            "webhook-secret-key",
		APIKeyEnabled:     true,
		PluginAuthEnabled: true,
		AppName:           "flowker",
	}, "")

	app := fiber.New()
	app.Post("/v1/webhooks/*", guard.With("webhooks", "execute", true), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/hooks/order", nil)
	req.Header.Set(HeaderAPIKey, "webhook-secret-key")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode,
		"valid API key should pass through even with plugin auth enabled")
}

// TestAuthGuard_With_ForceAPIKey_MissingKey verifies that With(forceAPIKeyAuth=true)
// rejects requests without an API key.
func TestAuthGuard_With_ForceAPIKey_MissingKey(t *testing.T) {
	t.Parallel()

	guard := newTestAuthGuard(t, AuthGuardConfig{
		APIKey:            "webhook-secret-key",
		APIKeyEnabled:     true,
		PluginAuthEnabled: true,
		AppName:           "flowker",
	}, "")

	app := fiber.New()
	app.Post("/v1/webhooks/*", guard.With("webhooks", "execute", true), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/hooks/order", nil)
	// No API key header

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"missing API key should return 401")
}

// TestAuthGuard_With_ForceAPIKey_InvalidKey verifies that With(forceAPIKeyAuth=true)
// rejects requests with a wrong API key.
func TestAuthGuard_With_ForceAPIKey_InvalidKey(t *testing.T) {
	t.Parallel()

	guard := newTestAuthGuard(t, AuthGuardConfig{
		APIKey:            "webhook-secret-key",
		APIKeyEnabled:     true,
		PluginAuthEnabled: true,
		AppName:           "flowker",
	}, "")

	app := fiber.New()
	app.Post("/v1/webhooks/*", guard.With("webhooks", "execute", true), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/hooks/order", nil)
	req.Header.Set(HeaderAPIKey, "wrong-key")

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"invalid API key should return 401")
}

// TestAuthGuard_With_NoForce_UsesPluginAuth verifies that With(forceAPIKeyAuth=false)
// delegates to Protect() which uses plugin auth when enabled.
func TestAuthGuard_With_NoForce_UsesPluginAuth(t *testing.T) {
	t.Parallel()

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("healthy"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authorized": true}`))
	}))
	defer mockServer.Close()

	guard := newTestAuthGuard(t, AuthGuardConfig{
		APIKey:            "some-key",
		APIKeyEnabled:     true,
		PluginAuthEnabled: true,
		AppName:           "flowker",
	}, mockServer.URL)

	app := fiber.New()
	app.Get("/v1/workflows", guard.With("workflows", "manage", false), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusOK)
	})

	token := createTestJWT(jwt.MapClaims{
		"sub":   "user-123",
		"type":  "normal-user",
		"owner": "org-456",
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"With(forceAPIKeyAuth=false) should use plugin auth, Bearer token should work")
}

// TestAuthGuard_With_ForceAPIKey_DisabledFallsBackToPluginAuth verifies that
// With(forceAPIKeyAuth=true) falls back to plugin auth when API keys are disabled,
// preventing the webhook route from being unauthenticated.
func TestAuthGuard_With_ForceAPIKey_DisabledFallsBackToPluginAuth(t *testing.T) {
	t.Parallel()

	// Mock Access Manager that requires Bearer token
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("healthy"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authorized": true}`))
	}))
	defer mockServer.Close()

	guard := newTestAuthGuard(t, AuthGuardConfig{
		APIKey:            "",
		APIKeyEnabled:     false,
		PluginAuthEnabled: true,
		AppName:           "flowker",
	}, mockServer.URL)

	app := fiber.New()
	app.Post("/v1/webhooks/*", guard.With("webhooks", "execute", true), func(c *fiber.Ctx) error {
		return c.SendStatus(http.StatusCreated)
	})

	// Without Bearer token, plugin auth should reject with "Missing Token"
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/hooks/order", nil)

	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
		"With(forceAPIKeyAuth=true) with APIKeyEnabled=false should fall back to plugin auth, not pass through")
}
