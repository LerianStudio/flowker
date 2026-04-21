// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/google/uuid"
)

// TransformationValidator validates transformation specifications.
// This interface breaks the import cycle between model and transformation packages.
type TransformationValidator interface {
	// ValidateMappings validates that field mappings can be converted to valid transformation specs.
	ValidateMappings(mappings []FieldMapping) error
	// ValidateOperations validates that Kazaam operations form a valid transformation spec.
	ValidateOperations(operations []KazaamOperation) error
}

// ValidateNodesWithCatalog validates executor nodes against the given catalog using each executor's JSON Schema.
// It validates that each executor node has a valid executorId (exists in catalog), a valid providerConfigId
// (valid UUID format, provider exists in catalog), and that the node data passes the executor's JSON Schema.
// It ignores non-executor nodes and returns nil when the catalog is nil.
func ValidateNodesWithCatalog(nodes []WorkflowNode, catalog executor.Catalog) error {
	if catalog == nil {
		return nil
	}

	for _, node := range nodes {
		if node.Type() != NodeTypeExecutor {
			continue
		}

		executorID := executor.ID(node.ExecutorID())
		if executorID == "" {
			return pkg.ValidationError{
				Code:    "WORKFLOW_INVALID_EXECUTOR_CONFIG",
				Message: "executorId is required",
			}
		}

		e, err := catalog.GetExecutor(executorID)
		if err != nil {
			return pkg.EntityNotFoundError{
				EntityType: "Executor",
				Code:       "WORKFLOW_UNKNOWN_EXECUTOR",
				Message:    fmt.Sprintf("unknown executor: %s", executorID),
				Err:        err,
			}
		}

		// Validate providerConfigId is present and valid UUID
		providerConfigID := node.ProviderConfigID()
		if providerConfigID == "" {
			return pkg.ValidationError{
				Code:    constant.ErrWorkflowInvalidProviderConfig.Error(),
				Message: fmt.Sprintf("node %s: providerConfigId is required for executor nodes", node.ID()),
			}
		}

		if _, err := uuid.Parse(providerConfigID); err != nil {
			return pkg.ValidationError{
				Code:    constant.ErrWorkflowInvalidProviderConfig.Error(),
				Message: fmt.Sprintf("node %s: providerConfigId is not a valid UUID: %s", node.ID(), providerConfigID),
			}
		}

		// Cross-validate: executor's provider must match (catalog-level only, DB check at service layer)
		executorProviderID := e.ProviderID()
		if executorProviderID != "" {
			// Verify the provider exists in catalog
			if _, provErr := catalog.GetProvider(executorProviderID); provErr != nil {
				return pkg.EntityNotFoundError{
					EntityType: "Provider",
					Code:       constant.ErrWorkflowInvalidProviderConfig.Error(),
					Message:    fmt.Sprintf("node %s: executor %s references unknown provider %s", node.ID(), executorID, executorProviderID),
					Err:        provErr,
				}
			}
		}

		// Remove workflow-internal fields from the configuration before schema validation.
		// If the remaining config is empty, skip validation — the node is purely declarative
		// (executor + provider config reference) and runtime data comes via input mappings.
		config := copyWithoutNodeInternalKeys(node.Data())

		if len(config) > 0 {
			if err := e.ValidateConfig(config); err != nil {
				return pkg.ValidationError{
					Code:    "WORKFLOW_INVALID_EXECUTOR_CONFIG",
					Message: fmt.Sprintf("invalid executor config: %v", err),
					Err:     err,
				}
			}
		}
	}

	return nil
}

// nodeInternalKeys are workflow-internal fields that should be stripped
// before validating node data against the executor's JSON Schema.
var nodeInternalKeys = map[string]bool{
	"executorId":       true,
	"providerConfigId": true,
	"endpointName":     true,
	"inputMapping":     true,
	"outputMapping":    true,
	"transforms":       true,
	"method":           true,
	"path":             true,
	"timeout":          true,
}

func copyWithoutNodeInternalKeys(data map[string]any) map[string]any {
	if len(data) == 0 {
		return map[string]any{}
	}

	result := make(map[string]any, len(data))

	for k, v := range data {
		if nodeInternalKeys[k] {
			continue
		}

		result[k] = v
	}

	return result
}

// ValidateNodeTransformations validates transformation specs for all executor nodes.
// It checks that inputMapping, outputMapping, and transforms can be converted to valid Kazaam specs.
// If validator is nil, transformation validation is skipped.
func ValidateNodeTransformations(nodes []WorkflowNode, validator TransformationValidator) error {
	if validator == nil {
		return nil
	}

	for _, node := range nodes {
		if node.Type() != NodeTypeExecutor {
			continue
		}

		nodeID := node.ID()

		// Validate inputMapping transformations
		if inputMapping := node.InputMapping(); len(inputMapping) > 0 {
			if err := validator.ValidateMappings(inputMapping); err != nil {
				return pkg.ValidationError{
					Code:    constant.ErrWorkflowInvalidInputMapping.Error(),
					Message: fmt.Sprintf("node %s: invalid inputMapping: %v", nodeID, err),
					Err:     err,
				}
			}
		}

		// Validate outputMapping transformations
		if outputMapping := node.OutputMapping(); len(outputMapping) > 0 {
			if err := validator.ValidateMappings(outputMapping); err != nil {
				return pkg.ValidationError{
					Code:    constant.ErrWorkflowInvalidOutputMapping.Error(),
					Message: fmt.Sprintf("node %s: invalid outputMapping: %v", nodeID, err),
					Err:     err,
				}
			}
		}

		// Validate transforms (raw Kazaam operations)
		if transforms := node.Transforms(); len(transforms) > 0 {
			if err := validator.ValidateOperations(transforms); err != nil {
				return pkg.ValidationError{
					Code:    constant.ErrWorkflowInvalidTransforms.Error(),
					Message: fmt.Sprintf("node %s: invalid transforms: %v", nodeID, err),
					Err:     err,
				}
			}
		}
	}

	return nil
}
