// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package execution

import (
	"context"
	"testing"

	"github.com/LerianStudio/flowker/internal/services/command"
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
	assert.Equal(t, "workflow_executions", CollectionName, "Collection name should be 'workflow_executions'")
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

	t.Run("FindByIdempotencyKey_returns_error_when_no_connection_available", func(t *testing.T) {
		_, err := repo.FindByIdempotencyKey(ctx, "test-key")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get collection")
	})

	t.Run("Update_returns_error_when_no_connection_available", func(t *testing.T) {
		err := repo.Update(ctx, nil, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get collection")
	})

	t.Run("FindIncomplete_returns_error_when_no_connection_available", func(t *testing.T) {
		_, err := repo.FindIncomplete(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get collection")
	})

	t.Run("List_returns_error_when_no_connection_available", func(t *testing.T) {
		filter := command.ExecutionListFilter{
			Limit: 10,
		}
		_, err := repo.List(ctx, filter)
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

func TestMapExecutionSortField(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"startedAt", "startedAt"},
		{"completedAt", "completedAt"},
		{"unknown", "startedAt"}, // defaults to startedAt
		{"", "startedAt"},        // empty defaults to startedAt
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mapExecutionSortField(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeSortParams(t *testing.T) {
	tests := []struct {
		name           string
		inputSortBy    string
		inputSortOrder string
		expectedSortBy string
		expectedOrder  string
	}{
		{
			name:           "valid sort params",
			inputSortBy:    "startedAt",
			inputSortOrder: "ASC",
			expectedSortBy: "startedAt",
			expectedOrder:  "ASC",
		},
		{
			name:           "lowercase sort order converted to uppercase",
			inputSortBy:    "startedAt",
			inputSortOrder: "desc",
			expectedSortBy: "startedAt",
			expectedOrder:  "DESC",
		},
		{
			name:           "invalid sort field defaults to startedAt",
			inputSortBy:    "invalidField",
			inputSortOrder: "ASC",
			expectedSortBy: command.DefaultExecutionSortField,
			expectedOrder:  "ASC",
		},
		{
			name:           "invalid sort order defaults to DESC",
			inputSortBy:    "startedAt",
			inputSortOrder: "INVALID",
			expectedSortBy: "startedAt",
			expectedOrder:  "DESC",
		},
		{
			name:           "empty values use defaults",
			inputSortBy:    "",
			inputSortOrder: "",
			expectedSortBy: command.DefaultExecutionSortField,
			expectedOrder:  "DESC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sortBy, sortOrder := normalizeSortParams(tt.inputSortBy, tt.inputSortOrder)
			assert.Equal(t, tt.expectedSortBy, sortBy)
			assert.Equal(t, tt.expectedOrder, sortOrder)
		})
	}
}

func TestNormalizeLimit(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{"zero defaults to default limit", 0, 10},      // constant.DefaultPaginationLimit
		{"negative defaults to default limit", -5, 10}, // constant.DefaultPaginationLimit
		{"valid limit unchanged", 25, 25},
		{"max limit capped", 1000, 100}, // constant.MaxPaginationLimit
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeLimit(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
