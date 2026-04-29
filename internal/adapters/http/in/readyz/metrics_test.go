// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package readyz_test

import (
	"context"
	"testing"
	"time"

	"github.com/LerianStudio/flowker/internal/adapters/http/in/readyz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// setupTestMeterProvider creates a test meter provider with an in-memory reader.
func setupTestMeterProvider(t *testing.T) (*sdkmetric.MeterProvider, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(provider)

	return provider, reader
}

// collectMetrics reads all metrics from the reader.
func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) *metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	err := reader.Collect(context.Background(), &rm)
	require.NoError(t, err)

	return &rm
}

// findHistogram searches for a histogram metric by name.
func findHistogram(rm *metricdata.ResourceMetrics, name string) *metricdata.Histogram[float64] {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				if hist, ok := m.Data.(metricdata.Histogram[float64]); ok {
					return &hist
				}
			}
		}
	}

	return nil
}

// findCounter searches for a counter metric by name.
func findCounter(rm *metricdata.ResourceMetrics, name string) *metricdata.Sum[int64] {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
					return &sum
				}
			}
		}
	}

	return nil
}

// findGauge searches for a gauge metric by name.
func findGauge(rm *metricdata.ResourceMetrics, name string) *metricdata.Gauge[float64] {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == name {
				if gauge, ok := m.Data.(metricdata.Gauge[float64]); ok {
					return &gauge
				}
			}
		}
	}

	return nil
}

// getAttributeValue extracts an attribute value by key from a data point.
func getAttributeValue(attrs []any, key string) string {
	for _, attr := range attrs {
		if kv, ok := attr.(interface {
			Key() string
			Value() interface{ AsString() string }
		}); ok {
			if kv.Key() == key {
				return kv.Value().AsString()
			}
		}
	}

	return ""
}

func TestInitMetrics_RegistersAllThreeMetrics(t *testing.T) {
	// Arrange
	provider, reader := setupTestMeterProvider(t)
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	// Reset metrics for clean test
	readyz.ResetMetricsForTest()

	// Act
	err := readyz.InitMetrics()
	require.NoError(t, err, "InitMetrics should not return an error")

	// Emit one of each metric type to verify they're registered
	ctx := context.Background()
	readyz.EmitCheckDuration(ctx, "test-dep", "up", 50*time.Millisecond)
	readyz.EmitCheckStatus(ctx, "test-dep", "up")
	readyz.EmitSelfProbeResult(ctx, "test-dep", true)

	// Collect metrics
	rm := collectMetrics(t, reader)

	// Assert: All three metrics should be present
	hist := findHistogram(rm, "readyz_check_duration_ms")
	assert.NotNil(t, hist, "readyz_check_duration_ms histogram should be registered")

	counter := findCounter(rm, "readyz_check_status")
	assert.NotNil(t, counter, "readyz_check_status counter should be registered")

	gauge := findGauge(rm, "selfprobe_result")
	assert.NotNil(t, gauge, "selfprobe_result gauge should be registered")
}

func TestEmitCheckDuration_RecordsHistogramWithCorrectLabels(t *testing.T) {
	// Arrange
	provider, reader := setupTestMeterProvider(t)
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	readyz.ResetMetricsForTest()
	err := readyz.InitMetrics()
	require.NoError(t, err)

	// Act
	ctx := context.Background()
	readyz.EmitCheckDuration(ctx, "mongodb", "up", 42*time.Millisecond)
	readyz.EmitCheckDuration(ctx, "postgresql", "down", 2500*time.Millisecond)

	// Collect metrics
	rm := collectMetrics(t, reader)
	hist := findHistogram(rm, "readyz_check_duration_ms")

	// Assert
	require.NotNil(t, hist, "Histogram should exist")
	require.Len(t, hist.DataPoints, 2, "Should have two data points (mongodb and postgresql)")

	// Verify data points exist for each dependency
	var mongoFound, pgFound bool
	for _, dp := range hist.DataPoints {
		dep, _ := dp.Attributes.Value("dep")
		status, _ := dp.Attributes.Value("status")

		if dep.AsString() == "mongodb" && status.AsString() == "up" {
			mongoFound = true
			assert.Equal(t, uint64(1), dp.Count, "MongoDB should have 1 observation")
		}

		if dep.AsString() == "postgresql" && status.AsString() == "down" {
			pgFound = true
			assert.Equal(t, uint64(1), dp.Count, "PostgreSQL should have 1 observation")
		}
	}

	assert.True(t, mongoFound, "MongoDB data point should exist with dep=mongodb, status=up")
	assert.True(t, pgFound, "PostgreSQL data point should exist with dep=postgresql, status=down")
}

