// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package in

import (
	"fmt"
	"time"

	"github.com/LerianStudio/flowker/internal/adapters/http/in/audit"
	"github.com/LerianStudio/flowker/internal/adapters/http/in/catalog"
	"github.com/LerianStudio/flowker/internal/adapters/http/in/dashboard"
	"github.com/LerianStudio/flowker/internal/adapters/http/in/execution"
	executorconfiguration "github.com/LerianStudio/flowker/internal/adapters/http/in/executor_configuration"
	"github.com/LerianStudio/flowker/internal/adapters/http/in/health"
	"github.com/LerianStudio/flowker/internal/adapters/http/in/middleware"
	providerconfiguration "github.com/LerianStudio/flowker/internal/adapters/http/in/provider_configuration"
	"github.com/LerianStudio/flowker/internal/adapters/http/in/readyz"
	"github.com/LerianStudio/flowker/internal/adapters/http/in/webhook"
	"github.com/LerianStudio/flowker/internal/adapters/http/in/workflow"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libHTTP "github.com/LerianStudio/lib-commons/v5/commons/net/http"
	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// RouteConfig holds HTTP route options.
type RouteConfig struct {
	CORSAllowedOrigins      string
	SkipLibCommonsTelemetry bool
	FaultInjectionEnabled   bool
}

// NewRoutes creates the Fiber application with all routes configured
// dbChecker should implement health.DatabaseChecker interface (e.g., bootstrap.DatabaseManager)
// readyzHandler is the canonical /readyz endpoint handler (must be mounted BEFORE auth middleware)
func NewRoutes(
	lg libLog.Logger,
	tl *libOtel.Telemetry,
	swaggerCfg SwaggerConfig,
	dbChecker health.DatabaseChecker,
	readyzHandler *readyz.Handler,
	routeCfg *RouteConfig,
	workflowHandler *workflow.Handler,
	catalogHandler *catalog.Handler,
	executorConfigHandler *executorconfiguration.Handler,
	providerConfigHandler *providerconfiguration.Handler,
	executionHandler *execution.Handler,
	dashboardHandler *dashboard.Handler,
	auditHandler *audit.Handler,
	webhookHandler *webhook.Handler,
	guard *middleware.AuthGuard,
) (*fiber.App, error) {
	f := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			return libHTTP.FiberErrorHandler(ctx, err)
		},
	})
	tlMid := libHTTP.NewTelemetryMiddleware(tl)

	// Allow skipping telemetry injection in specific test scenarios (same as tracer).
	skipTelemetry := routeCfg.SkipLibCommonsTelemetry

	// Middleware order per PROJECT_RULES.md Section 10
	if !skipTelemetry {
		f.Use(tlMid.WithTelemetry(tl)) // 1. FIRST - injects tracer/logger into context
	}

	f.Use(recover.New(recover.Config{EnableStackTrace: false})) // 2. Panic recovery

	// 3. CORS (configured; default restrictive)
	corsCfg := cors.Config{
		AllowOrigins:     getCORSAllowedOrigins(routeCfg),
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Request-ID,X-API-Key",
		AllowCredentials: false,
		MaxAge:           3600,
	}
	f.Use(cors.New(corsCfg))

	// 4. OpenTelemetry HTTP metrics/tracing
	f.Use(otelfiber.Middleware(
		otelfiber.WithNext(skipTelemetryPaths),
	))

	// 5. Client IP (before logging)
	f.Use(middleware.ClientIPMiddleware())

	// 6. HTTP logging
	if !skipTelemetry {
		f.Use(libHTTP.WithHTTPLogging(libHTTP.WithCustomLogger(lg)))
	}

	// 7. Fault injection (tests only, guarded by env)
	f.Use(middleware.FaultInjection(middleware.FaultInjectionConfig{
		Enabled:         routeCfg.FaultInjectionEnabled,
		TimeoutDuration: 100 * time.Millisecond,
	}))

	// API v1 routes
	v1 := f.Group("/v1")

	// Webhook routes (API key auth + optional verify_token per webhook)
	if webhookHandler != nil {
		webhookHandler.RegisterRoutes(v1.Group("/webhooks", guard.With("webhooks", "execute", true)))
	}

	// Protected management routes (Access Manager auth with API key fallback).
	// Workflows
	workflowHandler.RegisterRoutes(v1.Group("/", guard.Protect("workflows", "manage")))

	// Optional catalog routes
	if catalogHandler != nil {
		catalogHandler.RegisterRoutes(v1.Group("/", guard.Protect("catalog", "read")))
	}

	// Optional executor configuration routes
	if executorConfigHandler != nil {
		executorConfigHandler.RegisterRoutes(v1.Group("/", guard.Protect("executor-configurations", "manage")))
	}

	// Optional provider configuration routes
	if providerConfigHandler != nil {
		providerConfigHandler.RegisterRoutes(v1.Group("/", guard.Protect("provider-configurations", "manage")))
	}

	// Optional execution routes
	if executionHandler != nil {
		executionHandler.RegisterRoutes(v1.Group("/", guard.Protect("executions", "manage")))
	}

	// Optional dashboard routes
	if dashboardHandler != nil {
		dashboardHandler.RegisterRoutes(v1.Group("/", guard.Protect("dashboards", "read")))
	}

	// Audit routes (mandatory)
	auditHandler.RegisterRoutes(v1.Group("/", guard.Protect("audit-events", "read")))

	// Health check endpoints with database connectivity verification
	healthHandler, err := health.NewHealthHandler(dbChecker)
	if err != nil {
		return nil, fmt.Errorf("failed to create health handler: %w", err)
	}

	// /health - Combined health check (uptime, version, checks)
	// Gates on selfProbeOK: returns 503 if startup self-probe did not pass.
	f.Get("/health", healthHandler.Health)

	// /readyz - Kubernetes readiness probe (canonical contract)
	// MUST be registered BEFORE auth middleware per Ring Standards
	f.Get("/readyz", readyzHandler.Readyz)

	// Version
	f.Get("/version", libHTTP.Version)

	// Doc Swagger
	f.Get("/swagger/*", WithSwaggerConfig(swaggerCfg), fiberSwagger.WrapHandler)

	if !skipTelemetry {
		f.Use(tlMid.EndTracingSpans) // LAST - closes root spans
	}

	return f, nil
}
