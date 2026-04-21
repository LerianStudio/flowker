// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package executor

// ProviderID is a unique identifier for a provider type.
type ProviderID string

// Provider defines the contract for a workflow provider.
// Providers group related executors and declare a JSON Schema
// for configuration validation (e.g., credentials, URLs).
type Provider interface {
	// ID returns the unique identifier for this provider.
	ID() ProviderID

	// Name returns the human-readable name for this provider.
	Name() string

	// Description returns a brief description of this provider.
	Description() string

	// Version returns the provider version (e.g., "v1").
	Version() string

	// ConfigSchema returns the JSON Schema (Draft 2020-12) for
	// validating provider-level configuration (base URLs, credentials, etc.).
	ConfigSchema() string
}

// ExecutorRegistration pairs an Executor with its Runner for bulk registration.
type ExecutorRegistration struct {
	Executor Executor
	Runner   Runner
}
