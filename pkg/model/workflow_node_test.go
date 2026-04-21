// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package model

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransformationValidator is a test double for TransformationValidator.
type mockTransformationValidator struct {
	mappingsErr     error
	operationsErr   error
	mappingsCalls   int
	operationsCalls int
}

func (m *mockTransformationValidator) ValidateMappings(mappings []FieldMapping) error {
	m.mappingsCalls++
	return m.mappingsErr
}

func (m *mockTransformationValidator) ValidateOperations(operations []KazaamOperation) error {
	m.operationsCalls++
	return m.operationsErr
}

func TestNewWorkflowNode_Valid(t *testing.T) {
	name := "Test Node"
	node, err := NewWorkflowNode("node-1", NodeTypeExecutor, &name, Position{X: 100, Y: 200}, nil)

	require.NoError(t, err)
	assert.Equal(t, "node-1", node.ID())
	assert.Equal(t, NodeTypeExecutor, node.Type())
	assert.Equal(t, "Test Node", *node.Name())
	assert.Equal(t, 100, node.Position().X)
	assert.Equal(t, 200, node.Position().Y)
}

func TestNewWorkflowNode_EmptyID(t *testing.T) {
	_, err := NewWorkflowNode("", NodeTypeExecutor, nil, Position{}, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrNodeIDRequired, err)
}

func TestNewWorkflowNode_EmptyType(t *testing.T) {
	_, err := NewWorkflowNode("node-1", "", nil, Position{}, nil)

	assert.Error(t, err)
	assert.Equal(t, ErrNodeTypeRequired, err)
}

func TestWorkflowNode_ExecutorID(t *testing.T) {
	t.Run("returns executor ID for executor node", func(t *testing.T) {
		data := map[string]any{"executorId": "provider-123"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		assert.Equal(t, "provider-123", node.ExecutorID())
	})

	t.Run("returns empty for non-executor node", func(t *testing.T) {
		data := map[string]any{"executorId": "provider-123"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeTrigger, nil, Position{}, data)

		assert.Equal(t, "", node.ExecutorID())
	})

	t.Run("returns empty when data is nil", func(t *testing.T) {
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, nil)

		assert.Equal(t, "", node.ExecutorID())
	})

	t.Run("returns empty when executorId is not a string", func(t *testing.T) {
		data := map[string]any{"executorId": 123}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		assert.Equal(t, "", node.ExecutorID())
	})
}

func TestWorkflowNode_EndpointName(t *testing.T) {
	t.Run("returns endpoint name for executor node", func(t *testing.T) {
		data := map[string]any{"endpointName": "createAccount"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		assert.Equal(t, "createAccount", node.EndpointName())
	})

	t.Run("returns empty for non-executor node", func(t *testing.T) {
		data := map[string]any{"endpointName": "createAccount"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeTrigger, nil, Position{}, data)

		assert.Equal(t, "", node.EndpointName())
	})

	t.Run("returns empty when data is nil", func(t *testing.T) {
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, nil)

		assert.Equal(t, "", node.EndpointName())
	})
}

