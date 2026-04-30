// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libCommonsLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libCommonsOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
	libCommonsServer "github.com/LerianStudio/lib-commons/v5/commons/server"
	"github.com/gofiber/fiber/v2"
)

// HTTPServer represents the http server for Flowker services.
type HTTPServer struct {
	app           *fiber.App
	serverAddress string
	logger        libCommonsLog.Logger
	telemetry     *libCommonsOtel.Telemetry
}

// ServerAddress returns is a convenience method to return the server address.
func (s *HTTPServer) ServerAddress() string {
	return s.serverAddress
}

// NewHTTPServer creates an instance of HTTPServer.
func NewHTTPServer(cfg *Config, app *fiber.App, logger libCommonsLog.Logger, telemetry *libCommonsOtel.Telemetry) *HTTPServer {
	return &HTTPServer{
		app:           app,
		serverAddress: cfg.ServerAddress,
		logger:        logger,
		telemetry:     telemetry,
	}
}

// Run runs the server.
// Registers graceful shutdown with drain coupling: when SIGTERM/SIGINT is received,
// IsDraining() becomes true immediately, causing /readyz to return 503.
// This signals Kubernetes to stop routing new traffic before the server shuts down.
func (s *HTTPServer) Run(l *libCommons.Launcher) error {
	// Register graceful shutdown with drain coupling.
	// This sets IsDraining() = true immediately on signal receipt,
	// then waits for the grace period before actually shutting down.
	RegisterGracefulShutdown(GracefulShutdownConfig{
		App:              s.app,
		Logger:           s.logger,
		DrainGracePeriod: DefaultDrainGracePeriod,
		OnShutdown:       nil, // No additional cleanup needed; lib-commons handles telemetry flush
	})

	// Use lib-commons server manager for the actual startup and server management.
	// Note: Our RegisterGracefulShutdown handles SIGTERM/SIGINT with drain coupling,
	// so the server manager's graceful shutdown will be triggered by us calling App.Shutdown().
	libCommonsServer.NewServerManager(nil, s.telemetry, s.logger).
		WithHTTPServer(s.app, s.serverAddress).
		StartWithGracefulShutdown()

	return nil
}
