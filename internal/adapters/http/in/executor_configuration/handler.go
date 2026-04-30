// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package executorconfiguration contains the HTTP handler for executor configuration operations.
package executorconfiguration

import (
	"context"
	"errors"
	"strings"

	"github.com/LerianStudio/flowker/api"
	"github.com/LerianStudio/flowker/internal/services/query"
	pkg "github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	libHTTP "github.com/LerianStudio/lib-commons/v5/commons/net/http"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CommandService defines the interface for executor configuration command operations.
type CommandService interface {
	Update(ctx context.Context, id uuid.UUID, input *model.UpdateExecutorConfigurationInput) (*model.ExecutorConfiguration, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// QueryService defines the interface for executor configuration query operations.
type QueryService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error)
	GetByName(ctx context.Context, name string) (*model.ExecutorConfiguration, error)
	List(ctx context.Context, filter query.ExecutorConfigListFilter) (*query.ExecutorConfigListResult, error)
}

// Handler handles HTTP requests for executor configuration operations.
type Handler struct {
	cmdSvc   CommandService
	querySvc QueryService
}

// ErrExecutorConfigHandlerNilDependency is returned when a required dependency is nil.
var ErrExecutorConfigHandlerNilDependency = errors.New("executor configuration handler: required dependency cannot be nil")

// NewHandler creates a new executor configuration HTTP handler.
// Returns error if required dependencies are nil.
func NewHandler(cmdSvc CommandService, querySvc QueryService) (*Handler, error) {
	if cmdSvc == nil || querySvc == nil {
		return nil, ErrExecutorConfigHandlerNilDependency
	}

	return &Handler{
		cmdSvc:   cmdSvc,
		querySvc: querySvc,
	}, nil
}

// RegisterRoutes registers all executor configuration routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	executors := router.Group("/executors")

	executors.Get("/", h.List)
	executors.Get("/:id", h.GetByID)
	executors.Patch("/:id", h.Update)
	executors.Delete("/:id", h.Delete)
}

// GetByID handles GET /v1/executors/:id
// @Summary      Get executor configuration by ID
// @Description  Retrieves an executor configuration by its ID
// @Tags         Executors
// @Produce      json
// @Param        id   path      string  true  "Executor Configuration ID"  Format(uuid)
// @Success      200  {object}  model.ExecutorConfigurationOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/executors/{id} [get]
func (h *Handler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid executor configuration ID"})
	}

	executorConfig, err := h.querySvc.GetByID(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ExecutorConfigurationOutputFromDomain(executorConfig)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// List handles GET /v1/executors
// @Summary      List executor configurations
// @Description  Retrieves a paginated list of executor configurations with optional filtering
// @Tags         Executors
// @Produce      json
// @Param        status     query     string  false  "Filter by status"  Enums(unconfigured, configured, tested, active, disabled)
// @Param        limit      query     int     false  "Number of items per page"  default(10)  minimum(1)  maximum(100)
// @Param        cursor     query     string  false  "Pagination cursor"
// @Param        sortBy     query     string  false  "Sort field"  Enums(createdAt, updatedAt, name)  default(createdAt)
// @Param        sortOrder  query     string  false  "Sort order"  Enums(ASC, DESC)  default(DESC)
// @Success      200        {object}  model.ExecutorConfigurationListOutput
// @Failure      400        {object}  api.ErrorResponse
// @Failure      500        {object}  api.ErrorResponse
// @Router       /v1/executors [get]
func (h *Handler) List(c *fiber.Ctx) error {
	filter := query.ExecutorConfigListFilter{
		Cursor:    c.Query("cursor"),
		SortBy:    c.Query("sortBy", "createdAt"),
		SortOrder: c.Query("sortOrder", "DESC"),
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		limit := c.QueryInt("limit", 10)
		filter.Limit = limit
	} else {
		filter.Limit = 10
	}

	// Parse status filter
	if statusStr := c.Query("status"); statusStr != "" {
		status := model.ExecutorConfigurationStatus(statusStr)
		filter.Status = &status
	}

	result, err := h.querySvc.List(c.Context(), filter)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ExecutorConfigurationListOutputFromDomain(result.Items, result.NextCursor, result.HasMore)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// Update handles PATCH /v1/executors/:id
// @Summary      Update an executor configuration
// @Description  Updates an existing executor configuration (only unconfigured or configured executor configurations can be updated)
// @Tags         Executors
// @Accept       json
// @Produce      json
// @Param        id        path      string                                  true  "Executor Configuration ID"  Format(uuid)
// @Param        executorConfig  body      model.UpdateExecutorConfigurationInput  true  "Updated executor configuration"
// @Success      200       {object}  model.ExecutorConfigurationOutput
// @Failure      400       {object}  api.ErrorResponse
// @Failure      404       {object}  api.ErrorResponse
// @Failure      422       {object}  api.ErrorResponse
// @Failure      409       {object}  api.ErrorResponse
// @Failure      500       {object}  api.ErrorResponse
// @Router       /v1/executors/{id} [patch]
func (h *Handler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid executor configuration ID"})
	}

	var input model.UpdateExecutorConfigurationInput
	if err := c.BodyParser(&input); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidRequestBody.Error(), Title: "Bad Request", Message: "invalid request body"})
	}

	executorConfig, err := h.cmdSvc.Update(c.Context(), id, &input)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ExecutorConfigurationOutputFromDomain(executorConfig)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// Delete handles DELETE /v1/executors/:id
// @Summary      Delete an executor configuration
// @Description  Deletes an executor configuration (only unconfigured, configured, or disabled executor configurations can be deleted)
// @Tags         Executors
// @Param        id   path  string  true  "Executor Configuration ID"  Format(uuid)
// @Success      204  "No Content"
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/executors/{id} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid executor configuration ID"})
	}

	if err := h.cmdSvc.Delete(c.Context(), id); err != nil {
		return h.handleError(c, err)
	}

	return libHTTP.RespondStatus(c, fiber.StatusNoContent)
}

// handleError converts domain errors to appropriate HTTP responses.
func (h *Handler) handleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, constant.ErrExecutorConfigNotFound):
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrExecutorConfigNotFound.Error(), Title: "Not Found", Message: "executor configuration not found"})

	case errors.Is(err, constant.ErrExecutorConfigDuplicateName):
		return libHTTP.Respond(c, fiber.StatusConflict, api.ErrorResponse{Code: constant.ErrExecutorConfigDuplicateName.Error(), Title: "Conflict", Message: "executor configuration name already exists"})

	case errors.Is(err, constant.ErrExecutorConfigCannotModify):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrExecutorConfigCannotModify.Error(), Title: "Unprocessable Entity", Message: "cannot modify executor configuration in current status"})

	case isValidationErrorWithCode(err, constant.ErrExecutorConfigCannotModify.Error()):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrExecutorConfigCannotModify.Error(), Title: "Unprocessable Entity", Message: "cannot modify executor configuration in current status"})

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
