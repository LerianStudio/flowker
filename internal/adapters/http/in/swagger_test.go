// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package in

import (
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/api"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithSwaggerConfig(t *testing.T) {
	// Save original values
	originalTitle := api.SwaggerInfo.Title
	originalDescription := api.SwaggerInfo.Description
	originalVersion := api.SwaggerInfo.Version
	originalHost := api.SwaggerInfo.Host
	originalBasePath := api.SwaggerInfo.BasePath
	originalSchemes := api.SwaggerInfo.Schemes

	// Restore SwaggerInfo after test
	defer func() {
		api.SwaggerInfo.Title = originalTitle
		api.SwaggerInfo.Description = originalDescription
		api.SwaggerInfo.Version = originalVersion
		api.SwaggerInfo.Host = originalHost
		api.SwaggerInfo.BasePath = originalBasePath
		api.SwaggerInfo.Schemes = originalSchemes
	}()

	tests := []struct {
		name     string
		config   SwaggerConfig
		validate func(t *testing.T)
	}{
		{
			name: "sets swagger config from provided values",
			config: SwaggerConfig{
				Title:       "Test API",
				Description: "Test Description",
				Version:     "2.0.0",
				BasePath:    "/api/v2",
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "Test API", api.SwaggerInfo.Title)
				assert.Equal(t, "Test Description", api.SwaggerInfo.Description)
				assert.Equal(t, "2.0.0", api.SwaggerInfo.Version)
				assert.Equal(t, "/api/v2", api.SwaggerInfo.BasePath)
			},
		},
		{
			name: "does not override with empty config values",
			config: SwaggerConfig{
				Title: "",
			},
			validate: func(t *testing.T) {
				// Should keep original value when config is empty
				assert.Equal(t, originalTitle, api.SwaggerInfo.Title)
			},
		},
		{
			name: "sets schemes from config",
			config: SwaggerConfig{
				Schemes: "https",
			},
			validate: func(t *testing.T) {
				assert.Contains(t, api.SwaggerInfo.Schemes, "https")
			},
		},
		{
			name: "validates host address - invalid host",
			config: SwaggerConfig{
				Host: "invalid-host-without-port",
			},
			validate: func(t *testing.T) {
				// Invalid host should not be set
				assert.Equal(t, originalHost, api.SwaggerInfo.Host)
			},
		},
		{
			name: "sets valid host address",
			config: SwaggerConfig{
				Host: "localhost:8080",
			},
			validate: func(t *testing.T) {
				assert.Equal(t, "localhost:8080", api.SwaggerInfo.Host)
			},
		},
		{
			name: "handles schemes value",
			config: SwaggerConfig{
				Schemes: "http,https",
			},
			validate: func(t *testing.T) {
				// Verify schemes were set
				assert.NotEmpty(t, api.SwaggerInfo.Schemes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset to original values before each test
			api.SwaggerInfo.Title = originalTitle
			api.SwaggerInfo.Description = originalDescription
			api.SwaggerInfo.Version = originalVersion
			api.SwaggerInfo.Host = originalHost
			api.SwaggerInfo.BasePath = originalBasePath
			api.SwaggerInfo.Schemes = originalSchemes

			app := fiber.New()
			app.Get("/test", WithSwaggerConfig(tt.config), func(c *fiber.Ctx) error {
				return c.SendString("OK")
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)

			tt.validate(t)
		})
	}
}
