// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package workflow contains the MongoDB adapter for workflow persistence.
package workflow

import (
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MongoDBModel is the MongoDB document model for workflows.
// Implements ToEntity/FromEntity pattern per PROJECT_RULES.md.
type MongoDBModel struct {
	ObjectID    primitive.ObjectID `bson:"_id,omitempty"`
	WorkflowID  string             `bson:"workflowId"`
	Name        string             `bson:"name"`
	Description *string            `bson:"description,omitempty"`
	Status      string             `bson:"status"`
	Nodes       []NodeModel        `bson:"nodes"`
	Edges       []EdgeModel        `bson:"edges,omitempty"`
	Metadata    map[string]any     `bson:"metadata,omitempty"`
	CreatedAt   time.Time          `bson:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt"`
}

// NodeModel is the MongoDB model for workflow nodes.
type NodeModel struct {
	ID       string         `bson:"id"`
	Type     string         `bson:"type"`
	Name     *string        `bson:"name,omitempty"`
	Position PositionModel  `bson:"position"`
	Data     map[string]any `bson:"data,omitempty"`
}

// PositionModel is the MongoDB model for node position.
type PositionModel struct {
	X int `bson:"x"`
	Y int `bson:"y"`
}

// EdgeModel is the MongoDB model for workflow edges.
type EdgeModel struct {
	ID           string  `bson:"id"`
	Source       string  `bson:"source"`
	Target       string  `bson:"target"`
	SourceHandle *string `bson:"sourceHandle,omitempty"`
	Condition    *string `bson:"condition,omitempty"`
	Label        *string `bson:"label,omitempty"`
}

// ToEntity converts MongoDBModel to domain entity.
func (m *MongoDBModel) ToEntity() *model.Workflow {
	workflowID, _ := uuid.Parse(m.WorkflowID)

	nodes := make([]model.WorkflowNode, len(m.Nodes))
	for i, nodeModel := range m.Nodes {
		nodes[i] = nodeModel.ToEntity()
	}

	edges := make([]model.WorkflowEdge, len(m.Edges))
	for i, edgeModel := range m.Edges {
		edges[i] = edgeModel.ToEntity()
	}

	return model.NewWorkflowFromDB(
		workflowID,
		m.Name,
		m.Description,
		model.WorkflowStatus(m.Status),
		nodes,
		edges,
		m.Metadata,
		m.CreatedAt,
		m.UpdatedAt,
	)
}

// FromEntity populates MongoDBModel from domain entity.
func (m *MongoDBModel) FromEntity(w *model.Workflow) {
	m.WorkflowID = w.ID().String()
	m.Name = w.Name()
	m.Description = w.Description()
	m.Status = string(w.Status())
	m.Metadata = w.Metadata()
	m.CreatedAt = w.CreatedAt()
	m.UpdatedAt = w.UpdatedAt()

	m.Nodes = make([]NodeModel, len(w.Nodes()))
	for i, node := range w.Nodes() {
		m.Nodes[i] = NodeModelFromEntity(node)
	}

	m.Edges = make([]EdgeModel, len(w.Edges()))
	for i, edge := range w.Edges() {
		m.Edges[i] = EdgeModelFromEntity(edge)
	}
}

// ToEntity converts NodeModel to domain entity.
func (n *NodeModel) ToEntity() model.WorkflowNode {
	position := model.Position{X: n.Position.X, Y: n.Position.Y}

	return model.NewWorkflowNodeFromDB(
		n.ID,
		model.NodeType(n.Type),
		n.Name,
		position,
		normalizeMapData(n.Data),
	)
}

// normalizeMapData recursively converts BSON types (primitive.D, primitive.A)
// to standard Go types (map[string]any, []any) so that type assertions in
// domain logic (e.g., parseFieldMappings) work correctly after MongoDB round-trip.
func normalizeMapData(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}

	result := make(map[string]any, len(data))
	for k, v := range data {
		result[k] = normalizeBSONValue(v)
	}

	return result
}

// normalizeBSONValue converts a single BSON value to a standard Go type.
func normalizeBSONValue(v any) any {
	switch val := v.(type) {
	case primitive.D:
		m := make(map[string]any, len(val))
		for _, e := range val {
			m[e.Key] = normalizeBSONValue(e.Value)
		}

		return m
	case primitive.A:
		a := make([]any, len(val))
		for i, e := range val {
			a[i] = normalizeBSONValue(e)
		}

		return a
	case map[string]any:
		m := make(map[string]any, len(val))
		for k, e := range val {
			m[k] = normalizeBSONValue(e)
		}

		return m
	case []any:
		a := make([]any, len(val))
		for i, e := range val {
			a[i] = normalizeBSONValue(e)
		}

		return a
	default:
		return v
	}
}

// NodeModelFromEntity creates a NodeModel from domain entity.
func NodeModelFromEntity(n model.WorkflowNode) NodeModel {
	return NodeModel{
		ID:   n.ID(),
		Type: string(n.Type()),
		Name: n.Name(),
		Position: PositionModel{
			X: n.Position().X,
			Y: n.Position().Y,
		},
		Data: n.Data(),
	}
}

// ToEntity converts EdgeModel to domain entity.
func (e *EdgeModel) ToEntity() model.WorkflowEdge {
	return model.NewWorkflowEdgeFromDB(
		e.ID,
		e.Source,
		e.Target,
		e.SourceHandle,
		e.Condition,
		e.Label,
	)
}

// EdgeModelFromEntity creates an EdgeModel from domain entity.
func EdgeModelFromEntity(e model.WorkflowEdge) EdgeModel {
	return EdgeModel{
		ID:           e.ID(),
		Source:       e.Source(),
		Target:       e.Target(),
		SourceHandle: e.SourceHandle(),
		Condition:    e.Condition(),
		Label:        e.Label(),
	}
}

// NewMongoDBModelFromEntity creates a new MongoDBModel from domain entity.
func NewMongoDBModelFromEntity(w *model.Workflow) *MongoDBModel {
	m := &MongoDBModel{}
	m.FromEntity(w)

	return m
}
