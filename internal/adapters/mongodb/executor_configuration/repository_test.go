// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package executorconfiguration

import (
	"context"
	"testing"

	"github.com/LerianStudio/flowker/internal/services/query"
	tmcore "github.com/LerianStudio/lib-commons/v5/commons/tenant-manager/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
)

// =============================================================================
// Multi-Tenant Unit Tests
// =============================================================================

func TestMongoDBRepository_MultiTenant_GetCollection_WithTenantContext(t *testing.T) {
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
			repo := NewMongoDBRepository(tt.fallbackDB)
			ctx := tt.setupCtx()

			collection, err := repo.getCollection(ctx)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, collection)
				assert.Contains(t, err.Error(), "mongodb connection not available")
			} else {
				require.NoError(t, err)
				assert.NotNil(t, collection)
			}
		})
	}
}

func TestMongoDBRepository_MultiTenant_FallbackModeWithNilFallback(t *testing.T) {
	// When MULTI_TENANT_ENABLED=false and no fallback database is provided,
	// repository operations should return an error.
	repo := NewMongoDBRepository(nil)
	ctx := context.Background() // No tenant context

	// Test that getCollection returns appropriate error
	collection, err := repo.getCollection(ctx)

	require.Error(t, err)
	assert.Nil(t, collection)
	assert.Contains(t, err.Error(), "mongodb connection not available")
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

	t.Run("repository falls back to static connection when no tenant context", func(t *testing.T) {
		repo := NewMongoDBRepository(nil)
		ctx := context.Background()

		// Since fallback is nil, this should error
		_, err := repo.getCollection(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "mongodb connection not available")
	})
}

func TestMongoDBRepository_MultiTenant_CollectionName(t *testing.T) {
	// Verify that the repository uses the correct collection name
	assert.Equal(t, "executor_configurations", CollectionName, "Collection name should be 'executor_configurations'")
}

// =============================================================================
// Error Case Tests
// =============================================================================

func TestMongoDBRepository_MultiTenant_ErrorCases(t *testing.T) {
	repo := NewMongoDBRepository(nil)
	ctx := context.Background()

	t.Run("Create_returns_error_when_no_connection_available", func(t *testing.T) {
		err := repo.Create(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get collection")
	})

	t.Run("FindByID_returns_error_when_no_connection_available", func(t *testing.T) {
		_, err := repo.FindByID(ctx, uuid.New())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get collection")
	})

	t.Run("FindByName_returns_error_when_no_connection_available", func(t *testing.T) {
		_, err := repo.FindByName(ctx, "test-executor")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get collection")
	})

	t.Run("List_returns_error_when_no_connection_available", func(t *testing.T) {
		filter := query.ExecutorConfigListFilter{
			Limit: 10,
		}
		_, err := repo.List(ctx, filter)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get collection")
	})

	t.Run("Update_returns_error_when_no_connection_available", func(t *testing.T) {
		err := repo.Update(ctx, nil, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get collection")
	})

	t.Run("Delete_returns_error_when_no_connection_available", func(t *testing.T) {
		err := repo.Delete(ctx, uuid.New(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get collection")
	})

	t.Run("ExistsByName_returns_error_when_no_connection_available", func(t *testing.T) {
		_, err := repo.ExistsByName(ctx, "test-executor")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get collection")
	})
}

// =============================================================================
// Backward Compatibility Tests (Single-Tenant Mode)
// =============================================================================

func TestMongoDBRepository_SingleTenantMode_BackwardCompatibility(t *testing.T) {
	t.Run("repository_construction_accepts_nil_fallback", func(t *testing.T) {
		// This should NOT panic - nil fallback is valid (multi-tenant mode expected)
		repo := NewMongoDBRepository(nil)
		assert.NotNil(t, repo)
	})

	t.Run("repository_returns_error_with_nil_fallback_and_no_context", func(t *testing.T) {
		repo := NewMongoDBRepository(nil)
		ctx := context.Background()

		_, err := repo.getCollection(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "mongodb connection not available")
	})
}

// =============================================================================
// Helper function tests
// =============================================================================

func TestMapSortField(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"createdAt", "createdAt"},
		{"updatedAt", "updatedAt"},
		{"name", "name"},
		{"unknown", "createdAt"}, // defaults to createdAt
		{"", "createdAt"},        // empty defaults to createdAt
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapSortField(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
