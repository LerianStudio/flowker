// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swaggo/swag"
)

func TestSwaggerInfo(t *testing.T) {
	tests := []struct {
		name     string
		validate func(t *testing.T)
	}{
		{
			name: "has required fields",
			validate: func(t *testing.T) {
				require.NotNil(t, SwaggerInfo)
				assert.NotEmpty(t, SwaggerInfo.Version)
				assert.NotEmpty(t, SwaggerInfo.Host)
				assert.NotEmpty(t, SwaggerInfo.BasePath)
				assert.NotEmpty(t, SwaggerInfo.Title)
				assert.NotEmpty(t, SwaggerInfo.Description)
			},
		},
		{
			name: "has correct default values",
			validate: func(t *testing.T) {
				assert.Equal(t, "1.0.0", SwaggerInfo.Version)
				assert.Equal(t, "localhost:4021", SwaggerInfo.Host)
				assert.Equal(t, "/", SwaggerInfo.BasePath)
				assert.Equal(t, "swagger", SwaggerInfo.InfoInstanceName)
				assert.Equal(t, "{{", SwaggerInfo.LeftDelim)
				assert.Equal(t, "}}", SwaggerInfo.RightDelim)
			},
		},
		{
			name: "returns correct instance name",
			validate: func(t *testing.T) {
				assert.Equal(t, "swagger", SwaggerInfo.InstanceName())
			},
		},
		{
			name: "has non-empty template",
			validate: func(t *testing.T) {
				require.NotEmpty(t, SwaggerInfo.SwaggerTemplate)
				assert.Contains(t, SwaggerInfo.SwaggerTemplate, "swagger")
				assert.Contains(t, SwaggerInfo.SwaggerTemplate, "paths")
				assert.Contains(t, SwaggerInfo.SwaggerTemplate, "definitions")
			},
		},
		{
			name: "is registered with swag",
			validate: func(t *testing.T) {
				spec := swag.GetSwagger("swagger")
				require.NotNil(t, spec)
				assert.Equal(t, SwaggerInfo, spec)
			},
		},
		{
			name: "ReadDoc returns valid JSON",
			validate: func(t *testing.T) {
				doc := SwaggerInfo.ReadDoc()
				require.NotEmpty(t, doc)
				assert.Contains(t, doc, "swagger")
				assert.Contains(t, doc, "2.0")
				assert.Contains(t, doc, "/v1/workflows")
			},
		},
		{
			name: "schemes is empty by default",
			validate: func(t *testing.T) {
				assert.Empty(t, SwaggerInfo.Schemes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t)
		})
	}
}

func TestDocTemplate(t *testing.T) {
	tests := []struct {
		name            string
		expectedContent []string
	}{
		{
			name: "contains API paths",
			expectedContent: []string{
				"/v1/workflows",
				"/v1/workflows/{id}",
				"/v1/catalog/executors",
				"/v1/catalog/triggers",
				"/health",
			},
		},
		{
			name: "contains definitions",
			expectedContent: []string{
				"WorkflowCreateOutput",
				"WorkflowNodeInput",
				"ExecutorSummary",
				"TriggerDetail",
			},
		},
		{
			name: "contains HTTP methods",
			expectedContent: []string{
				`"get"`,
				`"post"`,
				`"put"`,
				`"delete"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, content := range tt.expectedContent {
				assert.Contains(t, docTemplate, content)
			}
		})
	}
}
