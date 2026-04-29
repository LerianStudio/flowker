// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package readyz

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const meterName = "github.com/LerianStudio/flowker/readyz"

var (
	metricsOnce sync.Once

	checkDuration metric.Float64Histogram
	checkStatus   metric.Int64Counter
	selfProbe     metric.Float64Gauge
)

// Histogram bucket boundaries in milliseconds.
// Covers cache-fast (1ms) to timeout-slow (5000ms) per readyz metrics contract.
var histogramBuckets = []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000}

// InitMetrics initializes the readyz metrics.
// This function is idempotent and safe to call multiple times.
// Call once at application startup after telemetry is initialized.
func InitMetrics() error {
	var initErr error

	metricsOnce.Do(func() {
		meter := otel.GetMeterProvider().Meter(meterName)

		// Histogram for check duration
		checkDuration, initErr = meter.Float64Histogram(
			"readyz_check_duration_ms",
			metric.WithDescription("Duration of /readyz dependency checks in milliseconds"),
			metric.WithUnit("ms"),
			metric.WithExplicitBucketBoundaries(histogramBuckets...),
		)
		if initErr != nil {
			return
		}

		// Counter for check status
		checkStatus, initErr = meter.Int64Counter(
			"readyz_check_status",
			metric.WithDescription("Count of /readyz check outcomes"),
		)
		if initErr != nil {
			return
		}

		// Gauge for self-probe result
		selfProbe, initErr = meter.Float64Gauge(
			"selfprobe_result",
			metric.WithDescription("Last self-probe result per dependency (1=up, 0=down)"),
		)
	})

	return initErr
}

// EmitCheckDuration records the duration of a dependency check.
// Labels: dep (dependency name), status (check outcome: up/down/degraded/skipped/n/a).
// Duration is recorded in milliseconds per the readyz metrics contract.
func EmitCheckDuration(ctx context.Context, dep, status string, duration time.Duration) {
	if checkDuration == nil {
		return // Metrics not initialized - graceful no-op
	}

	checkDuration.Record(ctx, float64(duration.Milliseconds()),
		metric.WithAttributes(
			attribute.String("dep", dep),
			attribute.String("status", status),
		),
	)
}

// EmitCheckStatus increments the counter for a dependency check outcome.
// Labels: dep (dependency name), status (check outcome: up/down/degraded/skipped/n/a).
func EmitCheckStatus(ctx context.Context, dep, status string) {
	if checkStatus == nil {
		return // Metrics not initialized - graceful no-op
	}

	checkStatus.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("dep", dep),
			attribute.String("status", status),
		),
	)
}

// EmitSelfProbeResult sets the gauge for a dependency's self-probe result.
// Label: dep (dependency name).
// Values: 1.0 (up), 0.0 (down).
// Called from RunSelfProbe() after startup health validation completes.
func EmitSelfProbeResult(ctx context.Context, dep string, up bool) {
	if selfProbe == nil {
		return // Metrics not initialized - graceful no-op
	}

	v := 0.0
	if up {
		v = 1.0
	}

	selfProbe.Record(ctx, v,
		metric.WithAttributes(
			attribute.String("dep", dep),
		),
	)
}

// ResetMetricsForTest resets the metrics state for testing purposes.
// This allows tests to re-initialize metrics with a fresh meter provider.
// WARNING: This function is intended for testing only and should not be
// called in production code.
func ResetMetricsForTest() {
	metricsOnce = sync.Once{}
	checkDuration = nil
	checkStatus = nil
	selfProbe = nil
}
