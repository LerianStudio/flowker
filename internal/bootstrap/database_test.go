// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package bootstrap

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabaseManager(t *testing.T) {
	dbManager := NewDatabaseManager()
	require.NotNil(t, dbManager, "DatabaseManager should not be nil")
	assert.NotNil(t, dbManager.config, "Config should not be nil")
}

func TestNewDatabaseManagerWithConfig(t *testing.T) {
	config := &MongoConfig{
		URI:         "mongodb://test:27017/test",
		Database:    "testdb",
		MaxPoolSize: 10,
	}

	dbManager := NewDatabaseManagerWithConfig(config)
	require.NotNil(t, dbManager)
	assert.Equal(t, config.URI, dbManager.config.URI)
	assert.Equal(t, config.Database, dbManager.config.Database)
	assert.Equal(t, uint64(10), dbManager.config.MaxPoolSize)
}

func TestNewDatabaseManagerWithConfig_NilConfig(t *testing.T) {
	dbManager := NewDatabaseManagerWithConfig(nil)
	require.NotNil(t, dbManager)
	// Should fall back to default config from environment
	assert.NotNil(t, dbManager.config, "Config should not be nil")
	assert.NotEmpty(t, dbManager.config.URI, "URI should not be empty")
	assert.NotEmpty(t, dbManager.config.Database, "Database name should not be empty")
}

func TestLoadMongoConfigFromEnv(t *testing.T) {
	// Set test values using t.Setenv for automatic cleanup
	t.Setenv("MONGO_URI", "mongodb://custom:27017/custom")
	t.Setenv("MONGO_DB_NAME", "customdb")
	t.Setenv("MONGO_MAX_POOL_SIZE", "50")

	config := loadMongoConfigFromEnv()

	assert.Equal(t, "mongodb://custom:27017/custom", config.URI)
	assert.Equal(t, "customdb", config.Database)
	assert.Equal(t, uint64(50), config.MaxPoolSize)
}

func TestLoadMongoConfigFromEnvDefaults(t *testing.T) {
	// Save and clear env vars
	origURI := os.Getenv("MONGO_URI")
	origDB := os.Getenv("MONGO_DB_NAME")
	origMaxPool := os.Getenv("MONGO_MAX_POOL_SIZE")
	origTLS := os.Getenv("MONGO_TLS_CA_CERT")

	os.Unsetenv("MONGO_URI")
	os.Unsetenv("MONGO_DB_NAME")
	os.Unsetenv("MONGO_MAX_POOL_SIZE")
	os.Unsetenv("MONGO_TLS_CA_CERT")

	defer func() {
		if origURI != "" {
			os.Setenv("MONGO_URI", origURI)
		}
		if origDB != "" {
			os.Setenv("MONGO_DB_NAME", origDB)
		}
		if origMaxPool != "" {
			os.Setenv("MONGO_MAX_POOL_SIZE", origMaxPool)
		}
		if origTLS != "" {
			os.Setenv("MONGO_TLS_CA_CERT", origTLS)
		}
	}()

	config := loadMongoConfigFromEnv()

	assert.Equal(t, "mongodb://localhost:27017/flowker", config.URI)
	assert.Equal(t, "flowker", config.Database)
	assert.Equal(t, uint64(20), config.MaxPoolSize)
	assert.Empty(t, config.TLSCACert)
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		defaultVal  string
		envValue    string
		setEnv      bool
		expectedVal string
	}{
		{
			name:        "returns env value when set",
			key:         "TEST_KEY_1",
			defaultVal:  "default",
			envValue:    "custom",
			setEnv:      true,
			expectedVal: "custom",
		},
		{
			name:        "returns default when env not set",
			key:         "TEST_KEY_2",
			defaultVal:  "default",
			envValue:    "",
			setEnv:      false,
			expectedVal: "default",
		},
		{
			name:        "returns default when env is empty string",
			key:         "TEST_KEY_3",
			defaultVal:  "default",
			envValue:    "",
			setEnv:      true,
			expectedVal: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvOrDefault(tt.key, tt.defaultVal)
			assert.Equal(t, tt.expectedVal, result)
		})
	}
}

