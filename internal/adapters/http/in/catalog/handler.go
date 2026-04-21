// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package catalog

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/LerianStudio/flowker/api"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/model"
	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libHTTP "github.com/LerianStudio/lib-commons/v4/commons/net/http"
	"github.com/gofiber/fiber/v2"
)

// ProviderConfigLister provides read-only access to provider configurations.
// Used by the catalog handler to enrich template schemas with available options.
type ProviderConfigLister interface {
	ListActiveByProvider(ctx context.Context, providerID string) ([]*model.ProviderConfiguration, error)
}

// Handler exposes read-only catalog endpoints for built-in executors, triggers, and providers.
type Handler struct {
	catalog      executor.Catalog
	configLister ProviderConfigLister // optional, nil = no dynamic options
}

// ErrCatalogHandlerNilDependency is returned when a required dependency is nil.
var ErrCatalogHandlerNilDependency = errors.New("catalog handler: catalog cannot be nil")

// NewHandler creates a new catalog handler.
// Returns error if catalog is nil. configLister is optional (pass nil to disable dynamic schema enrichment).
func NewHandler(catalog executor.Catalog, configLister ProviderConfigLister) (*Handler, error) {
	if catalog == nil {
		return nil, ErrCatalogHandlerNilDependency
	}

	return &Handler{catalog: catalog, configLister: configLister}, nil
}

// RegisterRoutes registers catalog routes.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	catalog := router.Group("/catalog")

	catalog.Get("/executors", h.ListExecutors)
	catalog.Get("/executors/:id", h.GetExecutor)
	catalog.Post("/executors/:id/validate", h.ValidateExecutor)

	catalog.Get("/triggers", h.ListTriggers)
	catalog.Get("/triggers/:id", h.GetTrigger)

	catalog.Get("/providers", h.ListProviders)
	catalog.Get("/providers/:id", h.GetProvider)
	catalog.Get("/providers/:id/executors", h.GetProviderExecutors)

	catalog.Get("/templates", h.ListTemplates)
	catalog.Get("/templates/:id", h.GetTemplate)
	catalog.Post("/templates/:id/validate", h.ValidateTemplateParams)
}

// ExecutorSummary represents an executor without schema.
type ExecutorSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	Version    string `json:"version"`
	ProviderID string `json:"providerId,omitempty"`
}

// ExecutorDetail represents an executor with schema included.
type ExecutorDetail struct {
	ExecutorSummary
	Schema string `json:"schema"`
}

// TriggerSummary represents a trigger without schema.
type TriggerSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// TriggerDetail represents a trigger with schema.
type TriggerDetail struct {
	TriggerSummary
	Schema string `json:"schema"`
}

// ProviderSummary represents a provider without config schema.
type ProviderSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// ProviderDetail represents a provider with config schema included.
type ProviderDetail struct {
	ProviderSummary
	ConfigSchema string `json:"configSchema"`
}

// TemplateSummary represents a template without param schema.
type TemplateSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Category    string `json:"category"`
}

// TemplateDetail represents a template with param schema included.
type TemplateDetail struct {
	TemplateSummary
	ParamSchema string `json:"paramSchema"`
}

// TemplateValidationRequest carries params to validate against a template schema.
type TemplateValidationRequest struct {
	Params map[string]any `json:"params"`
}

// ValidationRequest carries a config to validate against an executor schema.
type ValidationRequest struct {
	Config map[string]any `json:"config"`
}

// ValidationResponse indicates validation result.
type ValidationResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

// ListExecutors handles GET /v1/catalog/executors
// @Summary      List executors
// @Description  Returns all built-in executors registered in the catalog
// @Tags         Catalog
// @Produce      json
// @Success      200  {array}   ExecutorSummary
// @Router       /v1/catalog/executors [get]
func (h *Handler) ListExecutors(c *fiber.Ctx) error {
	executors := h.catalog.ListExecutors()

	result := make([]ExecutorSummary, 0, len(executors))
	for _, e := range executors {
		result = append(result, ExecutorSummary{
			ID:         string(e.ID()),
			Name:       e.Name(),
			Category:   e.Category(),
			Version:    e.Version(),
			ProviderID: string(e.ProviderID()),
		})
	}

	return libHTTP.Respond(c, fiber.StatusOK, result)
}

