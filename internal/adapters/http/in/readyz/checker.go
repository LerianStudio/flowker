// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package readyz

import (
	"context"
	"net/url"
)

// MongoDBPinger defines the interface for MongoDB health checks.
// This abstraction allows the readyz package to check MongoDB health
// without importing the bootstrap package directly.
type MongoDBPinger interface {
	IsConnected() bool
	Ping(ctx context.Context) error
}

// MongoDBConfig provides MongoDB configuration for TLS detection.
type MongoDBConfig interface {
	GetURI() string
	GetTLSCACert() string
}

// MongoDBChecker adapts a MongoDB connection to the HealthChecker interface.
type MongoDBChecker struct {
	pinger    MongoDBPinger
	tlsConfig MongoDBConfig
}

// NewMongoDBChecker creates a new MongoDBChecker.
func NewMongoDBChecker(pinger MongoDBPinger, tlsConfig MongoDBConfig) *MongoDBChecker {
	return &MongoDBChecker{
		pinger:    pinger,
		tlsConfig: tlsConfig,
	}
}

// Name returns the dependency name.
func (c *MongoDBChecker) Name() string {
	return "mongodb"
}

// Ping checks if MongoDB is reachable.
func (c *MongoDBChecker) Ping(ctx context.Context) error {
	if c.pinger == nil {
		return &SkippedError{Reason: "mongodb pinger not configured"}
	}

	if !c.pinger.IsConnected() {
		return errNotConnected
	}

	return c.pinger.Ping(ctx)
}

// IsTLSEnabled returns whether TLS is configured for MongoDB.
// Detection uses URL parsing (not substring matching per anti-pattern #4).
func (c *MongoDBChecker) IsTLSEnabled() bool {
	if c.tlsConfig == nil {
		return false
	}

	// Check if CA cert is configured (explicit TLS)
	if c.tlsConfig.GetTLSCACert() != "" {
		return true
	}

	// Parse URI and check for TLS query parameter
	// Anti-pattern #4: MUST NOT use strings.Contains(uri, "tls=true")
	uri := c.tlsConfig.GetURI()
	if uri == "" {
		return false
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return false
	}

	// Check scheme (mongodb+srv always uses TLS)
	if parsed.Scheme == "mongodb+srv" {
		return true
	}

	// Check tls or ssl query parameter (ssl is legacy, equivalent to tls)
	query := parsed.Query()

	return query.Get("tls") == "true" || query.Get("ssl") == "true"
}

// PostgreSQLPinger defines the interface for PostgreSQL health checks.
type PostgreSQLPinger interface {
	IsConnected() bool
	Ping(ctx context.Context) error
}

// PostgreSQLConfig provides PostgreSQL configuration for TLS detection.
type PostgreSQLConfig interface {
	GetSSLMode() string
}

// PostgreSQLChecker adapts a PostgreSQL connection to the HealthChecker interface.
type PostgreSQLChecker struct {
	pinger    PostgreSQLPinger
	tlsConfig PostgreSQLConfig
}

// NewPostgreSQLChecker creates a new PostgreSQLChecker.
func NewPostgreSQLChecker(pinger PostgreSQLPinger, tlsConfig PostgreSQLConfig) *PostgreSQLChecker {
	return &PostgreSQLChecker{
		pinger:    pinger,
		tlsConfig: tlsConfig,
	}
}

// Name returns the dependency name.
func (c *PostgreSQLChecker) Name() string {
	return "postgresql"
}

// Ping checks if PostgreSQL is reachable.
func (c *PostgreSQLChecker) Ping(ctx context.Context) error {
	if c.pinger == nil {
		return &SkippedError{Reason: "postgresql pinger not configured"}
	}

	if !c.pinger.IsConnected() {
		return errNotConnected
	}

	return c.pinger.Ping(ctx)
}

// IsTLSEnabled returns whether TLS is configured for PostgreSQL.
func (c *PostgreSQLChecker) IsTLSEnabled() bool {
	if c.tlsConfig == nil {
		return false
	}

	sslMode := c.tlsConfig.GetSSLMode()

	// sslmode values that enable TLS: require, verify-ca, verify-full
	// sslmode values that don't require TLS: disable, allow, prefer
	switch sslMode {
	case "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}

// errNotConnected is returned when the database is not connected.
var errNotConnected = &connectionError{message: "not connected"}

type connectionError struct {
	message string
}

func (e *connectionError) Error() string {
	return e.message
}
