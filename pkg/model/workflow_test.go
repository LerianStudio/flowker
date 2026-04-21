// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package model_test

import (
	"testing"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers for creating nodes and edges
func createTriggerNode(t *testing.T, id string) model.WorkflowNode {
	t.Helper()

	node, err := model.NewWorkflowNode(id, model.NodeTypeTrigger, nil, model.Position{X: 100, Y: 50}, map[string]any{
		"triggerType": "http",
	})
	require.NoError(t, err)

	return node
}

func createExecutorNode(t *testing.T, id string, name string) model.WorkflowNode {
	t.Helper()

	n := name
	node, err := model.NewWorkflowNode(id, model.NodeTypeExecutor, &n, model.Position{X: 100, Y: 150}, map[string]any{
		"executorId": "provider-001",
	})
	require.NoError(t, err)

	return node
}

func createConditionalNode(t *testing.T, id string, name string) model.WorkflowNode {
	t.Helper()

	n := name
	node, err := model.NewWorkflowNode(id, model.NodeTypeConditional, &n, model.Position{X: 100, Y: 250}, map[string]any{
		"condition": "workflow.score > 70",
	})
	require.NoError(t, err)

	return node
}

func createEdge(t *testing.T, id, source, target string) model.WorkflowEdge {
	t.Helper()

	edge, err := model.NewWorkflowEdge(id, source, target)
	require.NoError(t, err)

	return edge
}

func TestNewWorkflow_Success(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
		createExecutorNode(t, "provider-1", "Call KYC Provider"),
	}
	edges := []model.WorkflowEdge{
		createEdge(t, "edge-1", "trigger-1", "provider-1"),
	}
	description := "Test workflow description"

	workflow, err := model.NewWorkflow("Test Workflow", &description, nodes, edges)

	require.NoError(t, err)
	require.NotNil(t, workflow)
	assert.NotEqual(t, uuid.Nil, workflow.ID())
	assert.Equal(t, "Test Workflow", workflow.Name())
	assert.Equal(t, &description, workflow.Description())
	assert.Equal(t, model.WorkflowStatusDraft, workflow.Status())
	assert.Len(t, workflow.Nodes(), 2)
	assert.Len(t, workflow.Edges(), 1)
	assert.False(t, workflow.CreatedAt().IsZero())
	assert.False(t, workflow.UpdatedAt().IsZero())
}

func TestNewWorkflow_EmptyName(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}

	workflow, err := model.NewWorkflow("", nil, nodes, nil)

	require.ErrorIs(t, err, model.ErrWorkflowNameRequired)
	assert.Nil(t, workflow)
}

func TestNewWorkflow_NoNodes(t *testing.T) {
	workflow, err := model.NewWorkflow("Test", nil, nil, nil)

	require.NoError(t, err)
	require.NotNil(t, workflow)
	assert.Equal(t, "Test", workflow.Name())
	assert.Equal(t, model.WorkflowStatusDraft, workflow.Status())
	assert.Nil(t, workflow.Nodes())
}

func TestNewWorkflow_EmptyNodes(t *testing.T) {
	workflow, err := model.NewWorkflow("Test", nil, []model.WorkflowNode{}, nil)

	require.NoError(t, err)
	require.NotNil(t, workflow)
	assert.Equal(t, "Test", workflow.Name())
	assert.Equal(t, model.WorkflowStatusDraft, workflow.Status())
	assert.Empty(t, workflow.Nodes())
}

func TestNewWorkflow_TooManyNodes(t *testing.T) {
	nodes := make([]model.WorkflowNode, 101)
	nodes[0] = createTriggerNode(t, "trigger-1")
	for i := 1; i < len(nodes); i++ {
		name := "Node"
		node, _ := model.NewWorkflowNode("node", model.NodeTypeExecutor, &name, model.Position{X: 0, Y: 0}, nil)
		nodes[i] = node
	}

	workflow, err := model.NewWorkflow("Test", nil, nodes, nil)

	require.ErrorIs(t, err, model.ErrWorkflowTooManyNodes)
	assert.Nil(t, workflow)
}

func TestNewWorkflow_NoTrigger(t *testing.T) {
	nodes := []model.WorkflowNode{
		createExecutorNode(t, "provider-1", "Provider"),
	}

	workflow, err := model.NewWorkflow("Test", nil, nodes, nil)

	require.NoError(t, err)
	require.NotNil(t, workflow)
	assert.Equal(t, model.WorkflowStatusDraft, workflow.Status())
	assert.Len(t, workflow.Nodes(), 1)
}

