// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LerianStudio/flowker/pkg/executor"
	httpExecutor "github.com/LerianStudio/flowker/pkg/executors/http"
)

// writeJSON encodes v as JSON and writes it to w.
// Fails the test if encoding fails.
func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()

	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("failed to encode JSON: %v", err)
	}
}

// mockOIDCServer creates a mock OIDC server for testing.
func mockOIDCServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// OIDC Discovery endpoint
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}

		issuer := scheme + "://" + r.Host

		discovery := map[string]any{
			"issuer":                 issuer,
			"authorization_endpoint": issuer + "/auth",
			"token_endpoint":         issuer + "/token",
			"userinfo_endpoint":      issuer + "/userinfo",
			"jwks_uri":               issuer + "/jwks",
			"scopes_supported":       []string{"openid", "profile", "email", "api:read", "api:write"},
			"response_types_supported": []string{
				"code",
				"token",
				"id_token",
			},
			"grant_types_supported": []string{
				"authorization_code",
				"client_credentials",
				"password",
				"refresh_token",
			},
			"token_endpoint_auth_methods_supported": []string{
				"client_secret_basic",
				"client_secret_post",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, discovery)
	})

	// Token endpoint
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return
		}

		grantType := r.Form.Get("grant_type")

		// Validate client credentials
		var clientID, clientSecret string

		// Check Authorization header for Basic auth
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Basic ") {
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
			if err == nil {
				parts := strings.SplitN(string(decoded), ":", 2)
				if len(parts) == 2 {
					clientID = parts[0]
					clientSecret = parts[1]
				}
			}
		}

		// Or check form values
		if clientID == "" {
			clientID = r.Form.Get("client_id")
			clientSecret = r.Form.Get("client_secret")
		}

		switch grantType {
		case "client_credentials":
			// Validate client credentials
			if clientID != "test-client" || clientSecret != "test-secret" {
				w.WriteHeader(http.StatusUnauthorized)
				writeJSON(t, w, map[string]string{
					"error":             "invalid_client",
					"error_description": "Invalid client credentials",
				})

				return
			}

			token := map[string]any{
				"access_token": "mock-access-token-client-credentials",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"scope":        r.Form.Get("scope"),
			}

			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, token)

		case "password":
			username := r.Form.Get("username")
			password := r.Form.Get("password")

			// Validate user credentials
			if username != "test-user" || password != "test-password" {
				w.WriteHeader(http.StatusUnauthorized)
				writeJSON(t, w, map[string]string{
					"error":             "invalid_grant",
					"error_description": "Invalid user credentials",
				})

				return
			}

			token := map[string]any{
				"access_token":  "mock-access-token-password",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"refresh_token": "mock-refresh-token",
				"scope":         r.Form.Get("scope"),
			}

			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, token)

		case "refresh_token":
			refreshToken := r.Form.Get("refresh_token")
			if refreshToken != "mock-refresh-token" {
				w.WriteHeader(http.StatusUnauthorized)
				writeJSON(t, w, map[string]string{
					"error":             "invalid_grant",
					"error_description": "Invalid refresh token",
				})

				return
			}

			token := map[string]any{
				"access_token":  "mock-access-token-refreshed",
				"token_type":    "Bearer",
				"expires_in":    3600,
				"refresh_token": "mock-refresh-token-new",
			}

			w.Header().Set("Content-Type", "application/json")
			writeJSON(t, w, token)

		default:
			w.WriteHeader(http.StatusBadRequest)
			writeJSON(t, w, map[string]string{
				"error":             "unsupported_grant_type",
				"error_description": "Grant type not supported",
			})
		}
	})

	return httptest.NewServer(mux)
}

// mockTargetServer creates a mock target server that validates authentication.
func mockTargetServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	// Endpoint that requires API key
	mux.HandleFunc("/api-key", func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "test-api-key" {
			// Also check query parameter
			apiKey = r.URL.Query().Get("api_key")
			if apiKey != "test-api-key" {
				w.WriteHeader(http.StatusUnauthorized)
				writeJSON(t, w, map[string]string{"error": "invalid api key"})

				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]string{"message": "api key auth success"})
	})

	// Endpoint that requires bearer token
	mux.HandleFunc("/bearer", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(t, w, map[string]string{"error": "missing bearer token"})

			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]any{
			"message": "bearer auth success",
			"token":   token,
		})
	})

	// Endpoint that requires basic auth
	mux.HandleFunc("/basic", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Basic ") {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(t, w, map[string]string{"error": "missing basic auth"})

			return
		}

		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(t, w, map[string]string{"error": "invalid basic auth"})

			return
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 || parts[0] != "test-user" || parts[1] != "test-pass" {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(t, w, map[string]string{"error": "invalid credentials"})

			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]string{"message": "basic auth success"})
	})

	// Endpoint that validates OIDC tokens
	mux.HandleFunc("/protected", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(t, w, map[string]string{"error": "missing token"})

			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate mock tokens
		validTokens := []string{
			"mock-access-token-client-credentials",
			"mock-access-token-password",
			"mock-access-token-refreshed",
		}

		if !slices.Contains(validTokens, token) {
			w.WriteHeader(http.StatusUnauthorized)
			writeJSON(t, w, map[string]string{"error": "invalid token"})

			return
		}

		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]any{
			"message":      "oidc auth success",
			"token_prefix": token[:10],
		})
	})

	// Public endpoint (no auth)
	mux.HandleFunc("/public", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		writeJSON(t, w, map[string]string{"message": "public endpoint"})
	})

	return httptest.NewServer(mux)
}

