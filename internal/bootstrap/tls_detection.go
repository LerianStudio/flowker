// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"net/url"
)

// DetectMongoTLS checks if a MongoDB connection URI has TLS enabled.
// Returns (true, nil) if TLS is configured, (false, nil) if not, (false, error) for parse failures.
//
// TLS is considered enabled if:
//   - URI scheme is mongodb+srv:// (always uses TLS per MongoDB specification)
//   - URI has tls=true query parameter
//   - URI has ssl=true query parameter (legacy, equivalent to tls=true)
//
// IMPORTANT: This function uses url.Parse and url.Query().Get() to avoid
// anti-pattern #4 (substring matching like strings.Contains(uri, "tls=true"))
// which fails for URL-encoded parameters and matches false positives.
func DetectMongoTLS(uri string) (bool, error) {
	if uri == "" {
		return false, nil
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return false, err
	}

	// mongodb+srv scheme always uses TLS per MongoDB specification
	// https://www.mongodb.com/docs/manual/reference/connection-string/#dns-seed-list-connection-format
	if parsed.Scheme == "mongodb+srv" {
		return true, nil
	}

	// Check tls or ssl query parameter using url.Query().Get()
	// This correctly handles URL-encoded parameters and parameter boundaries
	query := parsed.Query()
	if query.Get("tls") == "true" || query.Get("ssl") == "true" {
		return true, nil
	}

	return false, nil
}

// DetectPostgresTLS checks if a PostgreSQL DSN has TLS enabled.
// Returns (true, nil) if TLS is configured, (false, nil) if not, (false, error) for parse failures.
//
// TLS is considered enabled ONLY for strict modes that GUARANTEE TLS:
// PostgreSQL sslmode values:
//   - disable: No SSL/TLS
//   - allow: Try non-TLS first, fallback to TLS (CAN connect without TLS)
//   - prefer: Try TLS first, fallback to non-TLS (CAN connect without TLS)
//   - require: Require TLS, no CA verification (ALWAYS TLS)
//   - verify-ca: Require TLS with CA verification (ALWAYS TLS)
//   - verify-full: Require TLS with CA + hostname verification (ALWAYS TLS)
//
// CRITICAL: Only "require", "verify-ca", "verify-full" guarantee TLS.
// "allow" and "prefer" can silently fallback to cleartext connections,
// which is unacceptable for SaaS deployments.
//
// IMPORTANT: This function uses url.Parse and url.Query().Get() to avoid
// anti-pattern #4 (substring matching like strings.Contains(dsn, "sslmode="))
// which fails for URL-encoded parameters and matches false positives.
func DetectPostgresTLS(dsn string) (bool, error) {
	if dsn == "" {
		return false, nil
	}

	parsed, err := url.Parse(dsn)
	if err != nil {
		return false, err
	}

	sslmode := parsed.Query().Get("sslmode")

	// Only strict modes that GUARANTEE TLS are considered TLS-enabled.
	// "allow" and "prefer" can fallback to cleartext, which is unacceptable for SaaS.
	switch sslmode {
	case "require", "verify-ca", "verify-full":
		return true, nil
	default:
		return false, nil
	}
}
