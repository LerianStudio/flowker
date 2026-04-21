// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package transformation

import (
	"context"
	"encoding/json"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
)

// BuildKazaamSpec converts FieldMappings to a Kazaam specification JSON string.
// It creates a shift operation for field mapping followed by any transformations.
func BuildKazaamSpec(mappings []model.FieldMapping) (string, error) {
	if len(mappings) == 0 {
		return "[]", nil
	}

	operations := make([]model.KazaamOperation, 0)

	// First, create the shift operation for all field mappings
	shiftSpec := make(map[string]any)
	hasRequired := false

	for _, m := range mappings {
		if m.Source != "" && m.Target != "" {
			shiftSpec[m.Target] = m.Source
		}

		if m.Required {
			hasRequired = true
		}
	}

	if len(shiftSpec) > 0 {
		shiftOp := model.KazaamOperation{
			Operation: "shift",
			Spec:      shiftSpec,
		}

		// If any mapping is required, enforce path existence in Kazaam
		if hasRequired {
			shiftOp.Require = true
		}

		operations = append(operations, shiftOp)
	}

	// Then, add transformation operations for each mapping that has one
	for _, m := range mappings {
		if m.Transformation != nil && m.Transformation.Type != "" {
			transformOp := model.KazaamOperation{
				Operation: m.Transformation.Type,
				Spec: map[string]any{
					"path": m.Target,
				},
			}

			// Merge transformation config into spec
			for k, v := range m.Transformation.Config {
				transformOp.Spec[k] = v
			}

			operations = append(operations, transformOp)
		}
	}

	specBytes, err := json.Marshal(operations)
	if err != nil {
		return "", pkg.ValidationError{
			Code:    constant.ErrWorkflowInvalidInputMapping.Error(),
			Message: "failed to build transformation spec from mappings",
			Err:     err,
		}
	}

	return string(specBytes), nil
}

// BuildKazaamSpecFromOperations converts KazaamOperations directly to a JSON string.
func BuildKazaamSpecFromOperations(operations []model.KazaamOperation) (string, error) {
	if len(operations) == 0 {
		return "[]", nil
	}

	specBytes, err := json.Marshal(operations)
	if err != nil {
		return "", pkg.ValidationError{
			Code:    constant.ErrWorkflowInvalidTransforms.Error(),
			Message: "failed to build transformation spec from operations",
			Err:     err,
		}
	}

	return string(specBytes), nil
}

// TransformWithMappings applies field mappings to input data using Kazaam.
// This is a convenience function that builds the spec and transforms in one step.
func (s *Service) TransformWithMappings(ctx context.Context, input []byte, mappings []model.FieldMapping) ([]byte, error) {
	spec, err := BuildKazaamSpec(mappings)
	if err != nil {
		return nil, err
	}

	return s.Transform(ctx, input, spec)
}

// TransformWithOperations applies Kazaam operations to input data.
// This is a convenience function that builds the spec and transforms in one step.
func (s *Service) TransformWithOperations(ctx context.Context, input []byte, operations []model.KazaamOperation) ([]byte, error) {
	spec, err := BuildKazaamSpecFromOperations(operations)
	if err != nil {
		return nil, err
	}

	return s.Transform(ctx, input, spec)
}

// ValidateMappings validates that field mappings can be converted to valid transformation specs.
// Implements model.TransformationValidator interface.
func (v *Validator) ValidateMappings(mappings []model.FieldMapping) error {
	spec, err := BuildKazaamSpec(mappings)
	if err != nil {
		return err
	}

	return v.service.ValidateSpec(spec)
}

// ValidateOperations validates that Kazaam operations form a valid transformation spec.
// Implements model.TransformationValidator interface.
func (v *Validator) ValidateOperations(operations []model.KazaamOperation) error {
	spec, err := BuildKazaamSpecFromOperations(operations)
	if err != nil {
		return err
	}

	return v.service.ValidateSpec(spec)
}