func TestNewWorkflow_InvalidEdgeRef(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}
	edges := []model.WorkflowEdge{
		createEdge(t, "edge-1", "trigger-1", "non-existent"),
	}

	workflow, err := model.NewWorkflow("Test", nil, nodes, edges)

	require.ErrorIs(t, err, model.ErrWorkflowInvalidEdgeRef)
	assert.Nil(t, workflow)
}

func TestNewWorkflow_NameTooLong(t *testing.T) {
	longName := make([]byte, 101)
	for i := range longName {
		longName[i] = 'a'
	}
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}

	workflow, err := model.NewWorkflow(string(longName), nil, nodes, nil)

	require.ErrorIs(t, err, model.ErrWorkflowNameTooLong)
	assert.Nil(t, workflow)
}

func TestWorkflow_Activate_FromDraft(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}
	workflow, _ := model.NewWorkflow("Test", nil, nodes, nil)

	err := workflow.Activate()

	require.NoError(t, err)
	assert.Equal(t, model.WorkflowStatusActive, workflow.Status())
}

func TestWorkflow_Activate_FromActive(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}
	workflow, _ := model.NewWorkflow("Test", nil, nodes, nil)
	_ = workflow.Activate()

	err := workflow.Activate()

	require.ErrorIs(t, err, model.ErrWorkflowCannotActivate)
}

func TestWorkflow_Activate_FromInactive(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}
	workflow, _ := model.NewWorkflow("Test", nil, nodes, nil)
	_ = workflow.Activate()
	_ = workflow.Deactivate()

	err := workflow.Activate()

	require.ErrorIs(t, err, model.ErrWorkflowCannotActivate)
}

func TestWorkflow_Deactivate_FromActive(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}
	workflow, _ := model.NewWorkflow("Test", nil, nodes, nil)
	_ = workflow.Activate()

	err := workflow.Deactivate()

	require.NoError(t, err)
	assert.Equal(t, model.WorkflowStatusInactive, workflow.Status())
}

func TestWorkflow_Deactivate_FromDraft(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}
	workflow, _ := model.NewWorkflow("Test", nil, nodes, nil)

	err := workflow.Deactivate()

	require.ErrorIs(t, err, model.ErrWorkflowCannotDeactivate)
}

func TestWorkflow_Update_Draft(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}
	workflow, _ := model.NewWorkflow("Test", nil, nodes, nil)
	originalUpdatedAt := workflow.UpdatedAt()
	time.Sleep(1 * time.Millisecond) // Ensure time difference

	newDescription := "New description"
	newNodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
		createExecutorNode(t, "provider-1", "New Provider"),
	}
	newEdges := []model.WorkflowEdge{
		createEdge(t, "edge-1", "trigger-1", "provider-1"),
	}

	err := workflow.Update("New Name", &newDescription, newNodes, newEdges)

	require.NoError(t, err)
	assert.Equal(t, "New Name", workflow.Name())
	assert.Equal(t, &newDescription, workflow.Description())
	assert.Len(t, workflow.Nodes(), 2)
	assert.Len(t, workflow.Edges(), 1)
	assert.True(t, workflow.UpdatedAt().After(originalUpdatedAt))
}

func TestWorkflow_Update_ActiveWorkflow(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}
	workflow, _ := model.NewWorkflow("Test", nil, nodes, nil)
	_ = workflow.Activate()

	err := workflow.Update("New Name", nil, nodes, nil)

	require.ErrorIs(t, err, model.ErrWorkflowCannotUpdate)
}

func TestWorkflow_Clone(t *testing.T) {
	description := "Original description"
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
		createExecutorNode(t, "provider-1", "Provider"),
	}
	edges := []model.WorkflowEdge{
		createEdge(t, "edge-1", "trigger-1", "provider-1"),
	}
	original, _ := model.NewWorkflow("Original", &description, nodes, edges)
	_ = original.Activate()

	cloned, err := original.Clone("Cloned Workflow")

	require.NoError(t, err)
	require.NotNil(t, cloned)
	assert.NotEqual(t, original.ID(), cloned.ID())
	assert.Equal(t, "Cloned Workflow", cloned.Name())
	assert.Equal(t, original.Description(), cloned.Description())
	assert.Equal(t, model.WorkflowStatusDraft, cloned.Status())
	assert.Len(t, cloned.Nodes(), len(original.Nodes()))
	assert.Len(t, cloned.Edges(), len(original.Edges()))
}

