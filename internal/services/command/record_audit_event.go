// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// RecordAuditEventCommand implements AuditWriter with a fire-and-forget pattern.
// Each recording is dispatched in a goroutine so that audit failures never
// block or fail the originating business operation.
type RecordAuditEventCommand struct {
	repo AuditWriteRepository
}

// NewRecordAuditEventCommand creates a new RecordAuditEventCommand.
// Returns error if the repository is nil.
func NewRecordAuditEventCommand(repo AuditWriteRepository) (*RecordAuditEventCommand, error) {
	if repo == nil {
		return nil, ErrAuditWriterNilRepo
	}

	return &RecordAuditEventCommand{repo: repo}, nil
}

// RecordWorkflowEvent records an audit event for a workflow lifecycle change.
func (c *RecordAuditEventCommand) RecordWorkflowEvent(
	ctx context.Context,
	eventType model.AuditEventType,
	action model.AuditAction,
	result model.AuditResult,
	workflowID uuid.UUID,
	metadata map[string]any,
) {
	c.recordEvent(ctx, eventType, action, result, workflowID.String(), model.AuditResourceTypeWorkflow, metadata)
}

// RecordExecutionEvent records an audit event for an execution lifecycle change.
func (c *RecordAuditEventCommand) RecordExecutionEvent(
	ctx context.Context,
	eventType model.AuditEventType,
	action model.AuditAction,
	result model.AuditResult,
	executionID uuid.UUID,
	metadata map[string]any,
) {
	c.recordEvent(ctx, eventType, action, result, executionID.String(), model.AuditResourceTypeExecution, metadata)
}

// RecordProviderConfigEvent records an audit event for a provider configuration change.
func (c *RecordAuditEventCommand) RecordProviderConfigEvent(
	ctx context.Context,
	eventType model.AuditEventType,
	action model.AuditAction,
	result model.AuditResult,
	configID uuid.UUID,
	metadata map[string]any,
) {
	c.recordEvent(ctx, eventType, action, result, configID.String(), model.AuditResourceTypeProviderConfig, metadata)
}

// recordEvent is the internal fire-and-forget dispatcher.
// It creates an AuditEntry and persists it in a background goroutine.
// context.WithoutCancel ensures the write completes even if the HTTP request context is cancelled.
func (c *RecordAuditEventCommand) recordEvent(
	ctx context.Context,
	eventType model.AuditEventType,
	action model.AuditAction,
	result model.AuditResult,
	resourceID string,
	resourceType model.AuditResourceType,
	metadata map[string]any,
) {
	logger := libCommons.NewLoggerFromContext(ctx)

	actor, err := model.NewAuditActor(model.AuditActorTypeSystem, "flowker", "")
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create audit actor",
			libLog.Any("error.message", err.Error()),
			libLog.Any("audit.event_type", string(eventType)),
		)

		return
	}

	entry, err := model.NewAuditEntry(eventType, action, result, resourceID, resourceType, actor)
	if err != nil {
		logger.Log(ctx, libLog.LevelError, "Failed to create audit entry",
			libLog.Any("error.message", err.Error()),
			libLog.Any("audit.event_type", string(eventType)),
			libLog.Any("audit.resource_id", resourceID),
		)

		return
	}

	if metadata != nil {
		entry.WithMetadata(metadata)
	}

	// Fire-and-forget: use context.WithoutCancel so the write survives request cancellation,
	// but add a 5s timeout to prevent goroutine leaks if the database is stuck.
	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)

	go func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				logger.Log(bgCtx, libLog.LevelError, "Panic in audit event recording",
					libLog.Any("panic", r),
					libLog.Any("audit.event_type", string(eventType)),
					libLog.Any("audit.resource_id", resourceID),
				)
			}
		}()

		if insertErr := c.repo.Insert(bgCtx, entry); insertErr != nil {
			logger.Log(bgCtx, libLog.LevelError, "Failed to persist audit entry",
				libLog.Any("error.message", insertErr.Error()),
				libLog.Any("audit.event_type", string(eventType)),
				libLog.Any("audit.resource_id", resourceID),
			)
		}
	}()
}
