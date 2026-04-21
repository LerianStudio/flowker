// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoneProvider(t *testing.T) {
	provider := NewNoneProvider()

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	err := provider.Apply(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, TypeNone, provider.Type())
	assert.Empty(t, req.Header.Get("Authorization"))
}

func TestAPIKeyProviderHeader(t *testing.T) {
	cfg := &APIKeyConfig{
		Key:        "my-api-key",
		HeaderName: "X-API-Key",
		Location:   "header",
	}

	provider, err := NewAPIKeyProvider(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	err = provider.Apply(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, TypeAPIKey, provider.Type())
	assert.Equal(t, "my-api-key", req.Header.Get("X-API-Key"))
}

func TestAPIKeyProviderHeaderWithPrefix(t *testing.T) {
	cfg := &APIKeyConfig{
		Key:        "my-api-key",
		HeaderName: "Authorization",
		Prefix:     "Api-Key ",
		Location:   "header",
	}

	provider, err := NewAPIKeyProvider(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	err = provider.Apply(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "Api-Key my-api-key", req.Header.Get("Authorization"))
}

func TestAPIKeyProviderQuery(t *testing.T) {
	cfg := &APIKeyConfig{
		Key:            "my-api-key",
		Location:       "query",
		QueryParamName: "apikey",
	}

	provider, err := NewAPIKeyProvider(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/path", nil)
	err = provider.Apply(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, "my-api-key", req.URL.Query().Get("apikey"))
}

func TestAPIKeyProviderDefaults(t *testing.T) {
	cfg := &APIKeyConfig{
		Key: "my-api-key",
	}

	provider, err := NewAPIKeyProvider(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	err = provider.Apply(context.Background(), req)

	require.NoError(t, err)
	// Default header name is X-API-Key, default location is header
	assert.Equal(t, "my-api-key", req.Header.Get("X-API-Key"))
}

func TestAPIKeyProviderInvalidLocation(t *testing.T) {
	cfg := &APIKeyConfig{
		Key:      "my-api-key",
		Location: "invalid",
	}

	_, err := NewAPIKeyProvider(cfg)
	require.ErrorIs(t, err, ErrAPIKeyInvalidLocation)
}

func TestAPIKeyProviderNilConfig(t *testing.T) {
	_, err := NewAPIKeyProvider(nil)
	require.ErrorIs(t, err, ErrAPIKeyConfigRequired)
}

func TestAPIKeyProviderMissingKey(t *testing.T) {
	cfg := &APIKeyConfig{
		Location: "header",
	}

	_, err := NewAPIKeyProvider(cfg)
	require.ErrorIs(t, err, ErrAPIKeyKeyRequired)
}

func TestBearerProvider(t *testing.T) {
	cfg := &BearerConfig{
		Token: "my-bearer-token",
	}

	provider, err := NewBearerProvider(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	err = provider.Apply(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, TypeBearer, provider.Type())
	assert.Equal(t, "Bearer my-bearer-token", req.Header.Get("Authorization"))
}

func TestBearerProviderNilConfig(t *testing.T) {
	_, err := NewBearerProvider(nil)
	require.ErrorIs(t, err, ErrBearerConfigRequired)
}

func TestBearerProviderMissingToken(t *testing.T) {
	cfg := &BearerConfig{}

	_, err := NewBearerProvider(cfg)
	require.ErrorIs(t, err, ErrBearerTokenRequired)
}

func TestBasicProvider(t *testing.T) {
	cfg := &BasicConfig{
		Username: "user",
		Password: "pass",
	}

	provider, err := NewBasicProvider(cfg)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	err = provider.Apply(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, TypeBasic, provider.Type())

	// user:pass in base64 = dXNlcjpwYXNz
	assert.Equal(t, "Basic dXNlcjpwYXNz", req.Header.Get("Authorization"))
}

func TestBasicProviderNilConfig(t *testing.T) {
	_, err := NewBasicProvider(nil)
	require.ErrorIs(t, err, ErrBasicConfigRequired)
}

func TestBasicProviderMissingUsername(t *testing.T) {
	cfg := &BasicConfig{
		Password: "pass",
	}

	_, err := NewBasicProvider(cfg)
	require.ErrorIs(t, err, ErrBasicUsernameRequired)
}

func TestBasicProviderMissingPassword(t *testing.T) {
	cfg := &BasicConfig{
		Username: "user",
	}

	_, err := NewBasicProvider(cfg)
	require.ErrorIs(t, err, ErrBasicPasswordRequired)
}