// GetExecutor handles GET /v1/catalog/executors/:id
// @Summary      Get executor
// @Description  Returns executor metadata and its JSON Schema
// @Tags         Catalog
// @Produce      json
// @Param        id   path      string  true  "Executor ID"
// @Success      200  {object}  ExecutorDetail
// @Failure      404  {object}  libCommons.Response
// @Router       /v1/catalog/executors/{id} [get]
func (h *Handler) GetExecutor(c *fiber.Ctx) error {
	id := executor.ID(c.Params("id"))

	e, err := h.catalog.GetExecutor(id)
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrExecutorNotFound.Error(), Title: "Not Found", Message: "executor not found"})
	}

	return libHTTP.Respond(c, fiber.StatusOK, ExecutorDetail{
		ExecutorSummary: ExecutorSummary{
			ID:         string(e.ID()),
			Name:       e.Name(),
			Category:   e.Category(),
			Version:    e.Version(),
			ProviderID: string(e.ProviderID()),
		},
		Schema: e.Schema(),
	})
}

// ValidateExecutor handles POST /v1/catalog/executors/:id/validate
// @Summary      Validate executor config
// @Description  Validates an executor configuration against its JSON Schema
// @Tags         Catalog
// @Accept       json
// @Produce      json
// @Param        id       path      string             true  "Executor ID"
// @Param        request  body      ValidationRequest  true  "Configuration to validate"
// @Success      200   {object}  ValidationResponse
// @Failure      400   {object}  ValidationResponse
// @Failure      404   {object}  libCommons.Response
// @Router       /v1/catalog/executors/{id}/validate [post]
func (h *Handler) ValidateExecutor(c *fiber.Ctx) error {
	id := executor.ID(c.Params("id"))

	e, err := h.catalog.GetExecutor(id)
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrExecutorNotFound.Error(), Title: "Not Found", Message: "executor not found"})
	}

	var req ValidationRequest
	if err := c.BodyParser(&req); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, libCommons.Response{
			Code:    constant.ErrInvalidRequestBody.Error(),
			Title:   "Bad Request",
			Message: "invalid request body",
		})
	}

	if err := e.ValidateConfig(req.Config); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, libCommons.Response{
			Code:    constant.ErrExecutorInvalidConfig.Error(),
			Title:   "Bad Request",
			Message: err.Error(),
		})
	}

	return libHTTP.Respond(c, fiber.StatusOK, ValidationResponse{Valid: true})
}

// ListTriggers handles GET /v1/catalog/triggers
// @Summary      List triggers
// @Description  Returns all built-in triggers registered in the catalog
// @Tags         Catalog
// @Produce      json
// @Success      200  {array}   TriggerSummary
// @Router       /v1/catalog/triggers [get]
func (h *Handler) ListTriggers(c *fiber.Ctx) error {
	triggers := h.catalog.ListTriggers()

	result := make([]TriggerSummary, 0, len(triggers))
	for _, t := range triggers {
		result = append(result, TriggerSummary{
			ID:      string(t.ID()),
			Name:    t.Name(),
			Version: t.Version(),
		})
	}

	return libHTTP.Respond(c, fiber.StatusOK, result)
}

// GetTrigger handles GET /v1/catalog/triggers/:id
// @Summary      Get trigger
// @Description  Returns trigger metadata and its JSON Schema
// @Tags         Catalog
// @Produce      json
// @Param        id   path      string  true  "Trigger ID"
// @Success      200  {object}  TriggerDetail
// @Failure      404  {object}  libCommons.Response
// @Router       /v1/catalog/triggers/{id} [get]
func (h *Handler) GetTrigger(c *fiber.Ctx) error {
	id := executor.TriggerID(c.Params("id"))

	t, err := h.catalog.GetTrigger(id)
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrTriggerNotFound.Error(), Title: "Not Found", Message: "trigger not found"})
	}

	return libHTTP.Respond(c, fiber.StatusOK, TriggerDetail{
		TriggerSummary: TriggerSummary{
			ID:      string(t.ID()),
			Name:    t.Name(),
			Version: t.Version(),
		},
		Schema: t.Schema(),
	})
}

