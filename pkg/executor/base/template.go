// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package base

import (
	"encoding/json"
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Template is a base implementation of the executor.Template interface.
// It provides JSON Schema validation for template parameters.
type Template struct {
	id                   executor.TemplateID
	name                 string
	description          string
	version              string
	category             string
	paramSchema          string
	validator            *jsonschema.Schema
	builder              func(params map[string]any) (any, error)
	providerConfigFields []executor.ProviderConfigField
}

// NewTemplate creates a new base Template with JSON Schema validation.
// Returns error if required fields are missing or the schema is invalid.
// providerConfigFields declares which schema parameters reference provider configurations;
// pass nil if the template has no provider config fields.
func NewTemplate(
	id executor.TemplateID,
	name, description, version, category, paramSchema string,
	builder func(params map[string]any) (any, error),
	providerConfigFields []executor.ProviderConfigField,
) (*Template, error) {
	if id == "" {
		return nil, fmt.Errorf("template id is required")
	}

	if name == "" {
		return nil, fmt.Errorf("template name is required")
	}

	if version == "" {
		return nil, fmt.Errorf("template version is required")
	}

	if category == "" {
		return nil, fmt.Errorf("template category is required")
	}

	if paramSchema == "" {
		return nil, fmt.Errorf("template param schema is required")
	}

	if builder == nil {
		return nil, fmt.Errorf("template builder function is required")
	}

	var schemaDoc any
	if err := json.Unmarshal([]byte(paramSchema), &schemaDoc); err != nil {
		return nil, fmt.Errorf("invalid param schema JSON for template %s: %w", id, err)
	}

	resourceID := fmt.Sprintf("urn:flowker:template:%s", id)
	compiler := jsonschema.NewCompiler()

	if err := compiler.AddResource(resourceID, schemaDoc); err != nil {
		return nil, fmt.Errorf("invalid param schema for template %s: %w", id, err)
	}

	validator, err := compiler.Compile(resourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to compile param schema for template %s: %w", id, err)
	}

	return &Template{
		id:                   id,
		name:                 name,
		description:          description,
		version:              version,
		category:             category,
		paramSchema:          paramSchema,
		validator:            validator,
		builder:              builder,
		providerConfigFields: providerConfigFields,
	}, nil
}

// ID returns the template's unique identifier.
func (t *Template) ID() executor.TemplateID {
	return t.id
}

// Name returns the template's human-readable name.
func (t *Template) Name() string {
	return t.name
}

// Description returns the template's description.
func (t *Template) Description() string {
	return t.description
}

// Version returns the template's version.
func (t *Template) Version() string {
	return t.version
}

// Category returns the template's category.
func (t *Template) Category() string {
	return t.category
}

// ParamSchema returns the template's JSON Schema for parameter validation.
func (t *Template) ParamSchema() string {
	return t.paramSchema
}

// ValidateParams validates the given parameters against the template's JSON Schema.
func (t *Template) ValidateParams(params map[string]any) error {
	if err := t.validator.Validate(params); err != nil {
		return executor.NewTemplateParamError(t.id, err)
	}

	return nil
}

// Build generates a complete workflow input from the given parameters.
func (t *Template) Build(params map[string]any) (any, error) {
	return t.builder(params)
}

// ProviderConfigFields returns the list of parameters that reference provider configurations.
func (t *Template) ProviderConfigFields() []executor.ProviderConfigField {
	if t.providerConfigFields == nil {
		return nil
	}

	result := make([]executor.ProviderConfigField, len(t.providerConfigFields))
	copy(result, t.providerConfigFields)

	return result
}

// Verify Template implements executor.Template interface.
var _ executor.Template = (*Template)(nil)
