// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package http

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryHeader_ToCursorPagination(t *testing.T) {
	tests := []struct {
		name     string
		qh       QueryHeader
		expected Pagination
	}{
		{
			name: "converts all fields correctly",
			qh: QueryHeader{
				Limit:     20,
				Cursor:    "cursor123",
				SortOrder: "asc",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			expected: Pagination{
				Limit:     20,
				Cursor:    "cursor123",
				SortOrder: "asc",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "converts with default values",
			qh:   QueryHeader{},
			expected: Pagination{
				Limit:     0,
				Cursor:    "",
				SortOrder: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.qh.ToCursorPagination()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateParameters(t *testing.T) {
	tests := []struct {
		name        string
		params      map[string]string
		expectError bool
		validate    func(t *testing.T, qh *QueryHeader)
	}{
		{
			name:        "empty params returns defaults",
			params:      map[string]string{},
			expectError: false,
			validate: func(t *testing.T, qh *QueryHeader) {
				assert.Equal(t, 10, qh.Limit)
				assert.Equal(t, "desc", qh.SortOrder)
				assert.False(t, qh.UseMetadata)
			},
		},
		{
			name: "custom limit",
			params: map[string]string{
				"limit": "25",
			},
			expectError: false,
			validate: func(t *testing.T, qh *QueryHeader) {
				assert.Equal(t, 25, qh.Limit)
			},
		},
		{
			name: "invalid limit returns error",
			params: map[string]string{
				"limit": "not-a-number",
			},
			expectError: true,
		},
		{
			name: "sort order asc",
			params: map[string]string{
				"sort_order": "ASC",
			},
			expectError: false,
			validate: func(t *testing.T, qh *QueryHeader) {
				assert.Equal(t, "asc", qh.SortOrder)
			},
		},
		{
			name: "sort order desc",
			params: map[string]string{
				"sort_order": "DESC",
			},
			expectError: false,
			validate: func(t *testing.T, qh *QueryHeader) {
				assert.Equal(t, "desc", qh.SortOrder)
			},
		},
		{
			name: "invalid sort order",
			params: map[string]string{
				"sort_order": "invalid",
			},
			expectError: true,
		},
		{
			name: "pagination limit exceeded",
			params: map[string]string{
				"limit": "500",
			},
			expectError: true,
		},
		{
			name: "metadata param sets useMetadata",
			params: map[string]string{
				"metadata.key": "value",
			},
			expectError: false,
			validate: func(t *testing.T, qh *QueryHeader) {
				assert.True(t, qh.UseMetadata)
				assert.NotNil(t, qh.Metadata)
			},
		},
		{
			name: "cursor param",
			params: map[string]string{
				"cursor": "eyJpZCI6IjEyMyIsInBuIjp0cnVlfQ==",
			},
			expectError: false,
			validate: func(t *testing.T, qh *QueryHeader) {
				assert.Equal(t, "eyJpZCI6IjEyMyIsInBuIjp0cnVlfQ==", qh.Cursor)
			},
		},
		{
			name: "invalid cursor param",
			params: map[string]string{
				"cursor": "invalid-cursor",
			},
			expectError: true,
		},
		{
			name: "invalid start_date format returns error",
			params: map[string]string{
				"start_date": "not-a-date",
				"end_date":   "2024-01-31",
			},
			expectError: true,
		},
		{
			name: "invalid end_date format returns error",
			params: map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "not-a-date",
			},
			expectError: true,
		},
		{
			name: "start_date only - error",
			params: map[string]string{
				"start_date": "2024-01-01",
			},
			expectError: true,
		},
		{
			name: "end_date only - error",
			params: map[string]string{
				"end_date": "2024-01-31",
			},
			expectError: true,
		},
		{
			name: "valid date range",
			params: map[string]string{
				"start_date": "2024-01-01",
				"end_date":   "2024-01-31",
			},
			expectError: false,
			validate: func(t *testing.T, qh *QueryHeader) {
				assert.False(t, qh.StartDate.IsZero())
				assert.False(t, qh.EndDate.IsZero())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateParameters(tt.params)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestPagination_Struct(t *testing.T) {
	tests := []struct {
		name     string
		p        Pagination
		validate func(t *testing.T, p Pagination)
	}{
		{
			name: "pagination fields are set correctly",
			p: Pagination{
				Limit:     10,
				Cursor:    "cursor123",
				SortOrder: "asc",
				StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				EndDate:   time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			},
			validate: func(t *testing.T, p Pagination) {
				assert.Equal(t, 10, p.Limit)
				assert.Equal(t, "cursor123", p.Cursor)
				assert.Equal(t, "asc", p.SortOrder)
			},
		},
		{
			name: "pagination with zero values",
			p:    Pagination{},
			validate: func(t *testing.T, p Pagination) {
				assert.Equal(t, 0, p.Limit)
				assert.Equal(t, "", p.Cursor)
				assert.Equal(t, "", p.SortOrder)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.p)
		})
	}
}
