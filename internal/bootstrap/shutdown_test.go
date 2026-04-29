// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package bootstrap_test

import (
	"testing"
	"time"

	"github.com/LerianStudio/flowker/internal/bootstrap"
	"github.com/stretchr/testify/assert"
)

func TestIsDraining_InitialValue(t *testing.T) {
	// Reset to known state for test isolation
	bootstrap.SetDraining(false)

	// Assert: Initial value should be false
	assert.False(t, bootstrap.IsDraining(), "IsDraining should be false initially")
}

func TestSetDraining_SetsToTrue(t *testing.T) {
	// Arrange: Start with false
	bootstrap.SetDraining(false)

	// Act: Set to true
	bootstrap.SetDraining(true)

	// Assert: Value should be true
	assert.True(t, bootstrap.IsDraining(), "IsDraining should be true after SetDraining(true)")
}

func TestSetDraining_SetsToFalse(t *testing.T) {
	// Arrange: Start with true
	bootstrap.SetDraining(true)

	// Act: Set to false
	bootstrap.SetDraining(false)

	// Assert: Value should be false
	assert.False(t, bootstrap.IsDraining(), "IsDraining should be false after SetDraining(false)")
}

func TestDefaultDrainGracePeriod(t *testing.T) {
	// Assert: Default grace period should be 12 seconds per Ring Standards
	// (>= K8s periodSeconds * failureThreshold + buffer)
	assert.Equal(t, 12*time.Second, bootstrap.DefaultDrainGracePeriod,
		"DefaultDrainGracePeriod should be 12 seconds")
}

func TestGracefulShutdownConfig_Fields(t *testing.T) {
	// Test that GracefulShutdownConfig struct has the expected fields
	cfg := bootstrap.GracefulShutdownConfig{
		App:              nil, // fiber.App
		Logger:           nil, // libLog.Logger
		DrainGracePeriod: 10 * time.Second,
		OnShutdown:       nil,
	}

	assert.Equal(t, 10*time.Second, cfg.DrainGracePeriod)
	assert.Nil(t, cfg.App)
	assert.Nil(t, cfg.Logger)
	assert.Nil(t, cfg.OnShutdown)
}
