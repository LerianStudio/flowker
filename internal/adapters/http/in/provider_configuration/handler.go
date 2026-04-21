// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package providerconfiguration contains the HTTP handler for provider configuration operations.
package providerconfiguration

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/LerianStudio/flowker/api"
	"github.com/LerianStudio/flowker/internal/services/query"
	pkg "github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	libHTTP "github.com/LerianStudio/lib-commons/v4/commons/net/http"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CommandService defines the interface for provider configuration command operations.
type CommandService interface {
	Create(ctx context.Context, input *model.CreateProviderConfigurationInput) (*model.ProviderConfiguration, error)
	Update(ctx context.Context, id uuid.UUID, input *model.UpdateProviderConfigurationInput) (*model.ProviderConfiguration, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Disable(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error)
	Enable(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error)
	TestConnectivity(ctx context.Context, id uuid.UUID) (*model.ProviderConfigTestResult, error)
}

// QueryService defines the interface for provider configuration query operations.
type QueryService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error)
	List(ctx context.Context, filter query.ProviderConfigListFilter) (*query.ProviderConfigListResult, error)
}

// Handler handles HTTP requests for provider configuration operations.
type Handler struct {
	cmdSvc   CommandService
	querySvc QueryService
}

// ErrProviderConfigHandlerNilDependency is returned when a required dependency is nil.
var ErrProviderConfigHandlerNilDependency = errors.New("provider configuration handler: required dependency cannot be nil")

// NewHandler creates a new provider configuration HTTP handler.
// Returns error if required dependencies are nil.
func NewHandler(cmdSvc CommandService, querySvc QueryService) (*Handler, error) {
	if cmdSvc == nil || querySvc == nil {
		return nil, ErrProviderConfigHandlerNilDependency
	}

	return &Handler{
		cmdSvc:   cmdSvc,
		querySvc: querySvc,
	}, nil
}

// RegisterRoutes registers all provider configuration routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	providerConfigs := router.Group("/provider-configurations")

	providerConfigs.Post("/", h.Create)
	providerConfigs.Get("/", h.List)
	providerConfigs.Get("/:id", h.GetByID)
	providerConfigs.Patch("/:id", h.Update)
	providerConfigs.Delete("/:id", h.Delete)
	providerConfigs.Post("/:id/disable", h.Disable)
	providerConfigs.Post("/:id/enable", h.Enable)
	providerConfigs.Post("/:id/test", h.TestConnectivity)
}

// Create handles POST /v1/provider-configurations
// @Summary      Create a provider configuration
// @Description  Creates a new provider configuration validated against the provider's JSON Schema
// @Tags         Provider Configurations
// @Accept       json
// @Produce      json
// @Param        providerConfig  body      model.CreateProviderConfigurationInput  true  "Provider Configuration Input"
// @Success      201  {object}  model.ProviderConfigurationCreateOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/provider-configurations [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var input model.CreateProviderConfigurationInput
	if err := c.BodyParser(&input); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidRequestBody.Error(), Title: "Bad Request", Message: "invalid request body"})
	}

	providerConfig, err := h.cmdSvc.Create(c.Context(), &input)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ProviderConfigurationCreateOutputFromDomain(providerConfig)

	return libHTTP.Respond(c, fiber.StatusCreated, output)
}

