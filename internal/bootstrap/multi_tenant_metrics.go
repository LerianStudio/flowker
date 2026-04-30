// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package bootstrap provides application initialization and configuration.
package bootstrap

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const multiTenantMeterName = "github.com/LerianStudio/flowker/multi-tenant"

var (
	multiTenantMetricsOnce    sync.Once
	multiTenantMetricsEnabled bool

	// tenantConnectionsTotal counts total tenant connections created.
	// Labels: tenant_id, resource_type (mongodb, postgresql)
	tenantConnectionsTotal metric.Int64Counter

	// tenantConnectionErrorsTotal counts connection failures per tenant.
	// Labels: tenant_id, resource_type (mongodb, postgresql), error_type
	tenantConnectionErrorsTotal metric.Int64Counter

	// tenantConsumersActive tracks active message consumers per tenant.
	// Labels: tenant_id
	// Note: Always 0 for Flowker (no RabbitMQ), but required for multi-tenant metrics contract.
	tenantConsumersActive metric.Int64Gauge

	// tenantMessagesProcessedTotal counts messages processed per tenant.
	// Labels: tenant_id, queue
	// Note: Always 0 for Flowker (no RabbitMQ), but required for multi-tenant metrics contract.
	tenantMessagesProcessedTotal metric.Int64Counter
)

// InitMultiTenantMetrics initializes multi-tenant metrics.
// When enabled=false, metrics are NOT registered (zero overhead in single-tenant mode).
// This function is idempotent and safe to call multiple times.
//
// Call once at application startup after telemetry is initialized:
//
//	if err := InitMultiTenantMetrics(cfg.MultiTenantEnabled); err != nil {
//	    logger.Log(ctx, libLog.LevelWarn, "Failed to initialize multi-tenant metrics", ...)
//	}
func InitMultiTenantMetrics(enabled bool) error {
	// No-op when disabled - zero overhead in single-tenant mode
	if !enabled {
		multiTenantMetricsEnabled = false
		return nil
	}

	var initErr error

	multiTenantMetricsOnce.Do(func() {
		meter := otel.GetMeterProvider().Meter(multiTenantMeterName)

		// Counter: tenant connections total
		tenantConnectionsTotal, initErr = meter.Int64Counter(
			"tenant_connections_total",
			metric.WithDescription("Total tenant connections created"),
		)
		if initErr != nil {
			return
		}

		// Counter: tenant connection errors total
		tenantConnectionErrorsTotal, initErr = meter.Int64Counter(
			"tenant_connection_errors_total",
			metric.WithDescription("Connection failures per tenant"),
		)
		if initErr != nil {
			return
		}

		// Gauge: tenant consumers active
		// Note: Always 0 for Flowker (no RabbitMQ), but required for multi-tenant metrics contract.
		tenantConsumersActive, initErr = meter.Int64Gauge(
			"tenant_consumers_active",
			metric.WithDescription("Active message consumers (0 for Flowker - no RabbitMQ)"),
		)
		if initErr != nil {
			return
		}

		// Counter: tenant messages processed total
		// Note: Always 0 for Flowker (no RabbitMQ), but required for multi-tenant metrics contract.
		tenantMessagesProcessedTotal, initErr = meter.Int64Counter(
			"tenant_messages_processed_total",
			metric.WithDescription("Messages processed per tenant (0 for Flowker - no RabbitMQ)"),
		)
		if initErr != nil {
			return
		}

		// Only enable metrics after all are successfully created to avoid partial state
		multiTenantMetricsEnabled = true
	})

	return initErr
}

// EmitTenantConnection increments the tenant connections counter.
// No-op when multi-tenant metrics are disabled.
//
// Parameters:
//   - tenantID: The tenant identifier
//   - resourceType: The resource type ("mongodb" or "postgresql")
func EmitTenantConnection(tenantID, resourceType string) {
	if !multiTenantMetricsEnabled || tenantConnectionsTotal == nil {
		return // No-op when disabled
	}

	tenantConnectionsTotal.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("tenant_id", tenantID),
			attribute.String("resource_type", resourceType),
		),
	)
}

// EmitTenantConnectionError increments the tenant connection errors counter.
// No-op when multi-tenant metrics are disabled.
//
// Parameters:
//   - tenantID: The tenant identifier
//   - resourceType: The resource type ("mongodb" or "postgresql")
//   - errorType: The error type (e.g., "timeout", "connection_refused", "auth_failed")
func EmitTenantConnectionError(tenantID, resourceType, errorType string) {
	if !multiTenantMetricsEnabled || tenantConnectionErrorsTotal == nil {
		return // No-op when disabled
	}

	tenantConnectionErrorsTotal.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("tenant_id", tenantID),
			attribute.String("resource_type", resourceType),
			attribute.String("error_type", errorType),
		),
	)
}

// EmitTenantConsumersActive sets the gauge for active message consumers per tenant.
// No-op when multi-tenant metrics are disabled.
//
// Note: Flowker does not use RabbitMQ, so this will always be 0.
// The metric exists for multi-tenant metrics contract compliance.
//
// Parameters:
//   - tenantID: The tenant identifier
//   - count: The number of active consumers (always 0 for Flowker)
func EmitTenantConsumersActive(tenantID string, count int64) {
	if !multiTenantMetricsEnabled || tenantConsumersActive == nil {
		return // No-op when disabled
	}

	tenantConsumersActive.Record(context.Background(), count,
		metric.WithAttributes(
			attribute.String("tenant_id", tenantID),
		),
	)
}

// EmitTenantMessagesProcessed increments the tenant messages processed counter.
// No-op when multi-tenant metrics are disabled.
//
// Note: Flowker does not use RabbitMQ, so this will always be 0.
// The metric exists for multi-tenant metrics contract compliance.
//
// Parameters:
//   - tenantID: The tenant identifier
//   - queue: The queue name
func EmitTenantMessagesProcessed(tenantID, queue string) {
	if !multiTenantMetricsEnabled || tenantMessagesProcessedTotal == nil {
		return // No-op when disabled
	}

	tenantMessagesProcessedTotal.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("tenant_id", tenantID),
			attribute.String("queue", queue),
		),
	)
}

// ResetMultiTenantMetricsForTest resets the metrics state for testing purposes.
// This allows tests to re-initialize metrics with a fresh meter provider.
// WARNING: This function is intended for testing only and should not be
// called in production code.
func ResetMultiTenantMetricsForTest() {
	multiTenantMetricsOnce = sync.Once{}
	multiTenantMetricsEnabled = false
	tenantConnectionsTotal = nil
	tenantConnectionErrorsTotal = nil
	tenantConsumersActive = nil
	tenantMessagesProcessedTotal = nil
}
