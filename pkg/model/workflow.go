// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package model contains domain entities and DTOs for Flowker.
package model

import (
	"time"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/google/uuid"
)

// WorkflowStatus represents the status of a workflow.
type WorkflowStatus string

const (
	// WorkflowStatusDraft indicates a workflow that is editable but not executable.
	WorkflowStatusDraft WorkflowStatus = "draft"
	// WorkflowStatusActive indicates a workflow that is executable but not editable.
	WorkflowStatusActive WorkflowStatus = "active"
	// WorkflowStatusInactive indicates a workflow that is archived.
	WorkflowStatusInactive WorkflowStatus = "inactive"
)

// Workflow validation errors
var (
	ErrWorkflowNameRequired = pkg.ValidationError{
		Code:    constant.ErrWorkflowNameRequired.Error(),
		Message: "name is required",
	}
	ErrWorkflowNameTooLong = pkg.ValidationError{
		Code:    constant.ErrWorkflowNameTooLong.Error(),
		Message: "name cannot exceed 100 characters",
	}
	ErrWorkflowNodesRequired = pkg.ValidationError{
		Code:    constant.ErrWorkflowNodesRequired.Error(),
		Message: "at least one node is required",
	}
	ErrWorkflowTooManyNodes = pkg.ValidationError{
		Code:    constant.ErrWorkflowTooManyNodes.Error(),
		Message: "cannot exceed 100 nodes",
	}
	ErrWorkflowTooManyEdges = pkg.ValidationError{
		Code:    constant.ErrWorkflowTooManyEdges.Error(),
		Message: "cannot exceed 200 edges",
	}
	ErrWorkflowInvalidEdgeRef = pkg.ValidationError{
		Code:    constant.ErrWorkflowInvalidEdgeRef.Error(),
		Message: "edge references non-existent node",
	}
	ErrWorkflowNoTrigger = pkg.ValidationError{
		Code:    constant.ErrWorkflowNoTrigger.Error(),
		Message: "workflow must have at least one trigger node",
	}
	ErrWorkflowCannotActivate = pkg.ValidationError{
		Code:    constant.ErrWorkflowCannotModify.Error(),
		Message: "only draft workflows can be activated",
	}
	ErrWorkflowCannotDeactivate = pkg.ValidationError{
		Code:    constant.ErrWorkflowCannotModify.Error(),
		Message: "only active workflows can be deactivated",
	}
	ErrWorkflowCannotUpdate = pkg.ValidationError{
		Code:    constant.ErrWorkflowCannotModify.Error(),
		Message: "can only update draft workflows",
	}
	ErrWorkflowCannotMoveToDraft = pkg.ValidationError{
		Code:    constant.ErrWorkflowCannotModify.Error(),
		Message: "only inactive workflows can be moved to draft",
	}
)

const (
	maxWorkflowNameLength = 100
	maxWorkflowNodes      = 100
	maxWorkflowEdges      = 200
)

// validateWorkflowDataForDraft validates name, nodes, and edges for draft workflows.
// Draft workflows allow empty nodes and missing trigger nodes.
func validateWorkflowDataForDraft(name string, nodes []WorkflowNode, edges []WorkflowEdge) error {
	if name == "" {
		return ErrWorkflowNameRequired
	}

	if len(name) > maxWorkflowNameLength {
		return ErrWorkflowNameTooLong
	}

	if len(nodes) > maxWorkflowNodes {
		return ErrWorkflowTooManyNodes
	}

	if len(edges) > maxWorkflowEdges {
		return ErrWorkflowTooManyEdges
	}

	// Validate edge references if edges are present
	if len(edges) > 0 {
		nodeIDs := make(map[string]bool)
		for _, node := range nodes {
			nodeIDs[node.ID()] = true
		}

		for _, edge := range edges {
			if !nodeIDs[edge.Source()] || !nodeIDs[edge.Target()] {
				return ErrWorkflowInvalidEdgeRef
			}
		}
	}

	return nil
}