// GetByID handles GET /v1/provider-configurations/:id
// @Summary      Get provider configuration by ID
// @Description  Retrieves a provider configuration by its ID
// @Tags         Provider Configurations
// @Produce      json
// @Param        id   path      string  true  "Provider Configuration ID"  Format(uuid)
// @Success      200  {object}  model.ProviderConfigurationOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/provider-configurations/{id} [get]
func (h *Handler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid provider configuration ID"})
	}

	providerConfig, err := h.querySvc.GetByID(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ProviderConfigurationOutputFromDomain(providerConfig)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// List handles GET /v1/provider-configurations
// @Summary      List provider configurations
// @Description  Retrieves a paginated list of provider configurations with optional filtering
// @Tags         Provider Configurations
// @Produce      json
// @Param        status       query     string  false  "Filter by status"  Enums(active, disabled)
// @Param        providerId   query     string  false  "Filter by provider ID"
// @Param        limit        query     int     false  "Number of items per page"  default(10)  minimum(1)  maximum(100)
// @Param        cursor       query     string  false  "Pagination cursor"
// @Param        sortBy       query     string  false  "Sort field"  Enums(createdAt, updatedAt, name)  default(createdAt)
// @Param        sortOrder    query     string  false  "Sort order"  Enums(ASC, DESC)  default(DESC)
// @Success      200          {object}  model.ProviderConfigurationListOutput
// @Failure      400          {object}  api.ErrorResponse
// @Failure      500          {object}  api.ErrorResponse
// @Router       /v1/provider-configurations [get]
func (h *Handler) List(c *fiber.Ctx) error {
	filter := query.ProviderConfigListFilter{
		Cursor:    c.Query("cursor"),
		SortBy:    c.Query("sortBy", "createdAt"),
		SortOrder: c.Query("sortOrder", "DESC"),
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > constant.MaxPaginationLimit {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
				Code:    constant.ErrInvalidQueryParameter.Error(),
				Title:   "Bad Request",
				Message: "invalid limit parameter: must be an integer between 1 and 100",
			})
		}

		filter.Limit = limit
	} else {
		filter.Limit = 10
	}

	// Parse status filter
	if statusStr := c.Query("status"); statusStr != "" {
		status := model.ProviderConfigurationStatus(statusStr)
		if status != model.ProviderConfigStatusActive && status != model.ProviderConfigStatusDisabled {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
				Code:    constant.ErrInvalidQueryParameter.Error(),
				Title:   "Bad Request",
				Message: "invalid status parameter: must be 'active' or 'disabled'",
			})
		}

		filter.Status = &status
	}

	// Parse providerId filter
	if providerID := c.Query("providerId"); providerID != "" {
		filter.ProviderID = &providerID
	}

	result, err := h.querySvc.List(c.Context(), filter)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ProviderConfigurationListOutputFromDomain(result.Items, result.NextCursor, result.HasMore)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// Update handles PATCH /v1/provider-configurations/:id
// @Summary      Update a provider configuration
// @Description  Updates an existing provider configuration. If config is changed, re-validates against provider JSON Schema.
// @Tags         Provider Configurations
// @Accept       json
// @Produce      json
// @Param        id              path      string                                    true  "Provider Configuration ID"  Format(uuid)
// @Param        providerConfig  body      model.UpdateProviderConfigurationInput     true  "Updated provider configuration"
// @Success      200             {object}  model.ProviderConfigurationOutput
// @Failure      400             {object}  api.ErrorResponse
// @Failure      404             {object}  api.ErrorResponse
// @Failure      409             {object}  api.ErrorResponse
// @Failure      422             {object}  api.ErrorResponse
// @Failure      500             {object}  api.ErrorResponse
// @Router       /v1/provider-configurations/{id} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid provider configuration ID"})
	}

	var input model.UpdateProviderConfigurationInput
	if err := c.BodyParser(&input); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidRequestBody.Error(), Title: "Bad Request", Message: "invalid request body"})
	}

	providerConfig, err := h.cmdSvc.Update(c.Context(), id, &input)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ProviderConfigurationOutputFromDomain(providerConfig)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// Delete handles DELETE /v1/provider-configurations/:id
// @Summary      Delete a provider configuration
// @Description  Deletes a provider configuration
// @Tags         Provider Configurations
// @Param        id   path  string  true  "Provider Configuration ID"  Format(uuid)
// @Success      204  "No Content"
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/provider-configurations/{id} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid provider configuration ID"})
	}

	if err := h.cmdSvc.Delete(c.Context(), id); err != nil {
		return h.handleError(c, err)
	}

	return libHTTP.RespondStatus(c, fiber.StatusNoContent)
}