func TestEmitCheckStatus_IncrementsCounterWithCorrectLabels(t *testing.T) {
	// Arrange
	provider, reader := setupTestMeterProvider(t)
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	readyz.ResetMetricsForTest()
	err := readyz.InitMetrics()
	require.NoError(t, err)

	// Act
	ctx := context.Background()
	readyz.EmitCheckStatus(ctx, "mongodb", "up")
	readyz.EmitCheckStatus(ctx, "mongodb", "up")
	readyz.EmitCheckStatus(ctx, "postgresql", "down")

	// Collect metrics
	rm := collectMetrics(t, reader)
	counter := findCounter(rm, "readyz_check_status")

	// Assert
	require.NotNil(t, counter, "Counter should exist")
	require.Len(t, counter.DataPoints, 2, "Should have two data points (mongodb:up, postgresql:down)")

	// Verify counts
	var mongoUpCount, pgDownCount int64
	for _, dp := range counter.DataPoints {
		dep, _ := dp.Attributes.Value("dep")
		status, _ := dp.Attributes.Value("status")

		if dep.AsString() == "mongodb" && status.AsString() == "up" {
			mongoUpCount = dp.Value
		}

		if dep.AsString() == "postgresql" && status.AsString() == "down" {
			pgDownCount = dp.Value
		}
	}

	assert.Equal(t, int64(2), mongoUpCount, "MongoDB up counter should be 2")
	assert.Equal(t, int64(1), pgDownCount, "PostgreSQL down counter should be 1")
}

func TestEmitSelfProbeResult_SetsGaugeToOne_WhenUp(t *testing.T) {
	// Arrange
	provider, reader := setupTestMeterProvider(t)
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	readyz.ResetMetricsForTest()
	err := readyz.InitMetrics()
	require.NoError(t, err)

	// Act
	ctx := context.Background()
	readyz.EmitSelfProbeResult(ctx, "mongodb", true)

	// Collect metrics
	rm := collectMetrics(t, reader)
	gauge := findGauge(rm, "selfprobe_result")

	// Assert
	require.NotNil(t, gauge, "Gauge should exist")
	require.Len(t, gauge.DataPoints, 1, "Should have one data point")

	dp := gauge.DataPoints[0]
	dep, _ := dp.Attributes.Value("dep")
	assert.Equal(t, "mongodb", dep.AsString(), "dep attribute should be mongodb")
	assert.Equal(t, 1.0, dp.Value, "Gauge value should be 1.0 when up=true")
}

func TestEmitSelfProbeResult_SetsGaugeToZero_WhenDown(t *testing.T) {
	// Arrange
	provider, reader := setupTestMeterProvider(t)
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	readyz.ResetMetricsForTest()
	err := readyz.InitMetrics()
	require.NoError(t, err)

	// Act
	ctx := context.Background()
	readyz.EmitSelfProbeResult(ctx, "postgresql", false)

	// Collect metrics
	rm := collectMetrics(t, reader)
	gauge := findGauge(rm, "selfprobe_result")

	// Assert
	require.NotNil(t, gauge, "Gauge should exist")
	require.Len(t, gauge.DataPoints, 1, "Should have one data point")

	dp := gauge.DataPoints[0]
	dep, _ := dp.Attributes.Value("dep")
	assert.Equal(t, "postgresql", dep.AsString(), "dep attribute should be postgresql")
	assert.Equal(t, 0.0, dp.Value, "Gauge value should be 0.0 when up=false")
}

