// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSensitiveConfig_NilConfig(t *testing.T) {
	result := maskSensitiveConfig(nil)
	assert.Nil(t, result)
}

func TestMaskSensitiveConfig_EmptyConfig(t *testing.T) {
	result := maskSensitiveConfig(map[string]any{})
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestMaskSensitiveConfig_MasksKnownSensitiveKeys(t *testing.T) {
	config := map[string]any{
		"api_key":       "sk-1234567890abcdef",
		"client_secret": "super-secret-value-xyz",
		"password":      "my-password-123",
		"token":         "tok_abcdefghij",
		"secret":        "s3cr3t-value",
		"base_url":      "https://api.example.com",
		"name":          "my-provider",
	}

	result := maskSensitiveConfig(config)

	// Sensitive fields should be masked with last 4 chars visible
	assert.Equal(t, "****cdef", result["api_key"])
	assert.Equal(t, "****-xyz", result["client_secret"])
	assert.Equal(t, "****-123", result["password"])
	assert.Equal(t, "****ghij", result["token"])
	assert.Equal(t, "****alue", result["secret"])

	// Non-sensitive fields should be unchanged
	assert.Equal(t, "https://api.example.com", result["base_url"])
	assert.Equal(t, "my-provider", result["name"])
}

func TestMaskSensitiveConfig_ShortValueFullyRedacted(t *testing.T) {
	config := map[string]any{
		"api_key": "abc",
	}

	result := maskSensitiveConfig(config)
	assert.Equal(t, "********", result["api_key"])
}

func TestMaskSensitiveConfig_ExactlyFourCharsFullyRedacted(t *testing.T) {
	config := map[string]any{
		"password": "abcd",
	}

	result := maskSensitiveConfig(config)
	assert.Equal(t, "********", result["password"])
}

func TestMaskSensitiveConfig_NonStringValueFullyRedacted(t *testing.T) {
	config := map[string]any{
		"secret": 12345,
	}

	result := maskSensitiveConfig(config)
	assert.Equal(t, "********", result["secret"])
}

func TestMaskSensitiveConfig_PreservesAllNonSensitiveFields(t *testing.T) {
	config := map[string]any{
		"base_url":    "https://api.example.com",
		"timeout":     30,
		"retry_count": 3,
		"enabled":     true,
	}

	result := maskSensitiveConfig(config)

	assert.Equal(t, "https://api.example.com", result["base_url"])
	assert.Equal(t, 30, result["timeout"])
	assert.Equal(t, 3, result["retry_count"])
	assert.Equal(t, true, result["enabled"])
}

func TestMaskSensitiveConfig_CamelCaseDetection(t *testing.T) {
	config := map[string]any{
		"clientSecret": "my-secret-value-1234",
		"apiKey":       "key-value-abcdefgh",
	}

	result := maskSensitiveConfig(config)

	// lib-commons IsSensitiveField normalizes camelCase
	assert.Equal(t, "****1234", result["clientSecret"])
	assert.Equal(t, "****efgh", result["apiKey"])
}

func TestProviderConfigurationOutputFromDomain_MasksSecrets(t *testing.T) {
	config := map[string]any{
		"base_url":      "https://api.example.com",
		"api_key":       "sk-1234567890abcdef",
		"client_secret": "super-secret-value",
	}

	pc, err := NewProviderConfiguration("test-provider", nil, "tracer", config)
	assert.NoError(t, err)

	output := ProviderConfigurationOutputFromDomain(pc)

	assert.Equal(t, "https://api.example.com", output.Config["base_url"])
	assert.Equal(t, "****cdef", output.Config["api_key"])
	assert.Equal(t, "****alue", output.Config["client_secret"])
}
