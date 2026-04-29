// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package bootstrap

import (
	"testing"

	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_Struct(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockLogger := createTestLogger(ctrl)

	cfg := &Config{
		ServerAddress: ":8080",
	}

	app := fiber.New()
	telemetry := &libOtel.Telemetry{
		TelemetryConfig: libOtel.TelemetryConfig{
			EnableTelemetry: false,
		},
	}

	httpServer := NewHTTPServer(cfg, app, mockLogger, telemetry)

	service := &Service{
		HTTPServer: httpServer,
		Logger:     mockLogger,
	}

	require.NotNil(t, service)
	assert.NotNil(t, service.HTTPServer)
	assert.NotNil(t, service.Logger)
}

func TestService_EmbeddedServers(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockLogger := createTestLogger(ctrl)

	cfg := &Config{
		ServerAddress: ":8080",
	}

	app := fiber.New()
	telemetry := &libOtel.Telemetry{
		TelemetryConfig: libOtel.TelemetryConfig{
			EnableTelemetry: false,
		},
	}

	httpServer := NewHTTPServer(cfg, app, mockLogger, telemetry)

	service := &Service{
		HTTPServer: httpServer,
		Logger:     mockLogger,
	}

	// Test embedded server address method
	assert.Equal(t, ":8080", service.HTTPServer.ServerAddress())
}