func TestWorkflowNode_TriggerType(t *testing.T) {
	t.Run("returns trigger type for trigger node", func(t *testing.T) {
		data := map[string]any{"triggerType": "http"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeTrigger, nil, Position{}, data)

		assert.Equal(t, "http", node.TriggerType())
	})

	t.Run("returns empty for non-trigger node", func(t *testing.T) {
		data := map[string]any{"triggerType": "http"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		assert.Equal(t, "", node.TriggerType())
	})
}

func TestWorkflowNode_Condition(t *testing.T) {
	t.Run("returns condition for conditional node", func(t *testing.T) {
		data := map[string]any{"condition": "data.amount > 1000"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeConditional, nil, Position{}, data)

		assert.Equal(t, "data.amount > 1000", node.Condition())
	})

	t.Run("returns empty for non-conditional node", func(t *testing.T) {
		data := map[string]any{"condition": "data.amount > 1000"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		assert.Equal(t, "", node.Condition())
	})
}

func TestWorkflowNode_InputMapping(t *testing.T) {
	t.Run("returns input mappings for executor node", func(t *testing.T) {
		data := map[string]any{
			"inputMapping": []any{
				map[string]any{
					"source":   "workflow.customer.cpf",
					"target":   "provider.document",
					"required": true,
				},
				map[string]any{
					"source": "workflow.customer.name",
					"target": "provider.fullName",
				},
			},
		}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		mappings := node.InputMapping()

		require.Len(t, mappings, 2)
		assert.Equal(t, "workflow.customer.cpf", mappings[0].Source)
		assert.Equal(t, "provider.document", mappings[0].Target)
		assert.True(t, mappings[0].Required)
		assert.Equal(t, "workflow.customer.name", mappings[1].Source)
		assert.Equal(t, "provider.fullName", mappings[1].Target)
		assert.False(t, mappings[1].Required)
	})

	t.Run("returns nil for non-executor node", func(t *testing.T) {
		data := map[string]any{
			"inputMapping": []any{
				map[string]any{"source": "a", "target": "b"},
			},
		}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeTrigger, nil, Position{}, data)

		assert.Nil(t, node.InputMapping())
	})

	t.Run("returns nil when data is nil", func(t *testing.T) {
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, nil)

		assert.Nil(t, node.InputMapping())
	})

	t.Run("returns nil when inputMapping is not a slice", func(t *testing.T) {
		data := map[string]any{"inputMapping": "invalid"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		assert.Nil(t, node.InputMapping())
	})
}

func TestWorkflowNode_InputMapping_WithTransformation(t *testing.T) {
	data := map[string]any{
		"inputMapping": []any{
			map[string]any{
				"source": "workflow.cpf",
				"target": "provider.document",
				"transformation": map[string]any{
					"type": "remove_characters",
					"config": map[string]any{
						"characters": ".-",
					},
				},
			},
		},
	}
	node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

	mappings := node.InputMapping()

	require.Len(t, mappings, 1)
	require.NotNil(t, mappings[0].Transformation)
	assert.Equal(t, "remove_characters", mappings[0].Transformation.Type)
	assert.Equal(t, ".-", mappings[0].Transformation.Config["characters"])
}

func TestWorkflowNode_OutputMapping(t *testing.T) {
	t.Run("returns output mappings for executor node", func(t *testing.T) {
		data := map[string]any{
			"outputMapping": []any{
				map[string]any{
					"source": "provider.accountId",
					"target": "workflow.result.accountId",
				},
			},
		}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		mappings := node.OutputMapping()

		require.Len(t, mappings, 1)
		assert.Equal(t, "provider.accountId", mappings[0].Source)
		assert.Equal(t, "workflow.result.accountId", mappings[0].Target)
	})

	t.Run("returns nil for non-executor node", func(t *testing.T) {
		data := map[string]any{
			"outputMapping": []any{
				map[string]any{"source": "a", "target": "b"},
			},
		}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeTrigger, nil, Position{}, data)

		assert.Nil(t, node.OutputMapping())
	})
}

func TestWorkflowNode_Transforms(t *testing.T) {
	t.Run("returns transforms for executor node", func(t *testing.T) {
		data := map[string]any{
			"transforms": []any{
				map[string]any{
					"operation": "shift",
					"spec": map[string]any{
						"out.id": "in.id",
					},
				},
				map[string]any{
					"operation": "to_uppercase",
					"spec": map[string]any{
						"path": "out.id",
					},
					"require": true,
				},
			},
		}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		transforms := node.Transforms()

		require.Len(t, transforms, 2)
		assert.Equal(t, "shift", transforms[0].Operation)
		assert.Equal(t, "in.id", transforms[0].Spec["out.id"])
		assert.False(t, transforms[0].Require)
		assert.Equal(t, "to_uppercase", transforms[1].Operation)
		assert.Equal(t, "out.id", transforms[1].Spec["path"])
		assert.True(t, transforms[1].Require)
	})

	t.Run("returns nil for non-executor node", func(t *testing.T) {
		data := map[string]any{
			"transforms": []any{
				map[string]any{"operation": "shift", "spec": map[string]any{}},
			},
		}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeTrigger, nil, Position{}, data)

		assert.Nil(t, node.Transforms())
	})

	t.Run("returns nil when data is nil", func(t *testing.T) {
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, nil)

		assert.Nil(t, node.Transforms())
	})

	t.Run("returns nil when transforms is not a slice", func(t *testing.T) {
		data := map[string]any{"transforms": "invalid"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		assert.Nil(t, node.Transforms())
	})
}

