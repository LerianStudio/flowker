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
func (s *HTTPServer) Run(l *libCommons.Launcher) error {
	libCommonsServer.NewServerManager(nil, s.telemetry, s.logger).
		WithHTTPServer(s.app, s.serverAddress).
		StartWithGracefulShutdown()

	return nil
}