// ListProviders handles GET /v1/catalog/providers
// @Summary      List providers
// @Description  Returns all registered providers from the static catalog
// @Tags         Catalog
// @Produce      json
// @Success      200  {array}   ProviderSummary
// @Router       /v1/catalog/providers [get]
func (h *Handler) ListProviders(c *fiber.Ctx) error {
	providers := h.catalog.ListProviders()

	result := make([]ProviderSummary, 0, len(providers))
	for _, p := range providers {
		result = append(result, ProviderSummary{
			ID:          string(p.ID()),
			Name:        p.Name(),
			Description: p.Description(),
			Version:     p.Version(),
		})
	}

	return libHTTP.Respond(c, fiber.StatusOK, result)
}

// GetProvider handles GET /v1/catalog/providers/:id
// @Summary      Get provider
// @Description  Returns provider metadata and its JSON Schema for configuration
// @Tags         Catalog
// @Produce      json
// @Param        id   path      string  true  "Provider ID"
// @Success      200  {object}  ProviderDetail
// @Failure      404  {object}  libCommons.Response
// @Router       /v1/catalog/providers/{id} [get]
func (h *Handler) GetProvider(c *fiber.Ctx) error {
	id := executor.ProviderID(c.Params("id"))

	p, err := h.catalog.GetProvider(id)
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrProviderNotFound.Error(), Title: "Not Found", Message: "provider not found"})
	}

	return libHTTP.Respond(c, fiber.StatusOK, ProviderDetail{
		ProviderSummary: ProviderSummary{
			ID:          string(p.ID()),
			Name:        p.Name(),
			Description: p.Description(),
			Version:     p.Version(),
		},
		ConfigSchema: p.ConfigSchema(),
	})
}

// GetProviderExecutors handles GET /v1/catalog/providers/:id/executors
// @Summary      List provider executors
// @Description  Returns all executors belonging to a specific provider
// @Tags         Catalog
// @Produce      json
// @Param        id   path      string  true  "Provider ID"
// @Success      200  {array}   ExecutorSummary
// @Failure      404  {object}  libCommons.Response
// @Router       /v1/catalog/providers/{id}/executors [get]
func (h *Handler) GetProviderExecutors(c *fiber.Ctx) error {
	id := executor.ProviderID(c.Params("id"))

	executors, err := h.catalog.GetProviderExecutors(id)
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrProviderNotFound.Error(), Title: "Not Found", Message: "provider not found"})
	}

	result := make([]ExecutorSummary, 0, len(executors))
	for _, e := range executors {
		result = append(result, ExecutorSummary{
			ID:         string(e.ID()),
			Name:       e.Name(),
			Category:   e.Category(),
			Version:    e.Version(),
			ProviderID: string(e.ProviderID()),
		})
	}

	return libHTTP.Respond(c, fiber.StatusOK, result)
}

// ListTemplates handles GET /v1/catalog/templates
// @Summary      List templates
// @Description  Returns all workflow templates registered in the catalog
// @Tags         Catalog
// @Produce      json
// @Success      200  {array}   TemplateSummary
// @Router       /v1/catalog/templates [get]
func (h *Handler) ListTemplates(c *fiber.Ctx) error {
	templates := h.catalog.ListTemplates()

	result := make([]TemplateSummary, 0, len(templates))
	for _, t := range templates {
		result = append(result, TemplateSummary{
			ID:          string(t.ID()),
			Name:        t.Name(),
			Description: t.Description(),
			Version:     t.Version(),
			Category:    t.Category(),
		})
	}

	return libHTTP.Respond(c, fiber.StatusOK, result)
}

