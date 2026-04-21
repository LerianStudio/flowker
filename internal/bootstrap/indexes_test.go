// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package bootstrap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIndexManager(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	assert.NotNil(t, indexManager, "IndexManager should not be nil")
}

func TestNewIndexManager_WithNilDbManager(t *testing.T) {
	indexManager := NewIndexManager(nil)

	assert.NotNil(t, indexManager, "IndexManager should not be nil even with nil dbManager")
}

func TestIndexManager_GetDatabaseManager(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	assert.Equal(t, dbManager, indexManager.GetDatabaseManager())
}

func TestIndexManager_GetIndexDefinitions(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	definitions := indexManager.GetIndexDefinitions()

	// Should have 21 indexes across 5 collections
	assert.Len(t, definitions, 21)

	// Count indexes per collection
	collectionCounts := make(map[string]int)
	for _, def := range definitions {
		collectionCounts[def.Collection]++
	}

	assert.Equal(t, 4, collectionCounts["workflows"], "workflows should have 4 indexes")
	assert.Equal(t, 4, collectionCounts["executor_configurations"], "executor_configurations should have 4 indexes")
	assert.Equal(t, 5, collectionCounts["provider_configurations"], "provider_configurations should have 5 indexes")
	assert.Equal(t, 6, collectionCounts["workflow_executions"], "workflow_executions should have 6 indexes")
	assert.Equal(t, 2, collectionCounts["audit_entries"], "audit_entries should have 2 indexes")
}

func TestIndexManager_GetIndexDefinitions_WorkflowsIndexes(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	definitions := indexManager.GetIndexDefinitions()

	// Find workflows indexes
	var workflowsIndexes []IndexDefinition
	for _, def := range definitions {
		if def.Collection == "workflows" {
			workflowsIndexes = append(workflowsIndexes, def)
		}
	}

	require.Len(t, workflowsIndexes, 4)

	// Verify workflowId index is unique
	var workflowIdIndex *IndexDefinition
	for i := range workflowsIndexes {
		if workflowsIndexes[i].Name == "idx_workflows_workflowId" {
			workflowIdIndex = &workflowsIndexes[i]
			break
		}
	}

	require.NotNil(t, workflowIdIndex)
	assert.True(t, workflowIdIndex.Unique, "workflowId index should be unique")
	assert.False(t, workflowIdIndex.Sparse, "workflowId index should not be sparse")
}

func TestIndexManager_GetIndexDefinitions_ExecutorConfigurationsIndexes(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	definitions := indexManager.GetIndexDefinitions()

	// Find executor_configurations indexes
	var executorConfigIndexes []IndexDefinition
	for _, def := range definitions {
		if def.Collection == "executor_configurations" {
			executorConfigIndexes = append(executorConfigIndexes, def)
		}
	}

	require.Len(t, executorConfigIndexes, 4)

	// Verify executorId index is unique
	var executorIdIndex *IndexDefinition
	for i := range executorConfigIndexes {
		if executorConfigIndexes[i].Name == "idx_executor_configurations_executorId" {
			executorIdIndex = &executorConfigIndexes[i]
			break
		}
	}

	require.NotNil(t, executorIdIndex)
	assert.True(t, executorIdIndex.Unique, "executorId index should be unique")

	// Verify name index is unique
	var nameIndex *IndexDefinition
	for i := range executorConfigIndexes {
		if executorConfigIndexes[i].Name == "idx_executor_configurations_name" {
			nameIndex = &executorConfigIndexes[i]
			break
		}
	}

	require.NotNil(t, nameIndex)
	assert.True(t, nameIndex.Unique, "name index should be unique")
}

func TestIndexManager_GetIndexDefinitions_ProviderConfigurationsIndexes(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	definitions := indexManager.GetIndexDefinitions()

	// Find provider_configurations indexes
	var providerConfigIndexes []IndexDefinition
	for _, def := range definitions {
		if def.Collection == "provider_configurations" {
			providerConfigIndexes = append(providerConfigIndexes, def)
		}
	}

	require.Len(t, providerConfigIndexes, 5)

	// Verify providerConfigId index is unique
	var providerConfigIdIndex *IndexDefinition
	for i := range providerConfigIndexes {
		if providerConfigIndexes[i].Name == "idx_provider_configurations_providerConfigId" {
			providerConfigIdIndex = &providerConfigIndexes[i]
			break
		}
	}

	require.NotNil(t, providerConfigIdIndex, "providerConfigId index should exist")
	assert.True(t, providerConfigIdIndex.Unique, "providerConfigId index should be unique")
	assert.False(t, providerConfigIdIndex.Sparse, "providerConfigId index should not be sparse")

	// Verify name index is unique
	var nameIndex *IndexDefinition
	for i := range providerConfigIndexes {
		if providerConfigIndexes[i].Name == "idx_provider_configurations_name" {
			nameIndex = &providerConfigIndexes[i]
			break
		}
	}

	require.NotNil(t, nameIndex)
	assert.True(t, nameIndex.Unique, "name index should be unique")
	assert.False(t, nameIndex.Sparse, "name index should not be sparse")

	// Verify providerId index exists and is not unique
	var providerIdIndex *IndexDefinition
	for i := range providerConfigIndexes {
		if providerConfigIndexes[i].Name == "idx_provider_configurations_providerId" {
			providerIdIndex = &providerConfigIndexes[i]
			break
		}
	}

	require.NotNil(t, providerIdIndex)
	assert.False(t, providerIdIndex.Unique, "providerId index should not be unique")
}

