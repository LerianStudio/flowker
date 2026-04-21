// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
)

// Node validation errors.
var (
	ErrNodeIDRequired = pkg.ValidationError{
		Code:    constant.ErrNodeIDRequired.Error(),
		Message: "node id is required",
	}
	ErrNodeTypeRequired = pkg.ValidationError{
		Code:    constant.ErrNodeTypeRequired.Error(),
		Message: "node type is required",
	}
)

// NodeType represents the type of workflow node.
type NodeType string

const (
	// NodeTypeTrigger indicates an entry point of the workflow.
	NodeTypeTrigger NodeType = "trigger"
	// NodeTypeExecutor indicates a node that calls an external executor.
	NodeTypeExecutor NodeType = "executor"
	// NodeTypeConditional indicates a node with conditional branching logic.
	NodeTypeConditional NodeType = "conditional"
	// NodeTypeAction indicates a generic action node.
	NodeTypeAction NodeType = "action"
)

// DefaultNodeTimeout is the default timeout for executor nodes in seconds.
const DefaultNodeTimeout = 30

// cloneAnyMap creates a shallow copy of a map to prevent external mutation.
func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}

	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}

	return out
}

// Position represents the canvas position for visual rendering.
type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// FieldMapping defines how workflow data maps to executor data.
// Used in executor nodes to transform data between workflow and executor formats.
type FieldMapping struct {
	Source         string                `json:"source"`                   // JSONPath source (e.g., "workflow.customer.cpf")
	Target         string                `json:"target"`                   // JSONPath target (e.g., "provider.document")
	Transformation *TransformationConfig `json:"transformation,omitempty"` // Optional transformation to apply
	Required       bool                  `json:"required,omitempty"`       // If true, source must exist
}

// TransformationConfig defines a transformation to apply during field mapping.
type TransformationConfig struct {
	Type   string         `json:"type"`   // Transformation type (e.g., "remove_characters", "to_uppercase")
	Config map[string]any `json:"config"` // Type-specific configuration
}

// KazaamOperation represents a Kazaam transformation operation.
// This is the format expected by Kazaam for JSON transformations.
type KazaamOperation struct {
	Operation string         `json:"operation"`         // Operation type (shift, concat, remove_characters, etc.)
	Spec      map[string]any `json:"spec"`              // Operation-specific specification
	Require   bool           `json:"require,omitempty"` // If true, paths must exist
}

// WorkflowNode represents a single node in the workflow graph.
// This is a value object embedded in Workflow.
type WorkflowNode struct {
	id       string
	nodeType NodeType
	name     *string
	position Position
	data     map[string]any
}

// NewWorkflowNode creates a new WorkflowNode with validation.
// Returns an error if id or nodeType are empty.
func NewWorkflowNode(id string, nodeType NodeType, name *string, position Position, data map[string]any) (WorkflowNode, error) {
	if id == "" {
		return WorkflowNode{}, ErrNodeIDRequired
	}

	if nodeType == "" {
		return WorkflowNode{}, ErrNodeTypeRequired
	}

	return WorkflowNode{
		id:       id,
		nodeType: nodeType,
		name:     name,
		position: position,
		data:     data,
	}, nil
}

// NewWorkflowNodeFromDB reconstructs a WorkflowNode from database values.
func NewWorkflowNodeFromDB(
	id string,
	nodeType NodeType,
	name *string,
	position Position,
	data map[string]any,
) WorkflowNode {
	return WorkflowNode{
		id:       id,
		nodeType: nodeType,
		name:     name,
		position: position,
		data:     data,
	}
}

// ID returns the node's identifier.
func (n WorkflowNode) ID() string {
	return n.id
}

// Type returns the node's type.
func (n WorkflowNode) Type() NodeType {
	return n.nodeType
}

// Name returns the node's display name.
func (n WorkflowNode) Name() *string {
	return n.name
}

// Position returns the node's canvas position.
func (n WorkflowNode) Position() Position {
	return n.position
}

// Data returns a copy of the node's configuration data.
func (n WorkflowNode) Data() map[string]any {
	if n.data == nil {
		return nil
	}

	result := make(map[string]any, len(n.data))
	for k, v := range n.data {
		result[k] = v
	}

	return result
}

// ExecutorID returns the executor ID if this is an executor node.
// Returns empty string if not found or not an executor node.
func (n WorkflowNode) ExecutorID() string {
	if n.nodeType != NodeTypeExecutor {
		return ""
	}

	if n.data == nil {
		return ""
	}

	if executorID, ok := n.data["executorId"].(string); ok {
		return executorID
	}

	return ""
}

