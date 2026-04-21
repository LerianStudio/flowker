// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// AuditReadRepository provides read access to the audit trail store.
type AuditReadRepository interface {
	FindByID(ctx context.Context, eventID uuid.UUID) (*model.AuditEntry, error)
	List(ctx context.Context, filter AuditListFilter) ([]*model.AuditEntry, string, bool, error)
	VerifyHashChain(ctx context.Context, eventID uuid.UUID) (*model.HashChainVerificationOutput, error)
}

// AuditListFilter defines filtering and pagination options for listing audit entries.
type AuditListFilter struct {
	EventType    *string
	Action       *string
	Result       *string
	ResourceType *string
	ResourceID   *uuid.UUID
	DateFrom     *time.Time
	DateTo       *time.Time
	Limit        int
	Cursor       string
	SortOrder    string
}