func TestWorkflowNode_Data(t *testing.T) {
	t.Run("returns copy of data", func(t *testing.T) {
		data := map[string]any{"key": "value"}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		result := node.Data()

		assert.Equal(t, "value", result["key"])

		// Verify it's a copy
		result["key"] = "modified"
		assert.Equal(t, "value", node.Data()["key"])
	})

	t.Run("returns nil when data is nil", func(t *testing.T) {
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, nil)

		assert.Nil(t, node.Data())
	})
}

func TestWorkflowNode_Clone(t *testing.T) {
	name := "Original"
	data := map[string]any{"key": "value"}
	node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, &name, Position{X: 10, Y: 20}, data)

	cloned := node.clone()

	// Verify values match
	assert.Equal(t, node.ID(), cloned.ID())
	assert.Equal(t, node.Type(), cloned.Type())
	assert.Equal(t, *node.Name(), *cloned.Name())
	assert.Equal(t, node.Position(), cloned.Position())
	assert.Equal(t, node.Data()["key"], cloned.Data()["key"])

	// Verify it's a deep copy (name)
	newName := "Modified"
	cloned.name = &newName
	assert.Equal(t, "Original", *node.Name())

	// Verify it's a deep copy (data)
	cloned.data["key"] = "modified"
	assert.Equal(t, "value", node.Data()["key"])
}

func TestParseFieldMappings_InvalidItems(t *testing.T) {
	t.Run("skips non-map items", func(t *testing.T) {
		data := map[string]any{
			"inputMapping": []any{
				"invalid-string",
				map[string]any{"source": "a", "target": "b"},
				123,
			},
		}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		mappings := node.InputMapping()

		require.Len(t, mappings, 1)
		assert.Equal(t, "a", mappings[0].Source)
	})
}

func TestParseKazaamOperations_InvalidItems(t *testing.T) {
	t.Run("skips non-map items", func(t *testing.T) {
		data := map[string]any{
			"transforms": []any{
				"invalid-string",
				map[string]any{"operation": "shift", "spec": map[string]any{}},
				123,
			},
		}
		node := NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, data)

		transforms := node.Transforms()

		require.Len(t, transforms, 1)
		assert.Equal(t, "shift", transforms[0].Operation)
	})
}

func TestValidateNodeTransformations_NilValidator(t *testing.T) {
	nodes := []WorkflowNode{
		NewWorkflowNodeFromDB("node-1", NodeTypeExecutor, nil, Position{}, map[string]any{
			"inputMapping": []any{
				map[string]any{"source": "a", "target": "b"},
			},
		}),
	}

	err := ValidateNodeTransformations(nodes, nil)

	assert.NoError(t, err)
}

func TestValidateNodeTransformations_SkipsNonProviderNodes(t *testing.T) {
	validator := &mockTransformationValidator{}
	nodes := []WorkflowNode{
		NewWorkflowNodeFromDB("trigger-1", NodeTypeTrigger, nil, Position{}, map[string]any{
			"inputMapping": []any{
				map[string]any{"source": "a", "target": "b"},
			},
		}),
		NewWorkflowNodeFromDB("conditional-1", NodeTypeConditional, nil, Position{}, map[string]any{
			"transforms": []any{
				map[string]any{"operation": "shift", "spec": map[string]any{}},
			},
		}),
	}

	err := ValidateNodeTransformations(nodes, validator)

	assert.NoError(t, err)
	assert.Equal(t, 0, validator.mappingsCalls)
	assert.Equal(t, 0, validator.operationsCalls)
}

