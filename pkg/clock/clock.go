// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package clock provides time-related utilities with support for testing.
package clock

import "time"

// Clock provides time operations. Allows injection for testing.
type Clock interface {
	Now() time.Time
}

// RealClock implements Clock using the real system time.
type RealClock struct{}

// Now returns the current time in UTC.
func (RealClock) Now() time.Time {
	return time.Now().UTC()
}

// New creates a new RealClock instance.
func New() Clock {
	return RealClock{}
}
