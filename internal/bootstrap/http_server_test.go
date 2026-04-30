// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package bootstrap

import (
	"testing"

	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v5/commons/opentelemetry"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func createTestLogger(ctrl *gomock.Controller) libLog.Logger {
	mockLogger := libLog.NewMockLogger(ctrl)
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().With(gomock.Any()).Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().WithGroup(gomock.Any()).Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().Enabled(gomock.Any()).Return(true).AnyTimes()
	mockLogger.EXPECT().Sync(gomock.Any()).Return(nil).AnyTimes()
	return mockLogger
}

func TestNewHTTPServer(t *testing.T) {
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

	server := NewHTTPServer(cfg, app, mockLogger, telemetry, nil)

	require.NotNil(t, server)
	assert.Equal(t, ":8080", server.ServerAddress())
}

func TestHTTPServer_ServerAddress(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockLogger := createTestLogger(ctrl)

	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "standard port",
			address:  ":8080",
			expected: ":8080",
		},
		{
			name:     "custom port",
			address:  ":3000",
			expected: ":3000",
		},
		{
			name:     "with host",
			address:  "localhost:8080",
			expected: "localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				ServerAddress: tt.address,
			}

			app := fiber.New()
			telemetry := &libOtel.Telemetry{
				TelemetryConfig: libOtel.TelemetryConfig{
					EnableTelemetry: false,
				},
			}

			server := NewHTTPServer(cfg, app, mockLogger, telemetry, nil)
			assert.Equal(t, tt.expected, server.ServerAddress())
		})
	}
}

func TestHTTPServer_Struct(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockLogger := createTestLogger(ctrl)

	cfg := &Config{
		ServerAddress: ":8080",
	}

	app := fiber.New()
	telemetry := &libOtel.Telemetry{
		TelemetryConfig: libOtel.TelemetryConfig{
			ServiceName:     "test-service",
			EnableTelemetry: false,
		},
	}

	server := NewHTTPServer(cfg, app, mockLogger, telemetry, nil)

	require.NotNil(t, server)
	assert.NotNil(t, server.app)
	assert.NotNil(t, server.logger)
	assert.Equal(t, "test-service", server.telemetry.ServiceName)
}
