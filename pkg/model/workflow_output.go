// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"time"

	"github.com/google/uuid"
)

// WorkflowOutput is the output DTO for a workflow.
type WorkflowOutput struct {
	ID          uuid.UUID            `json:"id" swaggertype:"string" format:"uuid"`
	Name        string               `json:"name"`
	Description *string              `json:"description,omitempty"`
	Status      string               `json:"status"`
	Nodes       []WorkflowNodeOutput `json:"nodes"`
	Edges       []WorkflowEdgeOutput `json:"edges"`
	Metadata    map[string]any       `json:"metadata,omitempty"`
	CreatedAt   time.Time            `json:"createdAt"`
	UpdatedAt   time.Time            `json:"updatedAt"`
}

// PositionOutput is the output DTO for node position.
type PositionOutput struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// WorkflowNodeOutput is the output DTO for a workflow node.
type WorkflowNodeOutput struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Name     *string        `json:"name,omitempty"`
	Position PositionOutput `json:"position"`
	Data     map[string]any `json:"data,omitempty"`
}

// WorkflowEdgeOutput is the output DTO for a workflow edge.
type WorkflowEdgeOutput struct {
	ID           string  `json:"id"`
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	SourceHandle *string `json:"sourceHandle,omitempty"`
	Condition    *string `json:"condition,omitempty"`
	Label        *string `json:"label,omitempty"`
}

// WorkflowCreateOutput is the minimal output for workflow creation.
type WorkflowCreateOutput struct {
	ID        uuid.UUID `json:"workflowId" swaggertype:"string" format:"uuid"`
	Version   string    `json:"version"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// WorkflowListOutput is the output DTO for listing workflows.
type WorkflowListOutput struct {
	Items      []WorkflowOutput `json:"items"`
	NextCursor string           `json:"nextCursor"`
	HasMore    bool             `json:"hasMore"`
}

// WorkflowOutputFromDomain converts a Workflow domain entity to WorkflowOutput.
func WorkflowOutputFromDomain(w *Workflow) WorkflowOutput {
	nodes := make([]WorkflowNodeOutput, len(w.Nodes()))
	for i, node := range w.Nodes() {
		nodes[i] = WorkflowNodeOutputFromDomain(node)
	}

	edges := make([]WorkflowEdgeOutput, len(w.Edges()))
	for i, edge := range w.Edges() {
		edges[i] = WorkflowEdgeOutputFromDomain(edge)
	}

	return WorkflowOutput{
		ID:          w.ID(),
		Name:        w.Name(),
		Description: w.Description(),
		Status:      string(w.Status()),
		Nodes:       nodes,
		Edges:       edges,
		Metadata:    w.Metadata(),
		CreatedAt:   w.CreatedAt(),
		UpdatedAt:   w.UpdatedAt(),
	}
}

// WorkflowNodeOutputFromDomain converts a WorkflowNode to WorkflowNodeOutput.
func WorkflowNodeOutputFromDomain(n WorkflowNode) WorkflowNodeOutput {
	return WorkflowNodeOutput{
		ID:   n.ID(),
		Type: string(n.Type()),
		Name: n.Name(),
		Position: PositionOutput{
			X: n.Position().X,
			Y: n.Position().Y,
		},
		Data: n.Data(),
	}
}

// WorkflowEdgeOutputFromDomain converts a WorkflowEdge to WorkflowEdgeOutput.
func WorkflowEdgeOutputFromDomain(e WorkflowEdge) WorkflowEdgeOutput {
	return WorkflowEdgeOutput{
		ID:           e.ID(),
		Source:       e.Source(),
		Target:       e.Target(),
		SourceHandle: e.SourceHandle(),
		Condition:    e.Condition(),
		Label:        e.Label(),
	}
}

// WorkflowCreateOutputFromDomain creates a minimal creation response.
func WorkflowCreateOutputFromDomain(w *Workflow) WorkflowCreateOutput {
	return WorkflowCreateOutput{
		ID:        w.ID(),
		Version:   "1.0.0",
		Status:    string(w.Status()),
		CreatedAt: w.CreatedAt(),
	}
}

// WorkflowListOutputFromDomain converts a list of workflows to list output.
func WorkflowListOutputFromDomain(workflows []*Workflow, nextCursor string, hasMore bool) WorkflowListOutput {
	items := make([]WorkflowOutput, len(workflows))
	for i, w := range workflows {
		items[i] = WorkflowOutputFromDomain(w)
	}

	return WorkflowListOutput{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}
