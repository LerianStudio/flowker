// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package bootstrap_test

import (
	"testing"

	"github.com/LerianStudio/flowker/internal/bootstrap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectMongoTLS(t *testing.T) {
	testCases := []struct {
		name        string
		uri         string
		expected    bool
		expectError bool
	}{
		{
			name:        "empty URI returns false",
			uri:         "",
			expected:    false,
			expectError: false,
		},
		{
			name:        "plain mongodb URI without TLS",
			uri:         "mongodb://localhost:27017",
			expected:    false,
			expectError: false,
		},
		{
			name:        "mongodb URI with database without TLS",
			uri:         "mongodb://localhost:27017/flowker",
			expected:    false,
			expectError: false,
		},
		{
			name:        "mongodb URI with tls=true query parameter",
			uri:         "mongodb://localhost:27017/db?tls=true",
			expected:    true,
			expectError: false,
		},
		{
			name:        "mongodb URI with ssl=true query parameter (legacy)",
			uri:         "mongodb://localhost:27017/db?ssl=true",
			expected:    true,
			expectError: false,
		},
		{
			name:        "mongodb+srv scheme always uses TLS",
			uri:         "mongodb+srv://cluster.mongodb.net/db",
			expected:    true,
			expectError: false,
		},
		{
			name:        "mongodb+srv with explicit tls=true",
			uri:         "mongodb+srv://cluster.mongodb.net/db?tls=true",
			expected:    true,
			expectError: false,
		},
		{
			name:        "mongodb URI with tls=false",
			uri:         "mongodb://localhost:27017/db?tls=false",
			expected:    false,
			expectError: false,
		},
		{
			name:        "mongodb URI with URL-encoded parameters",
			uri:         "mongodb://localhost:27017/db?tls=true&authSource=admin",
			expected:    true,
			expectError: false,
		},
		{
			name:        "mongodb URI with multiple query params and tls last",
			uri:         "mongodb://localhost:27017/db?authSource=admin&tls=true",
			expected:    true,
			expectError: false,
		},
		{
			name:        "mongodb URI with credentials and tls",
			uri:         "mongodb://user:password@localhost:27017/db?tls=true",
			expected:    true,
			expectError: false,
		},
		{
			name:        "malformed URI returns error",
			uri:         "://invalid-uri",
			expected:    false,
			expectError: true,
		},
		{
			name:        "mongodb URI with replica set and tls",
			uri:         "mongodb://host1:27017,host2:27017/db?replicaSet=rs0&tls=true",
			expected:    true,
			expectError: false,
		},
		// Anti-pattern #4 regression test: ensure substring-ambiguous cases work correctly
		{
			name:        "anti-pattern regression: tls parameter in path should not match",
			uri:         "mongodb://localhost:27017/tls_database",
			expected:    false,
			expectError: false,
		},
		{
			name:        "anti-pattern regression: tls=false with true in host",
			uri:         "mongodb://tls-true-host:27017/db?tls=false",
			expected:    false,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := bootstrap.DetectMongoTLS(tc.uri)

			if tc.expectError {
				require.Error(t, err, "expected error for URI: %s", tc.uri)
			} else {
				require.NoError(t, err, "unexpected error for URI: %s", tc.uri)
			}

			assert.Equal(t, tc.expected, result, "TLS detection mismatch for URI: %s", tc.uri)
		})
	}
}

func TestDetectPostgresTLS(t *testing.T) {
	testCases := []struct {
		name        string
		dsn         string
		expected    bool
		expectError bool
	}{
		{
			name:        "empty DSN returns false",
			dsn:         "",
			expected:    false,
			expectError: false,
		},
		{
			name:        "postgres DSN without sslmode",
			dsn:         "postgres://localhost:5432/db",
			expected:    false,
			expectError: false,
		},
		{
			name:        "postgres DSN with sslmode=disable",
			dsn:         "postgres://localhost:5432/db?sslmode=disable",
			expected:    false,
			expectError: false,
		},
		{
			name:        "postgres DSN with sslmode=allow returns false (can fallback to cleartext)",
			dsn:         "postgres://localhost:5432/db?sslmode=allow",
			expected:    false,
			expectError: false,
		},
		{
			name:        "postgres DSN with sslmode=prefer returns false (can fallback to cleartext)",
			dsn:         "postgres://localhost:5432/db?sslmode=prefer",
			expected:    false,
			expectError: false,
		},
		{
			name:        "postgres DSN with sslmode=require",
			dsn:         "postgres://localhost:5432/db?sslmode=require",
			expected:    true,
			expectError: false,
		},
		{
			name:        "postgres DSN with sslmode=verify-ca",
			dsn:         "postgres://localhost:5432/db?sslmode=verify-ca",
			expected:    true,
			expectError: false,
		},
		{
			name:        "postgres DSN with sslmode=verify-full",
			dsn:         "postgres://localhost:5432/db?sslmode=verify-full",
			expected:    true,
			expectError: false,
		},
		{
			name:        "postgres DSN with credentials and sslmode",
			dsn:         "postgres://user:password@localhost:5432/db?sslmode=require",
			expected:    true,
			expectError: false,
		},
		{
			name:        "postgres DSN with multiple query params",
			dsn:         "postgres://localhost:5432/db?sslmode=require&connect_timeout=10",
			expected:    true,
			expectError: false,
		},
		{
			name:        "malformed DSN returns error",
			dsn:         "://invalid-dsn",
			expected:    false,
			expectError: true,
		},
		// Anti-pattern #4 regression test: ensure substring-ambiguous cases work correctly
		{
			name:        "anti-pattern regression: sslmode in path should not match",
			dsn:         "postgres://localhost:5432/sslmode_database",
			expected:    false,
			expectError: false,
		},
		{
			name:        "anti-pattern regression: sslmode=disable with require in host",
			dsn:         "postgres://sslmode-require-host:5432/db?sslmode=disable",
			expected:    false,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := bootstrap.DetectPostgresTLS(tc.dsn)

			if tc.expectError {
				require.Error(t, err, "expected error for DSN: %s", tc.dsn)
			} else {
				require.NoError(t, err, "unexpected error for DSN: %s", tc.dsn)
			}

			assert.Equal(t, tc.expected, result, "TLS detection mismatch for DSN: %s", tc.dsn)
		})
	}
}
