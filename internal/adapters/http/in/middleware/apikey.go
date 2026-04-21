// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package middleware provides HTTP middleware helpers for Flowker APIs.
package middleware

import (
	"crypto/subtle"

	"github.com/LerianStudio/flowker/api"
	libHTTP "github.com/LerianStudio/lib-commons/v4/commons/net/http"
	"github.com/gofiber/fiber/v2"
)

// HeaderAPIKey is the HTTP header name for API key authentication.
const HeaderAPIKey = "X-API-Key"

// APIKeyConfig configures the API key middleware.
type APIKeyConfig struct {
	Key     string
	Enabled bool
}

// APIKeyAuth enforces API key authentication when enabled.
// Uses constant-time compare and returns the same message for missing/invalid keys.
func APIKeyAuth(cfg APIKeyConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !cfg.Enabled {
			return c.Next()
		}

		apiKey := c.Get(HeaderAPIKey)
		if apiKey == "" || subtle.ConstantTimeCompare([]byte(apiKey), []byte(cfg.Key)) != 1 {
			return libHTTP.Respond(c, fiber.StatusUnauthorized, api.ErrorResponse{Code: "UNAUTHORIZED", Title: "Unauthorized", Message: "API Key missing or invalid"})
		}

		return c.Next()
	}
}
