// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package catalog_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/internal/adapters/http/in/catalog"
	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testProvider implements executor.Provider for handler tests.
type testProvider struct {
	id           executor.ProviderID
	name         string
	description  string
	version      string
	configSchema string
}

func (p *testProvider) ID() executor.ProviderID { return p.id }
func (p *testProvider) Name() string            { return p.name }
func (p *testProvider) Description() string     { return p.description }
func (p *testProvider) Version() string         { return p.version }
func (p *testProvider) ConfigSchema() string    { return p.configSchema }

// testExecutor implements executor.Executor for handler tests.
type testExecutor struct {
	id         executor.ID
	name       string
	category   string
	version    string
	providerID executor.ProviderID
	schema     string
}

func (e *testExecutor) ID() executor.ID                       { return e.id }
func (e *testExecutor) Name() string                          { return e.name }
func (e *testExecutor) Category() string                      { return e.category }
func (e *testExecutor) Version() string                       { return e.version }
func (e *testExecutor) ProviderID() executor.ProviderID       { return e.providerID }
func (e *testExecutor) Schema() string                        { return e.schema }
func (e *testExecutor) ValidateConfig(_ map[string]any) error { return nil }

// testRunner implements executor.Runner for handler tests.
type testRunner struct {
	executorID executor.ID
}

func (r *testRunner) ExecutorID() executor.ID { return r.executorID }
func (r *testRunner) Execute(_ context.Context, _ executor.ExecutionInput) (executor.ExecutionResult, error) {
	return executor.ExecutionResult{}, nil
}

// setupCatalogWithProviders creates a catalog with test providers and returns the handler.
func setupCatalogWithProviders(t *testing.T) (*catalog.Handler, *fiber.App) {
	t.Helper()

	cat := executor.NewCatalog()

	err := cat.RegisterProvider(
		&testProvider{
			id:           "http",
			name:         "HTTP",
			description:  "Generic HTTP provider",
			version:      "v1",
			configSchema: `{"type":"object","properties":{"base_url":{"type":"string"}},"required":["base_url"]}`,
		},
		[]executor.ExecutorRegistration{
			{
				Executor: &testExecutor{
					id:         "http.request",
					name:       "HTTP Request",
					category:   "HTTP",
					version:    "v1",
					providerID: "http",
					schema:     `{"type":"object"}`,
				},
				Runner: &testRunner{executorID: "http.request"},
			},
		},
	)
	require.NoError(t, err)

	err = cat.RegisterProvider(
		&testProvider{
			id:           "midaz",
			name:         "Midaz",
			description:  "Midaz ledger provider",
			version:      "v1",
			configSchema: `{"type":"object","properties":{"base_url":{"type":"string"},"api_key":{"type":"string"}},"required":["base_url","api_key"]}`,
		},
		[]executor.ExecutorRegistration{
			{
				Executor: &testExecutor{
					id:         "midaz.create-transaction",
					name:       "Create Transaction",
					category:   "Midaz",
					version:    "v1",
					providerID: "midaz",
					schema:     `{"type":"object"}`,
				},
				Runner: &testRunner{executorID: "midaz.create-transaction"},
			},
		},
	)
	require.NoError(t, err)

	handler, err := catalog.NewHandler(cat, nil)
	require.NoError(t, err)

	app := fiber.New()
	v1 := app.Group("/v1")
	handler.RegisterRoutes(v1)

	return handler, app
}

func TestNewHandler_NilCatalog(t *testing.T) {
	handler, err := catalog.NewHandler(nil, nil)

	require.Error(t, err)
	assert.Nil(t, handler)
	assert.Equal(t, catalog.ErrCatalogHandlerNilDependency, err)
}

