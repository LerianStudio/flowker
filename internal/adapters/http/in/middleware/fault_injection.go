// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package middleware

import (
	"time"

	"github.com/gofiber/fiber/v2"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libHTTP "github.com/LerianStudio/lib-commons/v4/commons/net/http"
)

// Fault injection constants (headers and values)
const (
	FaultInjectionHeader = "X-Test-Fault-Injection"
	FaultTimeout         = "timeout"
	FaultUnavailable     = "unavailable"
)

// FaultInjectionConfig holds configuration for the fault injection middleware.
type FaultInjectionConfig struct {
	Enabled         bool
	TimeoutDuration time.Duration
}

// DefaultFaultInjectionConfig returns the default configuration with fault injection disabled.
func DefaultFaultInjectionConfig() FaultInjectionConfig {
	return FaultInjectionConfig{
		Enabled:         false,
		TimeoutDuration: 100 * time.Millisecond,
	}
}

// FaultInjection returns a middleware that can simulate infrastructure failures.
// ONLY for integration testing; keep disabled in production.
func FaultInjection(config ...FaultInjectionConfig) fiber.Handler {
	cfg := DefaultFaultInjectionConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		if !cfg.Enabled {
			return c.Next()
		}

		faultType := c.Get(FaultInjectionHeader)
		if faultType == "" {
			return c.Next()
		}

		switch faultType {
		case FaultTimeout:
			time.Sleep(cfg.TimeoutDuration)

			return libHTTP.Respond(c, fiber.StatusGatewayTimeout, libCommons.Response{
				Code:    "FLK-0800",
				Title:   "Gateway Timeout",
				Message: "simulated timeout",
			})
		case FaultUnavailable:
			return libHTTP.Respond(c, fiber.StatusServiceUnavailable, libCommons.Response{
				Code:    "FLK-0801",
				Title:   "Service Unavailable",
				Message: "simulated unavailability",
			})
		default:
			return c.Next()
		}
	}
}
