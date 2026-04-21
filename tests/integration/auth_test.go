// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuth_DisabledMode_AllowsRequests verifies that when PLUGIN_AUTH_ENABLED=false
// (configured in the test setup), management endpoints work without any token.
func TestAuth_DisabledMode_AllowsRequests(t *testing.T) {
	client := httpClient()

	// GET /v1/workflows without any Authorization header or API key.
	resp, err := client.Get(baseURL() + "/v1/workflows")
	require.NoError(t, err, "GET /v1/workflows should not fail")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"with auth disabled, GET /v1/workflows should return 200 without credentials")
}

// TestAuth_PublicEndpoints_NoAuthRequired verifies that health and swagger
// endpoints work without any authentication, regardless of auth mode.
func TestAuth_PublicEndpoints_NoAuthRequired(t *testing.T) {
	client := httpClient()

	tests := []struct {
		name string
		path string
	}{
		{name: "liveness probe", path: "/health/live"},
		{name: "readiness probe", path: "/health/ready"},
		{name: "combined health", path: "/health"},
		{name: "swagger docs", path: "/swagger/index.html"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Get(baseURL() + tt.path)
			require.NoError(t, err, "GET %s should not fail", tt.path)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode,
				"GET %s should return 200 without any authentication", tt.path)
		})
	}
}

// TestAuth_WebhookEndpoints_NoAuthRequired verifies that webhook endpoints
// are not behind Access Manager auth. When a webhook does not exist, it
// returns 404 (not 401/403), proving the auth middleware was not applied.
func TestAuth_WebhookEndpoints_NoAuthRequired(t *testing.T) {
	client := httpClient()

	// Call a non-existent webhook path with no auth headers.
	// If auth middleware were applied, we'd get 401. Instead, we get 404.
	resp, err := client.Post(
		baseURL()+"/v1/webhooks/auth-test-nonexistent",
		"application/json",
		nil,
	)
	require.NoError(t, err, "POST to webhook path should not fail")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode,
		"webhook endpoints should return 404 for unknown paths, not 401/403")
}
