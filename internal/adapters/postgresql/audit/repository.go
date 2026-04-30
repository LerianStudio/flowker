// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package audit contains the PostgreSQL repository implementation for audit trail operations.
package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// psql is the PostgreSQL-flavoured statement builder using $1, $2, ... placeholders.
var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// auditColumns lists the columns used in SELECT queries for audit_events.
var auditColumns = []string{
	"id", "event_id", "event_type", "action", "result",
	"resource_id", "resource_type",
	"actor_type", "actor_id", "actor_ip",
	"context", "metadata", "created_at", "hash", "previous_hash",
}

// PostgreSQLRepository implements both AuditWriteRepository and AuditReadRepository
// using pgxpool for PostgreSQL access.
// Supports both single-tenant (fallback) and multi-tenant (context-based) modes.
type PostgreSQLRepository struct {
	fallbackPool *pgxpool.Pool // Fallback for single-tenant mode
}

// NewPostgreSQLRepository creates a new PostgreSQL audit repository.
// The provided pool is used as fallback in single-tenant mode.
// In multi-tenant mode, the pool is resolved from context.
func NewPostgreSQLRepository(pool *pgxpool.Pool) (*PostgreSQLRepository, error) {
	if pool == nil {
		return nil, errors.New("pgxpool cannot be nil")
	}

	return &PostgreSQLRepository{fallbackPool: pool}, nil
}

// getPool returns the tenant-specific pool or fallback.
// In multi-tenant mode, it extracts the pool from context.
// In single-tenant mode, it uses the fallback pool.
//
// Note: Multi-tenant mode for PostgreSQL requires the tenant middleware to inject
// a *pgxpool.Pool into context. If GetPGContext returns a dbresolver.DB (sql-based),
// this method falls back to single-tenant mode since pgx-specific features are used.
//
// Current limitation: tmcore.GetPGContext returns dbresolver.DB (sql-based interface),
// but this repository uses pgxpool.Pool for pgx-specific features (e.g., native type handling).
// For full multi-tenant PostgreSQL support, consider either:
// 1. Refactoring to database/sql interface
// 2. Adding tmcore.GetPGXContext helper for pgx-specific pools
// Until then, this repository operates in single-tenant mode using the fallback pool.
func (r *PostgreSQLRepository) getPool(ctx context.Context) (*pgxpool.Pool, error) {
	// Note: We intentionally don't use tmcore.GetPGContext(ctx) here because:
	// - It returns dbresolver.DB (database/sql compatible)
	// - This repository requires pgxpool.Pool for pgx-specific features
	// - Type conversion is not possible without connection re-establishment
	// The fallback pool is used for all operations until pgx-specific multi-tenant support is added.

	if r.fallbackPool == nil {
		return nil, errors.New("postgresql connection not available")
	}

	return r.fallbackPool, nil
}

// Insert persists a new audit entry to PostgreSQL.
func (r *PostgreSQLRepository) Insert(ctx context.Context, entry *model.AuditEntry) error {
	pool, err := r.getPool(ctx)
	if err != nil {
		return fmt.Errorf("failed to get pool: %w", err)
	}

	if entry == nil {
		return errors.New("audit entry cannot be nil")
	}

	contextJSON, err := json.Marshal(entry.Context())
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	metadataJSON, err := json.Marshal(entry.Metadata())
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	sqlStr, args, err := psql.
		Insert("audit_events").
		Columns(
			"event_id", "event_type", "action", "result",
			"resource_id", "resource_type",
			"actor_type", "actor_id", "actor_ip",
			"context", "metadata",
		).
		Values(
			entry.EventID(),
			string(entry.EventType()),
			string(entry.Action()),
			string(entry.Result()),
			entry.ResourceID(),
			string(entry.ResourceType()),
			string(entry.Actor().Type()),
			entry.Actor().ID(),
			entry.Actor().IPAddress(),
			contextJSON,
			metadataJSON,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	_, err = pool.Exec(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to insert audit entry: %w", err)
	}

	return nil
}

// FindByID retrieves an audit entry by its event ID.
func (r *PostgreSQLRepository) FindByID(ctx context.Context, eventID uuid.UUID) (*model.AuditEntry, error) {
	pool, err := r.getPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool: %w", err)
	}

	sqlStr, args, err := psql.
		Select(auditColumns...).
		From("audit_events").
		Where(sq.Eq{"event_id": eventID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build find query: %w", err)
	}

	row := pool.QueryRow(ctx, sqlStr, args...)

	entry, err := scanAuditEntry(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, constant.ErrAuditEntryNotFound
		}

		return nil, fmt.Errorf("failed to find audit entry: %w", err)
	}

	return entry, nil
}

