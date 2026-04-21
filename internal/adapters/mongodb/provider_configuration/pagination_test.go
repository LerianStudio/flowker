// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package providerconfiguration

import (
	"testing"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/pagination"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Tests for pagination.ParseSortValue exercised through this repository's
// sort-field set; keeps parity with getSortValue and mapSortField below.
// ---------------------------------------------------------------------------

func TestParseSortValue_CreatedAt_ValidTime(t *testing.T) {
	ts := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	raw := ts.Format(pagination.SortTimeFormat)

	result, err := pagination.ParseSortValue(raw, "createdAt")

	require.NoError(t, err)
	parsed, ok := result.(time.Time)
	require.True(t, ok, "expected time.Time, got %T", result)
	assert.True(t, ts.Equal(parsed), "expected %v, got %v", ts, parsed)
}

func TestParseSortValue_UpdatedAt_ValidTime(t *testing.T) {
	ts := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	raw := ts.Format(pagination.SortTimeFormat)

	result, err := pagination.ParseSortValue(raw, "updatedAt")

	require.NoError(t, err)
	parsed, ok := result.(time.Time)
	require.True(t, ok, "expected time.Time, got %T", result)
	assert.True(t, ts.Equal(parsed), "expected %v, got %v", ts, parsed)
}

func TestParseSortValue_EmptyCreatedAt_ReturnsZeroTime(t *testing.T) {
	result, err := pagination.ParseSortValue("", "createdAt")

	require.NoError(t, err)
	parsed, ok := result.(time.Time)
	require.True(t, ok, "expected time.Time, got %T", result)
	assert.True(t, parsed.IsZero(), "expected zero time for empty string")
}

func TestParseSortValue_InvalidTimeFormat(t *testing.T) {
	_, err := pagination.ParseSortValue("not-a-date", "createdAt")

	assert.Error(t, err, "expected error for invalid time format")
}

func TestParseSortValue_NameField_ReturnsString(t *testing.T) {
	result, err := pagination.ParseSortValue("my-provider-config", "name")

	require.NoError(t, err)
	s, ok := result.(string)
	require.True(t, ok, "expected string, got %T", result)
	assert.Equal(t, "my-provider-config", s)
}

func TestParseSortValue_RoundTrip_CreatedAt(t *testing.T) {
	ts := time.Date(2025, 3, 10, 8, 45, 30, 0, time.UTC)

	pc := model.ReconstructProviderConfiguration(
		uuid.New(),
		"round-trip-provider",
		nil,
		"test-provider-id",
		map[string]any{"key": "value"},
		model.ProviderConfigStatusActive,
		nil,
		ts,
		ts,
	)

	// getSortValue produces the string that goes into the cursor.
	raw := getSortValue(pc, "createdAt")

	// pagination.ParseSortValue should convert it back to a time.Time.
	result, err := pagination.ParseSortValue(raw, "createdAt")

	require.NoError(t, err)
	parsed, ok := result.(time.Time)
	require.True(t, ok, "expected time.Time, got %T", result)
	assert.True(t, ts.Equal(parsed), "round-trip failed: expected %v, got %v", ts, parsed)
}

// ---------------------------------------------------------------------------
// Tests for mapSortField (already exists — should pass)
// ---------------------------------------------------------------------------

func TestMapSortField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "createdAt camelCase", input: "createdAt", expected: "createdAt"},
		{name: "updatedAt camelCase", input: "updatedAt", expected: "updatedAt"},
		{name: "name", input: "name", expected: "name"},
		{name: "snake_case not supported, defaults to createdAt", input: "updated_at", expected: "createdAt"},
		{name: "unknown defaults to createdAt", input: "unknown_field", expected: "createdAt"},
		{name: "empty defaults to createdAt", input: "", expected: "createdAt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapSortField(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Tests for getSortValue (already exists — should pass)
// ---------------------------------------------------------------------------

func newTestProviderConfig(t *testing.T) *model.ProviderConfiguration {
	t.Helper()

	createdAt := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 6, 16, 12, 30, 0, 0, time.UTC)

	return model.ReconstructProviderConfiguration(
		uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		"test-provider-config",
		nil,
		"test-provider-id",
		map[string]any{"endpoint": "https://example.com"},
		model.ProviderConfigStatusActive,
		nil,
		createdAt,
		updatedAt,
	)
}

func TestGetSortValue_CreatedAt(t *testing.T) {
	pc := newTestProviderConfig(t)

	result := getSortValue(pc, "createdAt")

	expected := "2025-06-15T10:00:00.000Z"
	assert.Equal(t, expected, result)
}

func TestGetSortValue_UpdatedAt(t *testing.T) {
	pc := newTestProviderConfig(t)

	result := getSortValue(pc, "updatedAt")

	expected := "2025-06-16T12:30:00.000Z"
	assert.Equal(t, expected, result)
}

func TestGetSortValue_Name(t *testing.T) {
	pc := newTestProviderConfig(t)

	result := getSortValue(pc, "name")

	assert.Equal(t, "test-provider-config", result)
}

func TestGetSortValue_Default(t *testing.T) {
	pc := newTestProviderConfig(t)

	// Unknown sortBy falls back to createdAt.
	result := getSortValue(pc, "something_unknown")

	expected := "2025-06-15T10:00:00.000Z"
	assert.Equal(t, expected, result)
}
