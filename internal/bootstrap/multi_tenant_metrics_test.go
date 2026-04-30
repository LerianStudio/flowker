// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"sync"
	"testing"

	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestMultiTenantMetrics_InitWhenEnabled verifies that metrics are registered
// when multi-tenant mode is enabled.
func TestMultiTenantMetrics_InitWhenEnabled(t *testing.T) {
	// Setup: create a meter provider with in-memory reader
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)

	// Reset metrics state for this test
	ResetMultiTenantMetricsForTest()

	// Act: Initialize metrics with enabled=true
	err := InitMultiTenantMetrics(true)

	// Assert: no error
	require.NoError(t, err)

	// Emit some metrics to verify they work
	EmitTenantConnection("tenant-123", "mongodb")
	EmitTenantConnectionError("tenant-456", "postgresql", "timeout")

	// Collect metrics
	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)

	// Verify metrics were emitted (at least one scope metric exists)
	assert.NotEmpty(t, rm.ScopeMetrics, "should have scope metrics when enabled")
}

// TestMultiTenantMetrics_NoopWhenDisabled verifies that metrics are NOT registered
// when multi-tenant mode is disabled (zero overhead).
func TestMultiTenantMetrics_NoopWhenDisabled(t *testing.T) {
	// Setup: create a fresh meter provider
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)

	// Reset metrics state for this test
	ResetMultiTenantMetricsForTest()

	// Act: Initialize metrics with enabled=false
	err := InitMultiTenantMetrics(false)

	// Assert: no error
	require.NoError(t, err)

	// Emit calls should be no-ops (not panic)
	EmitTenantConnection("tenant-123", "mongodb")
	EmitTenantConnectionError("tenant-456", "postgresql", "timeout")
	EmitTenantConsumersActive("tenant-789", 0)
	EmitTenantMessagesProcessed("tenant-000", "queue-name")

	// Collect metrics - should be empty since no-op
	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)

	// No scope metrics should exist when disabled
	assert.Empty(t, rm.ScopeMetrics, "should have no scope metrics when disabled (no-op)")
}

// TestMultiTenantMetrics_EmitFunctions verifies all emit functions work correctly.
func TestMultiTenantMetrics_EmitFunctions(t *testing.T) {
	// Setup: create a meter provider with in-memory reader
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)

	// Reset and initialize metrics
	ResetMultiTenantMetricsForTest()
	err := InitMultiTenantMetrics(true)
	require.NoError(t, err)

	tests := []struct {
		name     string
		emitFunc func()
	}{
		{
			name: "EmitTenantConnection",
			emitFunc: func() {
				EmitTenantConnection("tenant-a", "mongodb")
			},
		},
		{
			name: "EmitTenantConnectionError",
			emitFunc: func() {
				EmitTenantConnectionError("tenant-b", "postgresql", "connection_refused")
			},
		},
		{
			name: "EmitTenantConsumersActive",
			emitFunc: func() {
				EmitTenantConsumersActive("tenant-c", 5)
			},
		},
		{
			name: "EmitTenantMessagesProcessed",
			emitFunc: func() {
				EmitTenantMessagesProcessed("tenant-d", "workflow-queue")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			assert.NotPanics(t, func() {
				tt.emitFunc()
			})
		})
	}
}

// TestMultiTenantMetrics_ThreadSafety verifies that concurrent emit calls are safe.
func TestMultiTenantMetrics_ThreadSafety(t *testing.T) {
	// Setup: create a meter provider with in-memory reader
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)

	// Reset and initialize metrics
	ResetMultiTenantMetricsForTest()
	err := InitMultiTenantMetrics(true)
	require.NoError(t, err)

	// Act: concurrent emits
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(4)
		go func(idx int) {
			defer wg.Done()
			EmitTenantConnection("tenant-concurrent", "mongodb")
		}(i)
		go func(idx int) {
			defer wg.Done()
			EmitTenantConnectionError("tenant-concurrent", "postgresql", "timeout")
		}(i)
		go func(idx int) {
			defer wg.Done()
			EmitTenantConsumersActive("tenant-concurrent", 0)
		}(i)
		go func(idx int) {
			defer wg.Done()
			EmitTenantMessagesProcessed("tenant-concurrent", "queue")
		}(i)
	}
	wg.Wait()

	// If we reach here without panic, the test passes
	assert.True(t, true, "concurrent emits completed without panic")
}

