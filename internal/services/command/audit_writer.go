// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// AuditWriter provides a fire-and-forget interface for recording audit events
// from command handlers. Implementations must be safe for concurrent use.
type AuditWriter interface {
	// RecordWorkflowEvent records an audit event for a workflow lifecycle change.
	RecordWorkflowEvent(ctx context.Context, eventType model.AuditEventType, action model.AuditAction, result model.AuditResult, workflowID uuid.UUID, metadata map[string]any)

	// RecordExecutionEvent records an audit event for an execution lifecycle change.
	RecordExecutionEvent(ctx context.Context, eventType model.AuditEventType, action model.AuditAction, result model.AuditResult, executionID uuid.UUID, metadata map[string]any)

	// RecordProviderConfigEvent records an audit event for a provider configuration change.
	RecordProviderConfigEvent(ctx context.Context, eventType model.AuditEventType, action model.AuditAction, result model.AuditResult, configID uuid.UUID, metadata map[string]any)
}