// Disable handles POST /v1/provider-configurations/:id/disable
// @Summary      Disable a provider configuration
// @Description  Transitions a provider configuration from active to disabled status
// @Tags         Provider Configurations
// @Produce      json
// @Param        id   path      string  true  "Provider Configuration ID"  Format(uuid)
// @Success      200  {object}  model.ProviderConfigurationOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/provider-configurations/{id}/disable [post]
func (h *Handler) Disable(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid provider configuration ID"})
	}

	providerConfig, err := h.cmdSvc.Disable(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ProviderConfigurationOutputFromDomain(providerConfig)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// Enable handles POST /v1/provider-configurations/:id/enable
// @Summary      Enable a provider configuration
// @Description  Transitions a provider configuration from disabled to active status
// @Tags         Provider Configurations
// @Produce      json
// @Param        id   path      string  true  "Provider Configuration ID"  Format(uuid)
// @Success      200  {object}  model.ProviderConfigurationOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/provider-configurations/{id}/enable [post]
func (h *Handler) Enable(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid provider configuration ID"})
	}

	providerConfig, err := h.cmdSvc.Enable(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ProviderConfigurationOutputFromDomain(providerConfig)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// TestConnectivity handles POST /v1/provider-configurations/:id/test
// @Summary      Test provider configuration connectivity
// @Description  Tests connectivity, authentication, and end-to-end communication with the provider
// @Tags         Provider Configurations
// @Produce      json
// @Param        id   path      string  true  "Provider Configuration ID"  Format(uuid)
// @Success      200  {object}  model.ProviderConfigTestResultOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/provider-configurations/{id}/test [post]
func (h *Handler) TestConnectivity(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid provider configuration ID"})
	}

	result, err := h.cmdSvc.TestConnectivity(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ProviderConfigTestResultOutputFromDomain(result)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// handleError converts domain errors to appropriate HTTP responses.
func (h *Handler) handleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, constant.ErrProviderConfigNotFound):
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrProviderConfigNotFound.Error(), Title: "Not Found", Message: "provider configuration not found"})

	case errors.Is(err, constant.ErrProviderConfigDuplicateName):
		return libHTTP.Respond(c, fiber.StatusConflict, api.ErrorResponse{Code: constant.ErrProviderConfigDuplicateName.Error(), Title: "Conflict", Message: "provider configuration name already exists"})

	case errors.Is(err, constant.ErrProviderConfigCannotModify):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrProviderConfigCannotModify.Error(), Title: "Unprocessable Entity", Message: "cannot modify provider configuration in current status"})

	case errors.Is(err, constant.ErrProviderNotFoundInCatalog):
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrProviderNotFoundInCatalog.Error(), Title: "Not Found", Message: "provider not found in catalog"})

	case errors.Is(err, constant.ErrProviderConfigInvalidSchema):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrProviderConfigInvalidSchema.Error(), Title: "Unprocessable Entity", Message: err.Error()})

	case errors.Is(err, constant.ErrProviderConfigSSRFBlocked):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrProviderConfigSSRFBlocked.Error(), Title: "Unprocessable Entity", Message: "provider configuration base_url targets a restricted destination"})

	case errors.Is(err, constant.ErrProviderConfigMissingBaseURL):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrProviderConfigMissingBaseURL.Error(), Title: "Unprocessable Entity", Message: "provider configuration config missing required field 'base_url'"})

	case isValidationErrorWithCode(err, constant.ErrProviderConfigCannotModify.Error()):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrProviderConfigCannotModify.Error(), Title: "Unprocessable Entity", Message: "cannot modify provider configuration in current status"})

	case errors.Is(err, constant.ErrConflictStateChanged):
		return libHTTP.Respond(c, fiber.StatusConflict, api.ErrorResponse{Code: constant.ErrConflictStateChanged.Error(), Title: "Conflict", Message: "resource state changed concurrently; retry with latest version"})

	default:
		return libHTTP.Respond(c, fiber.StatusInternalServerError, api.ErrorResponse{Code: constant.ErrInternalServer.Error(), Title: "Internal Server Error", Message: "internal server error"})
	}
}

// isValidationErrorWithCode checks if the error is a ValidationError with the specified code.
func isValidationErrorWithCode(err error, code string) bool {
	var validationErr pkg.ValidationError
	if errors.As(err, &validationErr) {
		return strings.Contains(validationErr.Code, code)
	}

	return false
}
