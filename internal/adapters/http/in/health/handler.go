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

// Liveness handles liveness probe (GET /health/live)
// Returns 200 OK if the application process is running
// Used by Kubernetes to restart unhealthy pods
// @Summary      Kubernetes liveness probe
// @Description  Returns 200 OK if the application process is running
// @Tags         Health
// @Produce      json
// @Success      200 {object} HealthResponse
// @Router       /health/live [get]
func (h *HealthHandler) Liveness(c *fiber.Ctx) error {
	response := HealthResponse{
		Status:    "alive",
		Timestamp: time.Now(),
	}

	return libHTTP.Respond(c, fiber.StatusOK, response)
}

// Readiness handles readiness probe (GET /health/ready)
// Returns 200 OK if the application is ready to serve traffic
// Returns 503 Service Unavailable if dependencies are not ready
// Used by load balancers to route traffic
// @Summary      Kubernetes readiness probe
// @Description  Returns 200 OK if the application is ready to serve traffic
// @Tags         Health
// @Produce      json
// @Success      200 {object} HealthResponse
// @Failure      503 {object} HealthResponse
// @Router       /health/ready [get]
func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
	checks := make(map[string]CheckResult)

	// Check database connectivity
	dbStatus := h.checkDatabase()
	checks["database"] = dbStatus

	// Determine overall status
	overallStatus := "ready"
	statusCode := fiber.StatusOK

	if dbStatus.Status != "healthy" {
		overallStatus = "not_ready"
		statusCode = fiber.StatusServiceUnavailable
	}

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Checks:    checks,
	}

	return libHTTP.Respond(c, statusCode, response)
}

// Health handles combined health check (GET /health)
// Returns detailed health information including uptime and version
// @Summary      Combined health check
// @Description  Returns detailed health information including uptime and version
// @Tags         Health
// @Produce      json
// @Success      200 {object} HealthResponse
// @Failure      503 {object} HealthResponse
// @Router       /health [get]
func (h *HealthHandler) Health(c *fiber.Ctx) error {
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
