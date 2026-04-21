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

func TestNewCollectionManager(t *testing.T) {
	dbManager := NewDatabaseManager()
	require.NotNil(t, dbManager)

	collectionManager := NewCollectionManager(dbManager)
	assert.NotNil(t, collectionManager, "CollectionManager should not be nil")
	assert.Equal(t, dbManager, collectionManager.GetDatabaseManager())
}

func TestNewCollectionManager_WithNilDbManager(t *testing.T) {
	collectionManager := NewCollectionManager(nil)
	assert.NotNil(t, collectionManager, "CollectionManager should not be nil even with nil dbManager")
	assert.Nil(t, collectionManager.GetDatabaseManager())
}

func TestCollectionManager_GetCollectionDefinitions(t *testing.T) {
	dbManager := NewDatabaseManager()
	collectionManager := NewCollectionManager(dbManager)

	definitions := collectionManager.GetCollectionDefinitions()

	// Should have 5 collections
	assert.Len(t, definitions, 5)

	// Verify collection names
	expectedNames := []string{"workflows", "executor_configurations", "provider_configurations", "workflow_executions", "audit_entries"}
	for i, def := range definitions {
		assert.Equal(t, expectedNames[i], def.Name)
	}
}

func TestCollectionManager_GetCollectionDefinitions_WorkflowsHasValidator(t *testing.T) {
	dbManager := NewDatabaseManager()
	collectionManager := NewCollectionManager(dbManager)

	definitions := collectionManager.GetCollectionDefinitions()

	// Find workflows collection
	var workflowsDef *CollectionDefinition
	for _, def := range definitions {
		if def.Name == "workflows" {
			workflowsDef = &def
			break
		}
	}

	require.NotNil(t, workflowsDef)
	assert.NotNil(t, workflowsDef.Validator, "workflows should have validator")
	assert.Nil(t, workflowsDef.TimeSeries, "workflows should not be time-series")
}

func TestCollectionManager_GetCollectionDefinitions_ExecutorConfigurationsHasValidator(t *testing.T) {
	dbManager := NewDatabaseManager()
	collectionManager := NewCollectionManager(dbManager)

	definitions := collectionManager.GetCollectionDefinitions()

	// Find executor_configurations collection
	var executorConfigsDef *CollectionDefinition
	for _, def := range definitions {
		if def.Name == "executor_configurations" {
			executorConfigsDef = &def
			break
		}
	}

	require.NotNil(t, executorConfigsDef)
	assert.NotNil(t, executorConfigsDef.Validator, "executor_configurations should have validator")
	assert.Nil(t, executorConfigsDef.TimeSeries, "executor_configurations should not be time-series")
}

func TestCollectionManager_GetCollectionDefinitions_ProviderConfigurationsHasValidator(t *testing.T) {
	dbManager := NewDatabaseManager()
	collectionManager := NewCollectionManager(dbManager)

	definitions := collectionManager.GetCollectionDefinitions()

	var providerConfigsDef *CollectionDefinition
	for _, def := range definitions {
		if def.Name == "provider_configurations" {
			providerConfigsDef = &def
			break
		}
	}

	require.NotNil(t, providerConfigsDef)
	assert.NotNil(t, providerConfigsDef.Validator, "provider_configurations should have validator")
	assert.Nil(t, providerConfigsDef.TimeSeries, "provider_configurations should not be time-series")
}

func TestCollectionManager_GetCollectionDefinitions_WorkflowExecutionsHasValidator(t *testing.T) {
	dbManager := NewDatabaseManager()
	collectionManager := NewCollectionManager(dbManager)

	definitions := collectionManager.GetCollectionDefinitions()

	// Find workflow_executions collection
	var execDef *CollectionDefinition
	for _, def := range definitions {
		if def.Name == "workflow_executions" {
			execDef = &def
			break
		}
	}

	require.NotNil(t, execDef)
	assert.NotNil(t, execDef.Validator, "workflow_executions should have validator")
	assert.Nil(t, execDef.TimeSeries, "workflow_executions should not be time-series")
}