// List retrieves audit entries with filtering and cursor-based pagination.
func (r *PostgreSQLRepository) List(ctx context.Context, filter query.AuditListFilter) ([]*model.AuditEntry, string, bool, error) {
	pool, err := r.getPool(ctx)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to get pool: %w", err)
	}

	builder := psql.
		Select(auditColumns...).
		From("audit_events")

	if filter.EventType != nil {
		builder = builder.Where(sq.Eq{"event_type": *filter.EventType})
	}

	if filter.Action != nil {
		builder = builder.Where(sq.Eq{"action": *filter.Action})
	}

	if filter.Result != nil {
		builder = builder.Where(sq.Eq{"result": *filter.Result})
	}

	if filter.ResourceType != nil {
		builder = builder.Where(sq.Eq{"resource_type": *filter.ResourceType})
	}

	if filter.ResourceID != nil {
		builder = builder.Where(sq.Eq{"resource_id": filter.ResourceID.String()})
	}

	if filter.DateFrom != nil {
		builder = builder.Where(sq.GtOrEq{"created_at": *filter.DateFrom})
	}

	if filter.DateTo != nil {
		builder = builder.Where(sq.LtOrEq{"created_at": *filter.DateTo})
	}

	// Cursor-based pagination using id
	if filter.Cursor != "" {
		cursorID, err := strconv.ParseInt(filter.Cursor, 10, 64)
		if err != nil {
			return nil, "", false, fmt.Errorf("%w: %w", constant.ErrAuditInvalidCursor, err)
		}

		if filter.SortOrder == "ASC" {
			builder = builder.Where(sq.Gt{"id": cursorID})
		} else {
			builder = builder.Where(sq.Lt{"id": cursorID})
		}
	}

	// Sort order
	if filter.SortOrder == "ASC" {
		builder = builder.OrderBy("id ASC")
	} else {
		builder = builder.OrderBy("id DESC")
	}

	// Fetch one extra to determine hasMore (limit already validated by query layer)
	limit := filter.Limit
	builder = builder.Limit(uint64(limit + 1))

	sqlStr, args, err := builder.ToSql()
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to build list query: %w", err)
	}

	rows, err := pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to list audit entries: %w", err)
	}
	defer rows.Close()

	entries := make([]*model.AuditEntry, 0, limit)

	for rows.Next() {
		entry, scanErr := scanAuditEntryFromRows(rows)
		if scanErr != nil {
			return nil, "", false, fmt.Errorf("failed to scan audit entry: %w", scanErr)
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, "", false, fmt.Errorf("error iterating audit entries: %w", err)
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}

	var nextCursor string
	if len(entries) > 0 && hasMore {
		nextCursor = strconv.FormatInt(entries[len(entries)-1].InternalID(), 10)
	}

	return entries, nextCursor, hasMore, nil
}

