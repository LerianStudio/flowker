// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import "github.com/google/uuid"

// CreateWorkflowInput is the input DTO for creating a workflow.
// Uses validator tags per PROJECT_RULES.md.
type CreateWorkflowInput struct {
	Name        string              `json:"name" validate:"required,min=1,max=100"`
	Description *string             `json:"description,omitempty" validate:"omitempty,max=500"`
	Nodes       []WorkflowNodeInput `json:"nodes,omitempty" validate:"omitempty,max=100,dive"`
	Edges       []WorkflowEdgeInput `json:"edges,omitempty" validate:"omitempty,max=200,dive"`
	Metadata    map[string]any      `json:"metadata,omitempty" validate:"omitempty"`
}

// UpdateWorkflowInput is the input DTO for updating a workflow.
type UpdateWorkflowInput struct {
	Name        string              `json:"name" validate:"required,min=1,max=100"`
	Description *string             `json:"description,omitempty" validate:"omitempty,max=500"`
	Nodes       []WorkflowNodeInput `json:"nodes,omitempty" validate:"omitempty,max=100,dive"`
	Edges       []WorkflowEdgeInput `json:"edges,omitempty" validate:"omitempty,max=200,dive"`
	Metadata    map[string]any      `json:"metadata,omitempty" validate:"omitempty"`
}

// CloneWorkflowInput is the input DTO for cloning a workflow.
type CloneWorkflowInput struct {
	Name string `json:"name" validate:"required,min=1,max=100"`
}

// PositionInput is the input DTO for node position.
// X and Y do not use validate:"required" because 0 is a valid coordinate (top-left origin).
type PositionInput struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// WorkflowNodeInput is the input DTO for a workflow node.
type WorkflowNodeInput struct {
	ID       string         `json:"id" validate:"required,min=1,max=50"`
	Type     string         `json:"type" validate:"required,oneof=trigger executor conditional action"`
	Name     *string        `json:"name,omitempty" validate:"omitempty,max=50"`
	Position PositionInput  `json:"position" validate:"required"`
	Data     map[string]any `json:"data,omitempty"`
}

// WorkflowEdgeInput is the input DTO for a workflow edge.
type WorkflowEdgeInput struct {
	ID           string  `json:"id" validate:"required,min=1,max=50"`
	Source       string  `json:"source" validate:"required,min=1,max=50"`
	Target       string  `json:"target" validate:"required,min=1,max=50"`
	SourceHandle *string `json:"sourceHandle,omitempty" validate:"omitempty,max=50"`
	Condition    *string `json:"condition,omitempty" validate:"omitempty,max=500"`
	Label        *string `json:"label,omitempty" validate:"omitempty,max=50"`
}

// ToDomain converts CreateWorkflowInput to domain entity.
func (i *CreateWorkflowInput) ToDomain() (*Workflow, error) {
	nodes := make([]WorkflowNode, len(i.Nodes))
	for idx, nodeInput := range i.Nodes {
		nodes[idx] = nodeInput.ToDomain()
	}

	edges := make([]WorkflowEdge, len(i.Edges))
	for idx, edgeInput := range i.Edges {
		edges[idx] = edgeInput.ToDomain()
	}

	workflow, err := NewWorkflow(i.Name, i.Description, nodes, edges)
	if err != nil {
		return nil, err
	}

	// Set metadata if provided
	if i.Metadata != nil {
		for k, v := range i.Metadata {
			workflow.SetMetadata(k, v)
		}
	}

	return workflow, nil
}

// ToDomain converts WorkflowNodeInput to domain entity.
// Uses NewWorkflowNodeFromDB since input is already validated by struct tags.
func (i *WorkflowNodeInput) ToDomain() WorkflowNode {
	nodeType := NodeType(i.Type)
	position := Position{X: i.Position.X, Y: i.Position.Y}

	return NewWorkflowNodeFromDB(i.ID, nodeType, i.Name, position, i.Data)
}

// ToDomain converts WorkflowEdgeInput to domain entity.
// Uses NewWorkflowEdgeFromDB since input is already validated by struct tags.
func (i *WorkflowEdgeInput) ToDomain() WorkflowEdge {
	return NewWorkflowEdgeFromDB(i.ID, i.Source, i.Target, i.SourceHandle, i.Condition, i.Label)
}

// ToNodes converts UpdateWorkflowInput nodes to domain entities.
func (i *UpdateWorkflowInput) ToNodes() []WorkflowNode {
	nodes := make([]WorkflowNode, len(i.Nodes))
	for idx, nodeInput := range i.Nodes {
		nodes[idx] = nodeInput.ToDomain()
	}

	return nodes
}

// ToEdges converts UpdateWorkflowInput edges to domain entities.
func (i *UpdateWorkflowInput) ToEdges() []WorkflowEdge {
	edges := make([]WorkflowEdge, len(i.Edges))
	for idx, edgeInput := range i.Edges {
		edges[idx] = edgeInput.ToDomain()
	}

	return edges
}

// WorkflowFilterInput is the input DTO for listing workflows with filters.
type WorkflowFilterInput struct {
	Status    *string `query:"status" validate:"omitempty,oneof=draft active inactive"`
	Limit     int     `query:"limit" validate:"omitempty,min=1,max=100"`
	Cursor    string  `query:"cursor"`
	SortBy    string  `query:"sortBy" validate:"omitempty,oneof=createdAt updatedAt name"`
	SortOrder string  `query:"sortOrder" validate:"omitempty,oneof=ASC DESC"`
}

// GetWorkflowInput is the input DTO for getting a workflow by ID.
type GetWorkflowInput struct {
	ID uuid.UUID `params:"id" validate:"required"`
}
