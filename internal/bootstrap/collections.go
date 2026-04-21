// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CollectionManager manages database collections initialization
type CollectionManager struct {
	dbManager *DatabaseManager
}

// CollectionDefinition defines a collection with its configuration
type CollectionDefinition struct {
	Name       string
	TimeSeries *TimeSeriesConfig
	Validator  *bson.M
}

// TimeSeriesConfig holds time-series collection configuration
type TimeSeriesConfig struct {
	TimeField   string
	MetaField   string
	Granularity string
	ExpireAfter int64 // seconds
}

// NewCollectionManager creates a new CollectionManager instance
func NewCollectionManager(dbManager *DatabaseManager) *CollectionManager {
	return &CollectionManager{
		dbManager: dbManager,
	}
}

// InitializeCollections creates all required collections if they don't exist
// This operation is idempotent - safe to run multiple times
func (cm *CollectionManager) InitializeCollections(ctx context.Context) error {
	if cm.dbManager == nil || !cm.dbManager.IsConnected() {
		return ErrDatabaseNotConnected
	}

	collections := cm.GetCollectionDefinitions()

	for _, collDef := range collections {
		exists, err := cm.collectionExists(ctx, collDef.Name)
		if err != nil {
			return fmt.Errorf("failed to check if collection %s exists: %w", collDef.Name, err)
		}

		if exists {
			// Collection already exists, skip creation
			continue
		}

		// Create collection with appropriate configuration
		if err := cm.createCollection(ctx, collDef); err != nil {
			return fmt.Errorf("failed to create collection %s: %w", collDef.Name, err)
		}
	}

	return nil
}

// VerifyCollections checks if all required collections exist
// Returns list of missing collection names
func (cm *CollectionManager) VerifyCollections(ctx context.Context) ([]string, error) {
	if cm.dbManager == nil || !cm.dbManager.IsConnected() {
		return nil, ErrDatabaseNotConnected
	}

	collections := cm.GetCollectionDefinitions()

	var missing []string

	for _, collDef := range collections {
		exists, err := cm.collectionExists(ctx, collDef.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to check collection %s: %w", collDef.Name, err)
		}

		if !exists {
			missing = append(missing, collDef.Name)
		}
	}

	return missing, nil
}

// GetCollectionDefinitions returns all collection definitions for Flowker
func (cm *CollectionManager) GetCollectionDefinitions() []CollectionDefinition {
	return []CollectionDefinition{
		{
			Name: "workflows",
			Validator: &bson.M{
				"$jsonSchema": bson.M{
					"bsonType": "object",
					"required": []string{"workflowId", "name", "status", "createdAt"},
					"properties": bson.M{
						"workflowId": bson.M{"bsonType": "string"},
						"name":       bson.M{"bsonType": "string"},
						"status":     bson.M{"bsonType": "string"},
						"createdAt":  bson.M{"bsonType": "date"},
					},
				},
			},
		},
		{
			Name: "executor_configurations",
			Validator: &bson.M{
				"$jsonSchema": bson.M{
					"bsonType": "object",
					"required": []string{"executorId", "name", "status", "createdAt"},
					"properties": bson.M{
						"executorId": bson.M{"bsonType": "string"},
						"name":       bson.M{"bsonType": "string"},
						"status":     bson.M{"bsonType": "string"},
						"createdAt":  bson.M{"bsonType": "date"},
					},
				},
			},
		},
		{
			Name: "provider_configurations",
			Validator: &bson.M{
				"$jsonSchema": bson.M{
					"bsonType": "object",
					"required": []string{"name", "providerId", "status", "createdAt"},
					"properties": bson.M{
						"name":       bson.M{"bsonType": "string"},
						"providerId": bson.M{"bsonType": "string"},
						"status":     bson.M{"bsonType": "string"},
						"createdAt":  bson.M{"bsonType": "date"},
					},
				},
			},
		},
		{
			Name: "workflow_executions",
			Validator: &bson.M{
				"$jsonSchema": bson.M{
					"bsonType": "object",
					"required": []string{"executionId", "workflowId", "status", "startedAt"},
					"properties": bson.M{
						"executionId": bson.M{"bsonType": "string"},
						"workflowId":  bson.M{"bsonType": "string"},
						"status":      bson.M{"bsonType": "string"},
						"startedAt":   bson.M{"bsonType": "date"},
					},
				},
			},
		},
		{
			Name: "audit_entries",
			TimeSeries: &TimeSeriesConfig{
				TimeField:   "timestamp",
				MetaField:   "metadata",
				Granularity: "minutes",
				ExpireAfter: 220752000, // 7 years in seconds
			},
		},
	}
}

// collectionExists checks if a collection exists in the database
func (cm *CollectionManager) collectionExists(ctx context.Context, collectionName string) (bool, error) {
	db, err := cm.dbManager.GetDatabase(ctx)
	if err != nil {
		return false, err
	}

	collections, err := db.ListCollectionNames(ctx, bson.M{"name": collectionName})
	if err != nil {
		return false, err
	}

	return len(collections) > 0, nil
}

// createCollection creates a collection with appropriate configuration
func (cm *CollectionManager) createCollection(ctx context.Context, collDef CollectionDefinition) error {
	db, err := cm.dbManager.GetDatabase(ctx)
	if err != nil {
		return err
	}

	// Build collection options
	opts := options.CreateCollection()

	// Configure time-series collection if specified
	if collDef.TimeSeries != nil {
		opts.SetTimeSeriesOptions(options.TimeSeries().
			SetTimeField(collDef.TimeSeries.TimeField).
			SetMetaField(collDef.TimeSeries.MetaField).
			SetGranularity(collDef.TimeSeries.Granularity))

		// Set TTL index for expiration
		opts.SetExpireAfterSeconds(collDef.TimeSeries.ExpireAfter)
	}

	// Configure validator if specified
	if collDef.Validator != nil {
		opts.SetValidator(*collDef.Validator)
	}

	// Create the collection
	if err := db.CreateCollection(ctx, collDef.Name, opts); err != nil {
		// Check if error is "collection already exists" (NamespaceExists, code 48) - make idempotent
		var cmdErr mongo.CommandError
		if errors.As(err, &cmdErr) && cmdErr.Code == 48 {
			return nil
		}

		return err
	}

	return nil
}

// GetDatabaseManager returns the underlying database manager
func (cm *CollectionManager) GetDatabaseManager() *DatabaseManager {
	return cm.dbManager
}
