// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestMongoDBConnection tests basic MongoDB connectivity using testcontainers.
// This serves as a template for integration tests that require MongoDB.
func TestMongoDBConnection(t *testing.T) {
	if os.Getenv("DISABLE_TESTCONTAINERS") == "true" {
		t.Skip("Skipping testcontainers test - DISABLE_TESTCONTAINERS is set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start MongoDB container
	mongoContainer, err := mongodb.Run(ctx, "mongo:7.0")
	require.NoError(t, err, "Failed to start MongoDB container")

	defer func() {
		terminateCtx, terminateCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer terminateCancel()

		if err := mongoContainer.Terminate(terminateCtx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Get connection string
	connectionString, err := mongoContainer.ConnectionString(ctx)
	require.NoError(t, err, "Failed to get connection string")

	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(connectionString)
	client, err := mongo.Connect(ctx, clientOptions)
	require.NoError(t, err, "Failed to connect to MongoDB")

	defer func() {
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer disconnectCancel()

		if err := client.Disconnect(disconnectCtx); err != nil {
			t.Logf("Failed to disconnect: %v", err)
		}
	}()

	// Ping the database
	err = client.Ping(ctx, nil)
	require.NoError(t, err, "Failed to ping MongoDB")

	// Test basic CRUD operations
	t.Run("InsertAndFind", func(t *testing.T) {
		db := client.Database("test_db")
		collection := db.Collection("test_collection")

		// Insert a document
		doc := bson.M{
			"name":       "test",
			"created_at": time.Now(),
		}
		result, err := collection.InsertOne(ctx, doc)
		require.NoError(t, err, "Failed to insert document")
		assert.NotNil(t, result.InsertedID, "InsertedID should not be nil")

		// Find the document
		var found bson.M
		err = collection.FindOne(ctx, bson.M{"name": "test"}).Decode(&found)
		require.NoError(t, err, "Failed to find document")
		assert.Equal(t, "test", found["name"])
	})
}
