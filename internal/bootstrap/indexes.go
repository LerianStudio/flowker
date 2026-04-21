// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// IndexManager manages MongoDB indexes creation and verification
type IndexManager struct {
	dbManager *DatabaseManager
}

// IndexDefinition defines an index with its configuration
type IndexDefinition struct {
	Collection string
	Name       string
	Keys       []IndexKey
	Unique     bool
	Sparse     bool
}

// IndexKey represents a single field in an index
type IndexKey struct {
	Field string
	Order int // 1 for ascending, -1 for descending
}

// NewIndexManager creates a new IndexManager instance
func NewIndexManager(dbManager *DatabaseManager) *IndexManager {
	return &IndexManager{
		dbManager: dbManager,
	}
}

// CreateIndexes creates all required indexes for Flowker collections
// This operation is idempotent - safe to run multiple times
func (im *IndexManager) CreateIndexes(ctx context.Context) error {
	if im.dbManager == nil || !im.dbManager.IsConnected() {
		return ErrDatabaseNotConnected
	}

	indexes := im.GetIndexDefinitions()

	for _, indexDef := range indexes {
		exists, err := im.indexExists(ctx, indexDef.Collection, indexDef.Name)
		if err != nil {
			return fmt.Errorf("failed to check if index %s exists: %w", indexDef.Name, err)
		}

		if exists {
			// Index already exists, skip creation
			continue
		}

		// Create the index
		if err := im.createIndex(ctx, indexDef); err != nil {
			return fmt.Errorf("failed to create index %s: %w", indexDef.Name, err)
		}
	}

	return nil
}

// VerifyIndexes checks if all required indexes exist
// Returns list of missing index names
func (im *IndexManager) VerifyIndexes(ctx context.Context) ([]string, error) {
	if im.dbManager == nil || !im.dbManager.IsConnected() {
		return nil, ErrDatabaseNotConnected
	}

	indexes := im.GetIndexDefinitions()

	var missing []string

	for _, indexDef := range indexes {
		exists, err := im.indexExists(ctx, indexDef.Collection, indexDef.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check index %s: %w", indexDef.Name, err)
		}

		if !exists {
			missing = append(missing, fmt.Sprintf("%s.%s", indexDef.Collection, indexDef.Name))
		}
	}

	return missing, nil
}

// ListIndexes returns all indexes for a given collection
func (im *IndexManager) ListIndexes(ctx context.Context, collectionName string) ([]string, error) {
	if im.dbManager == nil || !im.dbManager.IsConnected() {
		return nil, ErrDatabaseNotConnected
	}

	db, err := im.dbManager.GetDatabase(ctx)
	if err != nil {
		return nil, err
	}

	collection := db.Collection(collectionName)

	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	var indexes []string

	for cursor.Next(ctx) {
		var index bson.M
		if err := cursor.Decode(&index); err != nil {
			return nil, err
		}

		if name, ok := index["name"].(string); ok {
			indexes = append(indexes, name)
		}
	}

	return indexes, cursor.Err()
}