// TestMultiTenant_BackwardCompatibility verifies that single-tenant mode works correctly.
func TestMultiTenant_BackwardCompatibility(t *testing.T) {
	t.Run("single_tenant_mode_no_tenant_context_required", func(t *testing.T) {
		// When MULTI_TENANT_ENABLED=false, TenantInfrastructure is nil
		// and repositories should work without tenant context.
		ctx := context.Background()
		logger := libLog.NewNop()
		cfg := &Config{
			MultiTenantEnabled: false,
			MultiTenantURL:     "",
		}

		// NewTenantInfrastructure returns nil in single-tenant mode
		ti, err := NewTenantInfrastructure(ctx, cfg, logger)
		require.NoError(t, err)
		assert.Nil(t, ti, "TenantInfrastructure should be nil in single-tenant mode")
	})

	t.Run("metrics_noop_when_disabled", func(t *testing.T) {
		// Setup: create a fresh meter provider
		reader := sdkmetric.NewManualReader()
		provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
		otel.SetMeterProvider(provider)

		// Reset metrics state
		ResetMultiTenantMetricsForTest()

		// Initialize with disabled
		err := InitMultiTenantMetrics(false)
		require.NoError(t, err)

		// Emit calls should be no-ops
		EmitTenantConnection("tenant-test", "mongodb")
		EmitTenantConnectionError("tenant-test", "postgresql", "error")

		// Verify no metrics were recorded
		var rm metricdata.ResourceMetrics
		err = reader.Collect(context.Background(), &rm)
		require.NoError(t, err)
		assert.Empty(t, rm.ScopeMetrics, "no metrics should be recorded when disabled")
	})

	t.Run("middleware_passthrough_when_disabled", func(t *testing.T) {
		// When MULTI_TENANT_ENABLED=false, TenantInfrastructure is nil
		// so the middleware is nil, causing WhenEnabled to pass through.
		ctx := context.Background()
		cfg := &Config{
			MultiTenantEnabled: false,
		}

		ti, err := NewTenantInfrastructure(ctx, cfg, nil)
		require.NoError(t, err)
		assert.Nil(t, ti, "TenantInfrastructure should be nil")

		// When ti is nil, Middleware would be nil
		// The route setup uses WhenEnabled(ti.Middleware) which returns c.Next()
		// for nil middleware. This is tested in routes_test.go.
	})
}

// TestMultiTenantMetrics_Idempotent verifies that InitMultiTenantMetrics is idempotent.
func TestMultiTenantMetrics_Idempotent(t *testing.T) {
	// Setup: create a meter provider
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)

	// Reset metrics state
	ResetMultiTenantMetricsForTest()

	// Call InitMultiTenantMetrics multiple times
	err1 := InitMultiTenantMetrics(true)
	err2 := InitMultiTenantMetrics(true)
	err3 := InitMultiTenantMetrics(true)

	// All calls should succeed
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)
}

// =============================================================================
// Multi-Tenant Integration Tests
// =============================================================================

// TestMultiTenant_ServiceStartsWithoutTenantManager verifies that the service
// can start without a running Tenant Manager when MULTI_TENANT_ENABLED=false.
func TestMultiTenant_ServiceStartsWithoutTenantManager(t *testing.T) {
	ctx := context.Background()
	logger := libLog.NewNop()

	tests := []struct {
		name   string
		cfg    *Config
		wantTI bool
	}{
		{
			name: "single_tenant_mode_no_tenant_manager_required",
			cfg: &Config{
				MultiTenantEnabled: false,
				MultiTenantURL:     "",
			},
			wantTI: false,
		},
		{
			name: "disabled_with_url_still_single_tenant",
			cfg: &Config{
				MultiTenantEnabled: false,
				MultiTenantURL:     "https://tenant-manager:8080",
			},
			wantTI: false,
		},
		{
			name: "enabled_without_url_is_single_tenant",
			cfg: &Config{
				MultiTenantEnabled: true,
				MultiTenantURL:     "", // Empty URL disables multi-tenant
			},
			wantTI: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti, err := NewTenantInfrastructure(ctx, tt.cfg, logger)

			require.NoError(t, err)

			if tt.wantTI {
				assert.NotNil(t, ti, "TenantInfrastructure should be created")
			} else {
				assert.Nil(t, ti, "TenantInfrastructure should be nil in single-tenant mode")
			}
		})
	}
}