// VerifyHashChain verifies the hash chain integrity up to the specified event ID.
func (r *PostgreSQLRepository) VerifyHashChain(ctx context.Context, eventID uuid.UUID) (*model.HashChainVerificationOutput, error) {
	pool, err := r.getPool(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool: %w", err)
	}

	// First, find the target entry to get its id
	findSQL, findArgs, err := psql.
		Select("id").
		From("audit_events").
		Where(sq.Eq{"event_id": eventID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build find query: %w", err)
	}

	var targetID int64

	err = pool.QueryRow(ctx, findSQL, findArgs...).Scan(&targetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, constant.ErrAuditEntryNotFound
		}

		return nil, fmt.Errorf("failed to find target audit entry: %w", err)
	}

	// Verify hash chain from beginning up to targetID
	verifySQL, verifyArgs, err := psql.
		Select("id", "hash", "previous_hash").
		From("audit_events").
		Where(sq.LtOrEq{"id": targetID}).
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build verify query: %w", err)
	}

	rows, err := pool.Query(ctx, verifySQL, verifyArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query hash chain: %w", err)
	}
	defer rows.Close()

	var totalChecked int64

	var previousHash string

	for rows.Next() {
		var (
			internalID int64
			hash       string
			prevHash   string
		)

		if err := rows.Scan(&internalID, &hash, &prevHash); err != nil {
			return nil, fmt.Errorf("failed to scan hash chain entry: %w", err)
		}

		totalChecked++

		// Verify chain: current entry's previousHash should match the last entry's hash
		if totalChecked > 1 && prevHash != previousHash {
			return &model.HashChainVerificationOutput{
				IsValid:        false,
				FirstInvalidID: &internalID,
				TotalChecked:   totalChecked,
				Message:        fmt.Sprintf("hash chain broken at id %d", internalID),
			}, nil
		}

		previousHash = hash
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating hash chain: %w", err)
	}

	return &model.HashChainVerificationOutput{
		IsValid:      true,
		TotalChecked: totalChecked,
		Message:      "hash chain is valid",
	}, nil
}

// scanAuditEntry scans a single row into an AuditEntry.
func scanAuditEntry(row pgx.Row) (*model.AuditEntry, error) {
	var (
		internalID   int64
		eventID      uuid.UUID
		eventType    string
		action       string
		result       string
		resourceID   string
		resourceType string
		actorType    string
		actorID      string
		actorIP      string
		contextJSON  []byte
		metadataJSON []byte
		ts           time.Time
		hash         string
		previousHash string
	)

	err := row.Scan(
		&internalID, &eventID, &eventType, &action, &result,
		&resourceID, &resourceType,
		&actorType, &actorID, &actorIP,
		&contextJSON, &metadataJSON, &ts, &hash, &previousHash,
	)
	if err != nil {
		return nil, err
	}

	return reconstructEntry(internalID, eventID, eventType, action, result,
		resourceID, resourceType, actorType, actorID, actorIP,
		contextJSON, metadataJSON, ts, hash, previousHash)
}

// scanAuditEntryFromRows scans the current row from pgx.Rows into an AuditEntry.
func scanAuditEntryFromRows(rows pgx.Rows) (*model.AuditEntry, error) {
	var (
		internalID   int64
		eventID      uuid.UUID
		eventType    string
		action       string
		result       string
		resourceID   string
		resourceType string
		actorType    string
		actorID      string
		actorIP      string
		contextJSON  []byte
		metadataJSON []byte
		ts           time.Time
		hash         string
		previousHash string
	)

	err := rows.Scan(
		&internalID, &eventID, &eventType, &action, &result,
		&resourceID, &resourceType,
		&actorType, &actorID, &actorIP,
		&contextJSON, &metadataJSON, &ts, &hash, &previousHash,
	)
	if err != nil {
		return nil, err
	}

	return reconstructEntry(internalID, eventID, eventType, action, result,
		resourceID, resourceType, actorType, actorID, actorIP,
		contextJSON, metadataJSON, ts, hash, previousHash)
}

// reconstructEntry builds an AuditEntry from scanned database values.
func reconstructEntry(
	internalID int64,
	eventID uuid.UUID,
	eventType, action, result, resourceID, resourceType string,
	actorType, actorID, actorIP string,
	contextJSON, metadataJSON []byte,
	ts time.Time,
	hash, previousHash string,
) (*model.AuditEntry, error) {
	var ctxMap map[string]any
	if len(contextJSON) > 0 {
		if err := json.Unmarshal(contextJSON, &ctxMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal context: %w", err)
		}
	}

	var metaMap map[string]any
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &metaMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	actor, err := model.NewAuditActor(model.AuditActorType(actorType), actorID, actorIP)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct actor: %w", err)
	}

	entry := model.ReconstructAuditEntry(
		internalID,
		eventID,
		model.AuditEventType(eventType),
		model.AuditAction(action),
		model.AuditResult(result),
		resourceID,
		model.AuditResourceType(resourceType),
		actor,
		ctxMap,
		metaMap,
		ts,
		hash,
		previousHash,
	)

	return entry, nil
}
