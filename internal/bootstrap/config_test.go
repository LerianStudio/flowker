// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package bootstrap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// clearOIDCEnvVars unsets all deprecated OIDC env vars for the duration of the test,
// restoring their original values on cleanup. Needed because t.Setenv cannot unset.
func clearOIDCEnvVars(t *testing.T) {
	t.Helper()
	for _, k := range []string{"OIDC_ENABLED", "OIDC_ISSUER_URL", "OIDC_JWKS_URL", "OIDC_AUDIENCE"} {
		if orig, ok := os.LookupEnv(k); ok {
			t.Cleanup(func() { _ = os.Setenv(k, orig) })
		} else {
			t.Cleanup(func() { _ = os.Unsetenv(k) })
		}
		_ = os.Unsetenv(k)
	}
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		validate func(t *testing.T, cfg Config)
	}{
		{
			name: "full configuration",
			config: Config{
				EnvName:                 "test",
				ServerAddress:           ":8080",
				LogLevel:                "info",
				OtelServiceName:         "test-service",
				OtelLibraryName:         "test-library",
				OtelServiceVersion:      "1.0.0",
				OtelDeploymentEnv:       "development",
				OtelColExporterEndpoint: "localhost:4317",
				EnableTelemetry:         true,
				MongoURI:                "mongodb://localhost:27017",
				MongoDBName:             "flowker",
				SwaggerTitle:            "Test API",
				SwaggerDescription:      "Test Description",
				SwaggerVersion:          "2.0.0",
				SwaggerHost:             "localhost:8080",
				SwaggerBasePath:         "/api/v1",
				SwaggerSchemes:          "https",
			},
			validate: func(t *testing.T, cfg Config) {
				assert.Equal(t, "test", cfg.EnvName)
				assert.Equal(t, ":8080", cfg.ServerAddress)
				assert.Equal(t, "info", cfg.LogLevel)
				assert.Equal(t, "test-service", cfg.OtelServiceName)
				assert.Equal(t, "test-library", cfg.OtelLibraryName)
				assert.Equal(t, "1.0.0", cfg.OtelServiceVersion)
				assert.Equal(t, "development", cfg.OtelDeploymentEnv)
				assert.Equal(t, "localhost:4317", cfg.OtelColExporterEndpoint)
				assert.True(t, cfg.EnableTelemetry)
				assert.Equal(t, "mongodb://localhost:27017", cfg.MongoURI)
				assert.Equal(t, "flowker", cfg.MongoDBName)
				assert.Equal(t, "Test API", cfg.SwaggerTitle)
				assert.Equal(t, "Test Description", cfg.SwaggerDescription)
				assert.Equal(t, "2.0.0", cfg.SwaggerVersion)
				assert.Equal(t, "localhost:8080", cfg.SwaggerHost)
				assert.Equal(t, "/api/v1", cfg.SwaggerBasePath)
				assert.Equal(t, "https", cfg.SwaggerSchemes)
			},
		},
		{
			name:   "default values (empty config)",
			config: Config{},
			validate: func(t *testing.T, cfg Config) {
				assert.Equal(t, "", cfg.EnvName)
				assert.Equal(t, "", cfg.ServerAddress)
				assert.False(t, cfg.EnableTelemetry)
				assert.Equal(t, "", cfg.MongoURI)
				assert.Equal(t, "", cfg.MongoDBName)
				assert.Equal(t, "", cfg.SwaggerTitle)
			},
		},
		{
			name: "partial configuration (server only)",
			config: Config{
				ServerAddress: ":8080",
				LogLevel:      "debug",
			},
			validate: func(t *testing.T, cfg Config) {
				assert.Equal(t, ":8080", cfg.ServerAddress)
				assert.Equal(t, "debug", cfg.LogLevel)
				assert.Equal(t, "", cfg.MongoURI)
			},
		},
		{
			name: "partial configuration (otel only)",
			config: Config{
				OtelServiceName:    "my-service",
				OtelServiceVersion: "1.0.0",
				EnableTelemetry:    true,
			},
			validate: func(t *testing.T, cfg Config) {
				assert.Equal(t, "my-service", cfg.OtelServiceName)
				assert.Equal(t, "1.0.0", cfg.OtelServiceVersion)
				assert.True(t, cfg.EnableTelemetry)
				assert.Equal(t, "", cfg.ServerAddress)
			},
		},
		{
			name: "partial configuration (swagger only)",
			config: Config{
				SwaggerTitle:   "API Title",
				SwaggerVersion: "3.0.0",
			},
			validate: func(t *testing.T, cfg Config) {
				assert.Equal(t, "API Title", cfg.SwaggerTitle)
				assert.Equal(t, "3.0.0", cfg.SwaggerVersion)
				assert.Equal(t, "", cfg.SwaggerHost)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.config)
		})
	}
}

func TestValidateAccessManagerConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "disabled returns nil",
			cfg:     Config{PluginAuthEnabled: false},
			wantErr: false,
		},
		{
			name:    "enabled without address returns error",
			cfg:     Config{PluginAuthEnabled: true, PluginAuthAddress: ""},
			wantErr: true,
			errMsg:  "PLUGIN_AUTH_ADDRESS must be set",
		},
		{
			name:    "enabled with address returns nil",
			cfg:     Config{PluginAuthEnabled: true, PluginAuthAddress: "http://auth.local:8080"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearOIDCEnvVars(t)

			ctrl := gomock.NewController(t)
			logger := createTestLogger(ctrl)

			err := ValidateAccessManagerConfig(&tt.cfg, logger)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestWarnDeprecatedOIDCEnvVars(t *testing.T) {
	// Each var set triggers exactly one Warn log entry via logger.With(...).Log(...).
	// We rely on the mock's AnyTimes matchers; this test asserts the function
	// completes without panic across the three relevant environment states.
	scenarios := []struct {
		name string
		vars map[string]string
	}{
		{name: "none set", vars: map[string]string{}},
		{name: "one set", vars: map[string]string{"OIDC_ENABLED": "true"}},
		{name: "all set", vars: map[string]string{
			"OIDC_ENABLED":    "true",
			"OIDC_ISSUER_URL": "https://idp.example.com",
			"OIDC_JWKS_URL":   "https://idp.example.com/.well-known/jwks.json",
			"OIDC_AUDIENCE":   "flowker",
		}},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			clearOIDCEnvVars(t)
			for k, v := range sc.vars {
				t.Setenv(k, v)
			}

			ctrl := gomock.NewController(t)
			logger := createTestLogger(ctrl)

			assert.NotPanics(t, func() {
				warnDeprecatedOIDCEnvVars(logger)
			})
		})
	}
}

func TestConfig_MultiTenantFields(t *testing.T) {
	// TDD-RED: Test that Config struct contains all 14 canonical multi-tenant fields
	// following lib-commons v5 naming conventions.
	tests := []struct {
		name     string
		config   Config
		validate func(t *testing.T, cfg Config)
	}{
		{
			name: "multi-tenant disabled (single-tenant mode)",
			config: Config{
				MultiTenantEnabled: false,
			},
			validate: func(t *testing.T, cfg Config) {
				assert.False(t, cfg.MultiTenantEnabled)
			},
		},
		{
			name: "multi-tenant enabled with full configuration",
			config: Config{
				MultiTenantEnabled:                     true,
				MultiTenantURL:                         "http://tenant-manager:8080",
				MultiTenantRedisHost:                   "redis.example.com",
				MultiTenantRedisPort:                   "6379",
				MultiTenantRedisPassword:               "secret",
				MultiTenantRedisTLS:                    true,
				MultiTenantMaxTenantPools:              100,
				MultiTenantIdleTimeoutSec:              300,
				MultiTenantTimeout:                     30,
				MultiTenantCircuitBreakerThreshold:     5,
				MultiTenantCircuitBreakerTimeoutSec:    30,
				MultiTenantServiceAPIKey:               "api-key-123",
				MultiTenantCacheTTLSec:                 120,
				MultiTenantConnectionsCheckIntervalSec: 30,
			},
			validate: func(t *testing.T, cfg Config) {
				assert.True(t, cfg.MultiTenantEnabled)
				assert.Equal(t, "http://tenant-manager:8080", cfg.MultiTenantURL)
				assert.Equal(t, "redis.example.com", cfg.MultiTenantRedisHost)
				assert.Equal(t, "6379", cfg.MultiTenantRedisPort)
				assert.Equal(t, "secret", cfg.MultiTenantRedisPassword)
				assert.True(t, cfg.MultiTenantRedisTLS)
				assert.Equal(t, 100, cfg.MultiTenantMaxTenantPools)
				assert.Equal(t, 300, cfg.MultiTenantIdleTimeoutSec)
				assert.Equal(t, 30, cfg.MultiTenantTimeout)
				assert.Equal(t, 5, cfg.MultiTenantCircuitBreakerThreshold)
				assert.Equal(t, 30, cfg.MultiTenantCircuitBreakerTimeoutSec)
				assert.Equal(t, "api-key-123", cfg.MultiTenantServiceAPIKey)
				assert.Equal(t, 120, cfg.MultiTenantCacheTTLSec)
				assert.Equal(t, 30, cfg.MultiTenantConnectionsCheckIntervalSec)
			},
		},
		{
			name: "multi-tenant default values",
			config: Config{
				MultiTenantEnabled: false,
			},
			validate: func(t *testing.T, cfg Config) {
				// When multi-tenant is disabled, all fields should have zero values
				assert.False(t, cfg.MultiTenantEnabled)
				assert.Equal(t, "", cfg.MultiTenantURL)
				assert.Equal(t, "", cfg.MultiTenantRedisHost)
				assert.Equal(t, "", cfg.MultiTenantRedisPort)
				assert.Equal(t, "", cfg.MultiTenantRedisPassword)
				assert.False(t, cfg.MultiTenantRedisTLS)
				assert.Equal(t, 0, cfg.MultiTenantMaxTenantPools)
				assert.Equal(t, 0, cfg.MultiTenantIdleTimeoutSec)
				assert.Equal(t, 0, cfg.MultiTenantTimeout)
				assert.Equal(t, 0, cfg.MultiTenantCircuitBreakerThreshold)
				assert.Equal(t, 0, cfg.MultiTenantCircuitBreakerTimeoutSec)
				assert.Equal(t, "", cfg.MultiTenantServiceAPIKey)
				assert.Equal(t, 0, cfg.MultiTenantCacheTTLSec)
				assert.Equal(t, 0, cfg.MultiTenantConnectionsCheckIntervalSec)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.config)
		})
	}
}
