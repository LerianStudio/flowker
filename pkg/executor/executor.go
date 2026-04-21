// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package executor defines the interfaces and types for workflow executors.
// Executors are built-in, statically compiled components that execute
// specific actions within a workflow (HTTP calls, gRPC, scripts, etc.).
package executor

// ID is a unique identifier for an executor type.
type ID string

// Executor defines the contract for a workflow executor.
// Executors are registered at compile time and validated using JSON Schema.
type Executor interface {
	// ID returns the unique identifier for this executor.
	ID() ID

	// Name returns the human-readable name for this executor.
	Name() string

	// Category returns the executor category (e.g., "HTTP", "gRPC", "Logic").
	Category() string

	// Version returns the executor version (e.g., "v1").
	Version() string

	// ProviderID returns the ID of the provider this executor belongs to.
	ProviderID() ProviderID

	// Schema returns the JSON Schema for validating executor configuration.
	Schema() string

	// ValidateConfig validates the given configuration against the executor's JSON Schema.
	ValidateConfig(config map[string]any) error
}

// TriggerID is a unique identifier for a trigger type.
type TriggerID string

// Trigger defines the contract for a workflow trigger.
// Triggers are entry points that start workflow execution.
type Trigger interface {
	// ID returns the unique identifier for this trigger.
	ID() TriggerID

	// Name returns the human-readable name for this trigger.
	Name() string

	// Version returns the trigger version (e.g., "v1").
	Version() string

	// Schema returns the JSON Schema for validating trigger configuration.
	Schema() string

	// ValidateConfig validates the given configuration against the trigger's JSON Schema.
	ValidateConfig(config map[string]any) error
}
