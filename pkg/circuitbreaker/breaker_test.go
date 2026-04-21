// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package circuitbreaker_test

import (
	"errors"
	"testing"

	"github.com/LerianStudio/flowker/pkg/circuitbreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	m := circuitbreaker.NewManager()
	assert.NotNil(t, m)
}

func TestManager_Execute_Success(t *testing.T) {
	m := circuitbreaker.NewManager()

	result, err := m.Execute("executor-1", func() (any, error) {
		return "ok", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestManager_Execute_Error(t *testing.T) {
	m := circuitbreaker.NewManager()
	expectedErr := errors.New("connection refused")

	result, err := m.Execute("executor-1", func() (any, error) {
		return nil, expectedErr
	})

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestManager_Execute_SameExecutorReusesBreaker(t *testing.T) {
	m := circuitbreaker.NewManager()
	callCount := 0

	for i := 0; i < 3; i++ {
		_, _ = m.Execute("executor-1", func() (any, error) {
			callCount++
			return "ok", nil
		})
	}

	assert.Equal(t, 3, callCount)
}

func TestManager_Execute_DifferentExecutorsIndependent(t *testing.T) {
	m := circuitbreaker.NewManager()
	errFail := errors.New("fail")

	// Fail executor-1 enough to trip circuit breaker (5 failures)
	for i := 0; i < int(circuitbreaker.DefaultFailureThreshold); i++ {
		_, _ = m.Execute("executor-1", func() (any, error) {
			return nil, errFail
		})
	}

	// executor-1 should now be open - next call fails with circuit breaker error
	_, err := m.Execute("executor-1", func() (any, error) {
		return "should not reach", nil
	})
	require.Error(t, err)

	// executor-2 should still work fine
	result, err := m.Execute("executor-2", func() (any, error) {
		return "ok", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

func TestManager_Execute_CircuitOpensAfterThreshold(t *testing.T) {
	m := circuitbreaker.NewManager()
	errFail := errors.New("fail")

	// Trip the circuit breaker
	for i := 0; i < int(circuitbreaker.DefaultFailureThreshold); i++ {
		_, _ = m.Execute("executor-trip", func() (any, error) {
			return nil, errFail
		})
	}

	// Next call should fail even though the function would succeed
	_, err := m.Execute("executor-trip", func() (any, error) {
		return "should not execute", nil
	})

	require.Error(t, err)
}

func TestManager_Execute_ConcurrentAccess(t *testing.T) {
	m := circuitbreaker.NewManager()
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			_, _ = m.Execute("concurrent-executor", func() (any, error) {
				return "ok", nil
			})
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
