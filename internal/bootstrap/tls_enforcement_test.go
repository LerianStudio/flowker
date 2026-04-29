// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSaaSTLS(t *testing.T) {
	// Save original env value and restore after test
	originalMode := os.Getenv("DEPLOYMENT_MODE")
	defer func() {
		if originalMode == "" {
			os.Unsetenv("DEPLOYMENT_MODE")
		} else {
			os.Setenv("DEPLOYMENT_MODE", originalMode)
		}
	}()

	tests := []struct {
		name           string
		deploymentMode string
		unsetMode      bool // if true, DEPLOYMENT_MODE is unset (not empty string)
		cfg            TLSConfig
		wantErr        bool
		errContains    string
	}{
		{
			name:           "saas mode with non-TLS MongoDB URI returns error",
			deploymentMode: "saas",
			cfg: TLSConfig{
				MongoURI:    "mongodb://localhost:27017/flowker",
				PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=require",
			},
			wantErr:     true,
			errContains: "mongodb",
		},
		{
			name:           "saas mode with non-TLS PostgreSQL DSN returns error",
			deploymentMode: "saas",
			cfg: TLSConfig{
				MongoURI:    "mongodb+srv://user:pass@cluster.mongodb.net/flowker",
				PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=disable",
			},
			wantErr:     true,
			errContains: "postgresql",
		},
		{
			name:           "saas mode with TLS enabled for all databases returns nil",
			deploymentMode: "saas",
			cfg: TLSConfig{
				MongoURI:    "mongodb+srv://user:pass@cluster.mongodb.net/flowker",
				PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=verify-full",
			},
			wantErr: false,
		},
		{
			name:           "local mode with non-TLS DSNs returns nil (no enforcement)",
			deploymentMode: "local",
			cfg: TLSConfig{
				MongoURI:    "mongodb://localhost:27017/flowker",
				PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=disable",
			},
			wantErr: false,
		},
		{
			name:           "byoc mode with non-TLS DSNs returns nil (no enforcement)",
			deploymentMode: "byoc",
			cfg: TLSConfig{
				MongoURI:    "mongodb://localhost:27017/flowker",
				PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=disable",
			},
			wantErr: false,
		},
		{
			name:           "saas mode with empty DSN returns nil (dependency not configured)",
			deploymentMode: "saas",
			cfg: TLSConfig{
				MongoURI:    "",
				PostgresDSN: "",
			},
			wantErr: false,
		},
		{
			name:           "saas mode with malformed MongoDB URI returns wrapped parse error",
			deploymentMode: "saas",
			cfg: TLSConfig{
				MongoURI:    "://invalid-uri",
				PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=require",
			},
			wantErr:     true,
			errContains: "mongodb",
		},
		{
			name:           "saas mode with malformed PostgreSQL DSN returns wrapped parse error",
			deploymentMode: "saas",
			cfg: TLSConfig{
				MongoURI:    "mongodb+srv://user:pass@cluster.mongodb.net/flowker",
				PostgresDSN: "://invalid-dsn",
			},
			wantErr:     true,
			errContains: "postgresql",
		},
		{
			name:      "DEPLOYMENT_MODE unset (defaults to local) with non-TLS DSNs returns nil",
			unsetMode: true,
			cfg: TLSConfig{
				MongoURI:    "mongodb://localhost:27017/flowker",
				PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=disable",
			},
			wantErr: false,
		},
		{
			name:           "saas mode with tls=true query param returns nil",
			deploymentMode: "saas",
			cfg: TLSConfig{
				MongoURI:    "mongodb://localhost:27017/flowker?tls=true",
				PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=require",
			},
			wantErr: false,
		},
		{
			name:           "saas mode with ssl=true (legacy) query param returns nil",
			deploymentMode: "saas",
			cfg: TLSConfig{
				MongoURI:    "mongodb://localhost:27017/flowker?ssl=true",
				PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=verify-ca",
			},
			wantErr: false,
		},
		{
			name:           "saas mode with only MongoDB configured (no PostgreSQL) validates MongoDB only",
			deploymentMode: "saas",
			cfg: TLSConfig{
				MongoURI:    "mongodb://localhost:27017/flowker", // non-TLS
				PostgresDSN: "",                                  // not configured
			},
			wantErr:     true,
			errContains: "mongodb",
		},
		{
			name:           "saas mode with only PostgreSQL configured (no MongoDB) validates PostgreSQL only",
			deploymentMode: "saas",
			cfg: TLSConfig{
				MongoURI:    "",                                                          // not configured
				PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=disable", // non-TLS
			},
			wantErr:     true,
			errContains: "postgresql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set or unset DEPLOYMENT_MODE based on test case
			if tt.unsetMode {
				os.Unsetenv("DEPLOYMENT_MODE")
			} else {
				os.Setenv("DEPLOYMENT_MODE", tt.deploymentMode)
			}

			err := ValidateSaaSTLS(tt.cfg)

			if tt.wantErr {
				require.Error(t, err, "expected error but got nil")
				assert.True(t, strings.Contains(strings.ToLower(err.Error()), tt.errContains),
					"error should mention %q, got: %s", tt.errContains, err.Error())
			} else {
				require.NoError(t, err, "expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateSaaSTLS_EdgeCases(t *testing.T) {
	originalMode := os.Getenv("DEPLOYMENT_MODE")
	defer func() {
		if originalMode == "" {
			os.Unsetenv("DEPLOYMENT_MODE")
		} else {
			os.Setenv("DEPLOYMENT_MODE", originalMode)
		}
	}()

	t.Run("saas mode case insensitivity", func(t *testing.T) {
		// DEPLOYMENT_MODE should be case-insensitive: "SAAS", "SaaS", "saas" all trigger enforcement
		os.Setenv("DEPLOYMENT_MODE", "SAAS")
		cfg := TLSConfig{
			MongoURI:    "mongodb://localhost:27017/flowker",
			PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=disable",
		}

		err := ValidateSaaSTLS(cfg)
		// "SAAS" (uppercase) should trigger enforcement just like "saas"
		assert.Error(t, err, "uppercase SAAS should trigger enforcement (case-insensitive)")
		assert.Contains(t, strings.ToLower(err.Error()), "mongodb")
	})

	t.Run("empty deployment mode treated as non-saas", func(t *testing.T) {
		os.Setenv("DEPLOYMENT_MODE", "")
		cfg := TLSConfig{
			MongoURI:    "mongodb://localhost:27017/flowker",
			PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=disable",
		}

		err := ValidateSaaSTLS(cfg)
		assert.NoError(t, err, "empty DEPLOYMENT_MODE should not trigger enforcement")
	})

	t.Run("PostgreSQL sslmode=prefer is NOT considered TLS enabled (can fallback to cleartext)", func(t *testing.T) {
		os.Setenv("DEPLOYMENT_MODE", "saas")
		cfg := TLSConfig{
			MongoURI:    "mongodb+srv://user:pass@cluster.mongodb.net/flowker",
			PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=prefer",
		}

		err := ValidateSaaSTLS(cfg)
		assert.Error(t, err, "sslmode=prefer can fallback to cleartext, should NOT be considered TLS enabled")
		assert.Contains(t, strings.ToLower(err.Error()), "postgresql")
	})

	t.Run("PostgreSQL sslmode=allow is NOT considered TLS enabled (can fallback to cleartext)", func(t *testing.T) {
		os.Setenv("DEPLOYMENT_MODE", "saas")
		cfg := TLSConfig{
			MongoURI:    "mongodb+srv://user:pass@cluster.mongodb.net/flowker",
			PostgresDSN: "postgres://user:pass@localhost:5432/audit?sslmode=allow",
		}

		err := ValidateSaaSTLS(cfg)
		assert.Error(t, err, "sslmode=allow can fallback to cleartext, should NOT be considered TLS enabled")
		assert.Contains(t, strings.ToLower(err.Error()), "postgresql")
	})
}
