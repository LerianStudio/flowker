// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package model contains domain entities and DTOs for Flowker.
package model

import (
	"strings"
	"time"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/google/uuid"
)

// AuditEventType represents the type of audit event.
type AuditEventType string

const (
	// Workflow lifecycle events
	AuditEventWorkflowCreated     AuditEventType = "WORKFLOW_CREATED"
	AuditEventWorkflowUpdated     AuditEventType = "WORKFLOW_UPDATED"
	AuditEventWorkflowActivated   AuditEventType = "WORKFLOW_ACTIVATED"
	AuditEventWorkflowDeactivated AuditEventType = "WORKFLOW_DEACTIVATED"
	AuditEventWorkflowDrafted     AuditEventType = "WORKFLOW_DRAFTED"
	AuditEventWorkflowDeleted     AuditEventType = "WORKFLOW_DELETED"

	// Execution lifecycle events
	AuditEventExecutionStarted   AuditEventType = "EXECUTION_STARTED"
	AuditEventExecutionCompleted AuditEventType = "EXECUTION_COMPLETED"
	AuditEventExecutionFailed    AuditEventType = "EXECUTION_FAILED"

	// Provider call events
	AuditEventProviderCallStarted   AuditEventType = "PROVIDER_CALL_STARTED"
	AuditEventProviderCallCompleted AuditEventType = "PROVIDER_CALL_COMPLETED"
	AuditEventProviderCallFailed    AuditEventType = "PROVIDER_CALL_FAILED"

	// Provider config events
	AuditEventProviderConfigCreated AuditEventType = "PROVIDER_CONFIG_CREATED"
	AuditEventProviderConfigUpdated AuditEventType = "PROVIDER_CONFIG_UPDATED"
	AuditEventProviderConfigDeleted AuditEventType = "PROVIDER_CONFIG_DELETED"
)

// IsValid checks if the AuditEventType is a valid enum value.
func (t AuditEventType) IsValid() bool {
	switch t {
	case AuditEventWorkflowCreated, AuditEventWorkflowUpdated, AuditEventWorkflowActivated,
		AuditEventWorkflowDeactivated, AuditEventWorkflowDrafted, AuditEventWorkflowDeleted,
		AuditEventExecutionStarted, AuditEventExecutionCompleted, AuditEventExecutionFailed,
		AuditEventProviderCallStarted, AuditEventProviderCallCompleted, AuditEventProviderCallFailed,
		AuditEventProviderConfigCreated, AuditEventProviderConfigUpdated, AuditEventProviderConfigDeleted:
		return true
	default:
		return false
	}
}

// AuditAction represents the action performed.
type AuditAction string

const (
	AuditActionCreate     AuditAction = "CREATE"
	AuditActionUpdate     AuditAction = "UPDATE"
	AuditActionDelete     AuditAction = "DELETE"
	AuditActionActivate   AuditAction = "ACTIVATE"
	AuditActionDeactivate AuditAction = "DEACTIVATE"
	AuditActionDraft      AuditAction = "DRAFT"
	AuditActionExecute    AuditAction = "EXECUTE"
)

// IsValid checks if the AuditAction is a valid enum value.
func (a AuditAction) IsValid() bool {
	switch a {
	case AuditActionCreate, AuditActionUpdate, AuditActionDelete,
		AuditActionActivate, AuditActionDeactivate, AuditActionDraft, AuditActionExecute:
		return true
	default:
		return false
	}
}

// AuditResult represents the result of an action.
type AuditResult string

const (
	AuditResultSuccess AuditResult = "SUCCESS"
	AuditResultFailed  AuditResult = "FAILED"
)

// IsValid checks if the AuditResult is a valid enum value.
func (r AuditResult) IsValid() bool {
	switch r {
	case AuditResultSuccess, AuditResultFailed:
		return true
	default:
		return false
	}
}

// AuditResourceType represents the type of resource affected.
type AuditResourceType string

const (
	AuditResourceTypeWorkflow       AuditResourceType = "workflow"
	AuditResourceTypeExecution      AuditResourceType = "execution"
	AuditResourceTypeProviderConfig AuditResourceType = "provider_config"
)

// IsValid checks if the AuditResourceType is a valid enum value.
func (r AuditResourceType) IsValid() bool {
	switch r {
	case AuditResourceTypeWorkflow, AuditResourceTypeExecution, AuditResourceTypeProviderConfig:
		return true
	default:
		return false
	}
}

// AuditActorType represents the type of actor that performed an action.
type AuditActorType string