func TestGetEnvAsIntOrDefault(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		defaultVal  int
		envValue    string
		setEnv      bool
		expectedVal int
	}{
		{
			name:        "returns parsed int when set",
			key:         "TEST_INT_1",
			defaultVal:  10,
			envValue:    "42",
			setEnv:      true,
			expectedVal: 42,
		},
		{
			name:        "returns default when env not set",
			key:         "TEST_INT_2",
			defaultVal:  10,
			envValue:    "",
			setEnv:      false,
			expectedVal: 10,
		},
		{
			name:        "returns default when env is not a number",
			key:         "TEST_INT_3",
			defaultVal:  10,
			envValue:    "notanumber",
			setEnv:      true,
			expectedVal: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnvAsIntOrDefault(tt.key, tt.defaultVal)
			assert.Equal(t, tt.expectedVal, result)
		})
	}
}

func TestDatabaseManager_IsConnected(t *testing.T) {
	dbManager := NewDatabaseManager()
	assert.False(t, dbManager.IsConnected(), "Should not be connected initially")
}

func TestDatabaseManager_GetClient_WhenNotConnected(t *testing.T) {
	dbManager := NewDatabaseManager()
	client, err := dbManager.GetClient(context.Background())
	assert.Nil(t, client, "Client should be nil when not connected")
	assert.ErrorIs(t, err, ErrDatabaseNotConnected)
}

func TestDatabaseManager_GetDatabase_WhenNotConnected(t *testing.T) {
	dbManager := NewDatabaseManager()
	db, err := dbManager.GetDatabase(context.Background())
	assert.Nil(t, db, "Database should be nil when not connected")
	assert.ErrorIs(t, err, ErrDatabaseNotConnected)
}

func TestDatabaseManager_GetConfig(t *testing.T) {
	config := &MongoConfig{
		URI:         "mongodb://test:27017",
		Database:    "testdb",
		MaxPoolSize: 25,
		TLSCACert:   "dGVzdA==",
	}

	dbManager := NewDatabaseManagerWithConfig(config)
	retrievedConfig := dbManager.GetConfig()

	assert.Equal(t, config.URI, retrievedConfig.URI)
	assert.Equal(t, config.Database, retrievedConfig.Database)
	assert.Equal(t, config.MaxPoolSize, retrievedConfig.MaxPoolSize)
	assert.Equal(t, config.TLSCACert, retrievedConfig.TLSCACert)
}

func TestDatabaseManager_Disconnect_WhenNotConnected(t *testing.T) {
	dbManager := NewDatabaseManager()
	err := dbManager.Disconnect(context.Background())
	assert.NoError(t, err, "Disconnect should not error when not connected")
}

func TestMongoConfig_Fields(t *testing.T) {
	config := &MongoConfig{
		URI:         "mongodb://user:pass@host:27017/db",
		Database:    "testdb",
		MaxPoolSize: 100,
		TLSCACert:   "dGVzdC1jYS1jZXJ0", // base64 of "test-ca-cert"
	}

	assert.Equal(t, "mongodb://user:pass@host:27017/db", config.URI)
	assert.Equal(t, "testdb", config.Database)
	assert.Equal(t, uint64(100), config.MaxPoolSize)
	assert.Equal(t, "dGVzdC1jYS1jZXJ0", config.TLSCACert)
}

func TestLoadMongoConfigFromEnv_TLSCACert(t *testing.T) {
	t.Setenv("MONGO_TLS_CA_CERT", "dGVzdC1jYS1jZXJ0")

	config := loadMongoConfigFromEnv()

	assert.Equal(t, "dGVzdC1jYS1jZXJ0", config.TLSCACert)
}