// TriggerType returns the trigger type if this is a trigger node.
// Returns empty string if not found or not a trigger node.
func (n WorkflowNode) TriggerType() string {
	if n.nodeType != NodeTypeTrigger {
		return ""
	}

	if n.data == nil {
		return ""
	}

	if triggerType, ok := n.data["triggerType"].(string); ok {
		return triggerType
	}

	return ""
}

// Condition returns the condition expression if this is a conditional node.
// Returns empty string if not found or not a conditional node.
func (n WorkflowNode) Condition() string {
	if n.nodeType != NodeTypeConditional {
		return ""
	}

	if n.data == nil {
		return ""
	}

	if condition, ok := n.data["condition"].(string); ok {
		return condition
	}

	return ""
}

// clone creates a copy of the node.
func (n WorkflowNode) clone() WorkflowNode {
	var dataCopy map[string]any

	if n.data != nil {
		dataCopy = make(map[string]any)
		for k, v := range n.data {
			dataCopy[k] = v
		}
	}

	var nameCopy *string

	if n.name != nil {
		nameVal := *n.name
		nameCopy = &nameVal
	}

	return WorkflowNode{
		id:       n.id,
		nodeType: n.nodeType,
		name:     nameCopy,
		position: n.position,
		data:     dataCopy,
	}
}

// ProviderConfigID returns the provider configuration UUID for executor nodes.
// This is the UUID of the ProviderConfiguration entity used at runtime.
func (n WorkflowNode) ProviderConfigID() string {
	if n.nodeType != NodeTypeExecutor {
		return ""
	}

	if n.data == nil {
		return ""
	}

	if configID, ok := n.data["providerConfigId"].(string); ok {
		return configID
	}

	return ""
}

// EndpointName returns the endpoint name for executor nodes.
// Returns empty string if not found or not an executor node.
func (n WorkflowNode) EndpointName() string {
	if n.nodeType != NodeTypeExecutor {
		return ""
	}

	if n.data == nil {
		return ""
	}

	if endpointName, ok := n.data["endpointName"].(string); ok {
		return endpointName
	}

	return ""
}

// InputMapping returns the input field mappings for executor nodes.
// Returns nil if not found or not an executor node.
func (n WorkflowNode) InputMapping() []FieldMapping {
	if n.nodeType != NodeTypeExecutor {
		return nil
	}

	if n.data == nil {
		return nil
	}

	return parseFieldMappings(n.data["inputMapping"])
}

// OutputMapping returns the output field mappings for executor nodes.
// Returns nil if not found or not an executor node.
func (n WorkflowNode) OutputMapping() []FieldMapping {
	if n.nodeType != NodeTypeExecutor {
		return nil
	}

	if n.data == nil {
		return nil
	}

	return parseFieldMappings(n.data["outputMapping"])
}

// Transforms returns the Kazaam transformation operations for executor nodes.
// Returns nil if not found or not an executor node.
func (n WorkflowNode) Transforms() []KazaamOperation {
	if n.nodeType != NodeTypeExecutor {
		return nil
	}

	if n.data == nil {
		return nil
	}

	return parseKazaamOperations(n.data["transforms"])
}

// parseFieldMappings converts a generic interface to []FieldMapping.
func parseFieldMappings(data any) []FieldMapping {
	if data == nil {
		return nil
	}

	slice, ok := data.([]any)
	if !ok {
		return nil
	}

	mappings := make([]FieldMapping, 0, len(slice))

	for _, item := range slice {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		mapping := FieldMapping{}

		if source, ok := m["source"].(string); ok {
			mapping.Source = source
		}

		if target, ok := m["target"].(string); ok {
			mapping.Target = target
		}

		if required, ok := m["required"].(bool); ok {
			mapping.Required = required
		}

		if transform, ok := m["transformation"].(map[string]any); ok {
			mapping.Transformation = &TransformationConfig{}
			if t, ok := transform["type"].(string); ok {
				mapping.Transformation.Type = t
			}

			if c, ok := transform["config"].(map[string]any); ok {
				mapping.Transformation.Config = cloneAnyMap(c)
			}
		}

		mappings = append(mappings, mapping)
	}

	return mappings
}

// parseKazaamOperations converts a generic interface to []KazaamOperation.
func parseKazaamOperations(data any) []KazaamOperation {
	if data == nil {
		return nil
	}

	slice, ok := data.([]any)
	if !ok {
		return nil
	}

	operations := make([]KazaamOperation, 0, len(slice))

	for _, item := range slice {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		op := KazaamOperation{}

		if operation, ok := m["operation"].(string); ok {
			op.Operation = operation
		}

		if spec, ok := m["spec"].(map[string]any); ok {
			op.Spec = cloneAnyMap(spec)
		}

		if require, ok := m["require"].(bool); ok {
			op.Require = require
		}

		operations = append(operations, op)
	}

	return operations
}
