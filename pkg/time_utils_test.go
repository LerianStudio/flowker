// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package pkg

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsValidDate(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		expected bool
	}{
		{
			name:     "valid date format",
			date:     "2024-01-15",
			expected: true,
		},
		{
			name:     "valid date at year boundary",
			date:     "2024-12-31",
			expected: true,
		},
		{
			name:     "valid date first day of year",
			date:     "2024-01-01",
			expected: true,
		},
		{
			name:     "invalid date format - wrong separator",
			date:     "2024/01/15",
			expected: false,
		},
		{
			name:     "invalid date format - day month reversed",
			date:     "15-01-2024",
			expected: false,
		},
		{
			name:     "invalid date - month out of range",
			date:     "2024-13-01",
			expected: false,
		},
		{
			name:     "invalid date - day out of range",
			date:     "2024-01-32",
			expected: false,
		},
		{
			name:     "invalid date - empty string",
			date:     "",
			expected: false,
		},
		{
			name:     "invalid date - random text",
			date:     "not-a-date",
			expected: false,
		},
		{
			name:     "valid leap year date",
			date:     "2024-02-29",
			expected: true,
		},
		{
			name:     "invalid leap year date - non-leap year",
			date:     "2023-02-29",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidDate(tt.date)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInitialDateBeforeFinalDate(t *testing.T) {
	tests := []struct {
		name     string
		initial  time.Time
		final    time.Time
		expected bool
	}{
		{
			name:     "initial before final",
			initial:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			final:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "initial equals final",
			initial:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			final:    time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "initial after final",
			initial:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			final:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInitialDateBeforeFinalDate(tt.initial, tt.final)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDateRangeWithinMonthLimit(t *testing.T) {
	tests := []struct {
		name     string
		initial  time.Time
		final    time.Time
		limit    int
		expected bool
	}{
		{
			name:     "range within limit",
			initial:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			final:    time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			limit:    3,
			expected: true,
		},
		{
			name:     "range exactly at limit",
			initial:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			final:    time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
			limit:    3,
			expected: true,
		},
		{
			name:     "range exceeds limit",
			initial:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			final:    time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC),
			limit:    3,
			expected: false,
		},
		{
			name:     "same day range",
			initial:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			final:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			limit:    1,
			expected: true,
		},
		{
			name:     "zero month limit with same day",
			initial:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			final:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			limit:    0,
			expected: true,
		},
		{
			name:     "negative month limit",
			initial:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			final:    time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			limit:    -1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDateRangeWithinMonthLimit(tt.initial, tt.final, tt.limit)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeDate(t *testing.T) {
	baseDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	addDays := 5
	subtractDays := -5
	zeroDays := 0

	tests := []struct {
		name     string
		date     time.Time
		days     *int
		expected string
	}{
		{
			name:     "normalize without adjustment",
			date:     baseDate,
			days:     nil,
			expected: "2024-01-15",
		},
		{
			name:     "normalize with positive days",
			date:     baseDate,
			days:     &addDays,
			expected: "2024-01-20",
		},
		{
			name:     "normalize with negative days",
			date:     baseDate,
			days:     &subtractDays,
			expected: "2024-01-10",
		},
		{
			name:     "normalize with zero days",
			date:     baseDate,
			days:     &zeroDays,
			expected: "2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeDate(tt.date, tt.days)
			assert.Equal(t, tt.expected, result)
		})
	}
}
