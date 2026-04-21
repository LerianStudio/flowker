// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"
	"errors"
	"fmt"

	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"github.com/google/uuid"
)

// VerifyAuditHashChainQuery handles verifying the hash chain integrity for an audit entry.
type VerifyAuditHashChainQuery struct {
	repo AuditReadRepository
}

// NewVerifyAuditHashChainQuery creates a new VerifyAuditHashChainQuery.
// Returns error if the repository is nil.
func NewVerifyAuditHashChainQuery(repo AuditReadRepository) (*VerifyAuditHashChainQuery, error) {
	if repo == nil {
		return nil, ErrAuditReadNilRepo
	}

	return &VerifyAuditHashChainQuery{
		repo: repo,
	}, nil
}

// Execute verifies the hash chain integrity up to the specified event ID.
func (q *VerifyAuditHashChainQuery) Execute(ctx context.Context, eventID uuid.UUID) (*model.HashChainVerificationOutput, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.audit.verify_hash_chain")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Verifying audit hash chain",
		libLog.Any("operation", "query.audit.verify_hash_chain"),
		libLog.Any("audit.event_id", eventID),
	)

	result, err := q.repo.VerifyHashChain(ctx, eventID)
	if err != nil {
		if errors.Is(err, constant.ErrAuditEntryNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Audit entry not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to verify hash chain", err)

		return nil, fmt.Errorf("failed to verify hash chain: %w", err)
	}

	return result, nil
}
