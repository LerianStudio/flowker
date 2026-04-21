// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package transformation_test

import (
	"context"
	"testing"

	"github.com/LerianStudio/flowker/pkg/transformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_Transform_Shift(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	input := []byte(`{"workflow":{"customer":{"id":"123","name":"João Silva"}}}`)
	spec := `[{
		"operation": "shift",
		"spec": {
			"provider.accountId": "workflow.customer.id",
			"provider.fullName": "workflow.customer.name"
		}
	}]`

	output, err := svc.Transform(ctx, input, spec)

	require.NoError(t, err)
	assert.JSONEq(t, `{"provider":{"accountId":"123","fullName":"João Silva"}}`, string(output))
}

func TestService_Transform_RemoveCharacters(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	// First shift, then remove characters
	input := []byte(`{"cpf":"123.456.789-00"}`)
	spec := `[
		{
			"operation": "shift",
			"spec": {
				"document": "cpf"
			}
		},
		{
			"operation": "remove_characters",
			"spec": {
				"path": "document",
				"characters": ".-"
			}
		}
	]`

	output, err := svc.Transform(ctx, input, spec)

	require.NoError(t, err)
	assert.JSONEq(t, `{"document":"12345678900"}`, string(output))
}

func TestService_Transform_AddPrefix(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	input := []byte(`{"id":"12345"}`)
	spec := `[{
		"operation": "add_prefix",
		"spec": {
			"path": "id",
			"prefix": "BR-"
		}
	}]`

	output, err := svc.Transform(ctx, input, spec)

	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"BR-12345"}`, string(output))
}

func TestService_Transform_AddSuffix(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	input := []byte(`{"id":"12345"}`)
	spec := `[{
		"operation": "add_suffix",
		"spec": {
			"path": "id",
			"suffix": "-2025"
		}
	}]`

	output, err := svc.Transform(ctx, input, spec)

	require.NoError(t, err)
	assert.JSONEq(t, `{"id":"12345-2025"}`, string(output))
}

func TestService_Transform_ToUppercase(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	input := []byte(`{"name":"joão silva"}`)
	spec := `[{
		"operation": "to_uppercase",
		"spec": {
			"path": "name"
		}
	}]`

	output, err := svc.Transform(ctx, input, spec)

	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"JOÃO SILVA"}`, string(output))
}

func TestService_Transform_ToLowercase(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	input := []byte(`{"email":"JOAO@EXAMPLE.COM"}`)
	spec := `[{
		"operation": "to_lowercase",
		"spec": {
			"path": "email"
		}
	}]`

	output, err := svc.Transform(ctx, input, spec)

	require.NoError(t, err)
	assert.JSONEq(t, `{"email":"joao@example.com"}`, string(output))
}

func TestService_Transform_Concat(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	input := []byte(`{"first":"João","last":"Silva"}`)
	spec := `[{
		"operation": "concat",
		"spec": {
			"sources": [
				{"path": "first"},
				{"value": " "},
				{"path": "last"}
			],
			"targetPath": "fullName",
			"delim": ""
		}
	}]`

	output, err := svc.Transform(ctx, input, spec)

	require.NoError(t, err)
	assert.Contains(t, string(output), `"fullName":"João Silva"`)
}

func TestService_Transform_ComplexWorkflow(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	// Simulate a real workflow transformation
	input := []byte(`{
		"workflow": {
			"customer": {
				"cpf": "123.456.789-00",
				"name": "joão silva",
				"email": "JOAO@EXAMPLE.COM"
			},
			"amount": 15050
		}
	}`)

	spec := `[
		{
			"operation": "shift",
			"spec": {
				"provider.document": "workflow.customer.cpf",
				"provider.fullName": "workflow.customer.name",
				"provider.email": "workflow.customer.email",
				"provider.valor": "workflow.amount"
			}
		},
		{
			"operation": "remove_characters",
			"spec": {
				"path": "provider.document",
				"characters": ".-"
			}
		},
		{
			"operation": "to_uppercase",
			"spec": {
				"path": "provider.fullName"
			}
		},
		{
			"operation": "to_lowercase",
			"spec": {
				"path": "provider.email"
			}
		}
	]`

	output, err := svc.Transform(ctx, input, spec)

	require.NoError(t, err)

	expected := `{
		"provider": {
			"document": "12345678900",
			"fullName": "JOÃO SILVA",
			"email": "joao@example.com",
			"valor": 15050
		}
	}`
	assert.JSONEq(t, expected, string(output))
}

func TestService_TransformMap(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	input := map[string]any{
		"name": "test",
		"id":   "123",
	}
	spec := `[{
		"operation": "shift",
		"spec": {
			"output.name": "name",
			"output.id": "id"
		}
	}]`

	output, err := svc.TransformMap(ctx, input, spec)

	require.NoError(t, err)
	require.NotNil(t, output)

	outputMap, ok := output["output"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "test", outputMap["name"])
	assert.Equal(t, "123", outputMap["id"])
}

func TestService_ValidateSpec_Valid(t *testing.T) {
	svc := transformation.NewService()

	spec := `[{"operation": "shift", "spec": {"out": "in"}}]`
	err := svc.ValidateSpec(spec)

	assert.NoError(t, err)
}

func TestService_ValidateSpec_Invalid(t *testing.T) {
	svc := transformation.NewService()

	spec := `[{"operation": "invalid_operation", "spec": {}}]`
	err := svc.ValidateSpec(spec)

	assert.Error(t, err)
}

func TestService_ValidateSpec_InvalidJSON(t *testing.T) {
	svc := transformation.NewService()

	spec := `not valid json`
	err := svc.ValidateSpec(spec)

	assert.Error(t, err)
}

func TestService_ClearCache(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	spec := `[{"operation": "shift", "spec": {"out": "in"}}]`

	// First transform - caches the spec
	_, err := svc.Transform(ctx, []byte(`{"in":"value"}`), spec)
	require.NoError(t, err)

	// Clear cache
	svc.ClearCache()

	// Should still work after clearing cache
	_, err = svc.Transform(ctx, []byte(`{"in":"value"}`), spec)
	require.NoError(t, err)
}

func TestService_Transform_PathNotExists(t *testing.T) {
	svc := transformation.NewService()
	ctx := context.Background()

	// When path doesn't exist, custom transforms should return data unchanged
	input := []byte(`{"other":"value"}`)
	spec := `[{
		"operation": "remove_characters",
		"spec": {
			"path": "nonexistent",
			"characters": ".-"
		}
	}]`

	output, err := svc.Transform(ctx, input, spec)

	require.NoError(t, err)
	assert.JSONEq(t, `{"other":"value"}`, string(output))
}
