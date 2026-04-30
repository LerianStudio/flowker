// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"testing"

	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTenantInfrastructure_SingleTenantMode(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
	}{
		{
			name: "returns nil when MultiTenantEnabled is false",
			cfg: &Config{
				MultiTenantEnabled: false,
				MultiTenantURL:     "https://tenant-manager:8080",
			},
		},
		{
			name: "returns nil when MultiTenantURL is empty",
			cfg: &Config{
				MultiTenantEnabled: true,
				MultiTenantURL:     "",
			},
		},
		{
			name: "returns nil when both disabled and empty URL",
			cfg: &Config{
				MultiTenantEnabled: false,
				MultiTenantURL:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			logger := libLog.NewNop()

			ti, err := NewTenantInfrastructure(ctx, tt.cfg, logger)

			assert.NoError(t, err)
			assert.Nil(t, ti, "TenantInfrastructure should be nil in single-tenant mode")
		})
	}
}

func TestNewTenantInfrastructure_RequiresServiceAPIKey(t *testing.T) {
	ctx := context.Background()
	logger := libLog.NewNop()

	cfg := &Config{
		MultiTenantEnabled:       true,
		MultiTenantURL:           "https://tenant-manager:8080",
		MultiTenantServiceAPIKey: "", // Missing API key
	}

	ti, err := NewTenantInfrastructure(ctx, cfg, logger)

	require.Error(t, err)
	assert.Nil(t, ti)
	assert.Contains(t, err.Error(), "MULTI_TENANT_SERVICE_API_KEY is required")
}

func TestTenantInfrastructure_Close_NilSafe(t *testing.T) {
	ctx := context.Background()

	// Close on nil receiver should not panic
	var ti *TenantInfrastructure
	err := ti.Close(ctx)

	assert.NoError(t, err, "Close on nil TenantInfrastructure should return nil")
}

func TestTenantInfrastructure_Close_WithNilComponents(t *testing.T) {
	ctx := context.Background()

	// TenantInfrastructure with all nil components should close without error
	ti := &TenantInfrastructure{
		Client:          nil,
		MongoManager:    nil,
		PostgresManager: nil,
		Middleware:      nil,
		EventListener:   nil,
		RedisClient:     nil,
	}

	err := ti.Close(ctx)

	assert.NoError(t, err, "Close with nil components should return nil")
}
