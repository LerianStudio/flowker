// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"
	"errors"
	"fmt"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	"github.com/google/uuid"
)

// GetAuditEntryByIDQuery handles retrieving an audit entry by its event ID.
type GetAuditEntryByIDQuery struct {
	repo AuditReadRepository
}

// NewGetAuditEntryByIDQuery creates a new GetAuditEntryByIDQuery.
// Returns error if the repository is nil.
func NewGetAuditEntryByIDQuery(repo AuditReadRepository) (*GetAuditEntryByIDQuery, error) {
	if repo == nil {
		return nil, ErrAuditReadNilRepo
	}

	return &GetAuditEntryByIDQuery{
		repo: repo,
	}, nil
}

// Execute retrieves an audit entry by its event ID.
func (q *GetAuditEntryByIDQuery) Execute(ctx context.Context, eventID uuid.UUID) (*model.AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.audit.get")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Getting audit entry by ID",
		libLog.Any("operation", "query.audit.get"),
		libLog.Any("audit.event_id", eventID),
	)

	entry, err := q.repo.FindByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, constant.ErrAuditEntryNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Audit entry not found", err)

			return nil, pkg.EntityNotFoundError{
				EntityType: "AuditEntry",
				Code:       constant.ErrAuditEntryNotFound.Error(),
				Title:      "Audit Entry Not Found",
				Message:    "No audit entry was found for the given event ID.",
				Err:        err,
			}
		}

		libOtel.HandleSpanError(span, "Failed to find audit entry", err)

		return nil, fmt.Errorf("failed to find audit entry: %w", err)
	}

	return entry, nil
}
