// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import "errors"

// Workflow query errors.
var (
	ErrGetWorkflowNilRepo       = errors.New("workflow repository cannot be nil")
	ErrGetWorkflowByNameNilRepo = errors.New("workflow repository cannot be nil")
	ErrListWorkflowsNilRepo     = errors.New("workflow repository cannot be nil")
)

// Execution query errors.
var (
	ErrGetExecutionNilRepo        = errors.New("execution repository cannot be nil")
	ErrGetExecutionResultsNilRepo = errors.New("execution repository cannot be nil")
	ErrListExecutionsNilRepo      = errors.New("execution repository cannot be nil")
)

// Executor configuration query errors.
var (
	ErrGetExecutorConfigNilRepo       = errors.New("executor configuration repository cannot be nil")
	ErrGetExecutorConfigByNameNilRepo = errors.New("executor configuration repository cannot be nil")
	ErrListExecutorConfigsNilRepo     = errors.New("executor configuration repository cannot be nil")
	ErrExistsExecutorConfigNilRepo    = errors.New("executor configuration repository cannot be nil")
)

// Provider configuration query errors.
var (
	ErrGetProviderConfigNilRepo   = errors.New("provider configuration repository cannot be nil")
	ErrListProviderConfigsNilRepo = errors.New("provider configuration repository cannot be nil")
)

// Dashboard query errors.
var (
	ErrDashboardNilRepo = errors.New("dashboard repository cannot be nil")
)

// Audit query errors.
var (
	ErrAuditReadNilRepo = errors.New("audit read repository cannot be nil")
)
