// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package workflow_test

import (
	"testing"
	"time"

	mongoworkflow "github.com/LerianStudio/flowker/internal/adapters/mongodb/workflow"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMongoDBModel_ToEntity(t *testing.T) {
	workflowID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	description := "Test description"
	createdAt := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	nodeName := "Call Provider"

	mongoModel := &mongoworkflow.MongoDBModel{
		WorkflowID:  workflowID.String(),
		Name:        "Test Workflow",
		Description: &description,
		Status:      "active",
		Nodes: []mongoworkflow.NodeModel{
			{
				ID:   "trigger-1",
				Type: "trigger",
				Position: mongoworkflow.PositionModel{
					X: 100,
					Y: 50,
				},
				Data: map[string]any{"triggerType": "http"},
			},
			{
				ID:   "provider-1",
				Type: "executor",
				Name: &nodeName,
				Position: mongoworkflow.PositionModel{
					X: 100,
					Y: 150,
				},
				Data: map[string]any{"executorId": "prov-123"},
			},
		},
		Edges: []mongoworkflow.EdgeModel{
			{
				ID:     "edge-1",
				Source: "trigger-1",
				Target: "provider-1",
			},
		},
		Metadata:  map[string]any{"key": "value"},
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	entity := mongoModel.ToEntity()

	require.NotNil(t, entity)
	assert.Equal(t, workflowID, entity.ID())
	assert.Equal(t, "Test Workflow", entity.Name())
	assert.Equal(t, &description, entity.Description())
	assert.Equal(t, model.WorkflowStatusActive, entity.Status())
	assert.Equal(t, createdAt, entity.CreatedAt())
	assert.Equal(t, updatedAt, entity.UpdatedAt())

	// Verify nodes
	require.Len(t, entity.Nodes(), 2)
	assert.Equal(t, "trigger-1", entity.Nodes()[0].ID())
	assert.Equal(t, model.NodeTypeTrigger, entity.Nodes()[0].Type())
	assert.Equal(t, 100, entity.Nodes()[0].Position().X)
	assert.Equal(t, 50, entity.Nodes()[0].Position().Y)
	assert.Equal(t, "provider-1", entity.Nodes()[1].ID())
	assert.Equal(t, model.NodeTypeExecutor, entity.Nodes()[1].Type())
	assert.Equal(t, &nodeName, entity.Nodes()[1].Name())

	// Verify edges
	require.Len(t, entity.Edges(), 1)
	assert.Equal(t, "edge-1", entity.Edges()[0].ID())
	assert.Equal(t, "trigger-1", entity.Edges()[0].Source())
	assert.Equal(t, "provider-1", entity.Edges()[0].Target())

	// Verify metadata
	assert.Equal(t, "value", entity.Metadata()["key"])
}

func TestMongoDBModel_FromEntity(t *testing.T) {
	triggerNode, err := model.NewWorkflowNode("trigger-1", model.NodeTypeTrigger, nil, model.Position{X: 100, Y: 50}, map[string]any{"triggerType": "http"})
	require.NoError(t, err)

	nodes := []model.WorkflowNode{
		triggerNode,
	}
	description := "Test description"

	workflow, err := model.NewWorkflow("Test Workflow", &description, nodes, nil)
	require.NoError(t, err)

	workflow.SetMetadata("env", "test")

	mongoModel := &mongoworkflow.MongoDBModel{}
	mongoModel.FromEntity(workflow)

	assert.Equal(t, workflow.ID().String(), mongoModel.WorkflowID)
	assert.Equal(t, "Test Workflow", mongoModel.Name)
	assert.Equal(t, &description, mongoModel.Description)
	assert.Equal(t, "draft", mongoModel.Status)
	assert.Len(t, mongoModel.Nodes, 1)
	assert.Equal(t, "trigger-1", mongoModel.Nodes[0].ID)
	assert.Equal(t, "trigger", mongoModel.Nodes[0].Type)
	assert.Equal(t, 100, mongoModel.Nodes[0].Position.X)
	assert.Equal(t, 50, mongoModel.Nodes[0].Position.Y)
	assert.Equal(t, "test", mongoModel.Metadata["env"])
}

func TestNewMongoDBModelFromEntity(t *testing.T) {
	triggerNode, err := model.NewWorkflowNode("trigger-1", model.NodeTypeTrigger, nil, model.Position{X: 100, Y: 50}, nil)
	require.NoError(t, err)

	nodes := []model.WorkflowNode{
		triggerNode,
	}

	workflow, err := model.NewWorkflow("My Workflow", nil, nodes, nil)
	require.NoError(t, err)

	mongoModel := mongoworkflow.NewMongoDBModelFromEntity(workflow)

	require.NotNil(t, mongoModel)
	assert.Equal(t, workflow.ID().String(), mongoModel.WorkflowID)
	assert.Equal(t, "My Workflow", mongoModel.Name)
	assert.Nil(t, mongoModel.Description)
	assert.Equal(t, "draft", mongoModel.Status)
}

func TestNodeModel_ToEntity(t *testing.T) {
	nodeName := "Check Balance"

	nodeModel := &mongoworkflow.NodeModel{
		ID:   "conditional-1",
		Type: "conditional",
		Name: &nodeName,
		Position: mongoworkflow.PositionModel{
			X: 200,
			Y: 300,
		},
		Data: map[string]any{"condition": "workflow.score > 70"},
	}

	entity := nodeModel.ToEntity()

	assert.Equal(t, "conditional-1", entity.ID())
	assert.Equal(t, model.NodeTypeConditional, entity.Type())
	assert.Equal(t, &nodeName, entity.Name())
	assert.Equal(t, 200, entity.Position().X)
	assert.Equal(t, 300, entity.Position().Y)
	assert.Equal(t, "workflow.score > 70", entity.Condition())
}

func TestNodeModelFromEntity(t *testing.T) {
	name := "My Node"
	node, err := model.NewWorkflowNode("node-1", model.NodeTypeExecutor, &name, model.Position{X: 150, Y: 250}, map[string]any{"executorId": "prov-001"})
	require.NoError(t, err)

	nodeModel := mongoworkflow.NodeModelFromEntity(node)

	assert.Equal(t, "node-1", nodeModel.ID)
	assert.Equal(t, "executor", nodeModel.Type)
	assert.Equal(t, &name, nodeModel.Name)
	assert.Equal(t, 150, nodeModel.Position.X)
	assert.Equal(t, 250, nodeModel.Position.Y)
	assert.Equal(t, "prov-001", nodeModel.Data["executorId"])
}

func TestEdgeModel_ToEntity(t *testing.T) {
	sourceHandle := "true"
	label := "Success"

	edgeModel := &mongoworkflow.EdgeModel{
		ID:           "edge-1",
		Source:       "conditional-1",
		Target:       "provider-1",
		SourceHandle: &sourceHandle,
		Label:        &label,
	}

	entity := edgeModel.ToEntity()

	assert.Equal(t, "edge-1", entity.ID())
	assert.Equal(t, "conditional-1", entity.Source())
	assert.Equal(t, "provider-1", entity.Target())
	assert.Equal(t, "true", *entity.SourceHandle())
	assert.Equal(t, "Success", *entity.Label())
}

func TestEdgeModelFromEntity(t *testing.T) {
	edge, _ := model.NewWorkflowEdge("edge-1", "source", "target")
	edge.WithSourceHandle("false")
	edge.WithLabel("Failure")

	edgeModel := mongoworkflow.EdgeModelFromEntity(edge)

	assert.Equal(t, "edge-1", edgeModel.ID)
	assert.Equal(t, "source", edgeModel.Source)
	assert.Equal(t, "target", edgeModel.Target)
	assert.Equal(t, "false", *edgeModel.SourceHandle)
	assert.Equal(t, "Failure", *edgeModel.Label)
}

func TestMongoDBModel_RoundTrip(t *testing.T) {
	// Create a workflow with all fields
	description := "Full workflow"
	providerName := "Call Provider"
	triggerNode, err := model.NewWorkflowNode("trigger-1", model.NodeTypeTrigger, nil, model.Position{X: 100, Y: 50}, map[string]any{"triggerType": "http"})
	require.NoError(t, err)
	providerNode, err := model.NewWorkflowNode("provider-1", model.NodeTypeExecutor, &providerName, model.Position{X: 100, Y: 150}, map[string]any{"executorId": "prov-001"})
	require.NoError(t, err)
	conditionalNode, err := model.NewWorkflowNode("conditional-1", model.NodeTypeConditional, nil, model.Position{X: 100, Y: 250}, map[string]any{"condition": "workflow.result == 'approved'"})
	require.NoError(t, err)
	nodes := []model.WorkflowNode{
		triggerNode,
		providerNode,
		conditionalNode,
	}
	edge1, _ := model.NewWorkflowEdge("edge-1", "trigger-1", "provider-1")
	edge2, _ := model.NewWorkflowEdge("edge-2", "provider-1", "conditional-1")
	edges := []model.WorkflowEdge{
		edge1,
		edge2,
	}

	original, err := model.NewWorkflow("Round Trip Test", &description, nodes, edges)
	require.NoError(t, err)

	original.SetMetadata("version", "1.0")

	// Convert to MongoDB model
	mongoModel := mongoworkflow.NewMongoDBModelFromEntity(original)

	// Convert back to entity
	restored := mongoModel.ToEntity()

	// Verify all fields match
	assert.Equal(t, original.ID(), restored.ID())
	assert.Equal(t, original.Name(), restored.Name())
	assert.Equal(t, original.Description(), restored.Description())
	assert.Equal(t, original.Status(), restored.Status())
	assert.Len(t, restored.Nodes(), 3)
	assert.Len(t, restored.Edges(), 2)
	assert.Equal(t, original.Metadata()["version"], restored.Metadata()["version"])
}
