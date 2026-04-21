// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package executor

// TemplateID is a unique identifier for a workflow template.
type TemplateID string

// ProviderConfigField describes a template parameter that references a provider configuration.
// Used by the catalog handler to enrich the schema with available options from the database.
type ProviderConfigField struct {
	ParamName  string     // JSON Schema property name (e.g., "tracerProviderConfigId")
	ProviderID ProviderID // Provider ID to filter configs (e.g., "tracer")
}

// Template defines the contract for a workflow template.
// Templates are pre-built workflow blueprints that users can instantiate
// by providing parameters (e.g., provider configuration IDs, thresholds).
//
// Build returns an any that must be a *model.CreateWorkflowInput. This avoids
// an import cycle between pkg/executor and pkg/model.
type Template interface {
	// ID returns the unique identifier for this template.
	ID() TemplateID

	// Name returns the human-readable name for this template.
	Name() string

	// Description returns a brief description of this template.
	Description() string

	// Version returns the template version (e.g., "v1").
	Version() string

	// Category returns the template category (e.g., "Compliance", "Payments").
	Category() string

	// ParamSchema returns the JSON Schema for validating template parameters.
	ParamSchema() string

	// ValidateParams validates the given parameters against the template's JSON Schema.
	ValidateParams(params map[string]any) error

	// Build generates a complete workflow input from the given parameters.
	// The returned value is a *model.CreateWorkflowInput.
	Build(params map[string]any) (any, error)

	// ProviderConfigFields returns the list of parameters that reference provider configurations.
	// The catalog handler uses this to enrich the schema with available options.
	ProviderConfigFields() []ProviderConfigField
}
