// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package base provides base implementations for executors and triggers
// with JSON Schema validation support.
package base

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Executor is a base implementation of the executor.Executor interface.
// It provides JSON Schema validation for executor configuration.
type Executor struct {
	id         executor.ID
	name       string
	category   string
	version    string
	providerID executor.ProviderID
	schema     string
	validator  *jsonschema.Schema
}

// NewExecutor creates a new base Executor with JSON Schema validation.
// Returns error if required fields are missing or the schema is invalid.
func NewExecutor(id executor.ID, name, category, version string, providerID executor.ProviderID, schema string) (*Executor, error) {
	if id == "" {
		return nil, fmt.Errorf("executor id is required")
	}

	if name == "" {
		return nil, fmt.Errorf("executor name is required")
	}

	if version == "" {
		return nil, fmt.Errorf("executor version is required")
	}

	if strings.TrimSpace(string(providerID)) == "" {
		return nil, fmt.Errorf("executor provider ID is required")
	}

	if schema == "" {
		return nil, fmt.Errorf("executor schema is required")
	}

	if strings.TrimSpace(category) == "" {
		category = "General"
	}

	var schemaDoc any
	if err := json.Unmarshal([]byte(schema), &schemaDoc); err != nil {
		return nil, fmt.Errorf("invalid schema JSON for executor %s: %w", id, err)
	}

	resourceID := fmt.Sprintf("urn:flowker:executor:%s", id)
	compiler := jsonschema.NewCompiler()

	if err := compiler.AddResource(resourceID, schemaDoc); err != nil {
		return nil, fmt.Errorf("invalid schema for executor %s: %w", id, err)
	}

	validator, err := compiler.Compile(resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema for executor %s: %w", id, err)
	}

	return &Executor{
		id:         id,
		name:       name,
		category:   category,
		version:    version,
		providerID: providerID,
		schema:     schema,
		validator:  validator,
	}, nil
}

// ID returns the executor's unique identifier.
func (e *Executor) ID() executor.ID {
	return e.id
}

// Name returns the executor's human-readable name.
func (e *Executor) Name() string {
	return e.name
}

// Category returns the executor's category.
func (e *Executor) Category() string {
	return e.category
}

// Version returns the executor's version.
func (e *Executor) Version() string {
	return e.version
}

// ProviderID returns the ID of the provider this executor belongs to.
func (e *Executor) ProviderID() executor.ProviderID {
	return e.providerID
}

// Schema returns the executor's JSON Schema.
func (e *Executor) Schema() string {
	return e.schema
}

// ValidateConfig validates the given configuration against the executor's JSON Schema.
func (e *Executor) ValidateConfig(config map[string]any) error {
	if err := e.validator.Validate(config); err != nil {
		return executor.NewExecutorConfigError(e.id, err)
	}

	return nil
}

// Verify Executor implements executor.Executor interface.
var _ executor.Executor = (*Executor)(nil)

// Trigger is a base implementation of the executor.Trigger interface.
// It provides JSON Schema validation for trigger configuration.
type Trigger struct {
	id        executor.TriggerID
	name      string
	version   string
	schema    string
	validator *jsonschema.Schema
}

// NewTrigger creates a new base Trigger with JSON Schema validation.
// Returns error if required fields are missing or the schema is invalid.
func NewTrigger(id executor.TriggerID, name, version, schema string) (*Trigger, error) {
	if id == "" {
		return nil, fmt.Errorf("trigger id is required")
	}

	if name == "" {
		return nil, fmt.Errorf("trigger name is required")
	}

	if version == "" {
		return nil, fmt.Errorf("trigger version is required")
	}

	if schema == "" {
		return nil, fmt.Errorf("trigger schema is required")
	}

	var schemaDoc any
	if err := json.Unmarshal([]byte(schema), &schemaDoc); err != nil {
		return nil, fmt.Errorf("invalid schema JSON for trigger %s: %w", id, err)
	}

	resourceID := fmt.Sprintf("urn:flowker:trigger:%s", id)
	compiler := jsonschema.NewCompiler()

	if err := compiler.AddResource(resourceID, schemaDoc); err != nil {
		return nil, fmt.Errorf("invalid schema for trigger %s: %w", id, err)
	}

	validator, err := compiler.Compile(resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema for trigger %s: %w", id, err)
	}

	return &Trigger{
		id:        id,
		name:      name,
		version:   version,
		schema:    schema,
		validator: validator,
	}, nil
}

// ID returns the trigger's unique identifier.
func (t *Trigger) ID() executor.TriggerID {
	return t.id
}

// Name returns the trigger's human-readable name.
func (t *Trigger) Name() string {
	return t.name
}

// Version returns the trigger's version.
func (t *Trigger) Version() string {
	return t.version
}

// Schema returns the trigger's JSON Schema.
func (t *Trigger) Schema() string {
	return t.schema
}

// ValidateConfig validates the given configuration against the trigger's JSON Schema.
func (t *Trigger) ValidateConfig(config map[string]any) error {
	if err := t.validator.Validate(config); err != nil {
		return executor.NewTriggerConfigError(t.id, err)
	}

	return nil
}

// Verify Trigger implements executor.Trigger interface.
var _ executor.Trigger = (*Trigger)(nil)
