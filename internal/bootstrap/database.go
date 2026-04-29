// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"errors"
	"os"
	"strconv"

	libMongo "github.com/LerianStudio/lib-commons/v5/commons/mongo"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	// ErrDatabaseNotConnected is returned when a database operation is attempted
	// before the connection has been established.
	ErrDatabaseNotConnected = errors.New("database not connected")

	// ErrDatabaseAlreadyConnected is returned when Connect is called on an
	// already-connected DatabaseManager.
	ErrDatabaseAlreadyConnected = errors.New("database already connected")
)

// DatabaseManager manages MongoDB connection via lib-commons mongo client.
// Implements connection pooling, TLS support, health checks, and graceful shutdown.
type DatabaseManager struct {
	libClient *libMongo.Client
	config    *MongoConfig
}

// MongoConfig holds MongoDB connection configuration
type MongoConfig struct {
	URI         string
	Database    string
	MaxPoolSize uint64
	TLSCACert   string
}

// NewDatabaseManager creates a new DatabaseManager instance
// Configuration loaded from environment variables following lib-commons pattern
func NewDatabaseManager() *DatabaseManager {
	config := loadMongoConfigFromEnv()

	return &DatabaseManager{
		config: config,
	}
}

// NewDatabaseManagerWithConfig creates a DatabaseManager with explicit configuration
// Useful for testing and dependency injection
func NewDatabaseManagerWithConfig(config *MongoConfig) *DatabaseManager {
	if config == nil {
		config = loadMongoConfigFromEnv()
	}

	return &DatabaseManager{
		config: config,
	}
}

// loadMongoConfigFromEnv loads MongoDB configuration from environment variables
// Follows lib-commons SetConfigFromEnvVars pattern
func loadMongoConfigFromEnv() *MongoConfig {
	maxPoolSize := getEnvAsIntOrDefault("MONGO_MAX_POOL_SIZE", 20)
	if maxPoolSize < 0 {
		maxPoolSize = 20
	}

	return &MongoConfig{
		URI:         getEnvOrDefault("MONGO_URI", "mongodb://localhost:27017/flowker"),
		Database:    getEnvOrDefault("MONGO_DB_NAME", "flowker"),
		MaxPoolSize: uint64(maxPoolSize),
		TLSCACert:   getEnvOrDefault("MONGO_TLS_CA_CERT", ""),
	}
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return defaultValue
}

// getEnvAsIntOrDefault gets environment variable as int or returns default
func getEnvAsIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}

	return defaultValue
}

// Connect establishes connection to MongoDB via lib-commons client.
// When MONGO_TLS_CA_CERT is set (base64-encoded PEM), TLS is configured
// automatically for secure connections (e.g., AWS DocumentDB).
func (dm *DatabaseManager) Connect(ctx context.Context) error {
	if dm.libClient != nil {
		return ErrDatabaseAlreadyConnected
	}

	cfg := libMongo.Config{
		URI:         dm.config.URI,
		Database:    dm.config.Database,
		MaxPoolSize: dm.config.MaxPoolSize,
	}

	if dm.config.TLSCACert != "" {
		cfg.TLS = &libMongo.TLSConfig{
			CACertBase64: dm.config.TLSCACert,
		}
	}

	client, err := libMongo.NewClient(ctx, cfg)
	if err != nil {
		return err
	}

	dm.libClient = client

	return nil
}

// Disconnect closes MongoDB connection gracefully
func (dm *DatabaseManager) Disconnect(ctx context.Context) error {
	if dm.libClient == nil {
		return nil
	}

	err := dm.libClient.Close(ctx)
	dm.libClient = nil

	return err
}

// Ping checks MongoDB connection health
// Used by health check endpoints following lib-commons pattern
func (dm *DatabaseManager) Ping(ctx context.Context) error {
	if dm.libClient == nil {
		return ErrDatabaseNotConnected
	}

	return dm.libClient.Ping(ctx)
}

// GetClient returns the underlying MongoDB client instance
// Used by repository implementations to access database
func (dm *DatabaseManager) GetClient(ctx context.Context) (*mongo.Client, error) {
	if dm.libClient == nil {
		return nil, ErrDatabaseNotConnected
	}

	return dm.libClient.Client(ctx)
}

// GetDatabase returns the MongoDB database instance
// Used by repository implementations for collection access
func (dm *DatabaseManager) GetDatabase(ctx context.Context) (*mongo.Database, error) {
	if dm.libClient == nil {
		return nil, ErrDatabaseNotConnected
	}

	return dm.libClient.Database(ctx)
}

// IsConnected checks if database connection is established
func (dm *DatabaseManager) IsConnected() bool {
	return dm.libClient != nil
}

// GetConfig returns the MongoDB configuration
func (dm *DatabaseManager) GetConfig() *MongoConfig {
	return dm.config
}