func TestListProviders(t *testing.T) {
	_, app := setupCatalogWithProviders(t)

	req := httptest.NewRequest("GET", "/v1/catalog/providers", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var providers []catalog.ProviderSummary
	err = json.Unmarshal(body, &providers)
	require.NoError(t, err)

	assert.Len(t, providers, 2, "should have two providers")

	// Providers should be sorted by ID
	assert.Equal(t, "http", providers[0].ID)
	assert.Equal(t, "HTTP", providers[0].Name)
	assert.Equal(t, "midaz", providers[1].ID)
	assert.Equal(t, "Midaz", providers[1].Name)
}

func TestGetProvider(t *testing.T) {
	tests := []struct {
		name       string
		providerID string
		wantStatus int
		wantName   string
	}{
		{
			name:       "existing provider returns detail",
			providerID: "http",
			wantStatus: fiber.StatusOK,
			wantName:   "HTTP",
		},
		{
			name:       "non-existing provider returns 404",
			providerID: "unknown",
			wantStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, app := setupCatalogWithProviders(t)

			req := httptest.NewRequest("GET", "/v1/catalog/providers/"+tt.providerID, nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantStatus == fiber.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var detail catalog.ProviderDetail
				err = json.Unmarshal(body, &detail)
				require.NoError(t, err)

				assert.Equal(t, tt.providerID, detail.ID)
				assert.Equal(t, tt.wantName, detail.Name)
				assert.NotEmpty(t, detail.ConfigSchema, "config schema should be present")
			}
		})
	}
}

func TestGetProviderExecutors(t *testing.T) {
	tests := []struct {
		name       string
		providerID string
		wantStatus int
		wantCount  int
		wantExecID string
	}{
		{
			name:       "existing provider returns its executors",
			providerID: "http",
			wantStatus: fiber.StatusOK,
			wantCount:  1,
			wantExecID: "http.request",
		},
		{
			name:       "midaz provider returns its executors",
			providerID: "midaz",
			wantStatus: fiber.StatusOK,
			wantCount:  1,
			wantExecID: "midaz.create-transaction",
		},
		{
			name:       "non-existing provider returns 404",
			providerID: "unknown",
			wantStatus: fiber.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, app := setupCatalogWithProviders(t)

			req := httptest.NewRequest("GET", "/v1/catalog/providers/"+tt.providerID+"/executors", nil)
			resp, err := app.Test(req, -1)
			require.NoError(t, err)

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantStatus == fiber.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var executors []catalog.ExecutorSummary
				err = json.Unmarshal(body, &executors)
				require.NoError(t, err)

				assert.Len(t, executors, tt.wantCount)

				if tt.wantCount > 0 {
					assert.Equal(t, tt.wantExecID, executors[0].ID)
					assert.Equal(t, tt.providerID, executors[0].ProviderID)
				}
			}
		})
	}
}

