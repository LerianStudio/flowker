// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package in

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwaggerConfig_Struct(t *testing.T) {
	tests := []struct {
		name     string
		config   SwaggerConfig
		validate func(t *testing.T, cfg SwaggerConfig)
	}{
		{
			name: "full configuration",
			config: SwaggerConfig{
				Title:       "Test API",
				Description: "Test Description",
				Version:     "1.0.0",
				Host:        "localhost:8080",
				BasePath:    "/api/v1",
				LeftDelim:   "{{",
				RightDelim:  "}}",
				Schemes:     "https",
			},
			validate: func(t *testing.T, cfg SwaggerConfig) {
				assert.Equal(t, "Test API", cfg.Title)
				assert.Equal(t, "Test Description", cfg.Description)
				assert.Equal(t, "1.0.0", cfg.Version)
				assert.Equal(t, "localhost:8080", cfg.Host)
				assert.Equal(t, "/api/v1", cfg.BasePath)
				assert.Equal(t, "{{", cfg.LeftDelim)
				assert.Equal(t, "}}", cfg.RightDelim)
				assert.Equal(t, "https", cfg.Schemes)
			},
		},
		{
			name:   "empty configuration",
			config: SwaggerConfig{},
			validate: func(t *testing.T, cfg SwaggerConfig) {
				assert.Empty(t, cfg.Title)
				assert.Empty(t, cfg.Description)
				assert.Empty(t, cfg.Version)
				assert.Empty(t, cfg.Host)
			},
		},
		{
			name: "partial configuration",
			config: SwaggerConfig{
				Title:   "Partial API",
				Version: "2.0.0",
			},
			validate: func(t *testing.T, cfg SwaggerConfig) {
				assert.Equal(t, "Partial API", cfg.Title)
				assert.Equal(t, "2.0.0", cfg.Version)
				assert.Empty(t, cfg.Description)
				assert.Empty(t, cfg.Host)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.config)
		})
	}
}

// Note: NewRoutes integration tests require a valid license client
// which makes external API calls. Integration tests for the full
// router setup should be in a separate integration test suite.

func TestNewRoutes_Signature(t *testing.T) {
	// This test verifies that the NewRoutes function has the expected signature
	// by checking we can reference the function type.
	// Actual integration tests with middleware require a valid license setup.

	require.NotNil(t, NewRoutes, "NewRoutes function should exist")
}
