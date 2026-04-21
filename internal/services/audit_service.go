// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package services

import (
	"context"

	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// AuditService is a facade that combines audit trail queries.
type AuditService struct {
	searchQuery     *query.SearchAuditLogsQuery
	getByIDQuery    *query.GetAuditEntryByIDQuery
	verifyHashQuery *query.VerifyAuditHashChainQuery
}

// NewAuditService creates a new AuditService facade.
func NewAuditService(
	searchQuery *query.SearchAuditLogsQuery,
	getByIDQuery *query.GetAuditEntryByIDQuery,
	verifyHashQuery *query.VerifyAuditHashChainQuery,
) (*AuditService, error) {
	if searchQuery == nil || getByIDQuery == nil || verifyHashQuery == nil {
		return nil, ErrAuditServiceNilDependency
	}

	return &AuditService{
		searchQuery:     searchQuery,
		getByIDQuery:    getByIDQuery,
		verifyHashQuery: verifyHashQuery,
	}, nil
}

// SearchLogs retrieves audit entries with optional filtering and pagination.
func (s *AuditService) SearchLogs(ctx context.Context, filter query.AuditListFilter) ([]*model.AuditEntry, string, bool, error) {
	return s.searchQuery.Execute(ctx, filter)
}

// GetByID retrieves a single audit entry by its event ID.
func (s *AuditService) GetByID(ctx context.Context, eventID uuid.UUID) (*model.AuditEntry, error) {
	return s.getByIDQuery.Execute(ctx, eventID)
}

// VerifyHashChain verifies the hash chain integrity up to the specified event ID.
func (s *AuditService) VerifyHashChain(ctx context.Context, eventID uuid.UUID) (*model.HashChainVerificationOutput, error) {
	return s.verifyHashQuery.Execute(ctx, eventID)
}
