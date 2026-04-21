// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockDiscoveryServer(t *testing.T, doc *OIDCDiscoveryDocument) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doc)
	}))
}

func TestDiscoveryClient(t *testing.T) {
	doc := &OIDCDiscoveryDocument{
		Issuer:                "https://auth.example.com",
		AuthorizationEndpoint: "https://auth.example.com/auth",
		TokenEndpoint:         "https://auth.example.com/token",
		UserinfoEndpoint:      "https://auth.example.com/userinfo",
		JwksURI:               "https://auth.example.com/jwks",
		ScopesSupported:       []string{"openid", "profile", "email"},
		GrantTypesSupported:   []string{"authorization_code", "client_credentials", "password"},
	}

	server := mockDiscoveryServer(t, doc)
	defer server.Close()

	client := NewDiscoveryClient(nil)
	result, err := client.Discover(context.Background(), server.URL)

	require.NoError(t, err)
	assert.Equal(t, doc.TokenEndpoint, result.TokenEndpoint)
	assert.Equal(t, doc.JwksURI, result.JwksURI)
	assert.Equal(t, doc.ScopesSupported, result.ScopesSupported)
}

func TestDiscoveryClientCache(t *testing.T) {
	callCount := 0

	doc := &OIDCDiscoveryDocument{
		Issuer:        "https://auth.example.com",
		TokenEndpoint: "https://auth.example.com/token",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doc)
	}))
	defer server.Close()

	client := NewDiscoveryClient(nil)

	// First call
	_, err := client.Discover(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Second call should use cache
	_, err = client.Discover(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount) // Still 1, cache was used
}

func TestDiscoveryClientInvalidateCache(t *testing.T) {
	callCount := 0

	doc := &OIDCDiscoveryDocument{
		Issuer:        "https://auth.example.com",
		TokenEndpoint: "https://auth.example.com/token",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doc)
	}))
	defer server.Close()

	client := NewDiscoveryClient(nil)

	// First call
	_, err := client.Discover(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Invalidate cache
	client.InvalidateCache(server.URL)

	// Third call should fetch again
	_, err = client.Discover(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestDiscoveryClientNormalizeURL(t *testing.T) {
	doc := &OIDCDiscoveryDocument{
		Issuer:        "https://auth.example.com",
		TokenEndpoint: "https://auth.example.com/token",
	}

	server := mockDiscoveryServer(t, doc)
	defer server.Close()

	client := NewDiscoveryClient(nil)

	// URL with trailing slash
	result, err := client.Discover(context.Background(), server.URL+"/")
	require.NoError(t, err)
	assert.Equal(t, doc.TokenEndpoint, result.TokenEndpoint)
}

func TestDiscoveryClientMissingTokenEndpoint(t *testing.T) {
	doc := &OIDCDiscoveryDocument{
		Issuer: "https://auth.example.com",
		// TokenEndpoint is missing
	}

	server := mockDiscoveryServer(t, doc)
	defer server.Close()

	client := NewDiscoveryClient(nil)
	_, err := client.Discover(context.Background(), server.URL)

	require.ErrorIs(t, err, ErrDiscoveryMissingTokenEndpoint)
}

func TestDiscoveryClientServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := NewDiscoveryClient(nil)
	_, err := client.Discover(context.Background(), server.URL)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestDiscoveryClientInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewDiscoveryClient(nil)
	_, err := client.Discover(context.Background(), server.URL)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}
