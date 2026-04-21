// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package transformation_test

import (
	"context"
	"testing"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/transformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildKazaamSpec_EmptyMappings(t *testing.T) {
	spec, err := transformation.BuildKazaamSpec(nil)

	require.NoError(t, err)
	assert.Equal(t, "[]", spec)
}

func TestBuildKazaamSpec_SimpleMapping(t *testing.T) {
	mappings := []model.FieldMapping{
		{Source: "workflow.id", Target: "provider.accountId"},
		{Source: "workflow.name", Target: "provider.fullName"},
	}

	spec, err := transformation.BuildKazaamSpec(mappings)

	require.NoError(t, err)
	assert.Contains(t, spec, `"operation":"shift"`)
	assert.Contains(t, spec, `"provider.accountId":"workflow.id"`)
	assert.Contains(t, spec, `"provider.fullName":"workflow.name"`)
}

func TestBuildKazaamSpec_WithTransformation(t *testing.T) {
	mappings := []model.FieldMapping{
		{
			Source: "workflow.cpf",
			Target: "provider.document",
			Transformation: &model.TransformationConfig{
				Type:   "remove_characters",
				Config: map[string]any{"characters": ".-"},
			},
		},
	}

	spec, err := transformation.BuildKazaamSpec(mappings)

	require.NoError(t, err)
	assert.Contains(t, spec, `"operation":"shift"`)
	assert.Contains(t, spec, `"operation":"remove_characters"`)
	assert.Contains(t, spec, `"path":"provider.document"`)
	assert.Contains(t, spec, `"characters":".-"`)
}

func TestBuildKazaamSpec_MultipleTransformations(t *testing.T) {
	mappings := []model.FieldMapping{
		{
			Source: "workflow.cpf",
			Target: "provider.document",
			Transformation: &model.TransformationConfig{
				Type:   "remove_characters",
				Config: map[string]any{"characters": ".-"},
			},
		},
		{
			Source: "workflow.name",
			Target: "provider.fullName",
			Transformation: &model.TransformationConfig{
				Type:   "to_uppercase",
				Config: map[string]any{},
			},
		},
	}

	spec, err := transformation.BuildKazaamSpec(mappings)

	require.NoError(t, err)
	assert.Contains(t, spec, `"operation":"shift"`)
	assert.Contains(t, spec, `"operation":"remove_characters"`)
	assert.Contains(t, spec, `"operation":"to_uppercase"`)
}

func TestBuildKazaamSpecFromOperations(t *testing.T) {
	operations := []model.KazaamOperation{
		{
			Operation: "shift",
			Spec: map[string]any{
				"out.id": "in.id",
			},
		},
		{
			Operation: "to_uppercase",
			Spec: map[string]any{
				"path": "out.id",
			},
		},
	}

	spec, err := transformation.BuildKazaamSpecFromOperations(operations)

	require.NoError(t, err)
	assert.Contains(t, spec, `"operation":"shift"`)
	assert.Contains(t, spec, `"operation":"to_uppercase"`)
}

func TestService_TransformWithMappings(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	input := []byte(`{"workflow":{"cpf":"123.456.789-00","name":"joão"}}`)
	mappings := []model.FieldMapping{
		{
			Source: "workflow.cpf",
			Target: "provider.document",
			Transformation: &model.TransformationConfig{
				Type:   "remove_characters",
				Config: map[string]any{"characters": ".-"},
			},
		},
		{
			Source: "workflow.name",
			Target: "provider.fullName",
			Transformation: &model.TransformationConfig{
				Type:   "to_uppercase",
				Config: map[string]any{},
			},
		},
	}

	output, err := svc.TransformWithMappings(ctx, input, mappings)

	require.NoError(t, err)
	assert.JSONEq(t, `{"provider":{"document":"12345678900","fullName":"JOÃO"}}`, string(output))
}

func TestService_TransformWithOperations(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	input := []byte(`{"in":{"value":"test"}}`)
	operations := []model.KazaamOperation{
		{
			Operation: "shift",
			Spec: map[string]any{
				"out.value": "in.value",
			},
		},
		{
			Operation: "to_uppercase",
			Spec: map[string]any{
				"path": "out.value",
			},
		},
	}

	output, err := svc.TransformWithOperations(ctx, input, operations)

	require.NoError(t, err)
	assert.JSONEq(t, `{"out":{"value":"TEST"}}`, string(output))
}
