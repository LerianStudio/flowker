// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package command contains command services for write operations.
package command

import "errors"

// Workflow command sentinel errors.
var (
	ErrCreateWorkflowNilRepo      = errors.New("workflow repository cannot be nil")
	ErrCreateWorkflowNilInput     = errors.New("create workflow input cannot be nil")
	ErrUpdateWorkflowNilRepo      = errors.New("workflow repository cannot be nil")
	ErrUpdateWorkflowNilInput     = errors.New("update workflow input cannot be nil")
	ErrCloneWorkflowNilRepo       = errors.New("workflow repository cannot be nil")
	ErrCloneWorkflowNilInput      = errors.New("clone workflow input cannot be nil")
	ErrActivateWorkflowNilRepo    = errors.New("workflow repository cannot be nil")
	ErrDeactivateWorkflowNilRepo  = errors.New("workflow repository cannot be nil")
	ErrMoveToDraftWorkflowNilRepo = errors.New("workflow repository cannot be nil")
	ErrDeleteWorkflowNilRepo      = errors.New("workflow repository cannot be nil")
)

// Workflow from template command sentinel errors.
var (
	ErrCreateWorkflowFromTemplateNilCatalog   = errors.New("executor catalog cannot be nil")
	ErrCreateWorkflowFromTemplateNilCreateCmd = errors.New("create workflow command cannot be nil")
	ErrCreateWorkflowFromTemplateNilInput     = errors.New("create workflow from template input cannot be nil")
)

// Execution command sentinel errors.
var (
	ErrExecuteWorkflowNilRepo               = errors.New("execution repository cannot be nil")
	ErrExecuteWorkflowNilWorkflowRepo       = errors.New("workflow repository cannot be nil")
	ErrExecuteWorkflowNilExecConfigRepo     = errors.New("executor config repository cannot be nil")
	ErrExecuteWorkflowNilCatalog            = errors.New("executor catalog cannot be nil")
	ErrExecuteWorkflowNilCircuitBreaker     = errors.New("circuit breaker manager cannot be nil")
	ErrExecuteWorkflowNilCondEvaluator      = errors.New("condition evaluator cannot be nil")
	ErrExecuteWorkflowNilInput              = errors.New("execute workflow input cannot be nil")
	ErrExecuteWorkflowNilTransformSvc       = errors.New("transformation service cannot be nil")
	ErrExecuteWorkflowNilProviderConfigRepo = errors.New("provider config repository cannot be nil")
)

// Executor configuration command sentinel errors.
var (
	ErrCreateExecutorConfigNilRepo     = errors.New("executor configuration repository cannot be nil")
	ErrCreateExecutorConfigNilInput    = errors.New("create executor configuration input cannot be nil")
	ErrUpdateExecutorConfigNilRepo     = errors.New("executor configuration repository cannot be nil")
	ErrUpdateExecutorConfigNilInput    = errors.New("update executor configuration input cannot be nil")
	ErrMarkConfiguredNilRepo           = errors.New("executor configuration repository cannot be nil")
	ErrMarkTestedNilRepo               = errors.New("executor configuration repository cannot be nil")
	ErrActivateExecutorConfigNilRepo   = errors.New("executor configuration repository cannot be nil")
	ErrDisableExecutorConfigNilRepo    = errors.New("executor configuration repository cannot be nil")
	ErrEnableExecutorConfigNilRepo     = errors.New("executor configuration repository cannot be nil")
	ErrDeleteExecutorConfigNilRepo     = errors.New("executor configuration repository cannot be nil")
	ErrTestExecutorConnectivityNilRepo = errors.New("executor configuration repository cannot be nil")
)

// Audit command sentinel errors.
var (
	ErrAuditWriterNilRepo = errors.New("audit write repository cannot be nil")
)

// AuditWriter nil-check sentinel errors (audit is mandatory for all commands).
var (
	ErrActivateWorkflowNilAuditWriter     = errors.New("audit writer cannot be nil")
	ErrCreateWorkflowNilAuditWriter       = errors.New("audit writer cannot be nil")
	ErrMoveToDraftWorkflowNilAuditWriter  = errors.New("audit writer cannot be nil")
	ErrDeactivateWorkflowNilAuditWriter   = errors.New("audit writer cannot be nil")
	ErrUpdateWorkflowNilAuditWriter       = errors.New("audit writer cannot be nil")
	ErrDeleteWorkflowNilAuditWriter       = errors.New("audit writer cannot be nil")
	ErrCreateProviderConfigNilAuditWriter = errors.New("audit writer cannot be nil")
	ErrUpdateProviderConfigNilAuditWriter = errors.New("audit writer cannot be nil")
	ErrDeleteProviderConfigNilAuditWriter = errors.New("audit writer cannot be nil")
	ErrExecuteWorkflowNilAuditWriter      = errors.New("audit writer cannot be nil")
)

// Provider configuration command sentinel errors.
var (
	ErrCreateProviderConfigNilRepo           = errors.New("provider configuration repository cannot be nil")
	ErrCreateProviderConfigNilCatalog        = errors.New("provider configuration catalog cannot be nil")
	ErrCreateProviderConfigNilInput          = errors.New("create provider configuration input cannot be nil")
	ErrUpdateProviderConfigNilRepo           = errors.New("provider configuration repository cannot be nil")
	ErrUpdateProviderConfigNilCatalog        = errors.New("provider configuration catalog cannot be nil")
	ErrUpdateProviderConfigNilInput          = errors.New("update provider configuration input cannot be nil")
	ErrDisableProviderConfigNilRepo          = errors.New("provider configuration repository cannot be nil")
	ErrEnableProviderConfigNilRepo           = errors.New("provider configuration repository cannot be nil")
	ErrDeleteProviderConfigNilRepo           = errors.New("provider configuration repository cannot be nil")
	ErrTestProviderConfigConnectivityNilRepo = errors.New("provider configuration repository cannot be nil")
)
