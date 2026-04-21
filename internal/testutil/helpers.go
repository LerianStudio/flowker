// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package testutil provides shared test helper functions for use across test files.
// These helpers should be used instead of duplicating pointer helper functions in each test file.
package testutil

import (
	"time"

	"github.com/google/uuid"
)

// UUIDPtr returns a pointer to the given UUID.
func UUIDPtr(u uuid.UUID) *uuid.UUID {
	return &u
}

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string {
	return &s
}

// Int64Ptr returns a pointer to the given int64.
func Int64Ptr(i int64) *int64 {
	return &i
}

// IntPtr returns a pointer to the given int.
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns a pointer to the given bool.
func BoolPtr(b bool) *bool {
	return &b
}

// TimePtr returns a pointer to the given time.Time.
func TimePtr(t time.Time) *time.Time {
	return &t
}

// Float64Ptr returns a pointer to the given float64.
func Float64Ptr(f float64) *float64 {
	return &f
}
