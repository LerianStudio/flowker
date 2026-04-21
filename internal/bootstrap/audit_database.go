// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"errors"
	"fmt"

	libPostgres "github.com/LerianStudio/lib-commons/v4/commons/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrAuditDatabaseNotConnected is returned when an audit database operation
// is attempted before the connection has been established.
var ErrAuditDatabaseNotConnected = errors.New("audit database not connected")

// AuditDBConfig holds PostgreSQL connection configuration for the audit database.
type AuditDBConfig struct {
	Host           string
	Port           string
	User           string
	Password       string
	DBName         string
	SSLMode        string
	MigrationsPath string
}

// DSN returns the PostgreSQL connection string.
func (c *AuditDBConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode,
	)
}

// AuditDatabaseManager manages PostgreSQL connection for the audit trail.
// Implements connection pooling, migrations, health checks, and graceful shutdown.
type AuditDatabaseManager struct {
	pool   *pgxpool.Pool
	config *AuditDBConfig
}

// NewAuditDatabaseManager creates a new AuditDatabaseManager instance.
// Configuration is loaded from environment variables.
func NewAuditDatabaseManager() *AuditDatabaseManager {
	config := loadAuditDBConfigFromEnv()

	return &AuditDatabaseManager{
		config: config,
	}
}

// NewAuditDatabaseManagerWithConfig creates an AuditDatabaseManager with explicit configuration.
// Useful for testing and dependency injection.
func NewAuditDatabaseManagerWithConfig(config *AuditDBConfig) *AuditDatabaseManager {
	if config == nil {
		config = loadAuditDBConfigFromEnv()
	}

	return &AuditDatabaseManager{
		config: config,
	}
}

// loadAuditDBConfigFromEnv loads audit database configuration from environment variables.
func loadAuditDBConfigFromEnv() *AuditDBConfig {
	return &AuditDBConfig{
		Host:           getEnvOrDefault("AUDIT_DB_HOST", "localhost"),
		Port:           getEnvOrDefault("AUDIT_DB_PORT", "5432"),
		User:           getEnvOrDefault("AUDIT_DB_USER", "flowker_audit"),
		Password:       getEnvOrDefault("AUDIT_DB_PASSWORD", "flowker_audit"),
		DBName:         getEnvOrDefault("AUDIT_DB_NAME", "flowker_audit"),
		SSLMode:        getEnvOrDefault("AUDIT_DB_SSL_MODE", "disable"),
		MigrationsPath: getEnvOrDefault("AUDIT_MIGRATIONS_PATH", "/migrations"),
	}
}

// Connect establishes connection to PostgreSQL and runs migrations.
func (m *AuditDatabaseManager) Connect(ctx context.Context) error {
	if m.pool != nil {
		return fmt.Errorf("audit database already connected")
	}

	poolConfig, err := pgxpool.ParseConfig(m.config.DSN())
	if err != nil {
		return fmt.Errorf("failed to parse audit database config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to audit database: %w", err)
	}

	// Verify connection with ping
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("failed to ping audit database: %w", err)
	}

	m.pool = pool

	// Run migrations
	if err = m.runMigrations(ctx); err != nil {
		pool.Close()
		m.pool = nil

		return fmt.Errorf("failed to run audit migrations: %w", err)
	}

	return nil
}

// Disconnect closes the PostgreSQL connection pool gracefully.
func (m *AuditDatabaseManager) Disconnect(_ context.Context) error {
	if m.pool == nil {
		return nil
	}

	m.pool.Close()
	m.pool = nil

	return nil
}

// GetPool returns the pgxpool.Pool instance.
func (m *AuditDatabaseManager) GetPool() *pgxpool.Pool {
	return m.pool
}

// Ping checks PostgreSQL connection health.
func (m *AuditDatabaseManager) Ping(ctx context.Context) error {
	if m.pool == nil {
		return ErrAuditDatabaseNotConnected
	}

	if err := m.pool.Ping(ctx); err != nil {
		return fmt.Errorf("audit database ping failed: %w", err)
	}

	return nil
}

// IsConnected checks if database connection is established.
func (m *AuditDatabaseManager) IsConnected() bool {
	return m.pool != nil
}

// GetConfig returns the audit database configuration.
func (m *AuditDatabaseManager) GetConfig() *AuditDBConfig {
	return m.config
}

// runMigrations executes version-tracked migrations against the audit database.
// Uses lib-commons v4's NewMigrator without AllowMultiStatements, since
// golang-migrate's multi-statement parser breaks PL/pgSQL $$-quoted functions.
func (m *AuditDatabaseManager) runMigrations(ctx context.Context) error {
	// AllowMultiStatements is intentionally false: golang-migrate's multi-statement
	// parser splits on ";" which breaks PL/pgSQL $$-quoted function bodies.
	// With AllowMultiStatements=false the entire migration file is sent as a single
	// Exec call, and PostgreSQL natively handles multiple statements with correct
	// dollar-quoting support.
	migrator, err := libPostgres.NewMigrator(libPostgres.MigrationConfig{
		PrimaryDSN:     m.config.DSN(),
		DatabaseName:   m.config.DBName,
		MigrationsPath: m.config.MigrationsPath,
	})
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := migrator.Up(ctx); err != nil {
		return fmt.Errorf("schema migrations failed: %w", err)
	}

	return nil
}