// GetTemplate handles GET /v1/catalog/templates/:id
// @Summary      Get template
// @Description  Returns template metadata and its JSON Schema for parameters
// @Tags         Catalog
// @Produce      json
// @Param        id   path      string  true  "Template ID"
// @Success      200  {object}  TemplateDetail
// @Failure      404  {object}  libCommons.Response
// @Router       /v1/catalog/templates/{id} [get]
func (h *Handler) GetTemplate(c *fiber.Ctx) error {
	id := executor.TemplateID(c.Params("id"))

	t, err := h.catalog.GetTemplate(id)
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrTemplateNotFound.Error(), Title: "Not Found", Message: "template not found"})
	}

	schema := t.ParamSchema()

	// Enrich schema with provider config options if available
	if h.configLister != nil {
		enrichedSchema, enrichErr := h.enrichSchemaWithOptions(c.Context(), t, schema)
		if enrichErr == nil {
			schema = enrichedSchema
		}
	}

	return libHTTP.Respond(c, fiber.StatusOK, TemplateDetail{
		TemplateSummary: TemplateSummary{
			ID:          string(t.ID()),
			Name:        t.Name(),
			Description: t.Description(),
			Version:     t.Version(),
			Category:    t.Category(),
		},
		ParamSchema: schema,
	})
}

// enrichSchemaWithOptions enriches JSON Schema properties that reference provider configurations
// with oneOf options from active provider configs in the database.
func (h *Handler) enrichSchemaWithOptions(ctx context.Context, t executor.Template, rawSchema string) (string, error) {
	fields := t.ProviderConfigFields()
	if len(fields) == 0 {
		return rawSchema, nil
	}

	var schema map[string]any
	if err := json.Unmarshal([]byte(rawSchema), &schema); err != nil {
		return rawSchema, err
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return rawSchema, nil
	}

	for _, field := range fields {
		prop, propOK := properties[field.ParamName].(map[string]any)
		if !propOK {
			continue
		}

		configs, listErr := h.configLister.ListActiveByProvider(ctx, string(field.ProviderID))
		if listErr != nil {
			continue // best-effort: skip if query fails
		}

		// Always set oneOf — empty array signals the frontend to show
		// "no provider configs available" instead of rendering a text input.
		options := make([]map[string]any, 0, len(configs))
		for _, cfg := range configs {
			options = append(options, map[string]any{
				"const": cfg.ID().String(),
				"title": cfg.Name(),
			})
		}

		prop["oneOf"] = options
	}

	enriched, err := json.Marshal(schema)
	if err != nil {
		return rawSchema, err
	}

	return string(enriched), nil
}

// ValidateTemplateParams handles POST /v1/catalog/templates/:id/validate
// @Summary      Validate template params
// @Description  Validates template parameters against its JSON Schema
// @Tags         Catalog
// @Accept       json
// @Produce      json
// @Param        id       path      string                     true  "Template ID"
// @Param        request  body      TemplateValidationRequest  true  "Parameters to validate"
// @Success      200   {object}  ValidationResponse
// @Failure      400   {object}  ValidationResponse
// @Failure      404   {object}  libCommons.Response
// @Router       /v1/catalog/templates/{id}/validate [post]
func (h *Handler) ValidateTemplateParams(c *fiber.Ctx) error {
	id := executor.TemplateID(c.Params("id"))

	t, err := h.catalog.GetTemplate(id)
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrTemplateNotFound.Error(), Title: "Not Found", Message: "template not found"})
	}

	var req TemplateValidationRequest
	if err := c.BodyParser(&req); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, libCommons.Response{
			Code:    constant.ErrInvalidRequestBody.Error(),
			Title:   "Bad Request",
			Message: "invalid request body",
		})
	}

	if err := t.ValidateParams(req.Params); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, libCommons.Response{
			Code:    constant.ErrTemplateInvalidParams.Error(),
			Title:   "Bad Request",
			Message: err.Error(),
		})
	}

	return libHTTP.Respond(c, fiber.StatusOK, ValidationResponse{Valid: true})
}
