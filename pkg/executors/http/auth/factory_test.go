// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFromConfigNil(t *testing.T) {
	provider, err := NewFromConfig(nil, nil)

	require.NoError(t, err)
	assert.Equal(t, TypeNone, provider.Type())
}

func TestNewFromConfigNoneType(t *testing.T) {
	cfg := map[string]any{
		"type": "none",
	}

	provider, err := NewFromConfig(cfg, nil)

	require.NoError(t, err)
	assert.Equal(t, TypeNone, provider.Type())
}

func TestNewFromConfigEmptyType(t *testing.T) {
	cfg := map[string]any{
		"type": "",
	}

	provider, err := NewFromConfig(cfg, nil)

	require.NoError(t, err)
	assert.Equal(t, TypeNone, provider.Type())
}

func TestNewFromConfigAPIKey(t *testing.T) {
	cfg := map[string]any{
		"type": "api_key",
		"config": map[string]any{
			"key":         "my-key",
			"header_name": "X-Custom-Key",
		},
	}

	provider, err := NewFromConfig(cfg, nil)

	require.NoError(t, err)
	assert.Equal(t, TypeAPIKey, provider.Type())
}

func TestNewFromConfigAPIKeyMissingKey(t *testing.T) {
	cfg := map[string]any{
		"type":   "api_key",
		"config": map[string]any{},
	}

	_, err := NewFromConfig(cfg, nil)

	require.ErrorIs(t, err, ErrAPIKeyKeyRequired)
}

func TestNewFromConfigBearer(t *testing.T) {
	cfg := map[string]any{
		"type": "bearer",
		"config": map[string]any{
			"token": "my-token",
		},
	}

	provider, err := NewFromConfig(cfg, nil)

	require.NoError(t, err)
	assert.Equal(t, TypeBearer, provider.Type())
}

func TestNewFromConfigBearerMissingToken(t *testing.T) {
	cfg := map[string]any{
		"type":   "bearer",
		"config": map[string]any{},
	}

	_, err := NewFromConfig(cfg, nil)

	require.ErrorIs(t, err, ErrBearerTokenRequired)
}

func TestNewFromConfigBasic(t *testing.T) {
	cfg := map[string]any{
		"type": "basic",
		"config": map[string]any{
			"username": "user",
			"password": "pass",
		},
	}

	provider, err := NewFromConfig(cfg, nil)

	require.NoError(t, err)
	assert.Equal(t, TypeBasic, provider.Type())
}

func TestNewFromConfigBasicMissingUsername(t *testing.T) {
	cfg := map[string]any{
		"type": "basic",
		"config": map[string]any{
			"password": "pass",
		},
	}

	_, err := NewFromConfig(cfg, nil)

	require.ErrorIs(t, err, ErrBasicUsernameRequired)
}

func TestNewFromConfigBasicMissingPassword(t *testing.T) {
	cfg := map[string]any{
		"type": "basic",
		"config": map[string]any{
			"username": "user",
		},
	}

	_, err := NewFromConfig(cfg, nil)

	require.ErrorIs(t, err, ErrBasicPasswordRequired)
}

func TestNewFromConfigOIDCClientCredentials(t *testing.T) {
	cfg := map[string]any{
		"type": "oidc_client_credentials",
		"config": map[string]any{
			"issuer_url":    "https://auth.example.com",
			"client_id":     "my-client",
			"client_secret": "my-secret",
		},
	}

	provider, err := NewFromConfig(cfg, nil)

	require.NoError(t, err)
	assert.Equal(t, TypeOIDCClientCredentials, provider.Type())
}

func TestNewFromConfigOIDCClientCredentialsMissingIssuer(t *testing.T) {
	cfg := map[string]any{
		"type": "oidc_client_credentials",
		"config": map[string]any{
			"client_id":     "my-client",
			"client_secret": "my-secret",
		},
	}

	_, err := NewFromConfig(cfg, nil)

	require.ErrorIs(t, err, ErrOIDCClientCredentialsIssuerRequired)
}

func TestNewFromConfigOIDCUser(t *testing.T) {
	cfg := map[string]any{
		"type": "oidc_user",
		"config": map[string]any{
			"issuer_url": "https://auth.example.com",
			"client_id":  "my-client",
			"username":   "user",
			"password":   "pass",
		},
	}

	provider, err := NewFromConfig(cfg, nil)

	require.NoError(t, err)
	assert.Equal(t, TypeOIDCUser, provider.Type())
}

func TestNewFromConfigOIDCUserMissingUsername(t *testing.T) {
	cfg := map[string]any{
		"type": "oidc_user",
		"config": map[string]any{
			"issuer_url": "https://auth.example.com",
			"client_id":  "my-client",
			"password":   "pass",
		},
	}

	_, err := NewFromConfig(cfg, nil)

	require.ErrorIs(t, err, ErrOIDCUserUsernameRequired)
}

func TestNewFromConfigUnknownType(t *testing.T) {
	cfg := map[string]any{
		"type": "unknown",
	}

	_, err := NewFromConfig(cfg, nil)

	require.ErrorIs(t, err, ErrUnknownAuthType)
}