func TestCollectionManager_GetCollectionDefinitions_AuditEntriesIsTimeSeries(t *testing.T) {
	dbManager := NewDatabaseManager()
	collectionManager := NewCollectionManager(dbManager)

	definitions := collectionManager.GetCollectionDefinitions()

	// Find audit_entries collection
	var auditDef *CollectionDefinition
	for _, def := range definitions {
		if def.Name == "audit_entries" {
			auditDef = &def
			break
		}
	}

	require.NotNil(t, auditDef)
	assert.Nil(t, auditDef.Validator, "audit_entries should not have validator")
	require.NotNil(t, auditDef.TimeSeries, "audit_entries should be time-series")

	// Verify time-series configuration
	assert.Equal(t, "timestamp", auditDef.TimeSeries.TimeField)
	assert.Equal(t, "metadata", auditDef.TimeSeries.MetaField)
	assert.Equal(t, "minutes", auditDef.TimeSeries.Granularity)
	assert.Equal(t, int64(220752000), auditDef.TimeSeries.ExpireAfter) // 7 years
}

func TestCollectionDefinition_Fields(t *testing.T) {
	def := CollectionDefinition{
		Name: "test_collection",
		TimeSeries: &TimeSeriesConfig{
			TimeField:   "ts",
			MetaField:   "meta",
			Granularity: "hours",
			ExpireAfter: 3600,
		},
	}

	assert.Equal(t, "test_collection", def.Name)
	assert.NotNil(t, def.TimeSeries)
	assert.Equal(t, "ts", def.TimeSeries.TimeField)
	assert.Equal(t, "meta", def.TimeSeries.MetaField)
	assert.Equal(t, "hours", def.TimeSeries.Granularity)
	assert.Equal(t, int64(3600), def.TimeSeries.ExpireAfter)
}

func TestTimeSeriesConfig_Fields(t *testing.T) {
	config := TimeSeriesConfig{
		TimeField:   "created_at",
		MetaField:   "metadata",
		Granularity: "seconds",
		ExpireAfter: 86400, // 1 day
	}

	assert.Equal(t, "created_at", config.TimeField)
	assert.Equal(t, "metadata", config.MetaField)
	assert.Equal(t, "seconds", config.Granularity)
	assert.Equal(t, int64(86400), config.ExpireAfter)
}

func TestCollectionManager_InitializeCollections_ErrorWhenNotConnected(t *testing.T) {
	dbManager := NewDatabaseManager()
	collectionManager := NewCollectionManager(dbManager)

	// Database is not connected, should return error
	err := collectionManager.InitializeCollections(context.Background())
	require.ErrorIs(t, err, ErrDatabaseNotConnected)
}

func TestCollectionManager_VerifyCollections_ErrorWhenNotConnected(t *testing.T) {
	dbManager := NewDatabaseManager()
	collectionManager := NewCollectionManager(dbManager)

	// Database is not connected, should return error
	missing, err := collectionManager.VerifyCollections(context.Background())
	require.ErrorIs(t, err, ErrDatabaseNotConnected)
	assert.Nil(t, missing)
}

func TestCollectionManager_InitializeCollections_ErrorWithNilDbManager(t *testing.T) {
	collectionManager := NewCollectionManager(nil)

	err := collectionManager.InitializeCollections(context.Background())
	require.ErrorIs(t, err, ErrDatabaseNotConnected)
}

func TestCollectionManager_VerifyCollections_ErrorWithNilDbManager(t *testing.T) {
	collectionManager := NewCollectionManager(nil)

	missing, err := collectionManager.VerifyCollections(context.Background())
	require.ErrorIs(t, err, ErrDatabaseNotConnected)
	assert.Nil(t, missing)
}
