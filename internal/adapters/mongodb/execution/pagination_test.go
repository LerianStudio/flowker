// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package execution

import (
	"testing"
	"time"

	nethttp "github.com/LerianStudio/flowker/pkg/net/http"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSortValue_StartedAt_ValidTime(t *testing.T) {
	value := "2024-06-15T10:30:45.123Z"

	result, err := parseSortValue(value, "startedAt")

	require.NoError(t, err)

	parsed, ok := result.(time.Time)
	require.True(t, ok, "expected time.Time, got %T", result)
	assert.Equal(t, 2024, parsed.Year())
	assert.Equal(t, time.June, parsed.Month())
	assert.Equal(t, 15, parsed.Day())
	assert.Equal(t, 10, parsed.Hour())
	assert.Equal(t, 30, parsed.Minute())
	assert.Equal(t, 45, parsed.Second())
}

func TestParseSortValue_CompletedAt_ValidTime(t *testing.T) {
	value := "2025-12-31T23:59:59.999Z"

	result, err := parseSortValue(value, "completedAt")

	require.NoError(t, err)

	parsed, ok := result.(time.Time)
	require.True(t, ok, "expected time.Time, got %T", result)
	assert.Equal(t, 2025, parsed.Year())
	assert.Equal(t, time.December, parsed.Month())
}

func TestParseSortValue_EmptyStartedAt_ReturnsError(t *testing.T) {
	_, err := parseSortValue("", "startedAt")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid empty time value")
}

func TestParseSortValue_EmptyCompletedAt_ReturnsZeroTime(t *testing.T) {
	result, err := parseSortValue("", "completedAt")

	require.NoError(t, err)

	parsed, ok := result.(time.Time)
	require.True(t, ok, "expected time.Time, got %T", result)
	assert.True(t, parsed.IsZero(), "expected zero time for nil completed_at cursor")
}

func TestParseSortValue_EmptyUnknownField_ReturnsEmptyString(t *testing.T) {
	result, err := parseSortValue("", "unknown_field")

	require.NoError(t, err)
	assert.Equal(t, "", result)
}

func TestParseSortValue_InvalidTimeFormat(t *testing.T) {
	_, err := parseSortValue("not-a-time", "startedAt")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid time value in cursor")
}

func TestParseSortValue_UnknownSortField_ReturnsString(t *testing.T) {
	value := "some-value"

	result, err := parseSortValue(value, "unknown_field")

	require.NoError(t, err)
	assert.Equal(t, "some-value", result)
}

func TestGetExecutionSortValue_StartedAt(t *testing.T) {
	started := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	exec := model.NewWorkflowExecutionFromDB(
		uuid.New(), uuid.New(), model.ExecutionStatusRunning,
		nil, nil, nil, 0, 0, nil, nil, started, nil,
	)

	value := getExecutionSortValue(exec, "startedAt")

	assert.Equal(t, "2024-03-15T14:30:00.000Z", value)
}

func TestGetExecutionSortValue_CompletedAt_NonNil(t *testing.T) {
	started := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	completed := time.Date(2024, 3, 15, 14, 31, 5, 0, time.UTC)
	exec := model.NewWorkflowExecutionFromDB(
		uuid.New(), uuid.New(), model.ExecutionStatusCompleted,
		nil, nil, nil, 0, 0, nil, nil, started, &completed,
	)

	value := getExecutionSortValue(exec, "completedAt")

	assert.Equal(t, "2024-03-15T14:31:05.000Z", value)
}

func TestGetExecutionSortValue_CompletedAt_Nil(t *testing.T) {
	started := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	exec := model.NewWorkflowExecutionFromDB(
		uuid.New(), uuid.New(), model.ExecutionStatusRunning,
		nil, nil, nil, 0, 0, nil, nil, started, nil,
	)

	value := getExecutionSortValue(exec, "completedAt")

	assert.Equal(t, "", value)
}

func TestGetExecutionSortValue_DefaultSortField(t *testing.T) {
	started := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	exec := model.NewWorkflowExecutionFromDB(
		uuid.New(), uuid.New(), model.ExecutionStatusRunning,
		nil, nil, nil, 0, 0, nil, nil, started, nil,
	)

	value := getExecutionSortValue(exec, "anything")

	assert.Equal(t, "2024-01-01T00:00:00.000Z", value)
}

func TestParseSortValue_RoundTrip_StartedAt(t *testing.T) {
	// Simulate: getExecutionSortValue → encode cursor → decode cursor → parseSortValue
	original := time.Date(2024, 6, 15, 10, 30, 45, 123000000, time.UTC)
	formatted := original.Format(cursorTimeFormat)

	parsed, err := parseSortValue(formatted, "startedAt")

	require.NoError(t, err)

	result, ok := parsed.(time.Time)
	require.True(t, ok)
	assert.True(t, result.Equal(original), "round-trip failed: expected %v, got %v", original, result)
}

func TestBuildNextCursor_RoundTrip_DecodesCorrectly(t *testing.T) {
	started := time.Date(2024, 6, 15, 10, 30, 45, 123000000, time.UTC)
	execID := uuid.New()
	exec := model.NewWorkflowExecutionFromDB(
		execID, uuid.New(), model.ExecutionStatusCompleted,
		nil, nil, nil, 0, 0, nil, nil, started, nil,
	)

	items := []*model.WorkflowExecution{exec}
	cursorStr, err := buildNextCursor(items, true, "startedAt", "DESC")
	require.NoError(t, err)
	require.NotEmpty(t, cursorStr)

	// Decode the cursor
	cur, err := nethttp.DecodeCursor(cursorStr)
	require.NoError(t, err)
	assert.Equal(t, execID.String(), cur.ID)
	assert.Equal(t, "startedAt", cur.SortBy)
	assert.Equal(t, "DESC", cur.SortOrder)
	assert.True(t, cur.PointsNext)

	// Parse the sort value back — should be time.Time
	parsed, err := parseSortValue(cur.SortValue, cur.SortBy)
	require.NoError(t, err)

	result, ok := parsed.(time.Time)
	require.True(t, ok, "expected time.Time after round-trip, got %T", parsed)
	assert.True(t, result.Equal(started), "time mismatch: expected %v, got %v", started, result)
}

func TestBuildNextCursor_NoMore_ReturnsEmpty(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	exec := model.NewWorkflowExecutionFromDB(
		uuid.New(), uuid.New(), model.ExecutionStatusCompleted,
		nil, nil, nil, 0, 0, nil, nil, fixedTime, nil,
	)

	cursorStr, err := buildNextCursor([]*model.WorkflowExecution{exec}, false, "startedAt", "DESC")

	require.NoError(t, err)
	assert.Empty(t, cursorStr)
}

func TestBuildNextCursor_EmptyItems_ReturnsEmpty(t *testing.T) {
	cursorStr, err := buildNextCursor(nil, true, "startedAt", "DESC")

	require.NoError(t, err)
	assert.Empty(t, cursorStr)
}

func TestMapExecutionSortField(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"startedAt", "startedAt"},
		{"completedAt", "completedAt"},
		{"unknown", "startedAt"},
		{"", "startedAt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, mapExecutionSortField(tt.input))
		})
	}
}