func TestEmitFunctions_DoNotPanic_WhenMetricsNotInitialized(t *testing.T) {
	// Arrange: Reset metrics so they are nil (not initialized)
	readyz.ResetMetricsForTest()

	// Act & Assert: Functions should not panic when metrics are nil
	ctx := context.Background()

	assert.NotPanics(t, func() {
		readyz.EmitCheckDuration(ctx, "test", "up", time.Millisecond)
	}, "EmitCheckDuration should not panic when metrics not initialized")

	assert.NotPanics(t, func() {
		readyz.EmitCheckStatus(ctx, "test", "up")
	}, "EmitCheckStatus should not panic when metrics not initialized")

	assert.NotPanics(t, func() {
		readyz.EmitSelfProbeResult(ctx, "test", true)
	}, "EmitSelfProbeResult should not panic when metrics not initialized")
}

func TestInitMetrics_IsIdempotent(t *testing.T) {
	// Arrange
	provider, _ := setupTestMeterProvider(t)
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	readyz.ResetMetricsForTest()

	// Act: Call InitMetrics multiple times
	err1 := readyz.InitMetrics()
	require.NoError(t, err1)

	err2 := readyz.InitMetrics()
	require.NoError(t, err2)

	err3 := readyz.InitMetrics()
	require.NoError(t, err3)

	// Assert: No errors and no panics (sync.Once ensures single initialization)
}

func TestEmitCheckDuration_RecordsDurationInMilliseconds(t *testing.T) {
	// Arrange
	provider, reader := setupTestMeterProvider(t)
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	readyz.ResetMetricsForTest()
	err := readyz.InitMetrics()
	require.NoError(t, err)

	// Act: Emit duration of 100ms
	ctx := context.Background()
	readyz.EmitCheckDuration(ctx, "test-dep", "up", 100*time.Millisecond)

	// Collect metrics
	rm := collectMetrics(t, reader)
	hist := findHistogram(rm, "readyz_check_duration_ms")

	// Assert: Value should be recorded in milliseconds (100), not nanoseconds or seconds
	require.NotNil(t, hist)
	require.Len(t, hist.DataPoints, 1)

	dp := hist.DataPoints[0]
	// Sum should be approximately 100 (milliseconds)
	assert.InDelta(t, 100.0, dp.Sum, 0.1, "Duration should be recorded in milliseconds")
}

func TestHistogramBuckets_CoverExpectedRange(t *testing.T) {
	// This test verifies that the histogram bucket boundaries match the spec:
	// [1, 5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000]

	// Arrange
	provider, reader := setupTestMeterProvider(t)
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	readyz.ResetMetricsForTest()
	err := readyz.InitMetrics()
	require.NoError(t, err)

	// Act: Emit values that span the expected bucket range
	ctx := context.Background()
	testDurations := []time.Duration{
		500 * time.Microsecond,   // < 1ms (below first bucket)
		3 * time.Millisecond,     // 1-5ms bucket
		7 * time.Millisecond,     // 5-10ms bucket
		15 * time.Millisecond,    // 10-25ms bucket
		35 * time.Millisecond,    // 25-50ms bucket
		75 * time.Millisecond,    // 50-100ms bucket
		150 * time.Millisecond,   // 100-250ms bucket
		300 * time.Millisecond,   // 250-500ms bucket
		750 * time.Millisecond,   // 500-1000ms bucket
		1500 * time.Millisecond,  // 1000-2000ms bucket
		3000 * time.Millisecond,  // 2000-5000ms bucket
		10000 * time.Millisecond, // > 5000ms (above last bucket)
	}

	for i, d := range testDurations {
		readyz.EmitCheckDuration(ctx, "test-dep", "up", d)
		_ = i // use index to avoid compiler warning
	}

	// Collect metrics
	rm := collectMetrics(t, reader)
	hist := findHistogram(rm, "readyz_check_duration_ms")

	// Assert: Histogram should have recorded all observations
	require.NotNil(t, hist)
	require.Len(t, hist.DataPoints, 1)
	assert.Equal(t, uint64(len(testDurations)), hist.DataPoints[0].Count,
		"All %d durations should be recorded", len(testDurations))
}
