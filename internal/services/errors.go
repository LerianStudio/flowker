// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package services

import (
	"errors"

	"github.com/LerianStudio/flowker/pkg/constant"
)

// Common database errors
var (
	// ErrDatabaseItemNotFound is thrown when an item was not found in the database.
	ErrDatabaseItemNotFound = errors.New("errDatabaseItemNotFound")
)

// Workflow-specific errors with FLK- prefix per api-design.md
var (
	// ErrWorkflowNotFound is returned when a workflow is not found.
	ErrWorkflowNotFound = constant.ErrWorkflowNotFound

	// ErrWorkflowDuplicateName is returned when a workflow with the same name already exists.
	ErrWorkflowDuplicateName = constant.ErrWorkflowDuplicateName

	// ErrWorkflowInvalidStatus is returned when a workflow status transition is invalid.
	ErrWorkflowInvalidStatus = constant.ErrWorkflowInvalidStatus

	// ErrWorkflowCannotModify is returned when trying to modify a non-draft workflow.
	ErrWorkflowCannotModify = constant.ErrWorkflowCannotModify

	// ErrWorkflowExecutorNotFound is returned when a referenced executor doesn't exist.
	ErrWorkflowExecutorNotFound = constant.ErrWorkflowExecutorNotFound

	// ErrWorkflowInvalidCondition is returned when a conditional expression is invalid.
	ErrWorkflowInvalidCondition = constant.ErrWorkflowInvalidCondition
)

// Facade constructor sentinel errors
var (
	ErrWorkflowServiceNilDependency       = errors.New("workflow service: required dependency cannot be nil")
	ErrExecutorConfigServiceNilDependency = errors.New("executor configuration service: required dependency cannot be nil")
	ErrExecutionServiceNilDependency      = errors.New("execution service: required dependency cannot be nil")
	ErrProviderConfigServiceNilDependency = errors.New("provider configuration service: required dependency cannot be nil")
	ErrDashboardServiceNilDependency      = errors.New("dashboard service: required dependency cannot be nil")
	ErrAuditServiceNilDependency          = errors.New("audit service: required dependency cannot be nil")
)

// ExecutorConfiguration-specific errors with FLK- prefix per api-design.md
var (
	// ErrExecutorConfigNotFound is returned when an executor configuration is not found.
	ErrExecutorConfigNotFound = constant.ErrExecutorConfigNotFound

	// ErrExecutorConfigDuplicateName is returned when an executor configuration with the same name already exists.
	ErrExecutorConfigDuplicateName = constant.ErrExecutorConfigDuplicateName

	// ErrExecutorConfigCannotModify is returned when trying to modify an executor configuration in an invalid state.
	ErrExecutorConfigCannotModify = constant.ErrExecutorConfigCannotModify
)