// GetIndexDefinitions returns all index definitions for Flowker
// Excludes TTL indexes (handled by collection initialization)
func (im *IndexManager) GetIndexDefinitions() []IndexDefinition {
	return []IndexDefinition{
		// Workflows indexes
		{
			Collection: "workflows",
			Name:       "idx_workflows_workflowId",
			Keys:       []IndexKey{{Field: "workflowId", Order: 1}},
			Unique:     true,
			Sparse:     false,
		},
		{
			Collection: "workflows",
			Name:       "idx_workflows_name",
			Keys:       []IndexKey{{Field: "name", Order: 1}},
			Unique:     true,
			Sparse:     false,
		},
		{
			Collection: "workflows",
			Name:       "idx_workflows_status",
			Keys:       []IndexKey{{Field: "status", Order: 1}},
			Unique:     false,
			Sparse:     false,
		},
		{
			Collection: "workflows",
			Name:       "idx_workflows_createdAt",
			Keys:       []IndexKey{{Field: "createdAt", Order: -1}},
			Unique:     false,
			Sparse:     false,
		},

		// Executor Configurations indexes
		{
			Collection: "executor_configurations",
			Name:       "idx_executor_configurations_executorId",
			Keys:       []IndexKey{{Field: "executorId", Order: 1}},
			Unique:     true,
			Sparse:     false,
		},
		{
			Collection: "executor_configurations",
			Name:       "idx_executor_configurations_name",
			Keys:       []IndexKey{{Field: "name", Order: 1}},
			Unique:     true,
			Sparse:     false,
		},
		{
			Collection: "executor_configurations",
			Name:       "idx_executor_configurations_status",
			Keys:       []IndexKey{{Field: "status", Order: 1}},
			Unique:     false,
			Sparse:     false,
		},
		{
			Collection: "executor_configurations",
			Name:       "idx_executor_configurations_createdAt",
			Keys:       []IndexKey{{Field: "createdAt", Order: -1}},
			Unique:     false,
			Sparse:     false,
		},

		// Workflow Executions indexes
		{
			Collection: "workflow_executions",
			Name:       "idx_executions_executionId",
			Keys:       []IndexKey{{Field: "executionId", Order: 1}},
			Unique:     true,
			Sparse:     false,
		},
		{
			Collection: "workflow_executions",
			Name:       "idx_executions_workflowId",
			Keys:       []IndexKey{{Field: "workflowId", Order: 1}},
			Unique:     false,
			Sparse:     false,
		},
		{
			Collection: "workflow_executions",
			Name:       "idx_executions_status",
			Keys:       []IndexKey{{Field: "status", Order: 1}},
			Unique:     false,
			Sparse:     false,
		},
		{
			Collection: "workflow_executions",
			Name:       "idx_executions_startedAt",
			Keys:       []IndexKey{{Field: "startedAt", Order: -1}},
			Unique:     false,
			Sparse:     false,
		},
		{
			Collection: "workflow_executions",
			Name:       "idx_executions_workflowId_status",
			Keys:       []IndexKey{{Field: "workflowId", Order: 1}, {Field: "status", Order: 1}},
			Unique:     false,
			Sparse:     false,
		},
		{
			Collection: "workflow_executions",
			Name:       "idx_executions_idempotencyKey",
			Keys:       []IndexKey{{Field: "idempotencyKey", Order: 1}},
			Unique:     true,
			Sparse:     true,
		},

		// Provider Configurations indexes
		{
			Collection: "provider_configurations",
			Name:       "idx_provider_configurations_providerConfigId",
			Keys:       []IndexKey{{Field: "providerConfigId", Order: 1}},
			Unique:     true,
			Sparse:     false,
		},
		{
			Collection: "provider_configurations",
			Name:       "idx_provider_configurations_name",
			Keys:       []IndexKey{{Field: "name", Order: 1}},
			Unique:     true,
			Sparse:     false,
		},
		{
			Collection: "provider_configurations",
			Name:       "idx_provider_configurations_providerId",
			Keys:       []IndexKey{{Field: "providerId", Order: 1}},
			Unique:     false,
			Sparse:     false,
		},
		{
			Collection: "provider_configurations",
			Name:       "idx_provider_configurations_status",
			Keys:       []IndexKey{{Field: "status", Order: 1}},
			Unique:     false,
			Sparse:     false,
		},
		{
			Collection: "provider_configurations",
			Name:       "idx_provider_configurations_createdAt",
			Keys:       []IndexKey{{Field: "createdAt", Order: -1}},
			Unique:     false,
			Sparse:     false,
		},

		// Audit Entries indexes (time-series collection - limited indexing)
		{
			Collection: "audit_entries",
			Name:       "idx_audit_entityId",
			Keys:       []IndexKey{{Field: "metadata.entityId", Order: 1}},
			Unique:     false,
			Sparse:     true,
		},
		{
			Collection: "audit_entries",
			Name:       "idx_audit_entityType",
			Keys:       []IndexKey{{Field: "metadata.entityType", Order: 1}},
			Unique:     false,
			Sparse:     true,
		},
	}
}

// GetDatabaseManager returns the underlying database manager
func (im *IndexManager) GetDatabaseManager() *DatabaseManager {
	return im.dbManager
}

// indexExists checks if an index exists in a collection
func (im *IndexManager) indexExists(ctx context.Context, collectionName, indexName string) (bool, error) {
	db, err := im.dbManager.GetDatabase(ctx)
	if err != nil {
		return false, err
	}

	collection := db.Collection(collectionName)

	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return false, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var index bson.M
		if err := cursor.Decode(&index); err != nil {
			return false, err
		}

		if name, ok := index["name"].(string); ok && name == indexName {
			return true, nil
		}
	}

	return false, cursor.Err()
}

// createIndex creates a single index
func (im *IndexManager) createIndex(ctx context.Context, indexDef IndexDefinition) error {
	db, err := im.dbManager.GetDatabase(ctx)
	if err != nil {
		return err
	}

	collection := db.Collection(indexDef.Collection)

	// Convert IndexKey slice to bson.D
	keys := bson.D{}
	for _, key := range indexDef.Keys {
		keys = append(keys, bson.E{Key: key.Field, Value: key.Order})
	}

	indexModel := mongo.IndexModel{
		Keys: keys,
		Options: options.Index().
			SetName(indexDef.Name).
			SetUnique(indexDef.Unique).
			SetSparse(indexDef.Sparse),
	}

	_, err = collection.Indexes().CreateOne(ctx, indexModel)

	return err
}
