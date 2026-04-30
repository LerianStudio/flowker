// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package readyz implements the canonical /readyz endpoint following Ring Standards.
// The response contract is non-negotiable and matches the Lerian /readyz specification exactly.
package readyz

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	libHTTP "github.com/LerianStudio/lib-commons/v5/commons/net/http"
	"github.com/gofiber/fiber/v2"
)

// Version is set during build via ldflags. Used in /readyz response.
var Version = "dev"

// IsDrainingFunc is a function that returns whether the service is draining.
// This is set by the bootstrap package to avoid import cycles.
// Default: always returns false (not draining).
var IsDrainingFunc = func() bool { return false }

// HealthChecker defines the interface for dependency health checks.
// Each dependency (MongoDB, PostgreSQL, Redis, etc.) must implement this interface.
type HealthChecker interface {
	// Name returns the dependency name for the checks map key (e.g., "mongodb", "postgresql")
	Name() string

	// Ping checks if the dependency is reachable.
	// Returns nil for "up", error for "down".
	// Special error types:
	//   - *SkippedError: returns "skipped" status (requires Reason)
	//   - *NotApplicableError: returns "n/a" status (requires Reason)
	//   - *DegradedError: returns "degraded" status (circuit breaker half-open)
	Ping(ctx context.Context) error

	// IsTLSEnabled returns whether TLS is configured for this dependency.
	// This reflects configured posture, not runtime certificate validity.
	IsTLSEnabled() bool
}

// SkippedError indicates the dependency is intentionally skipped (disabled via config).
type SkippedError struct {
	Reason string
}

func (e *SkippedError) Error() string { return "skipped: " + e.Reason }

// NotApplicableError indicates the dependency is not applicable in current mode.
type NotApplicableError struct {
	Reason string
}

func (e *NotApplicableError) Error() string { return "n/a: " + e.Reason }

// DegradedError indicates the dependency is in degraded state (e.g., circuit breaker half-open).
type DegradedError struct {
	BreakerState string
}

func (e *DegradedError) Error() string { return "degraded: " + e.BreakerState }

// Response is the canonical /readyz response contract.
// This shape is NON-NEGOTIABLE per Ring Standards.
type Response struct {
	Status         string                     `json:"status"` // "healthy" or "unhealthy"
	Checks         map[string]DependencyCheck `json:"checks"`
	Version        string                     `json:"version"`
	DeploymentMode string                     `json:"deployment_mode"`
}

// DependencyCheck represents a single dependency's health status.
type DependencyCheck struct {
	Status       string `json:"status"`                  // up/down/degraded/skipped/n/a (closed set)
	LatencyMs    int64  `json:"latency_ms,omitempty"`    // when status is up or degraded
	TLS          bool   `json:"tls,omitempty"`           // when dependency supports TLS
	Error        string `json:"error,omitempty"`         // when status is down or degraded
	Reason       string `json:"reason,omitempty"`        // when status is skipped or n/a
	BreakerState string `json:"breaker_state,omitempty"` // when dep is breaker-wrapped
}

// Handler handles the /readyz endpoint.
type Handler struct {
	checkers       []HealthChecker
	version        string
	deploymentMode string
	checkTimeout   time.Duration

	// Note: Drain state is now checked via IsDrainingFunc() in the Readyz handler.
	// When draining, /readyz immediately returns 503 to signal K8s to stop routing traffic.
}

// Option configures the Handler.
type Option func(*Handler)

// WithChecker adds a health checker to the handler.
func WithChecker(checker HealthChecker) Option {
	return func(h *Handler) {
		h.checkers = append(h.checkers, checker)
	}
}

// WithVersion sets the version to include in the response.
func WithVersion(version string) Option {
	return func(h *Handler) {
		h.version = version
	}
}

// WithDeploymentMode sets the deployment mode to include in the response.
func WithDeploymentMode(mode string) Option {
	return func(h *Handler) {
		h.deploymentMode = mode
	}
}

// WithCheckTimeout sets the per-dependency check timeout.
func WithCheckTimeout(timeout time.Duration) Option {
	return func(h *Handler) {
		h.checkTimeout = timeout
	}
}