func TestListExecutors_IncludesProviderID(t *testing.T) {
	_, app := setupCatalogWithProviders(t)

	req := httptest.NewRequest("GET", "/v1/catalog/executors", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var executors []catalog.ExecutorSummary
	err = json.Unmarshal(body, &executors)
	require.NoError(t, err)

	assert.Len(t, executors, 2)

	// Each executor should have its providerID populated.
	for _, e := range executors {
		assert.NotEmpty(t, e.ProviderID, "executor %s should have a providerID", e.ID)
	}
}

func TestGetExecutor_IncludesProviderID(t *testing.T) {
	_, app := setupCatalogWithProviders(t)

	req := httptest.NewRequest("GET", "/v1/catalog/executors/http.request", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var detail catalog.ExecutorDetail
	err = json.Unmarshal(body, &detail)
	require.NoError(t, err)

	assert.Equal(t, "http.request", detail.ID)
	assert.Equal(t, "http", detail.ProviderID)
}

// testTemplate implements executor.Template for handler tests.
type testTemplate struct {
	id                   executor.TemplateID
	name                 string
	description          string
	version              string
	category             string
	paramSchema          string
	providerConfigFields []executor.ProviderConfigField
}

func (t *testTemplate) ID() executor.TemplateID               { return t.id }
func (t *testTemplate) Name() string                          { return t.name }
func (t *testTemplate) Description() string                   { return t.description }
func (t *testTemplate) Version() string                       { return t.version }
func (t *testTemplate) Category() string                      { return t.category }
func (t *testTemplate) ParamSchema() string                   { return t.paramSchema }
func (t *testTemplate) ValidateParams(_ map[string]any) error { return nil }
func (t *testTemplate) Build(_ map[string]any) (any, error)   { return nil, nil }
func (t *testTemplate) ProviderConfigFields() []executor.ProviderConfigField {
	return t.providerConfigFields
}

// mockConfigLister implements catalog.ProviderConfigLister for tests.
type mockConfigLister struct {
	configs map[string][]*model.ProviderConfiguration // keyed by providerID
	err     error
}

func (m *mockConfigLister) ListActiveByProvider(_ context.Context, providerID string) ([]*model.ProviderConfiguration, error) {
	if m.err != nil {
		return nil, m.err
	}

	return m.configs[providerID], nil
}

func TestGetTemplate_EnrichesSchemaWithProviderConfigOptions(t *testing.T) {
	cat := executor.NewCatalog()

	err := cat.RegisterTemplate(&testTemplate{
		id:          "test-template",
		name:        "Test Template",
		description: "A test template",
		version:     "v1",
		category:    "Testing",
		paramSchema: `{
			"type": "object",
			"properties": {
				"tracerProviderConfigId": {
					"type": "string",
					"format": "uuid",
					"description": "Provider config for Tracer"
				},
				"midazProviderConfigId": {
					"type": "string",
					"format": "uuid",
					"description": "Provider config for Midaz"
				}
			}
		}`,
		providerConfigFields: []executor.ProviderConfigField{
			{ParamName: "tracerProviderConfigId", ProviderID: "tracer"},
			{ParamName: "midazProviderConfigId", ProviderID: "midaz"},
		},
	})
	require.NoError(t, err)

	tracerConfig, err := model.NewProviderConfiguration("prod-tracer", nil, "tracer", map[string]any{"url": "https://tracer.example.com"})
	require.NoError(t, err)

	midazConfig, err := model.NewProviderConfiguration("prod-midaz", nil, "midaz", map[string]any{"url": "https://midaz.example.com"})
	require.NoError(t, err)

	lister := &mockConfigLister{
		configs: map[string][]*model.ProviderConfiguration{
			"tracer": {tracerConfig},
			"midaz":  {midazConfig},
		},
	}

	handler, err := catalog.NewHandler(cat, lister)
	require.NoError(t, err)

	app := fiber.New()
	v1 := app.Group("/v1")
	handler.RegisterRoutes(v1)

	req := httptest.NewRequest("GET", "/v1/catalog/templates/test-template", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var detail catalog.TemplateDetail
	err = json.Unmarshal(body, &detail)
	require.NoError(t, err)

	// Parse the enriched param schema
	var schema map[string]any
	err = json.Unmarshal([]byte(detail.ParamSchema), &schema)
	require.NoError(t, err)

	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok)

	// Check tracerProviderConfigId has oneOf
	tracerProp, ok := properties["tracerProviderConfigId"].(map[string]any)
	require.True(t, ok)
	tracerOneOf, ok := tracerProp["oneOf"].([]any)
	require.True(t, ok, "tracerProviderConfigId should have oneOf")
	require.Len(t, tracerOneOf, 1)

	tracerOption, ok := tracerOneOf[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, tracerConfig.ID().String(), tracerOption["const"])
	assert.Equal(t, "prod-tracer", tracerOption["title"])

	// Check midazProviderConfigId has oneOf
	midazProp, ok := properties["midazProviderConfigId"].(map[string]any)
	require.True(t, ok)
	midazOneOf, ok := midazProp["oneOf"].([]any)
	require.True(t, ok, "midazProviderConfigId should have oneOf")
	require.Len(t, midazOneOf, 1)

	midazOption, ok := midazOneOf[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, midazConfig.ID().String(), midazOption["const"])
	assert.Equal(t, "prod-midaz", midazOption["title"])
}

func TestGetTemplate_NilConfigLister_ReturnsStaticSchema(t *testing.T) {
	cat := executor.NewCatalog()

	staticSchema := `{"type":"object","properties":{"configId":{"type":"string"}}}`
	err := cat.RegisterTemplate(&testTemplate{
		id:          "static-template",
		name:        "Static Template",
		description: "Template without enrichment",
		version:     "v1",
		category:    "Testing",
		paramSchema: staticSchema,
		providerConfigFields: []executor.ProviderConfigField{
			{ParamName: "configId", ProviderID: "some-provider"},
		},
	})
	require.NoError(t, err)

	handler, err := catalog.NewHandler(cat, nil)
	require.NoError(t, err)

	app := fiber.New()
	v1 := app.Group("/v1")
	handler.RegisterRoutes(v1)

	req := httptest.NewRequest("GET", "/v1/catalog/templates/static-template", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var detail catalog.TemplateDetail
	err = json.Unmarshal(body, &detail)
	require.NoError(t, err)

	// Schema should be returned as-is (no enrichment)
	assert.Equal(t, staticSchema, detail.ParamSchema)
}

func TestGetTemplate_ConfigListerError_FallsBackToStaticSchema(t *testing.T) {
	cat := executor.NewCatalog()

	err := cat.RegisterTemplate(&testTemplate{
		id:          "fallback-template",
		name:        "Fallback Template",
		description: "Template with failing lister",
		version:     "v1",
		category:    "Testing",
		paramSchema: `{"type":"object","properties":{"configId":{"type":"string","format":"uuid"}}}`,
		providerConfigFields: []executor.ProviderConfigField{
			{ParamName: "configId", ProviderID: "failing-provider"},
		},
	})
	require.NoError(t, err)

	lister := &mockConfigLister{
		err: fmt.Errorf("database connection error"),
	}

	handler, err := catalog.NewHandler(cat, lister)
	require.NoError(t, err)

	app := fiber.New()
	v1 := app.Group("/v1")
	handler.RegisterRoutes(v1)

	req := httptest.NewRequest("GET", "/v1/catalog/templates/fallback-template", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var detail catalog.TemplateDetail
	err = json.Unmarshal(body, &detail)
	require.NoError(t, err)

	// Schema should still be returned (best-effort, skip failed fields)
	var schema map[string]any
	err = json.Unmarshal([]byte(detail.ParamSchema), &schema)
	require.NoError(t, err)

	// configId should NOT have oneOf since the lister failed
	properties := schema["properties"].(map[string]any)
	configProp := properties["configId"].(map[string]any)
	_, hasOneOf := configProp["oneOf"]
	assert.False(t, hasOneOf, "should not have oneOf when lister fails")
}

func TestGetTemplate_NoProviderConfigFields_ReturnsStaticSchema(t *testing.T) {
	cat := executor.NewCatalog()

	staticSchema := `{"type":"object","properties":{"name":{"type":"string"}}}`
	err := cat.RegisterTemplate(&testTemplate{
		id:                   "no-fields-template",
		name:                 "No Fields Template",
		description:          "Template with no provider config fields",
		version:              "v1",
		category:             "Testing",
		paramSchema:          staticSchema,
		providerConfigFields: nil,
	})
	require.NoError(t, err)

	lister := &mockConfigLister{
		configs: map[string][]*model.ProviderConfiguration{},
	}

	handler, err := catalog.NewHandler(cat, lister)
	require.NoError(t, err)

	app := fiber.New()
	v1 := app.Group("/v1")
	handler.RegisterRoutes(v1)

	req := httptest.NewRequest("GET", "/v1/catalog/templates/no-fields-template", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var detail catalog.TemplateDetail
	err = json.Unmarshal(body, &detail)
	require.NoError(t, err)

	// Schema should be returned as-is
	assert.Equal(t, staticSchema, detail.ParamSchema)
}
