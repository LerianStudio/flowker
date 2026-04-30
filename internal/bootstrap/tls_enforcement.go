// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"fmt"
	"os"
	"strings"

	"github.com/LerianStudio/flowker/pkg"
)

// TLSConfig holds connection information for TLS validation.
// Used by ValidateSaaSTLS to verify TLS is enabled before connections are opened.
type TLSConfig struct {
	// MongoURI is the MongoDB connection URI.
	MongoURI string

	// PostgresDSN is the PostgreSQL connection string.
	PostgresDSN string
}

// ValidateSaaSTLS validates that all database connections use TLS when
// DEPLOYMENT_MODE=saas. Returns an error if any connection is non-TLS.
//
// MUST be called BEFORE any database connection is opened.
// This is a hard-fail at startup - the service will not start without TLS in SaaS mode.
//
// Deployment modes:
//   - saas: TLS MANDATORY for all DB connections
//   - byoc: TLS recommended but not hard-enforced
//   - local: TLS optional (developer workstation)
//   - (unset): defaults to local behavior
//
// Error messages specify which dependency caused the violation for operator diagnostics.
func ValidateSaaSTLS(cfg TLSConfig) error {
	mode := os.Getenv("DEPLOYMENT_MODE")
	if !strings.EqualFold(mode, "saas") {
		return nil // Only enforce in SaaS mode
	}

	// Check MongoDB TLS
	if cfg.MongoURI != "" {
		tls, err := DetectMongoTLS(cfg.MongoURI)
		if err != nil {
			return fmt.Errorf("validate TLS for mongodb: %w", err)
		}

		if !tls {
			return pkg.ValidationError{
				EntityType: "bootstrap",
				Code:       "TLS_REQUIRED_MONGODB",
				Title:      "TLS Required for MongoDB",
				Message:    "DEPLOYMENT_MODE=saas: TLS required for mongodb but not configured. Use mongodb+srv:// scheme or add tls=true/ssl=true query parameter.",
			}
		}
	}

	// Check PostgreSQL TLS
	if cfg.PostgresDSN != "" {
		tls, err := DetectPostgresTLS(cfg.PostgresDSN)
		if err != nil {
			return fmt.Errorf("validate TLS for postgresql: %w", err)
		}

		if !tls {
			return pkg.ValidationError{
				EntityType: "bootstrap",
				Code:       "TLS_REQUIRED_POSTGRESQL",
				Title:      "TLS Required for PostgreSQL",
				Message:    "DEPLOYMENT_MODE=saas: TLS required for postgresql but not configured. Use sslmode=require, sslmode=verify-ca, or sslmode=verify-full.",
			}
		}
	}

	return nil
}