// ValidateWorkflowStructure validates the full structural requirements of a workflow.
// Used at activation time to ensure: non-empty name, at least one node, at least one
// trigger, valid edge references, and size limits. Draft creation uses the permissive
// validateWorkflowDataForDraft instead.
func ValidateWorkflowStructure(name string, nodes []WorkflowNode, edges []WorkflowEdge) error {
	if name == "" {
		return ErrWorkflowNameRequired
	}

	if len(name) > maxWorkflowNameLength {
		return ErrWorkflowNameTooLong
	}

	if len(nodes) == 0 {
		return ErrWorkflowNodesRequired
	}

	if len(nodes) > maxWorkflowNodes {
		return ErrWorkflowTooManyNodes
	}

	if len(edges) > maxWorkflowEdges {
		return ErrWorkflowTooManyEdges
	}

	// Validate edge references and trigger existence
	nodeIDs := make(map[string]bool)
	hasTrigger := false

	for _, node := range nodes {
		nodeIDs[node.ID()] = true
		if node.Type() == NodeTypeTrigger {
			hasTrigger = true
		}
	}

	if !hasTrigger {
		return ErrWorkflowNoTrigger
	}

	for _, edge := range edges {
		if !nodeIDs[edge.Source()] || !nodeIDs[edge.Target()] {
			return ErrWorkflowInvalidEdgeRef
		}
	}

	return nil
}

// Workflow represents a workflow definition (Rich Domain Model).
// Fields are private with validation in constructor per PROJECT_RULES.md.
// Uses graph-based structure with nodes and edges for visual canvas rendering.
type Workflow struct {
	id          uuid.UUID
	name        string
	description *string
	status      WorkflowStatus
	nodes       []WorkflowNode
	edges       []WorkflowEdge
	metadata    map[string]any
	createdAt   time.Time
	updatedAt   time.Time
}

// NewWorkflow creates a new Workflow with draft validation.
// Draft workflows allow empty nodes and missing trigger nodes.
// Full validation (nodes required, trigger required) happens at activation time.
func NewWorkflow(name string, description *string, nodes []WorkflowNode, edges []WorkflowEdge) (*Workflow, error) {
	if err := validateWorkflowDataForDraft(name, nodes, edges); err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	return &Workflow{
		id:          uuid.New(),
		name:        name,
		description: description,
		status:      WorkflowStatusDraft,
		nodes:       cloneNodes(nodes),
		edges:       cloneEdges(edges),
		metadata:    make(map[string]any),
		createdAt:   now,
		updatedAt:   now,
	}, nil
}

// NewWorkflowFromDB reconstructs a Workflow from database values.
// Used by repository adapters - bypasses validation since data is already valid.
func NewWorkflowFromDB(
	id uuid.UUID,
	name string,
	description *string,
	status WorkflowStatus,
	nodes []WorkflowNode,
	edges []WorkflowEdge,
	metadata map[string]any,
	createdAt, updatedAt time.Time,
) *Workflow {
	return &Workflow{
		id:          id,
		name:        name,
		description: description,
		status:      status,
		nodes:       cloneNodes(nodes),
		edges:       cloneEdges(edges),
		metadata:    cloneMetadata(metadata),
		createdAt:   createdAt,
		updatedAt:   updatedAt,
	}
}

// ID returns the workflow's unique identifier.
func (w *Workflow) ID() uuid.UUID {
	return w.id
}

// Name returns the workflow's name.
func (w *Workflow) Name() string {
	return w.name
}

// Description returns the workflow's description.
func (w *Workflow) Description() *string {
	return w.description
}

// Status returns the workflow's current status.
func (w *Workflow) Status() WorkflowStatus {
	return w.status
}

// Nodes returns a copy of the workflow's nodes.
func (w *Workflow) Nodes() []WorkflowNode {
	if w.nodes == nil {
		return nil
	}

	result := make([]WorkflowNode, len(w.nodes))
	copy(result, w.nodes)

	return result
}

// Edges returns a copy of the workflow's edges.
func (w *Workflow) Edges() []WorkflowEdge {
	if w.edges == nil {
		return nil
	}

	result := make([]WorkflowEdge, len(w.edges))
	copy(result, w.edges)

	return result
}

// Metadata returns a copy of the workflow's metadata.
func (w *Workflow) Metadata() map[string]any {
	if w.metadata == nil {
		return nil
	}

	result := make(map[string]any, len(w.metadata))
	for k, v := range w.metadata {
		result[k] = v
	}

	return result
}

