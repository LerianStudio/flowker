// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package constant

import "errors"

// Generic / parsing
var (
	ErrInvalidRequestBody = errors.New("FLK-0001")
	ErrInvalidID          = errors.New("FLK-0002")
	ErrInternalServer     = errors.New("FLK-9999")
)

// Request validation / pagination / metadata
var (
	ErrUnexpectedFieldsInTheRequest = errors.New("FLK-0300")
	ErrMissingFieldsInRequest       = errors.New("FLK-0301")
	ErrBadRequest                   = errors.New("FLK-0302")
	ErrCalculationFieldType         = errors.New("FLK-0303")
	ErrInvalidQueryParameter        = errors.New("FLK-0304")
	ErrInvalidDateFormat            = errors.New("FLK-0305")
	ErrInvalidFinalDate             = errors.New("FLK-0306")
	ErrDateRangeExceedsLimit        = errors.New("FLK-0307")
	ErrInvalidDateRange             = errors.New("FLK-0308")
	ErrPaginationLimitExceeded      = errors.New("FLK-0309")
	ErrInvalidSortOrder             = errors.New("FLK-0310")
	ErrInvalidPathParameter         = errors.New("FLK-0311")
	ErrMetadataKeyLengthExceeded    = errors.New("FLK-0312")
	ErrMetadataValueLengthExceeded  = errors.New("FLK-0313")
	ErrInvalidMetadataNesting       = errors.New("FLK-0314")
	ErrMetadataEntriesExceeded      = errors.New("FLK-0315")
)

// Generic entity/business
var (
	ErrEntityNotFound          = errors.New("FLK-0400")
	ErrActionNotPermitted      = errors.New("FLK-0401")
	ErrParentExampleIDNotFound = errors.New("FLK-0402")
)

// Concurrency / state conflicts
var (
	ErrConflictStateChanged = errors.New("FLK-0350")
)

// Webhook errors
var (
	ErrWebhookPathAlreadyRegistered = errors.New("FLK-0360")
	ErrWebhookRouteNotFound         = errors.New("FLK-0361")
	ErrWebhookTokenInvalid          = errors.New("FLK-0362")
	ErrWebhookPayloadTooLarge       = errors.New("FLK-0363")
)

// Workflow domain (WF)
var (
	ErrWorkflowNotFound         = errors.New("FLK-0100")
	ErrWorkflowDuplicateName    = errors.New("FLK-0101")
	ErrWorkflowInvalidStatus    = errors.New("FLK-0102")
	ErrWorkflowCannotModify     = errors.New("FLK-0103")
	ErrWorkflowExecutorNotFound = errors.New("FLK-0104")
	ErrWorkflowInvalidCondition = errors.New("FLK-0105")

	// Workflow validation errors
	ErrWorkflowNameRequired   = errors.New("FLK-0110")
	ErrWorkflowNameTooLong    = errors.New("FLK-0111")
	ErrWorkflowNodesRequired  = errors.New("FLK-0112")
	ErrWorkflowTooManyNodes   = errors.New("FLK-0113")
	ErrWorkflowTooManyEdges   = errors.New("FLK-0114")
	ErrWorkflowInvalidEdgeRef = errors.New("FLK-0115")
	ErrWorkflowNoTrigger      = errors.New("FLK-0116")

	// Workflow node validation errors (FLK-0120 to FLK-0129)
	ErrNodeIDRequired   = errors.New("FLK-0120")
	ErrNodeTypeRequired = errors.New("FLK-0121")

	// Workflow edge validation errors (FLK-0130 to FLK-0139)
	ErrEdgeIDRequired     = errors.New("FLK-0130")
	ErrEdgeSourceRequired = errors.New("FLK-0131")
	ErrEdgeTargetRequired = errors.New("FLK-0132")

	// Workflow transformation validation errors (FLK-0140 to FLK-0149)
	ErrWorkflowInvalidInputMapping  = errors.New("FLK-0140")
	ErrWorkflowInvalidOutputMapping = errors.New("FLK-0141")
	ErrWorkflowInvalidTransforms    = errors.New("FLK-0142")

	// Workflow provider config validation errors (FLK-0150 to FLK-0159)
	ErrWorkflowInvalidProviderConfig  = errors.New("FLK-0150")
	ErrWorkflowProviderConfigMismatch = errors.New("FLK-0151")
)

// Catalog / Executors / Triggers / Runners
var (
	ErrExecutorNotFound      = errors.New("FLK-0200")
	ErrExecutorInvalidConfig = errors.New("FLK-0201")

	ErrTriggerNotFound      = errors.New("FLK-0210")
	ErrTriggerInvalidConfig = errors.New("FLK-0211")

	ErrRunnerNotFound = errors.New("FLK-0220")

	ErrProviderNotFound      = errors.New("FLK-0230")
	ErrProviderDuplicate     = errors.New("FLK-0231")
	ErrProviderInvalidConfig = errors.New("FLK-0232")
	ErrProviderNilExecutor   = errors.New("FLK-0233")

	ErrTemplateNotFound      = errors.New("FLK-0240")
	ErrTemplateDuplicate     = errors.New("FLK-0241")
	ErrTemplateInvalidParams = errors.New("FLK-0242")
	ErrTemplateBuildFailed   = errors.New("FLK-0243")
)

