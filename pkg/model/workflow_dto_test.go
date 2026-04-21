// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package model_test

import (
	"testing"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validate = validator.New()

// CreateWorkflowInput validation tests
func TestCreateWorkflowInput_Valid(t *testing.T) {
	input := model.CreateWorkflowInput{
		Name:        "Test Workflow",
		Description: ptrString("A test workflow"),
		Nodes: []model.WorkflowNodeInput{
			{
				ID:       "trigger-1",
				Type:     "trigger",
				Position: model.PositionInput{X: 100, Y: 50},
				Data:     map[string]any{"triggerType": "http"},
			},
			{
				ID:       "provider-1",
				Type:     "executor",
				Name:     ptrString("Call Executor"),
				Position: model.PositionInput{X: 100, Y: 150},
				Data:     map[string]any{"executorId": "prov-123"},
			},
		},
		Edges: []model.WorkflowEdgeInput{
			{
				ID:     "edge-1",
				Source: "trigger-1",
				Target: "provider-1",
			},
		},
	}

	err := validate.Struct(input)

	require.NoError(t, err)
}

func TestCreateWorkflowInput_MissingName(t *testing.T) {
	input := model.CreateWorkflowInput{
		Name: "",
		Nodes: []model.WorkflowNodeInput{
			{ID: "trigger-1", Type: "trigger", Position: model.PositionInput{X: 0, Y: 0}},
		},
	}

	err := validate.Struct(input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Name")
}

func TestCreateWorkflowInput_NameTooLong(t *testing.T) {
	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}

	input := model.CreateWorkflowInput{
		Name: string(longName),
		Nodes: []model.WorkflowNodeInput{
			{ID: "trigger-1", Type: "trigger", Position: model.PositionInput{X: 0, Y: 0}},
		},
	}

	err := validate.Struct(input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Name")
}

func TestCreateWorkflowInput_NoNodes(t *testing.T) {
	input := model.CreateWorkflowInput{
		Name:  "Test",
		Nodes: []model.WorkflowNodeInput{},
	}

	err := validate.Struct(input)

	// Draft workflows allow empty nodes; validation happens at activation
	require.NoError(t, err)
}

func TestCreateWorkflowInput_NilNodes(t *testing.T) {
	input := model.CreateWorkflowInput{
		Name: "Test",
	}

	err := validate.Struct(input)

	// Draft workflows allow nil nodes; validation happens at activation
	require.NoError(t, err)
}

func TestCreateWorkflowInput_TooManyNodes(t *testing.T) {
	nodes := make([]model.WorkflowNodeInput, 101)
	nodes[0] = model.WorkflowNodeInput{
		ID:       "trigger-1",
		Type:     "trigger",
		Position: model.PositionInput{X: 0, Y: 0},
	}
	for i := 1; i < len(nodes); i++ {
		name := "Node"
		nodes[i] = model.WorkflowNodeInput{
			ID:       "node",
			Type:     "executor",
			Name:     &name,
			Position: model.PositionInput{X: 0, Y: 0},
		}
	}

	input := model.CreateWorkflowInput{
		Name:  "Test",
		Nodes: nodes,
	}

	err := validate.Struct(input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Nodes")
}

func TestCreateWorkflowInput_InvalidNodeType(t *testing.T) {
	input := model.CreateWorkflowInput{
		Name: "Test",
		Nodes: []model.WorkflowNodeInput{
			{ID: "node-1", Type: "invalid_type", Position: model.PositionInput{X: 0, Y: 0}},
		},
	}

	err := validate.Struct(input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Type")
}

func TestCreateWorkflowInput_NodeMissingID(t *testing.T) {
	input := model.CreateWorkflowInput{
		Name: "Test",
		Nodes: []model.WorkflowNodeInput{
			{ID: "", Type: "trigger", Position: model.PositionInput{X: 0, Y: 0}},
		},
	}

	err := validate.Struct(input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ID")
}

// UpdateWorkflowInput validation tests
func TestUpdateWorkflowInput_Valid(t *testing.T) {
	input := model.UpdateWorkflowInput{
		Name:        "Updated Workflow",
		Description: ptrString("Updated description"),
		Nodes: []model.WorkflowNodeInput{
			{ID: "trigger-1", Type: "trigger", Position: model.PositionInput{X: 100, Y: 50}},
			{ID: "conditional-1", Type: "conditional", Position: model.PositionInput{X: 100, Y: 150}},
		},
		Edges: []model.WorkflowEdgeInput{
			{ID: "edge-1", Source: "trigger-1", Target: "conditional-1"},
		},
	}

	err := validate.Struct(input)

	require.NoError(t, err)
}

func TestUpdateWorkflowInput_NoNodes(t *testing.T) {
	input := model.UpdateWorkflowInput{
		Name:  "Updated Workflow",
		Nodes: []model.WorkflowNodeInput{},
	}

	err := validate.Struct(input)

	// Draft workflows allow empty nodes; validation happens at activation
	require.NoError(t, err)
}

func TestUpdateWorkflowInput_NilNodes(t *testing.T) {
	input := model.UpdateWorkflowInput{
		Name: "Updated Workflow",
	}

	err := validate.Struct(input)

	// Draft workflows allow nil nodes; validation happens at activation
	require.NoError(t, err)
}

// WorkflowOutput conversion tests
func TestWorkflowOutput_FromDomain(t *testing.T) {
	id := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	description := "Test description"
	triggerNode, _ := model.NewWorkflowNode("trigger-1", model.NodeTypeTrigger, nil, model.Position{X: 100, Y: 50}, map[string]any{"triggerType": "http"})
	providerNode, _ := model.NewWorkflowNode("provider-1", model.NodeTypeExecutor, ptrString("Call Provider"), model.Position{X: 100, Y: 150}, map[string]any{"executorId": "prov-123"})
	nodes := []model.WorkflowNode{
		triggerNode,
		providerNode,
	}
	edge1, _ := model.NewWorkflowEdge("edge-1", "trigger-1", "provider-1")
	edges := []model.WorkflowEdge{
		edge1,
	}

	workflow := model.NewWorkflowFromDB(
		id,
		"Test Workflow",
		&description,
		model.WorkflowStatusActive,
		nodes,
		edges,
		map[string]any{"meta": "data"},
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	)

	output := model.WorkflowOutputFromDomain(workflow)

	assert.Equal(t, id, output.ID)
	assert.Equal(t, "Test Workflow", output.Name)
	assert.Equal(t, &description, output.Description)
	assert.Equal(t, "active", output.Status)
	require.Len(t, output.Nodes, 2)
	require.Len(t, output.Edges, 1)
	assert.Equal(t, "trigger-1", output.Nodes[0].ID)
	assert.Equal(t, "trigger", output.Nodes[0].Type)
	assert.Equal(t, "provider-1", output.Nodes[1].ID)
	assert.Equal(t, "executor", output.Nodes[1].Type)
}

func TestWorkflowNodeOutput_FromDomain(t *testing.T) {
	name := "Test Node"
	node, err := model.NewWorkflowNode(
		"node-1",
		model.NodeTypeExecutor,
		&name,
		model.Position{X: 200, Y: 300},
		map[string]any{"executorId": "prov-123"},
	)
	require.NoError(t, err)

	output := model.WorkflowNodeOutputFromDomain(node)

	assert.Equal(t, "node-1", output.ID)
	assert.Equal(t, "executor", output.Type)
	assert.Equal(t, &name, output.Name)
	assert.Equal(t, 200, output.Position.X)
	assert.Equal(t, 300, output.Position.Y)
	assert.Equal(t, "prov-123", output.Data["executorId"])
}

func TestWorkflowEdgeOutput_FromDomain(t *testing.T) {
	edge, _ := model.NewWorkflowEdge("edge-1", "source-node", "target-node")
	edge.WithSourceHandle("true")
	edge.WithLabel("Success")

	output := model.WorkflowEdgeOutputFromDomain(edge)

	assert.Equal(t, "edge-1", output.ID)
	assert.Equal(t, "source-node", output.Source)
	assert.Equal(t, "target-node", output.Target)
	assert.Equal(t, "true", *output.SourceHandle)
	assert.Equal(t, "Success", *output.Label)
}

// CloneWorkflowInput tests
func TestCloneWorkflowInput_Valid(t *testing.T) {
	input := model.CloneWorkflowInput{
		Name: "Cloned Workflow",
	}

	err := validate.Struct(input)

	require.NoError(t, err)
}

func TestCloneWorkflowInput_MissingName(t *testing.T) {
	input := model.CloneWorkflowInput{
		Name: "",
	}

	err := validate.Struct(input)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Name")
}

// Helper
func ptrString(s string) *string {
	return &s
}