func TestHTTPProviderAuthNone(t *testing.T) {
	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/public",
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Equal(t, 200, result.Data["status"])
}

func TestHTTPProviderAuthAPIKeyHeader(t *testing.T) {
	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/api-key",
			"auth": map[string]any{
				"type": "api_key",
				"config": map[string]any{
					"key":         "test-api-key",
					"header_name": "X-API-Key",
					"location":    "header",
				},
			},
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Equal(t, 200, result.Data["status"])

	body := result.Data["body"].(map[string]any)
	assert.Equal(t, "api key auth success", body["message"])
}

func TestHTTPProviderAuthAPIKeyQuery(t *testing.T) {
	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/api-key",
			"auth": map[string]any{
				"type": "api_key",
				"config": map[string]any{
					"key":              "test-api-key",
					"location":         "query",
					"query_param_name": "api_key",
				},
			},
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Equal(t, 200, result.Data["status"])
}

func TestHTTPProviderAuthBearer(t *testing.T) {
	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/bearer",
			"auth": map[string]any{
				"type": "bearer",
				"config": map[string]any{
					"token": "my-static-bearer-token",
				},
			},
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Equal(t, 200, result.Data["status"])

	body := result.Data["body"].(map[string]any)
	assert.Equal(t, "bearer auth success", body["message"])
	assert.Equal(t, "my-static-bearer-token", body["token"])
}

func TestHTTPProviderAuthBasic(t *testing.T) {
	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/basic",
			"auth": map[string]any{
				"type": "basic",
				"config": map[string]any{
					"username": "test-user",
					"password": "test-pass",
				},
			},
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Equal(t, 200, result.Data["status"])

	body := result.Data["body"].(map[string]any)
	assert.Equal(t, "basic auth success", body["message"])
}

func TestHTTPProviderAuthOIDCClientCredentials(t *testing.T) {
	oidcServer := mockOIDCServer(t)
	defer oidcServer.Close()

	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/protected",
			"auth": map[string]any{
				"type": "oidc_client_credentials",
				"config": map[string]any{
					"issuer_url":    oidcServer.URL,
					"client_id":     "test-client",
					"client_secret": "test-secret",
					"scopes":        []any{"api:read"},
				},
				"cache": map[string]any{
					"enabled":                       true,
					"refresh_before_expiry_seconds": 60,
				},
			},
		},
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Equal(t, 200, result.Data["status"])

	body := result.Data["body"].(map[string]any)
	assert.Equal(t, "oidc auth success", body["message"])
}

func TestHTTPProviderAuthOIDCUser(t *testing.T) {
	oidcServer := mockOIDCServer(t)
	defer oidcServer.Close()

	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/protected",
			"auth": map[string]any{
				"type": "oidc_user",
				"config": map[string]any{
					"issuer_url": oidcServer.URL,
					"client_id":  "test-client",
					"username":   "test-user",
					"password":   "test-password",
					"scopes":     []any{"openid", "profile"},
				},
				"cache": map[string]any{
					"enabled":           true,
					"use_refresh_token": true,
				},
			},
		},
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Equal(t, 200, result.Data["status"])

	body := result.Data["body"].(map[string]any)
	assert.Equal(t, "oidc auth success", body["message"])
}

func TestHTTPProviderAuthOIDCClientCredentialsInvalidClient(t *testing.T) {
	oidcServer := mockOIDCServer(t)
	defer oidcServer.Close()

	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/protected",
			"auth": map[string]any{
				"type": "oidc_client_credentials",
				"config": map[string]any{
					"issuer_url":    oidcServer.URL,
					"client_id":     "wrong-client",
					"client_secret": "wrong-secret",
				},
			},
		},
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusError, result.Status)
	assert.Contains(t, result.Error, "failed to apply authentication")
}

func TestHTTPProviderAuthOIDCUserInvalidCredentials(t *testing.T) {
	oidcServer := mockOIDCServer(t)
	defer oidcServer.Close()

	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/protected",
			"auth": map[string]any{
				"type": "oidc_user",
				"config": map[string]any{
					"issuer_url": oidcServer.URL,
					"client_id":  "test-client",
					"username":   "wrong-user",
					"password":   "wrong-password",
				},
			},
		},
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusError, result.Status)
	assert.Contains(t, result.Error, "failed to apply authentication")
}

func TestHTTPProviderAuthInvalidType(t *testing.T) {
	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/public",
			"auth": map[string]any{
				"type": "invalid_type",
			},
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusError, result.Status)
	assert.Contains(t, result.Error, "failed to create auth provider")
}

func TestHTTPProviderAuthMissingConfig(t *testing.T) {
	target := mockTargetServer(t)
	defer target.Close()

	runner := httpExecutor.NewRunner()

	// Missing key for api_key
	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    target.URL + "/api-key",
			"auth": map[string]any{
				"type":   "api_key",
				"config": map[string]any{},
			},
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusError, result.Status)
	assert.Contains(t, result.Error, "key is required")
}
