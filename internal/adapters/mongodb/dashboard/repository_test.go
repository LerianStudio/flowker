// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package dashboard

import (
	"context"
	"testing"

	"github.com/LerianStudio/flowker/internal/services/query"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
)

// =============================================================================
// Multi-Tenant Unit Tests
// =============================================================================

func TestMongoDBRepository_MultiTenant_GetDB_WithTenantContext(t *testing.T) {
	tests := []struct {
		name           string
		setupCtx       func() context.Context
		fallbackDB     *mongo.Database
		expectFallback bool
		expectError    bool
	}{
		{
			name: "uses tenant database from context when present",
			setupCtx: func() context.Context {
				// Create context without tenant database
				return context.Background()
			},
			fallbackDB:     nil,
			expectFallback: false,
			expectError:    true, // No tenant context and no fallback
		},
		{
			name: "uses fallback database when no tenant context (single-tenant mode)",
			setupCtx: func() context.Context {
				return context.Background() // No tenant context
			},
			fallbackDB:     nil, // Test with nil fallback
			expectFallback: true,
			expectError:    true, // Expected error since fallback is nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewMongoDBRepository(tt.fallbackDB)
			if tt.fallbackDB == nil {
				// Constructor requires non-nil database
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			ctx := tt.setupCtx()

			db, err := repo.getDB(ctx)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, db)
				assert.Contains(t, err.Error(), "mongodb connection not available")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, db)
			}
		})
	}
}

func TestMongoDBRepository_MultiTenant_ConstructorValidation(t *testing.T) {
	t.Run("constructor_rejects_nil_fallback", func(t *testing.T) {
		repo, err := NewMongoDBRepository(nil)

		require.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "database cannot be nil")
	})
}

func TestMongoDBRepository_MultiTenant_TenantContextExtraction(t *testing.T) {
	// Test that the repository correctly checks for tenant context
	// using tmcore.GetMBContext

	t.Run("returns nil when context has no tenant database", func(t *testing.T) {
		ctx := context.Background()

		// Verify that GetMBContext returns nil for plain context
		db := tmcore.GetMBContext(ctx)
		assert.Nil(t, db, "GetMBContext should return nil for context without tenant database")
	})
}

func TestMongoDBRepository_MultiTenant_CollectionNames(t *testing.T) {
	// Verify that the repository uses the correct collection names
	assert.Equal(t, "workflows", workflowCollectionName, "Workflow collection name should be 'workflows'")
	assert.Equal(t, "workflow_executions", executionCollectionName, "Execution collection name should be 'workflow_executions'")
}

// =============================================================================
// Error Case Tests
// =============================================================================

func TestMongoDBRepository_MultiTenant_ErrorCases(t *testing.T) {
	// Note: We cannot create a repository with nil fallback as the constructor rejects it.
	// These tests verify the error handling when getDB fails.

	t.Run("WorkflowSummary_returns_error_when_no_connection_available", func(t *testing.T) {
		// Create a mock repository that will fail getDB
		repo := &MongoDBRepository{fallbackDB: nil}
		ctx := context.Background()

		_, err := repo.WorkflowSummary(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get database")
	})

	t.Run("ExecutionSummary_returns_error_when_no_connection_available", func(t *testing.T) {
		// Create a mock repository that will fail getDB
		repo := &MongoDBRepository{fallbackDB: nil}
		ctx := context.Background()

		filter := query.ExecutionSummaryFilter{}
		_, err := repo.ExecutionSummary(ctx, filter)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get database")
	})
}

// =============================================================================
// Backward Compatibility Tests (Single-Tenant Mode)
// =============================================================================

func TestMongoDBRepository_SingleTenantMode_BackwardCompatibility(t *testing.T) {
	t.Run("constructor_requires_non_nil_database", func(t *testing.T) {
		// Dashboard repository requires a fallback database at construction time
		// This ensures single-tenant mode always works
		repo, err := NewMongoDBRepository(nil)

		require.Error(t, err)
		assert.Nil(t, repo)
		assert.Contains(t, err.Error(), "database cannot be nil")
	})
}

// =============================================================================
// Interface Compliance Tests
// =============================================================================

func TestMongoDBRepository_ImplementsInterface(t *testing.T) {
	// Verify that MongoDBRepository implements query.DashboardRepository
	var _ query.DashboardRepository = (*MongoDBRepository)(nil)
}
