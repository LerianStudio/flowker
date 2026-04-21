// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package in

import "github.com/gofiber/fiber/v2"

// getCORSAllowedOrigins resolves the CORS origins with a restrictive default.
func getCORSAllowedOrigins(cfg *RouteConfig) string {
	if cfg == nil || cfg.CORSAllowedOrigins == "" {
		// Restrictive default to avoid accidental exposure.
		return ""
	}

	return cfg.CORSAllowedOrigins
}

// skipTelemetryPaths avoids instrumenting noisy endpoints.
func skipTelemetryPaths(c *fiber.Ctx) bool {
	switch c.Path() {
	case "/health", "/health/live", "/health/ready":
		return true
	default:
		return false
	}
}
