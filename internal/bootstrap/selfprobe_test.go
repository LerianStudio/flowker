// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package bootstrap_test

import (
	"testing"

	"github.com/LerianStudio/flowker/internal/bootstrap"
	"github.com/stretchr/testify/assert"
)

func TestSelfProbeOK_InitialValue(t *testing.T) {
	// Reset to known state for test isolation
	bootstrap.SetSelfProbeOK(false)

	// Assert: Initial value should be false (unhealthy until proven otherwise)
	assert.False(t, bootstrap.SelfProbeOK(), "SelfProbeOK should be false initially")
}

func TestSetSelfProbeOK_SetsToTrue(t *testing.T) {
	// Arrange: Start with false
	bootstrap.SetSelfProbeOK(false)

	// Act: Set to true
	bootstrap.SetSelfProbeOK(true)

	// Assert: Value should be true
	assert.True(t, bootstrap.SelfProbeOK(), "SelfProbeOK should be true after SetSelfProbeOK(true)")
}

func TestSetSelfProbeOK_SetsToFalse(t *testing.T) {
	// Arrange: Start with true
	bootstrap.SetSelfProbeOK(true)

	// Act: Set to false
	bootstrap.SetSelfProbeOK(false)

	// Assert: Value should be false
	assert.False(t, bootstrap.SelfProbeOK(), "SelfProbeOK should be false after SetSelfProbeOK(false)")
}

func TestSelfProbeResult_StructFields(t *testing.T) {
	// Test that SelfProbeResult struct has the expected fields
	result := bootstrap.SelfProbeResult{
		Name:   "mongodb",
		Status: "up",
		Error:  nil,
	}

	assert.Equal(t, "mongodb", result.Name)
	assert.Equal(t, "up", result.Status)
	assert.Nil(t, result.Error)
}

func TestSelfProbeResult_WithError(t *testing.T) {
	// Test SelfProbeResult with an error
	err := assert.AnError
	result := bootstrap.SelfProbeResult{
		Name:   "postgresql",
		Status: "down",
		Error:  err,
	}

	assert.Equal(t, "postgresql", result.Name)
	assert.Equal(t, "down", result.Status)
	assert.Equal(t, err, result.Error)
}