const (
	AuditActorTypeUser   AuditActorType = "user"
	AuditActorTypeSystem AuditActorType = "system"
	AuditActorTypeAPIKey AuditActorType = "api_key"
)

// IsValid checks if the AuditActorType is a valid enum value.
func (a AuditActorType) IsValid() bool {
	switch a {
	case AuditActorTypeUser, AuditActorTypeSystem, AuditActorTypeAPIKey:
		return true
	default:
		return false
	}
}

// AuditEntry validation errors
var (
	ErrAuditEntryInvalidEventType = pkg.ValidationError{
		Code:    constant.ErrAuditEntryInvalidEventType.Error(),
		Message: "audit event type is invalid",
	}
	ErrAuditEntryInvalidAction = pkg.ValidationError{
		Code:    constant.ErrAuditEntryInvalidAction.Error(),
		Message: "audit action is invalid",
	}
	ErrAuditEntryInvalidResult = pkg.ValidationError{
		Code:    constant.ErrAuditEntryInvalidResult.Error(),
		Message: "audit result is invalid",
	}
	ErrAuditEntryResourceIDRequired = pkg.ValidationError{
		Code:    constant.ErrAuditEntryResourceIDRequired.Error(),
		Message: "resource ID is required",
	}
	ErrAuditEntryResourceIDTooLong = pkg.ValidationError{
		Code:    constant.ErrAuditEntryResourceIDTooLong.Error(),
		Message: "resource ID must not exceed 255 characters",
	}
	ErrAuditEntryInvalidResourceType = pkg.ValidationError{
		Code:    constant.ErrAuditEntryInvalidResourceType.Error(),
		Message: "resource type is invalid",
	}
	ErrAuditEntryActorIDRequired = pkg.ValidationError{
		Code:    constant.ErrAuditEntryActorIDRequired.Error(),
		Message: "actor ID is required",
	}
	ErrAuditEntryActorIDTooLong = pkg.ValidationError{
		Code:    constant.ErrAuditEntryActorIDTooLong.Error(),
		Message: "actor ID must not exceed 255 characters",
	}
	ErrAuditEntryInvalidActorType = pkg.ValidationError{
		Code:    constant.ErrAuditEntryInvalidActorType.Error(),
		Message: "actor type is invalid",
	}
)

// AuditActor represents who performed the action.
type AuditActor struct {
	actorType AuditActorType
	id        string
	ipAddress string
}

// NewAuditActor creates a new AuditActor with validation.
func NewAuditActor(actorType AuditActorType, id string, ipAddress string) (AuditActor, error) {
	normalizedID := strings.TrimSpace(id)

	if !actorType.IsValid() {
		return AuditActor{}, ErrAuditEntryInvalidActorType
	}

	if normalizedID == "" {
		return AuditActor{}, ErrAuditEntryActorIDRequired
	}

	if len(normalizedID) > 255 {
		return AuditActor{}, ErrAuditEntryActorIDTooLong
	}

	ip := strings.TrimSpace(ipAddress)
	if ip == "" {
		ip = "0.0.0.0"
	}

	return AuditActor{
		actorType: actorType,
		id:        normalizedID,
		ipAddress: ip,
	}, nil
}

// Type returns the actor type.
func (a AuditActor) Type() AuditActorType {
	return a.actorType
}

// ID returns the actor ID.
func (a AuditActor) ID() string {
	return a.id
}

// IPAddress returns the actor's IP address.
func (a AuditActor) IPAddress() string {
	return a.ipAddress
}

// AuditEntry represents an immutable audit record (Rich Domain Model).
// Fields are private with validation in constructor per PROJECT_RULES.md.
// Hash chain is computed by PostgreSQL trigger, not in Go.
type AuditEntry struct {
	internalID   int64
	eventID      uuid.UUID
	eventType    AuditEventType
	action       AuditAction
	result       AuditResult
	resourceID   string
	resourceType AuditResourceType
	actor        AuditActor
	context      map[string]any
	metadata     map[string]any
	timestamp    time.Time
	hash         string
	previousHash string
}

