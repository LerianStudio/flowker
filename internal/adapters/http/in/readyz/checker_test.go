// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package readyz_test

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/flowker/internal/adapters/http/in/readyz"
	"github.com/stretchr/testify/assert"
)

// --- MongoDB Checker Tests ---

type mockMongoPinger struct {
	connected bool
	pingErr   error
}

func (m *mockMongoPinger) IsConnected() bool          { return m.connected }
func (m *mockMongoPinger) Ping(ctx context.Context) error { return m.pingErr }

type mockMongoConfig struct {
	uri       string
	tlsCACert string
}

func (m *mockMongoConfig) GetURI() string        { return m.uri }
func (m *mockMongoConfig) GetTLSCACert() string  { return m.tlsCACert }

func TestMongoDBChecker_Name(t *testing.T) {
	checker := readyz.NewMongoDBChecker(nil, nil)
	assert.Equal(t, "mongodb", checker.Name())
}

func TestMongoDBChecker_Ping_Success(t *testing.T) {
	pinger := &mockMongoPinger{connected: true, pingErr: nil}
	checker := readyz.NewMongoDBChecker(pinger, nil)

	err := checker.Ping(context.Background())
	assert.NoError(t, err)
}

func TestMongoDBChecker_Ping_NotConnected(t *testing.T) {
	pinger := &mockMongoPinger{connected: false}
	checker := readyz.NewMongoDBChecker(pinger, nil)

	err := checker.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestMongoDBChecker_Ping_PingFails(t *testing.T) {
	pinger := &mockMongoPinger{connected: true, pingErr: errors.New("connection refused")}
	checker := readyz.NewMongoDBChecker(pinger, nil)

	err := checker.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestMongoDBChecker_Ping_NilPinger(t *testing.T) {
	checker := readyz.NewMongoDBChecker(nil, nil)

	err := checker.Ping(context.Background())
	assert.Error(t, err)

	var skippedErr *readyz.SkippedError
	assert.True(t, errors.As(err, &skippedErr))
}

func TestMongoDBChecker_IsTLSEnabled_WithCACert(t *testing.T) {
	config := &mockMongoConfig{tlsCACert: "base64-encoded-cert"}
	checker := readyz.NewMongoDBChecker(nil, config)

	assert.True(t, checker.IsTLSEnabled())
}

func TestMongoDBChecker_IsTLSEnabled_WithMongoDBSRV(t *testing.T) {
	config := &mockMongoConfig{uri: "mongodb+srv://user:pass@cluster.example.com/db"}
	checker := readyz.NewMongoDBChecker(nil, config)

	assert.True(t, checker.IsTLSEnabled())
}

func TestMongoDBChecker_IsTLSEnabled_WithTLSQueryParam(t *testing.T) {
	// Anti-pattern #4: Must use url.Parse, not strings.Contains
	config := &mockMongoConfig{uri: "mongodb://host:27017/db?tls=true"}
	checker := readyz.NewMongoDBChecker(nil, config)

	assert.True(t, checker.IsTLSEnabled())
}

func TestMongoDBChecker_IsTLSEnabled_NoTLS(t *testing.T) {
	config := &mockMongoConfig{uri: "mongodb://localhost:27017/db"}
	checker := readyz.NewMongoDBChecker(nil, config)

	assert.False(t, checker.IsTLSEnabled())
}

func TestMongoDBChecker_IsTLSEnabled_NilConfig(t *testing.T) {
	checker := readyz.NewMongoDBChecker(nil, nil)

	assert.False(t, checker.IsTLSEnabled())
}

// --- PostgreSQL Checker Tests ---

type mockPostgresPinger struct {
	connected bool
	pingErr   error
}

func (m *mockPostgresPinger) IsConnected() bool          { return m.connected }
func (m *mockPostgresPinger) Ping(ctx context.Context) error { return m.pingErr }

type mockPostgresConfig struct {
	sslMode string
}

func (m *mockPostgresConfig) GetSSLMode() string { return m.sslMode }

func TestPostgreSQLChecker_Name(t *testing.T) {
	checker := readyz.NewPostgreSQLChecker(nil, nil)
	assert.Equal(t, "postgresql", checker.Name())
}

func TestPostgreSQLChecker_Ping_Success(t *testing.T) {
	pinger := &mockPostgresPinger{connected: true, pingErr: nil}
	checker := readyz.NewPostgreSQLChecker(pinger, nil)

	err := checker.Ping(context.Background())
	assert.NoError(t, err)
}

func TestPostgreSQLChecker_Ping_NotConnected(t *testing.T) {
	pinger := &mockPostgresPinger{connected: false}
	checker := readyz.NewPostgreSQLChecker(pinger, nil)

	err := checker.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not connected")
}

func TestPostgreSQLChecker_Ping_PingFails(t *testing.T) {
	pinger := &mockPostgresPinger{connected: true, pingErr: errors.New("connection refused")}
	checker := readyz.NewPostgreSQLChecker(pinger, nil)

	err := checker.Ping(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestPostgreSQLChecker_Ping_NilPinger(t *testing.T) {
	checker := readyz.NewPostgreSQLChecker(nil, nil)

	err := checker.Ping(context.Background())
	assert.Error(t, err)

	var skippedErr *readyz.SkippedError
	assert.True(t, errors.As(err, &skippedErr))
}

func TestPostgreSQLChecker_IsTLSEnabled_SSLModes(t *testing.T) {
	testCases := []struct {
		sslMode  string
		expected bool
	}{
		{"require", true},
		{"verify-ca", true},
		{"verify-full", true},
		{"disable", false},
		{"allow", false},
		{"prefer", false},
		{"", false},
	}

	for _, tc := range testCases {
		t.Run(tc.sslMode, func(t *testing.T) {
			config := &mockPostgresConfig{sslMode: tc.sslMode}
			checker := readyz.NewPostgreSQLChecker(nil, config)

			assert.Equal(t, tc.expected, checker.IsTLSEnabled())
		})
	}
}

func TestPostgreSQLChecker_IsTLSEnabled_NilConfig(t *testing.T) {
	checker := readyz.NewPostgreSQLChecker(nil, nil)

	assert.False(t, checker.IsTLSEnabled())
}