func TestIndexManager_GetIndexDefinitions_ExecutionsIndexes(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	definitions := indexManager.GetIndexDefinitions()

	// Find workflow_executions indexes
	var execIndexes []IndexDefinition
	for _, def := range definitions {
		if def.Collection == "workflow_executions" {
			execIndexes = append(execIndexes, def)
		}
	}

	require.Len(t, execIndexes, 6)

	// Verify executionId index is unique
	var execIdIndex *IndexDefinition
	for i := range execIndexes {
		if execIndexes[i].Name == "idx_executions_executionId" {
			execIdIndex = &execIndexes[i]
			break
		}
	}

	require.NotNil(t, execIdIndex)
	assert.True(t, execIdIndex.Unique, "executionId index should be unique")

	// Verify compound index exists
	var compoundIndex *IndexDefinition
	for i := range execIndexes {
		if execIndexes[i].Name == "idx_executions_workflowId_status" {
			compoundIndex = &execIndexes[i]
			break
		}
	}

	require.NotNil(t, compoundIndex, "compound index workflowId_status should exist")
	assert.False(t, compoundIndex.Unique, "compound index should not be unique")
	assert.Len(t, compoundIndex.Keys, 2, "compound index should have 2 keys")

	// Verify idempotencyKey index is unique and sparse
	var idempotencyIndex *IndexDefinition
	for i := range execIndexes {
		if execIndexes[i].Name == "idx_executions_idempotencyKey" {
			idempotencyIndex = &execIndexes[i]
			break
		}
	}

	require.NotNil(t, idempotencyIndex, "idempotencyKey index should exist")
	assert.True(t, idempotencyIndex.Unique, "idempotencyKey index should be unique")
	assert.True(t, idempotencyIndex.Sparse, "idempotencyKey index should be sparse")
}

func TestIndexManager_GetIndexDefinitions_AuditEntriesIndexes(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	definitions := indexManager.GetIndexDefinitions()

	// Find audit_entries indexes
	var auditIndexes []IndexDefinition
	for _, def := range definitions {
		if def.Collection == "audit_entries" {
			auditIndexes = append(auditIndexes, def)
		}
	}

	require.Len(t, auditIndexes, 2)

	// All audit indexes should be sparse (time-series collection)
	for _, idx := range auditIndexes {
		assert.True(t, idx.Sparse, "audit index %s should be sparse", idx.Name)
		assert.False(t, idx.Unique, "audit index %s should not be unique", idx.Name)
	}
}

func TestIndexManager_CreateIndexes_ErrorWhenNotConnected(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	// Database is not connected, should return error
	err := indexManager.CreateIndexes(context.Background())
	require.ErrorIs(t, err, ErrDatabaseNotConnected)
}

func TestIndexManager_VerifyIndexes_ErrorWhenNotConnected(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	// Database is not connected, should return error
	missing, err := indexManager.VerifyIndexes(context.Background())
	require.ErrorIs(t, err, ErrDatabaseNotConnected)
	assert.Nil(t, missing)
}

func TestIndexManager_ListIndexes_ErrorWhenNotConnected(t *testing.T) {
	dbManager := NewDatabaseManager()
	indexManager := NewIndexManager(dbManager)

	// Database is not connected, should return error
	indexes, err := indexManager.ListIndexes(context.Background(), "workflows")
	require.ErrorIs(t, err, ErrDatabaseNotConnected)
	assert.Nil(t, indexes)
}

func TestIndexManager_CreateIndexes_ErrorWithNilDbManager(t *testing.T) {
	indexManager := NewIndexManager(nil)

	err := indexManager.CreateIndexes(context.Background())
	require.ErrorIs(t, err, ErrDatabaseNotConnected)
}

func TestIndexDefinition_Fields(t *testing.T) {
	def := IndexDefinition{
		Collection: "test_collection",
		Name:       "idx_test",
		Keys:       []IndexKey{{Field: "testField", Order: 1}},
		Unique:     true,
		Sparse:     false,
	}

	assert.Equal(t, "test_collection", def.Collection)
	assert.Equal(t, "idx_test", def.Name)
	assert.Len(t, def.Keys, 1)
	assert.Equal(t, "testField", def.Keys[0].Field)
	assert.Equal(t, 1, def.Keys[0].Order)
	assert.True(t, def.Unique)
	assert.False(t, def.Sparse)
}

func TestIndexKey_Fields(t *testing.T) {
	key := IndexKey{
		Field: "createdAt",
		Order: -1, // Descending
	}

	assert.Equal(t, "createdAt", key.Field)
	assert.Equal(t, -1, key.Order)
}