// NewAuditEntry creates a new AuditEntry with validation.
// Returns error if any required field is invalid.
// eventID is generated via uuid.New(), timestamp is set to now.
// Hash and previousHash are left empty (computed by PostgreSQL trigger).
func NewAuditEntry(
	eventType AuditEventType,
	action AuditAction,
	result AuditResult,
	resourceID string,
	resourceType AuditResourceType,
	actor AuditActor,
) (*AuditEntry, error) {
	normalizedResourceID := strings.TrimSpace(resourceID)

	if !eventType.IsValid() {
		return nil, ErrAuditEntryInvalidEventType
	}

	if !action.IsValid() {
		return nil, ErrAuditEntryInvalidAction
	}

	if !result.IsValid() {
		return nil, ErrAuditEntryInvalidResult
	}

	if normalizedResourceID == "" {
		return nil, ErrAuditEntryResourceIDRequired
	}

	if len(normalizedResourceID) > 255 {
		return nil, ErrAuditEntryResourceIDTooLong
	}

	if !resourceType.IsValid() {
		return nil, ErrAuditEntryInvalidResourceType
	}

	// Actor was already validated in NewAuditActor, but re-validate for safety
	if !actor.actorType.IsValid() {
		return nil, ErrAuditEntryInvalidActorType
	}

	if strings.TrimSpace(actor.id) == "" {
		return nil, ErrAuditEntryActorIDRequired
	}

	return &AuditEntry{
		eventID:      uuid.New(),
		eventType:    eventType,
		action:       action,
		result:       result,
		resourceID:   normalizedResourceID,
		resourceType: resourceType,
		actor:        actor,
		context:      make(map[string]any),
		metadata:     make(map[string]any),
		timestamp:    time.Now().UTC(),
	}, nil
}

// ReconstructAuditEntry reconstructs an AuditEntry from database values.
// Used by repository adapters - bypasses validation since data is already valid.
func ReconstructAuditEntry(
	internalID int64,
	eventID uuid.UUID,
	eventType AuditEventType,
	action AuditAction,
	result AuditResult,
	resourceID string,
	resourceType AuditResourceType,
	actor AuditActor,
	ctx map[string]any,
	meta map[string]any,
	timestamp time.Time,
	hash string,
	previousHash string,
) *AuditEntry {
	return &AuditEntry{
		internalID:   internalID,
		eventID:      eventID,
		eventType:    eventType,
		action:       action,
		result:       result,
		resourceID:   resourceID,
		resourceType: resourceType,
		actor:        actor,
		context:      cloneMap(ctx),
		metadata:     cloneMap(meta),
		timestamp:    timestamp,
		hash:         hash,
		previousHash: previousHash,
	}
}

// InternalID returns the internal database sequence ID.
func (e *AuditEntry) InternalID() int64 {
	return e.internalID
}

// EventID returns the unique event identifier.
func (e *AuditEntry) EventID() uuid.UUID {
	return e.eventID
}

// EventType returns the audit event type.
func (e *AuditEntry) EventType() AuditEventType {
	return e.eventType
}

// Action returns the audit action.
func (e *AuditEntry) Action() AuditAction {
	return e.action
}

// Result returns the audit result.
func (e *AuditEntry) Result() AuditResult {
	return e.result
}

// ResourceID returns the affected resource ID.
func (e *AuditEntry) ResourceID() string {
	return e.resourceID
}

// ResourceType returns the affected resource type.
func (e *AuditEntry) ResourceType() AuditResourceType {
	return e.resourceType
}

// Actor returns the actor who performed the action.
func (e *AuditEntry) Actor() AuditActor {
	return e.actor
}

// Context returns a copy of the context data.
func (e *AuditEntry) Context() map[string]any {
	if e.context == nil {
		return nil
	}

	result := make(map[string]any, len(e.context))
	for k, v := range e.context {
		result[k] = v
	}

	return result
}

// Metadata returns a copy of the metadata.
func (e *AuditEntry) Metadata() map[string]any {
	if e.metadata == nil {
		return nil
	}

	result := make(map[string]any, len(e.metadata))
	for k, v := range e.metadata {
		result[k] = v
	}

	return result
}

// Timestamp returns when the event occurred.
func (e *AuditEntry) Timestamp() time.Time {
	return e.timestamp
}

// Hash returns the hash chain value.
func (e *AuditEntry) Hash() string {
	return e.hash
}

// PreviousHash returns the previous hash in the chain.
func (e *AuditEntry) PreviousHash() string {
	return e.previousHash
}

// WithContext sets the context data and returns the entry for chaining.
func (e *AuditEntry) WithContext(ctx map[string]any) *AuditEntry {
	e.context = cloneMap(ctx)

	return e
}

// WithMetadata sets additional metadata and returns the entry for chaining.
func (e *AuditEntry) WithMetadata(meta map[string]any) *AuditEntry {
	e.metadata = cloneMap(meta)

	return e
}

// cloneMap creates a shallow copy of a map. Returns an empty map if input is nil.
func cloneMap(m map[string]any) map[string]any {
	if m == nil {
		return make(map[string]any)
	}

	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}

	return result
}
