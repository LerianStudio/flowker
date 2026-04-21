// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// seedExecutorConfig inserts an executor configuration directly into MongoDB
// with "active" status, bypassing the HTTP API. This is needed because
// POST /v1/executors and lifecycle transition routes are deprecated (405).
func seedExecutorConfig(t *testing.T, name, baseURL string, endpoints []map[string]any) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err, "connect to mongo for seeding")

	defer func() {
		_ = client.Disconnect(ctx)
	}()

	collection := client.Database("flowker_test").Collection("executor_configurations")

	executorID := uuid.New().String()
	now := time.Now()

	endpointDocs := make([]bson.M, len(endpoints))
	for i, ep := range endpoints {
		endpointDocs[i] = bson.M{
			"name":    ep["name"],
			"path":    ep["path"],
			"method":  ep["method"],
			"timeout": ep["timeout"],
		}
	}

	doc := bson.M{
		"executorId":  executorID,
		"name":        name,
		"description": "Integration test executor (seeded)",
		"baseUrl":     baseURL,
		"endpoints":   endpointDocs,
		"authentication": bson.M{
			"type": "none",
		},
		"status":    "active",
		"createdAt": now,
		"updatedAt": now,
	}

	_, err = collection.InsertOne(ctx, doc)
	require.NoError(t, err, "seed executor config into MongoDB")

	return executorID
}

// seedProviderConfig inserts a provider configuration directly into MongoDB
// with "active" status, bypassing the HTTP API. Returns the generated UUID string.
func seedProviderConfig(t *testing.T, name, providerID string, config map[string]any) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err, "connect to mongo for seeding provider config")

	defer func() {
		_ = client.Disconnect(ctx)
	}()

	collection := client.Database("flowker_test").Collection("provider_configurations")

	providerConfigID := uuid.New().String()
	now := time.Now()

	doc := bson.M{
		"providerConfigId": providerConfigID,
		"name":             name,
		"providerId":       providerID,
		"config":           config,
		"status":           "active",
		"createdAt":        now,
		"updatedAt":        now,
	}

	_, err = collection.InsertOne(ctx, doc)
	require.NoError(t, err, "seed provider config into MongoDB")

	return providerConfigID
}

// seedDeleteProviderConfig removes a provider configuration directly from MongoDB.
func seedDeleteProviderConfig(t *testing.T, providerConfigID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err, "connect to mongo for cleanup")

	defer func() {
		_ = client.Disconnect(ctx)
	}()

	collection := client.Database("flowker_test").Collection("provider_configurations")

	res, err := collection.DeleteOne(ctx, bson.M{"providerConfigId": providerConfigID})
	require.NoError(t, err, "delete seeded provider config from MongoDB")
	require.Equal(t, int64(1), res.DeletedCount, "expected to delete exactly one seeded provider config")
}

// seedDeleteExecutorConfig removes an executor configuration directly from MongoDB.
func seedDeleteExecutorConfig(t *testing.T, executorID string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err, "connect to mongo for cleanup")

	defer func() {
		_ = client.Disconnect(ctx)
	}()

	collection := client.Database("flowker_test").Collection("executor_configurations")

	res, err := collection.DeleteOne(ctx, bson.M{"executorId": executorID})
	require.NoError(t, err, "delete seeded executor config from MongoDB")
	require.Equal(t, int64(1), res.DeletedCount, "expected to delete exactly one seeded executor config")
}
