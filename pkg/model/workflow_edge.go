// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
)

// Edge validation errors.
var (
	ErrEdgeIDRequired = pkg.ValidationError{
		Code:    constant.ErrEdgeIDRequired.Error(),
		Message: "edge id is required",
	}
	ErrEdgeSourceRequired = pkg.ValidationError{
		Code:    constant.ErrEdgeSourceRequired.Error(),
		Message: "edge source is required",
	}
	ErrEdgeTargetRequired = pkg.ValidationError{
		Code:    constant.ErrEdgeTargetRequired.Error(),
		Message: "edge target is required",
	}
)

// WorkflowEdge represents a connection between two nodes in the workflow graph.
// This is a value object embedded in Workflow.
type WorkflowEdge struct {
	id           string
	source       string
	target       string
	sourceHandle *string
	condition    *string
	label        *string
}

// NewWorkflowEdge creates a new WorkflowEdge with validation.
// Returns an error if id, source, or target are empty.
func NewWorkflowEdge(id, source, target string) (WorkflowEdge, error) {
	if id == "" {
		return WorkflowEdge{}, ErrEdgeIDRequired
	}

	if source == "" {
		return WorkflowEdge{}, ErrEdgeSourceRequired
	}

	if target == "" {
		return WorkflowEdge{}, ErrEdgeTargetRequired
	}

	return WorkflowEdge{
		id:     id,
		source: source,
		target: target,
	}, nil
}

// NewWorkflowEdgeFromDB reconstructs a WorkflowEdge from database values.
func NewWorkflowEdgeFromDB(
	id, source, target string,
	sourceHandle, condition, label *string,
) WorkflowEdge {
	return WorkflowEdge{
		id:           id,
		source:       source,
		target:       target,
		sourceHandle: sourceHandle,
		condition:    condition,
		label:        label,
	}
}

// ID returns the edge's identifier.
func (e WorkflowEdge) ID() string {
	return e.id
}

// Source returns the source node ID.
func (e WorkflowEdge) Source() string {
	return e.source
}

// Target returns the target node ID.
func (e WorkflowEdge) Target() string {
	return e.target
}

// SourceHandle returns the output handle on the source node (e.g., "true", "false" for conditionals).
func (e WorkflowEdge) SourceHandle() *string {
	return e.sourceHandle
}

// Condition returns the optional condition for edge traversal.
func (e WorkflowEdge) Condition() *string {
	return e.condition
}

// Label returns the display label for the edge.
func (e WorkflowEdge) Label() *string {
	return e.label
}

// WithSourceHandle sets the source handle and returns the modified edge.
func (e *WorkflowEdge) WithSourceHandle(handle string) *WorkflowEdge {
	e.sourceHandle = &handle

	return e
}

// WithCondition sets the condition and returns the modified edge.
func (e *WorkflowEdge) WithCondition(condition string) *WorkflowEdge {
	e.condition = &condition

	return e
}

// WithLabel sets the label and returns the modified edge.
func (e *WorkflowEdge) WithLabel(label string) *WorkflowEdge {
	e.label = &label

	return e
}

// clone creates a copy of the edge.
func (e WorkflowEdge) clone() WorkflowEdge {
	var sourceHandleCopy, conditionCopy, labelCopy *string

	if e.sourceHandle != nil {
		val := *e.sourceHandle
		sourceHandleCopy = &val
	}

	if e.condition != nil {
		val := *e.condition
		conditionCopy = &val
	}

	if e.label != nil {
		val := *e.label
		labelCopy = &val
	}

	return WorkflowEdge{
		id:           e.id,
		source:       e.source,
		target:       e.target,
		sourceHandle: sourceHandleCopy,
		condition:    conditionCopy,
		label:        labelCopy,
	}
}