func TestWorkflow_IsActive(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}
	workflow, _ := model.NewWorkflow("Test", nil, nodes, nil)

	assert.False(t, workflow.IsActive())

	_ = workflow.Activate()

	assert.True(t, workflow.IsActive())
}

func TestWorkflow_IsDraft(t *testing.T) {
	nodes := []model.WorkflowNode{
		createTriggerNode(t, "trigger-1"),
	}
	workflow, _ := model.NewWorkflow("Test", nil, nodes, nil)

	assert.True(t, workflow.IsDraft())

	_ = workflow.Activate()

	assert.False(t, workflow.IsDraft())
}

// WorkflowNode tests
func TestNewWorkflowNode(t *testing.T) {
	data := map[string]any{
		"executorId":   "provider-123",
		"endpointName": "validate",
	}
	name := "Call Provider"
	position := model.Position{X: 100, Y: 150}

	node, err := model.NewWorkflowNode("node-1", model.NodeTypeExecutor, &name, position, data)

	require.NoError(t, err)
	assert.Equal(t, "node-1", node.ID())
	assert.Equal(t, model.NodeTypeExecutor, node.Type())
	assert.Equal(t, &name, node.Name())
	assert.Equal(t, position, node.Position())
	assert.Equal(t, data, node.Data())
}

func TestNewWorkflowNode_EmptyID(t *testing.T) {
	_, err := model.NewWorkflowNode("", model.NodeTypeExecutor, nil, model.Position{}, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrNodeIDRequired)
}

func TestNewWorkflowNode_EmptyType(t *testing.T) {
	_, err := model.NewWorkflowNode("node-1", "", nil, model.Position{}, nil)

	require.Error(t, err)
	assert.ErrorIs(t, err, model.ErrNodeTypeRequired)
}

func TestWorkflowNode_ExecutorID(t *testing.T) {
	data := map[string]any{
		"executorId": "provider-123",
	}
	node, err := model.NewWorkflowNode("node-1", model.NodeTypeExecutor, nil, model.Position{}, data)

	require.NoError(t, err)
	assert.Equal(t, "provider-123", node.ExecutorID())
}

func TestWorkflowNode_TriggerType(t *testing.T) {
	data := map[string]any{
		"triggerType": "http",
	}
	node, err := model.NewWorkflowNode("node-1", model.NodeTypeTrigger, nil, model.Position{}, data)

	require.NoError(t, err)
	assert.Equal(t, "http", node.TriggerType())
}

func TestWorkflowNode_Condition(t *testing.T) {
	data := map[string]any{
		"condition": "workflow.score > 70",
	}
	node, err := model.NewWorkflowNode("node-1", model.NodeTypeConditional, nil, model.Position{}, data)

	require.NoError(t, err)
	assert.Equal(t, "workflow.score > 70", node.Condition())
}

// WorkflowEdge tests
func TestNewWorkflowEdge(t *testing.T) {
	edge, err := model.NewWorkflowEdge("edge-1", "source-node", "target-node")

	require.NoError(t, err)
	assert.Equal(t, "edge-1", edge.ID())
	assert.Equal(t, "source-node", edge.Source())
	assert.Equal(t, "target-node", edge.Target())
	assert.Nil(t, edge.SourceHandle())
	assert.Nil(t, edge.Condition())
	assert.Nil(t, edge.Label())
}

func TestNewWorkflowEdge_Validation(t *testing.T) {
	t.Run("empty id", func(t *testing.T) {
		_, err := model.NewWorkflowEdge("", "source", "target")
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrEdgeIDRequired)
	})

	t.Run("empty source", func(t *testing.T) {
		_, err := model.NewWorkflowEdge("edge-1", "", "target")
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrEdgeSourceRequired)
	})

	t.Run("empty target", func(t *testing.T) {
		_, err := model.NewWorkflowEdge("edge-1", "source", "")
		require.Error(t, err)
		assert.ErrorIs(t, err, model.ErrEdgeTargetRequired)
	})
}

func TestWorkflowEdge_WithSourceHandle(t *testing.T) {
	edge, err := model.NewWorkflowEdge("edge-1", "source", "target")
	require.NoError(t, err)
	edge.WithSourceHandle("true")

	assert.Equal(t, "true", *edge.SourceHandle())
}

func TestWorkflowEdge_WithLabel(t *testing.T) {
	edge, err := model.NewWorkflowEdge("edge-1", "source", "target")
	require.NoError(t, err)
	edge.WithLabel("Success Path")

	assert.Equal(t, "Success Path", *edge.Label())
}
