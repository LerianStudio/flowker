// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package http

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeCursor(t *testing.T) {
	tests := []struct {
		name        string
		cursor      string
		expected    Cursor
		expectError bool
	}{
		{
			name: "valid cursor with all fields",
			cursor: func() string {
				c := Cursor{ID: "123", SortValue: "2024-01-01", SortBy: "createdAt", SortOrder: "DESC", PointsNext: true}
				b, _ := json.Marshal(c)

				return base64.StdEncoding.EncodeToString(b)
			}(),
			expected:    Cursor{ID: "123", SortValue: "2024-01-01", SortBy: "createdAt", SortOrder: "DESC", PointsNext: true},
			expectError: false,
		},
		{
			name: "valid cursor with minimal fields",
			cursor: func() string {
				c := Cursor{ID: "456", PointsNext: false}
				b, _ := json.Marshal(c)

				return base64.StdEncoding.EncodeToString(b)
			}(),
			expected:    Cursor{ID: "456", PointsNext: false},
			expectError: false,
		},
		{
			name: "valid cursor with empty ID",
			cursor: func() string {
				c := Cursor{ID: "", PointsNext: true}
				b, _ := json.Marshal(c)

				return base64.StdEncoding.EncodeToString(b)
			}(),
			expected:    Cursor{ID: "", PointsNext: true},
			expectError: false,
		},
		{
			name:        "invalid base64 encoding",
			cursor:      "not-valid-base64!!!",
			expected:    Cursor{},
			expectError: true,
		},
		{
			name:        "valid base64 but invalid JSON",
			cursor:      base64.StdEncoding.EncodeToString([]byte("not json")),
			expected:    Cursor{},
			expectError: true,
		},
		{
			name:        "empty cursor string",
			cursor:      "",
			expected:    Cursor{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DecodeCursor(tt.cursor)

			if tt.expectError {
				require.Error(t, err)
				assert.Equal(t, Cursor{}, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestEncodeCursor(t *testing.T) {
	tests := []struct {
		name        string
		cursor      Cursor
		expectError bool
	}{
		{
			name: "encode cursor with all fields",
			cursor: Cursor{
				ID:         "test-123",
				SortValue:  "2024-01-15",
				SortBy:     "createdAt",
				SortOrder:  "ASC",
				PointsNext: true,
			},
			expectError: false,
		},
		{
			name: "encode cursor with minimal fields",
			cursor: Cursor{
				ID:         "test-456",
				PointsNext: false,
			},
			expectError: false,
		},
		{
			name:        "encode empty cursor",
			cursor:      Cursor{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded, err := EncodeCursor(tt.cursor)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, encoded)

				// Verify round-trip
				decoded, err := DecodeCursor(encoded)
				require.NoError(t, err)
				assert.Equal(t, tt.cursor, decoded)
			}
		})
	}
}

func TestCursor_Struct(t *testing.T) {
	tests := []struct {
		name   string
		cursor Cursor
	}{
		{
			name: "cursor with all fields set",
			cursor: Cursor{
				ID:         "test-id",
				SortValue:  "sort-value",
				SortBy:     "name",
				SortOrder:  "DESC",
				PointsNext: true,
			},
		},
		{
			name:   "cursor with zero values",
			cursor: Cursor{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON serialization round-trip
			data, err := json.Marshal(tt.cursor)
			require.NoError(t, err)

			var decoded Cursor
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.cursor, decoded)
		})
	}
}
