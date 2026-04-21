// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package middleware

import (
	"context"
	"net"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/LerianStudio/flowker/pkg/contextutil"
)

// ClientIPMiddleware extracts the client's IP address and injects it into the request context.
// Order: place after otelfiber (tracing) and before logging so logs can include client IP.
func ClientIPMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		clientIP := extractClientIP(c)

		ctx := context.WithValue(c.UserContext(), contextutil.ContextKeyClientIP{}, clientIP)
		c.SetUserContext(ctx)

		return c.Next()
	}
}

// extractClientIP extracts the real client IP address from the request.
// Priority order:
//  1. X-Forwarded-For (leftmost IP)
//  2. X-Real-IP
//  3. c.IP() fallback
//
// Returns "0.0.0.0" if no valid IP is found.
func extractClientIP(c *fiber.Ctx) string {
	if xff := c.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if isValidIP(ip) {
				return ip
			}
		}
	}

	if xri := c.Get("X-Real-IP"); xri != "" {
		if isValidIP(xri) {
			return xri
		}
	}

	ip := c.IP()
	if isValidIP(ip) {
		return ip
	}

	return "0.0.0.0"
}

// isValidIP checks if the given string is a valid IP address (IPv4 or IPv6).
func isValidIP(ip string) bool {
	if ip == "" {
		return false
	}

	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	return net.ParseIP(ip) != nil
}