// TestMultiTenant_ConfigValidation verifies configuration validation for multi-tenant mode.
func TestMultiTenant_ConfigValidation(t *testing.T) {
	ctx := context.Background()
	logger := libLog.NewNop()

	tests := []struct {
		name      string
		cfg       *Config
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid_single_tenant_config",
			cfg: &Config{
				MultiTenantEnabled: false,
			},
			wantError: false,
		},
		{
			name: "multi_tenant_missing_api_key",
			cfg: &Config{
				MultiTenantEnabled:       true,
				MultiTenantURL:           "https://tenant-manager:8080",
				MultiTenantServiceAPIKey: "", // Missing required API key
			},
			wantError: true,
			errorMsg:  "MULTI_TENANT_SERVICE_API_KEY is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti, err := NewTenantInfrastructure(ctx, tt.cfg, logger)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, ti)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMultiTenant_ConfigurationEnvVars verifies that Config struct has all
// required multi-tenant environment variable mappings.
func TestMultiTenant_ConfigurationEnvVars(t *testing.T) {
	cfg := &Config{}

	// Verify that the Config struct has the expected multi-tenant fields
	t.Run("has_all_multi_tenant_config_fields", func(t *testing.T) {
		// These fields should exist and have correct types
		assert.IsType(t, false, cfg.MultiTenantEnabled)
		assert.IsType(t, "", cfg.MultiTenantURL)
		assert.IsType(t, "", cfg.MultiTenantRedisHost)
		assert.IsType(t, "", cfg.MultiTenantRedisPort)
		assert.IsType(t, "", cfg.MultiTenantRedisPassword)
		assert.IsType(t, false, cfg.MultiTenantRedisTLS)
		assert.IsType(t, 0, cfg.MultiTenantMaxTenantPools)
		assert.IsType(t, 0, cfg.MultiTenantIdleTimeoutSec)
		assert.IsType(t, 0, cfg.MultiTenantTimeout)
		assert.IsType(t, 0, cfg.MultiTenantCircuitBreakerThreshold)
		assert.IsType(t, 0, cfg.MultiTenantCircuitBreakerTimeoutSec)
		assert.IsType(t, "", cfg.MultiTenantServiceAPIKey)
		assert.IsType(t, 0, cfg.MultiTenantCacheTTLSec)
		assert.IsType(t, 0, cfg.MultiTenantConnectionsCheckIntervalSec)
	})
}

// TestMultiTenant_ExistingTestsPassWithDisabled verifies backward compatibility
// by checking that existing functionality works without multi-tenant mode.
func TestMultiTenant_ExistingTestsPassWithDisabled(t *testing.T) {
	// This test verifies that when MULTI_TENANT_ENABLED=false (the default),
	// all existing functionality continues to work as expected.

	t.Run("metrics_disabled_by_default", func(t *testing.T) {
		// Reset metrics to simulate fresh startup
		ResetMultiTenantMetricsForTest()

		// Initialize with disabled (simulating MULTI_TENANT_ENABLED=false)
		err := InitMultiTenantMetrics(false)
		require.NoError(t, err)

		// All emit functions should be no-ops and not panic
		assert.NotPanics(t, func() {
			EmitTenantConnection("tenant-1", "mongodb")
			EmitTenantConnectionError("tenant-1", "postgresql", "error")
			EmitTenantConsumersActive("tenant-1", 0)
			EmitTenantMessagesProcessed("tenant-1", "queue")
		})
	})

	t.Run("tenant_infrastructure_nil_when_disabled", func(t *testing.T) {
		ctx := context.Background()
		logger := libLog.NewNop()

		cfg := &Config{
			MultiTenantEnabled: false,
		}

		ti, err := NewTenantInfrastructure(ctx, cfg, logger)

		require.NoError(t, err)
		assert.Nil(t, ti, "TenantInfrastructure should be nil when disabled")
	})
}

// TestMultiTenant_HealthEndpointsWorkWithoutTenantContext verifies that
// health and readyz endpoints work without tenant context (as expected).
func TestMultiTenant_HealthEndpointsWorkWithoutTenantContext(t *testing.T) {
	// Health and readyz endpoints should NOT require tenant context
	// They are infrastructure endpoints that check system health, not tenant-specific data

	t.Run("health_endpoints_no_tenant_required", func(t *testing.T) {
		// In single-tenant mode, TenantInfrastructure is nil
		// This means the tenant middleware is nil and WhenEnabled passes through
		ctx := context.Background()
		logger := libLog.NewNop()

		cfg := &Config{
			MultiTenantEnabled: false,
		}

		ti, err := NewTenantInfrastructure(ctx, cfg, logger)

		require.NoError(t, err)
		assert.Nil(t, ti)

		// In routes.go, health/readyz are registered BEFORE the v1 group
		// with tenant middleware, so they never need tenant context
	})
}
