// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package testutil provides testing utilities for the flowker application.
package testutil

import (
	"time"

	"github.com/LerianStudio/flowker/pkg/clock"
)

// DefaultTestTime is the standard fixed time used in tests.
// Value: 2024-01-15 10:30:00 UTC
var DefaultTestTime = time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

// MockClock is a test double for clock.Clock that returns a fixed time.
type MockClock struct {
	FixedTime time.Time
}

// Now returns the fixed time.
func (m MockClock) Now() time.Time {
	return m.FixedTime
}

// NewMockClock creates a new MockClock with the given fixed time.
func NewMockClock(t time.Time) clock.Clock {
	return MockClock{FixedTime: t}
}

// NewDefaultMockClock creates a MockClock with the default test time.
func NewDefaultMockClock() clock.Clock {
	return MockClock{FixedTime: DefaultTestTime}
}

// Ptr returns a pointer to any value. Generic helper for tests.
func Ptr[T any](v T) *T {
	return &v
}
