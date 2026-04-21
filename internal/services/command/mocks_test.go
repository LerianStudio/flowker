// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package command

import (
	"context"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source=workflow_repository.go -destination=workflow_repository_mock_test.go -package=command
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source=executor_config_repository.go -destination=executor_config_repository_mock_test.go -package=command
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source=execution_repository.go -destination=execution_repository_mock_test.go -package=command
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source=provider_config_repository.go -destination=provider_config_repository_mock_test.go -package=command

// noopAuditWriter is a no-op implementation of AuditWriter for tests.
type noopAuditWriter struct{}

func (n *noopAuditWriter) RecordWorkflowEvent(_ context.Context, _ model.AuditEventType, _ model.AuditAction, _ model.AuditResult, _ uuid.UUID, _ map[string]any) {
}

func (n *noopAuditWriter) RecordExecutionEvent(_ context.Context, _ model.AuditEventType, _ model.AuditAction, _ model.AuditResult, _ uuid.UUID, _ map[string]any) {
}

func (n *noopAuditWriter) RecordProviderConfigEvent(_ context.Context, _ model.AuditEventType, _ model.AuditAction, _ model.AuditResult, _ uuid.UUID, _ map[string]any) {
}

// newNoopAuditWriter returns a no-op AuditWriter for use in tests.
func newNoopAuditWriter() AuditWriter {
	return &noopAuditWriter{}
}
