// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package query

import (
	"context"
	"strings"

	"github.com/LerianStudio/flowker/pkg/model"
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
)

// SearchAuditLogsQuery handles listing audit log entries with filtering and pagination.
type SearchAuditLogsQuery struct {
	repo AuditReadRepository
}

// NewSearchAuditLogsQuery creates a new SearchAuditLogsQuery.
// Returns error if the repository is nil.
func NewSearchAuditLogsQuery(repo AuditReadRepository) (*SearchAuditLogsQuery, error) {
	if repo == nil {
		return nil, ErrAuditReadNilRepo
	}

	return &SearchAuditLogsQuery{
		repo: repo,
	}, nil
}

// Execute retrieves audit log entries with optional filtering and pagination.
func (q *SearchAuditLogsQuery) Execute(ctx context.Context, filter AuditListFilter) ([]*model.AuditEntry, string, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", false, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "query.audit.search")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Searching audit logs",
		libLog.Any("operation", "query.audit.search"),
		libLog.Any("filter.limit", filter.Limit),
		libLog.Any("filter.sort_order", filter.SortOrder),
	)

	// Apply defaults and constraints
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	filter.SortOrder = strings.ToUpper(filter.SortOrder)
	if filter.SortOrder != "ASC" && filter.SortOrder != "DESC" {
		filter.SortOrder = "DESC"
	}

	entries, nextCursor, hasMore, err := q.repo.List(ctx, filter)
	if err != nil {
		libOtel.HandleSpanError(span, "failed to search audit logs", err)
		return nil, "", false, err
	}

	return entries, nextCursor, hasMore, nil
}
