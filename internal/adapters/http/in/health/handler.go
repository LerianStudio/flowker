// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package health

import (
	"context"
	"time"

	libHTTP "github.com/LerianStudio/lib-commons/v5/commons/net/http"
	"github.com/gofiber/fiber/v2"
)

// SelfProbeOKFunc is a function that returns whether the startup self-probe passed.
// This is set by the bootstrap package to avoid import cycles.
// Default: always returns true (for backward compatibility).
var SelfProbeOKFunc = func() bool { return true }

// Version is set during build via ldflags
var Version = "dev"

// DatabaseChecker defines the interface for database health checks
// This allows the health handler to verify database connectivity without
// creating import cycles with the bootstrap package
type DatabaseChecker interface {
	IsConnected() bool
	Ping(ctx context.Context) error
}

// HealthHandler handles health check endpoints
type HealthHandler struct {
	dbChecker DatabaseChecker
	startTime time.Time
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Version   string                 `json:"version,omitempty"`
	Uptime    string                 `json:"uptime,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]CheckResult `json:"checks,omitempty"`
}

// CheckResult represents individual health check result
type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// NewHealthHandler creates a new HealthHandler instance.
// dbChecker can be any type that implements DatabaseChecker (e.g., bootstrap.DatabaseManager).
// A nil dbChecker is accepted; health checks will report the database as unhealthy.
func NewHealthHandler(dbChecker DatabaseChecker) (*HealthHandler, error) {
	return &HealthHandler{
		dbChecker: dbChecker,
		startTime: time.Now(),
	}, nil
}

// Health handles combined health check (GET /health)
// Returns detailed health information including uptime and version.
// GATE: Returns 503 immediately if startup self-probe did not pass.
// NOTE: Infrastructure routes (/health, /readyz) are excluded from OpenAPI spec.
func (h *HealthHandler) Health(c *fiber.Ctx) error {
	// Gate on self-probe result: if startup probe failed, return 503 immediately.
	// This ensures the service doesn't accept traffic until all dependencies
	// were verified healthy at startup.
	if !SelfProbeOKFunc() {
		return libHTTP.Respond(c, fiber.StatusServiceUnavailable, HealthResponse{
			Status:    "unhealthy",
			Version:   Version,
			Uptime:    time.Since(h.startTime).String(),
			Timestamp: time.Now(),
			Checks: map[string]CheckResult{
				"self_probe": {
					Status:  "failed",
					Message: "startup self-probe did not pass",
				},
			},
		})
	}

	checks := make(map[string]CheckResult)

	// Check database connectivity
	dbStatus := h.checkDatabase()
	checks["database"] = dbStatus

	// Calculate uptime
	uptime := time.Since(h.startTime)

	// Determine overall status
	overallStatus := "healthy"
	statusCode := fiber.StatusOK

	if dbStatus.Status != "healthy" {
		overallStatus = "unhealthy"
		statusCode = fiber.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:    overallStatus,
		Version:   Version,
		Uptime:    uptime.String(),
		Timestamp: time.Now(),
		Checks:    checks,
	}

	return libHTTP.Respond(c, statusCode, response)
}

// checkDatabase verifies database connectivity
func (h *HealthHandler) checkDatabase() CheckResult {
	if h.dbChecker == nil || !h.dbChecker.IsConnected() {
		return CheckResult{
			Status:  "unhealthy",
			Message: "database not connected",
		}
	}

	// Ping database with 2-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := h.dbChecker.Ping(ctx); err != nil {
		return CheckResult{
			Status:  "unhealthy",
			Message: "database ping failed: " + err.Error(),
		}
	}

	return CheckResult{
		Status:  "healthy",
		Message: "database connection ok",
	}
}