// CreatedAt returns when the workflow was created.
func (w *Workflow) CreatedAt() time.Time {
	return w.createdAt
}

// UpdatedAt returns when the workflow was last updated.
func (w *Workflow) UpdatedAt() time.Time {
	return w.updatedAt
}

// IsActive returns true if the workflow status is active.
func (w *Workflow) IsActive() bool {
	return w.status == WorkflowStatusActive
}

// IsDraft returns true if the workflow status is draft.
func (w *Workflow) IsDraft() bool {
	return w.status == WorkflowStatusDraft
}

// IsInactive returns true if the workflow status is inactive.
func (w *Workflow) IsInactive() bool {
	return w.status == WorkflowStatusInactive
}

// Activate transitions the workflow from draft to active status.
// Only draft workflows can be activated.
func (w *Workflow) Activate() error {
	if w.status != WorkflowStatusDraft {
		return ErrWorkflowCannotActivate
	}

	w.status = WorkflowStatusActive
	w.updatedAt = time.Now().UTC()

	return nil
}

// Deactivate transitions the workflow from active to inactive status.
// Only active workflows can be deactivated.
func (w *Workflow) Deactivate() error {
	if w.status != WorkflowStatusActive {
		return ErrWorkflowCannotDeactivate
	}

	w.status = WorkflowStatusInactive
	w.updatedAt = time.Now().UTC()

	return nil
}

// MoveToDraft transitions the workflow from inactive to draft status.
// Only inactive workflows can be moved back to draft for editing.
func (w *Workflow) MoveToDraft() error {
	if w.status != WorkflowStatusInactive {
		return ErrWorkflowCannotMoveToDraft
	}

	w.status = WorkflowStatusDraft
	w.updatedAt = time.Now().UTC()

	return nil
}

// Update modifies the workflow's name, description, nodes, and edges.
// Only draft workflows can be updated.
func (w *Workflow) Update(name string, description *string, nodes []WorkflowNode, edges []WorkflowEdge) error {
	if w.status != WorkflowStatusDraft {
		return ErrWorkflowCannotUpdate
	}

	if err := validateWorkflowDataForDraft(name, nodes, edges); err != nil {
		return err
	}

	w.name = name
	w.description = description
	w.nodes = cloneNodes(nodes)
	w.edges = cloneEdges(edges)
	w.updatedAt = time.Now().UTC()

	return nil
}

// Clone creates a copy of the workflow with a new name and draft status.
func (w *Workflow) Clone(newName string) (*Workflow, error) {
	// Deep copy nodes
	clonedNodes := make([]WorkflowNode, len(w.nodes))
	for i, node := range w.nodes {
		clonedNodes[i] = node.clone()
	}

	// Deep copy edges
	clonedEdges := make([]WorkflowEdge, len(w.edges))
	for i, edge := range w.edges {
		clonedEdges[i] = edge.clone()
	}

	return NewWorkflow(newName, w.description, clonedNodes, clonedEdges)
}

// SetMetadata sets a metadata key-value pair.
func (w *Workflow) SetMetadata(key string, value any) {
	if w.metadata == nil {
		w.metadata = make(map[string]any)
	}

	w.metadata[key] = value
	w.updatedAt = time.Now().UTC()
}

// cloneNodes creates a defensive copy of a nodes slice.
func cloneNodes(nodes []WorkflowNode) []WorkflowNode {
	if nodes == nil {
		return nil
	}

	result := make([]WorkflowNode, len(nodes))
	for i, node := range nodes {
		result[i] = node.clone()
	}

	return result
}

// cloneEdges creates a defensive copy of an edges slice.
func cloneEdges(edges []WorkflowEdge) []WorkflowEdge {
	if edges == nil {
		return nil
	}

	result := make([]WorkflowEdge, len(edges))
	for i, edge := range edges {
		result[i] = edge.clone()
	}

	return result
}

// cloneMetadata creates a defensive copy of a metadata map.
func cloneMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}

	result := make(map[string]any, len(metadata))
	for k, v := range metadata {
		result[k] = v
	}

	return result
}
