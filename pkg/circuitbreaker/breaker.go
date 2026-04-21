// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package circuitbreaker provides a per-executor circuit breaker manager.
package circuitbreaker

import (
	"sync"
	"time"

	"github.com/sony/gobreaker/v2"
)

// DefaultMaxRequests is the max number of requests allowed in the half-open state.
const DefaultMaxRequests = 1

// DefaultInterval is the cyclic period of the closed state for clearing failure counts.
const DefaultInterval = 0 // 0 means never clear

// DefaultTimeout is the period of the open state before transitioning to half-open.
const DefaultTimeout = 30 * time.Second

// DefaultFailureThreshold is the number of failures before opening the circuit.
const DefaultFailureThreshold = 5

// Manager manages per-executor circuit breakers.
type Manager struct {
	mu       sync.RWMutex
	breakers map[string]*gobreaker.CircuitBreaker[any]
}

// NewManager creates a new circuit breaker Manager.
func NewManager() *Manager {
	return &Manager{
		breakers: make(map[string]*gobreaker.CircuitBreaker[any]),
	}
}

// Execute runs the function through the circuit breaker identified by executorID.
// If no breaker exists for the executorID, one is created with default settings.
func (m *Manager) Execute(executorID string, fn func() (any, error)) (any, error) {
	cb := m.getOrCreate(executorID)

	return cb.Execute(fn)
}

// getOrCreate returns an existing circuit breaker or creates a new one.
func (m *Manager) getOrCreate(executorID string) *gobreaker.CircuitBreaker[any] {
	m.mu.RLock()
	cb, ok := m.breakers[executorID]
	m.mu.RUnlock()

	if ok {
		return cb
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, ok = m.breakers[executorID]; ok {
		return cb
	}

	settings := gobreaker.Settings{
		Name:        executorID,
		MaxRequests: DefaultMaxRequests,
		Interval:    DefaultInterval,
		Timeout:     DefaultTimeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= uint32(DefaultFailureThreshold)
		},
	}

	cb = gobreaker.NewCircuitBreaker[any](settings)
	m.breakers[executorID] = cb

	return cb
}