// Provider Configuration (PC)
var (
	ErrProviderConfigNotFound      = errors.New("FLK-0290")
	ErrProviderConfigDuplicateName = errors.New("FLK-0291")
	ErrProviderConfigCannotModify  = errors.New("FLK-0292")
	ErrProviderConfigInvalidSchema = errors.New("FLK-0293")
	ErrProviderNotFoundInCatalog   = errors.New("FLK-0294")

	// Provider Configuration validation errors (FLK-0295 to FLK-0299)
	ErrProviderConfigNameRequired       = errors.New("FLK-0295")
	ErrProviderConfigNameTooLong        = errors.New("FLK-0296")
	ErrProviderConfigProviderIDRequired = errors.New("FLK-0297")
	ErrProviderConfigConfigRequired     = errors.New("FLK-0298")
	ErrProviderConfigDescriptionTooLong = errors.New("FLK-0299")
	ErrProviderConfigIDRequired         = errors.New("FLK-0300")

	// Provider Configuration Connectivity Testing errors (FLK-0340 to FLK-0349)
	ErrProviderConfigMissingBaseURL = errors.New("FLK-0340")
	ErrProviderConfigSSRFBlocked    = errors.New("FLK-0341")
)

// Executor Configuration (EC)
var (
	ErrExecutorConfigNotFound      = errors.New("FLK-0250")
	ErrExecutorConfigDuplicateName = errors.New("FLK-0251")
	ErrExecutorConfigCannotModify  = errors.New("FLK-0252")

	// Executor Configuration validation errors (FLK-0260 to FLK-0279)
	ErrExecutorConfigNameRequired           = errors.New("FLK-0260")
	ErrExecutorConfigNameTooLong            = errors.New("FLK-0261")
	ErrExecutorConfigBaseURLRequired        = errors.New("FLK-0262")
	ErrExecutorConfigBaseURLInvalid         = errors.New("FLK-0263")
	ErrExecutorConfigBaseURLTooLong         = errors.New("FLK-0264")
	ErrExecutorConfigEndpointsRequired      = errors.New("FLK-0265")
	ErrExecutorConfigAuthRequired           = errors.New("FLK-0266")
	ErrExecutorConfigDescriptionTooLong     = errors.New("FLK-0267")
	ErrExecutorConfigAuthTypeRequired       = errors.New("FLK-0268")
	ErrExecutorConfigAuthTypeInvalid        = errors.New("FLK-0269")
	ErrExecutorConfigEndpointNameRequired   = errors.New("FLK-0270")
	ErrExecutorConfigEndpointPathRequired   = errors.New("FLK-0271")
	ErrExecutorConfigEndpointMethodRequired = errors.New("FLK-0272")
	ErrExecutorConfigEndpointMethodInvalid  = errors.New("FLK-0273")

	// Executor Connectivity Testing errors (FLK-0280 to FLK-0289)
	ErrExecutorConnectivityTestFailed = errors.New("FLK-0280")
	ErrExecutorEndpointUnreachable    = errors.New("FLK-0281")
	ErrExecutorAuthTestFailed         = errors.New("FLK-0282")
	ErrExecutorTransformTestFailed    = errors.New("FLK-0283")
)

// Workflow Execution (EX)
var (
	ErrExecutionNotFound      = errors.New("FLK-0500")
	ErrExecutionNotActive     = errors.New("FLK-0501")
	ErrExecutionInProgress    = errors.New("FLK-0502")
	ErrExecutionTimeout       = errors.New("FLK-0503")
	ErrExecutionNodeFailed    = errors.New("FLK-0504")
	ErrExecutionDuplicate     = errors.New("FLK-0505")
	ErrExecutionInputTooLarge = errors.New("FLK-0506")
	ErrExecutionCircuitOpen   = errors.New("FLK-0507")
	ErrExecutionCycleDetected = errors.New("FLK-0508")
	ErrMissingIdempotencyKey  = errors.New("FLK-0509")
)

// Audit Trail (AT)
var (
	ErrAuditEntryNotFound            = errors.New("FLK-0600")
	ErrAuditEntryInvalidEventType    = errors.New("FLK-0601")
	ErrAuditEntryInvalidAction       = errors.New("FLK-0602")
	ErrAuditEntryInvalidResult       = errors.New("FLK-0603")
	ErrAuditEntryResourceIDRequired  = errors.New("FLK-0604")
	ErrAuditEntryInvalidResourceType = errors.New("FLK-0605")
	ErrAuditEntryActorIDRequired     = errors.New("FLK-0606")
	ErrAuditEntryInvalidActorType    = errors.New("FLK-0607")
	ErrAuditEntryResourceIDTooLong   = errors.New("FLK-0608")
	ErrAuditEntryActorIDTooLong      = errors.New("FLK-0609")
	ErrAuditInvalidCursor            = errors.New("FLK-0610")
	ErrAuditInvalidFilters           = errors.New("FLK-0611")
	ErrAuditDatabaseNotConnected     = errors.New("FLK-0612")
)