func TestValidateNodeTransformations_ValidatesInputMapping(t *testing.T) {
	validator := &mockTransformationValidator{}
	nodes := []WorkflowNode{
		NewWorkflowNodeFromDB("provider-1", NodeTypeExecutor, nil, Position{}, map[string]any{
			"inputMapping": []any{
				map[string]any{"source": "workflow.cpf", "target": "provider.document"},
			},
		}),
	}

	err := ValidateNodeTransformations(nodes, validator)

	assert.NoError(t, err)
	assert.Equal(t, 1, validator.mappingsCalls)
}

func TestValidateNodeTransformations_ValidatesOutputMapping(t *testing.T) {
	validator := &mockTransformationValidator{}
	nodes := []WorkflowNode{
		NewWorkflowNodeFromDB("provider-1", NodeTypeExecutor, nil, Position{}, map[string]any{
			"outputMapping": []any{
				map[string]any{"source": "provider.result", "target": "workflow.output"},
			},
		}),
	}

	err := ValidateNodeTransformations(nodes, validator)

	assert.NoError(t, err)
	assert.Equal(t, 1, validator.mappingsCalls)
}

func TestValidateNodeTransformations_ValidatesTransforms(t *testing.T) {
	validator := &mockTransformationValidator{}
	nodes := []WorkflowNode{
		NewWorkflowNodeFromDB("provider-1", NodeTypeExecutor, nil, Position{}, map[string]any{
			"transforms": []any{
				map[string]any{"operation": "shift", "spec": map[string]any{"out": "in"}},
			},
		}),
	}

	err := ValidateNodeTransformations(nodes, validator)

	assert.NoError(t, err)
	assert.Equal(t, 1, validator.operationsCalls)
}

func TestValidateNodeTransformations_InputMappingError(t *testing.T) {
	validator := &mockTransformationValidator{
		mappingsErr: errors.New("invalid mapping"),
	}
	nodes := []WorkflowNode{
		NewWorkflowNodeFromDB("provider-1", NodeTypeExecutor, nil, Position{}, map[string]any{
			"inputMapping": []any{
				map[string]any{"source": "a", "target": "b"},
			},
		}),
	}

	err := ValidateNodeTransformations(nodes, validator)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid inputMapping")
	assert.Contains(t, err.Error(), "provider-1")
}

func TestValidateNodeTransformations_OutputMappingError(t *testing.T) {
	validator := &mockTransformationValidator{
		mappingsErr: errors.New("invalid mapping"),
	}
	nodes := []WorkflowNode{
		NewWorkflowNodeFromDB("provider-1", NodeTypeExecutor, nil, Position{}, map[string]any{
			"outputMapping": []any{
				map[string]any{"source": "a", "target": "b"},
			},
		}),
	}

	err := ValidateNodeTransformations(nodes, validator)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid outputMapping")
}

func TestValidateNodeTransformations_TransformsError(t *testing.T) {
	validator := &mockTransformationValidator{
		operationsErr: errors.New("invalid operation"),
	}
	nodes := []WorkflowNode{
		NewWorkflowNodeFromDB("provider-1", NodeTypeExecutor, nil, Position{}, map[string]any{
			"transforms": []any{
				map[string]any{"operation": "invalid_op", "spec": map[string]any{}},
			},
		}),
	}

	err := ValidateNodeTransformations(nodes, validator)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transforms")
}

func TestValidateNodeTransformations_MultipleNodesValidation(t *testing.T) {
	validator := &mockTransformationValidator{}
	nodes := []WorkflowNode{
		NewWorkflowNodeFromDB("provider-1", NodeTypeExecutor, nil, Position{}, map[string]any{
			"inputMapping": []any{
				map[string]any{"source": "a", "target": "b"},
			},
		}),
		NewWorkflowNodeFromDB("provider-2", NodeTypeExecutor, nil, Position{}, map[string]any{
			"outputMapping": []any{
				map[string]any{"source": "x", "target": "y"},
			},
			"transforms": []any{
				map[string]any{"operation": "shift", "spec": map[string]any{}},
			},
		}),
	}

	err := ValidateNodeTransformations(nodes, validator)

	assert.NoError(t, err)
	assert.Equal(t, 2, validator.mappingsCalls)   // inputMapping + outputMapping
	assert.Equal(t, 1, validator.operationsCalls) // transforms
}