// NewHandler creates a new Handler with the given options.
func NewHandler(opts ...Option) *Handler {
	h := &Handler{
		checkers:       make([]HealthChecker, 0),
		version:        getVersionFromEnv(),
		deploymentMode: getDeploymentModeFromEnv(),
		checkTimeout:   2 * time.Second, // Default per Ring Standards
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// Readyz handles GET /readyz requests.
// Returns 200 with status "healthy" iff every check is up/skipped/n/a.
// Returns 503 with status "unhealthy" if ANY check is down or degraded.
// GATE: Returns 503 immediately if service is draining (graceful shutdown).
//
// @Summary      Kubernetes readiness probe
// @Description  Returns dependency health status following canonical contract
// @Tags         Health
// @Accept       json
// @Produce      json
// @Success      200 {object} Response "Service is healthy - all dependencies up/skipped/n-a"
// @Failure      503 {object} Response "Service is unhealthy - one or more dependencies down/degraded"
// @Router       /readyz [get]
func (h *Handler) Readyz(c *fiber.Ctx) error {
	// Check draining state first - short-circuit to 503 during graceful shutdown.
	// This signals K8s to stop routing new traffic to this pod.
	if IsDrainingFunc() {
		return libHTTP.Respond(c, fiber.StatusServiceUnavailable, Response{
			Status:         "unhealthy",
			Checks:         map[string]DependencyCheck{},
			Version:        h.version,
			DeploymentMode: h.deploymentMode,
		})
	}

	checks := h.runChecks(c.Context())

	// Aggregation rule: 503 iff any check is "down" or "degraded"
	status := "healthy"
	statusCode := fiber.StatusOK

	for _, check := range checks {
		if check.Status == "down" || check.Status == "degraded" {
			status = "unhealthy"
			statusCode = fiber.StatusServiceUnavailable

			break
		}
	}

	response := Response{
		Status:         status,
		Checks:         checks,
		Version:        h.version,
		DeploymentMode: h.deploymentMode,
	}

	return libHTTP.Respond(c, statusCode, response)
}

// runChecks executes all health checks in parallel with per-dep timeout.
func (h *Handler) runChecks(ctx context.Context) map[string]DependencyCheck {
	checks := make(map[string]DependencyCheck)

	if len(h.checkers) == 0 {
		return checks
	}

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	for _, checker := range h.checkers {
		wg.Add(1)

		go func(chk HealthChecker) {
			defer wg.Done()

			check := h.runSingleCheck(ctx, chk)

			mu.Lock()
			checks[chk.Name()] = check
			mu.Unlock()
		}(checker)
	}

	wg.Wait()

	return checks
}

// runSingleCheck executes a single health check with timeout.
// Emits readyz_check_duration_ms and readyz_check_status metrics for every check.
func (h *Handler) runSingleCheck(ctx context.Context, checker HealthChecker) DependencyCheck {
	checkCtx, cancel := context.WithTimeout(ctx, h.checkTimeout)
	defer cancel()

	start := time.Now()
	err := checker.Ping(checkCtx)
	duration := time.Since(start)
	latency := duration.Milliseconds()

	check := DependencyCheck{
		TLS: checker.IsTLSEnabled(),
	}

	switch {
	case err == nil:
		// Status: up
		check.Status = "up"
		check.LatencyMs = latency

	case errors.Is(err, context.DeadlineExceeded):
		// Timeout = down
		check.Status = "down"
		check.Error = "timeout exceeded"

	default:
		// Check for special error types
		var (
			skippedErr  *SkippedError
			naErr       *NotApplicableError
			degradedErr *DegradedError
		)

		switch {
		case errors.As(err, &skippedErr):
			check.Status = "skipped"
			check.Reason = skippedErr.Reason
			// No latency for skipped

		case errors.As(err, &naErr):
			check.Status = "n/a"
			check.Reason = naErr.Reason
			// No latency for n/a

		case errors.As(err, &degradedErr):
			check.Status = "degraded"
			check.BreakerState = degradedErr.BreakerState
			check.LatencyMs = latency

		default:
			// Generic error = down
			check.Status = "down"
			check.Error = err.Error()
		}
	}

	// Emit readyz metrics for every check (per metrics contract)
	depName := checker.Name()
	EmitCheckDuration(ctx, depName, check.Status, duration)
	EmitCheckStatus(ctx, depName, check.Status)

	return check
}

// SelfProbeResult represents the result of a dependency's self-probe.
type SelfProbeResult struct {
	Name   string // Dependency name (e.g., "mongodb", "postgresql")
	Status string // up, down, skipped
	Err    error  // Non-nil when status is "down"
}

// RunSelfProbe executes health checks for all dependencies and emits selfprobe_result metrics.
// This is called once at startup after telemetry is initialized.
// Unlike Readyz, this only emits the gauge metric for monitoring startup health state.
// Returns the results for each dependency so the caller can determine overall health.
func (h *Handler) RunSelfProbe(ctx context.Context) []SelfProbeResult {
	results := make([]SelfProbeResult, 0, len(h.checkers))

	for _, checker := range h.checkers {
		checkCtx, cancel := context.WithTimeout(ctx, h.checkTimeout)
		err := checker.Ping(checkCtx)

		cancel()

		result := SelfProbeResult{
			Name: checker.Name(),
		}

		if err == nil {
			result.Status = "up"
		} else {
			// Check for special error types (same handling as runCheck)
			var (
				skippedErr  *SkippedError
				naErr       *NotApplicableError
				degradedErr *DegradedError
			)

			switch {
			case errors.As(err, &skippedErr):
				result.Status = "skipped"
			case errors.As(err, &naErr):
				result.Status = "n/a"
			case errors.As(err, &degradedErr):
				result.Status = "degraded"
				result.Err = err
			default:
				result.Status = "down"
				result.Err = err
			}
		}

		results = append(results, result)

		// Emit selfprobe_result metric: 1.0 for healthy (up or skipped), 0.0 for unhealthy
		// "skipped" is healthy because it means the dependency was intentionally disabled
		isHealthy := result.Status == "up" || result.Status == "skipped"
		EmitSelfProbeResult(ctx, checker.Name(), isHealthy)
	}

	return results
}

// getVersionFromEnv returns version from env or default.
func getVersionFromEnv() string {
	if v := os.Getenv("VERSION"); v != "" {
		return v
	}

	return Version
}

// getDeploymentModeFromEnv returns deployment mode from env or default.
func getDeploymentModeFromEnv() string {
	if m := os.Getenv("DEPLOYMENT_MODE"); m != "" {
		return m
	}

	return "local"
}
