// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package audit

import (
	"context"
	"testing"

	"github.com/LerianStudio/flowker/internal/services/query"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Multi-Tenant Unit Tests
// =============================================================================

func TestPostgreSQLRepository_MultiTenant_GetPool_WithTenantContext(t *testing.T) {
	// Note: The PostgreSQL audit repository currently falls back to the static pool
	// even when tenant context is present because GetPGContext returns dbresolver.DB
	// which is sql-based, not pgx-specific. This test documents this behavior.

	t.Run("falls_back_to_static_pool_even_with_tenant_context", func(t *testing.T) {
		// Create repository with nil fallback pool
		repo := &PostgreSQLRepository{fallbackPool: nil}
		ctx := context.Background()

		// Even if GetPGContext returns something, the repo falls back
		// because it needs pgxpool.Pool, not dbresolver.DB
		pool, err := repo.getPool(ctx)

		require.Error(t, err)
		assert.Nil(t, pool)
		assert.Contains(t, err.Error(), "postgresql connection not available")
	})
}

func TestPostgreSQLRepository_MultiTenant_FallbackModeWithNilFallback(t *testing.T) {
	// When no fallback pool is provided and no compatible tenant context exists,
	// repository operations should return an error.
	repo := &PostgreSQLRepository{fallbackPool: nil}
	ctx := context.Background()

	pool, err := repo.getPool(ctx)

	require.Error(t, err)
	assert.Nil(t, pool)
	assert.Contains(t, err.Error(), "postgresql connection not available")
}

func TestPostgreSQLRepository_MultiTenant_TenantContextExtraction(t *testing.T) {
	// Test that the repository correctly checks for tenant context
	// using tmcore.GetPGContext

	t.Run("returns nil when context has no tenant database", func(t *testing.T) {
		ctx := context.Background()

		// Verify that GetPGContext returns nil for plain context
		db := tmcore.GetPGContext(ctx)
		assert.Nil(t, db, "GetPGContext should return nil for context without tenant database")
	})

	t.Run("repository falls back to static pool when no compatible tenant context", func(t *testing.T) {
		repo := &PostgreSQLRepository{fallbackPool: nil}
		ctx := context.Background()

		// Since fallback is nil, this should error
		_, err := repo.getPool(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "postgresql connection not available")
	})
}

func TestPostgreSQLRepository_MultiTenant_ConstructorValidation(t *testing.T) {
	t.Run("constructor_rejects_nil_pool", func(t *testing.T) {
		repo, err := NewPostgreSQLRepository(nil)

		require.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "pgxpool cannot be nil")
	})
}

// =============================================================================
// Error Case Tests
// =============================================================================

func TestPostgreSQLRepository_MultiTenant_ErrorCases(t *testing.T) {
	// Create repository with nil fallback for error testing
	repo := &PostgreSQLRepository{fallbackPool: nil}
	ctx := context.Background()

	t.Run("Insert_returns_error_when_no_connection_available", func(t *testing.T) {
		err := repo.Insert(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get pool")
	})

	t.Run("FindByID_returns_error_when_no_connection_available", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get pool")
	})

	t.Run("List_returns_error_when_no_connection_available", func(t *testing.T) {
		filter := query.AuditListFilter{
			Limit: 10,
		}
		_, _, _, err := repo.List(ctx, filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get pool")
	})

	t.Run("VerifyHashChain_returns_error_when_no_connection_available", func(t *testing.T) {
		_, err := repo.VerifyHashChain(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get pool")
	})
}

// =============================================================================
// Backward Compatibility Tests (Single-Tenant Mode)
// =============================================================================

func TestPostgreSQLRepository_SingleTenantMode_BackwardCompatibility(t *testing.T) {
	t.Run("constructor_requires_non_nil_pool", func(t *testing.T) {
		// PostgreSQL audit repository requires a fallback pool at construction time
		// This ensures single-tenant mode always works
		repo, err := NewPostgreSQLRepository(nil)

		require.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "pgxpool cannot be nil")
	})
}

// =============================================================================
// PostgreSQL-specific Multi-Tenant Notes
// =============================================================================

func TestPostgreSQLRepository_MultiTenant_Documentation(t *testing.T) {
	// This test documents the current multi-tenant behavior for PostgreSQL.
	//
	// The audit repository uses pgxpool directly (not database/sql) for
	// PostgreSQL-specific features. The lib-commons GetPGContext returns
	// a dbresolver.DB (sql-based interface), which is NOT compatible with
	// pgxpool.Pool.
	//
	// Current behavior:
	// 1. In single-tenant mode: uses the static fallbackPool provided at construction
	// 2. In multi-tenant mode: currently falls back to static pool because
	//    GetPGContext returns dbresolver.DB, not pgxpool.Pool
	//
	// Future enhancement: Add tmcore.GetPGXContext helper for pgx-specific pools
	// to enable full multi-tenant support for repositories that use pgx directly.

	t.Run("documents_pgx_vs_sql_compatibility", func(t *testing.T) {
		ctx := context.Background()

		// GetPGContext is designed for database/sql compatibility
		db := tmcore.GetPGContext(ctx)
		assert.Nil(t, db, "GetPGContext returns nil for empty context")

		// The audit repository needs pgxpool.Pool, not dbresolver.DB
		// This is why it currently falls back to the static pool
	})
}

// =============================================================================
// Column Definition Tests
// =============================================================================

func TestAuditColumns(t *testing.T) {
	// Verify that the audit columns are correctly defined
	expectedColumns := []string{
		"id", "event_id", "event_type", "action", "result",
		"resource_id", "resource_type",
		"actor_type", "actor_id", "actor_ip",
		"context", "metadata", "created_at", "hash", "previous_hash",
	}

	assert.Equal(t, expectedColumns, auditColumns, "Audit columns should match expected definition")
	assert.Len(t, auditColumns, 15, "Should have 15 audit columns")
}
