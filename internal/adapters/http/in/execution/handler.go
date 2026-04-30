// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package execution contains the HTTP handler for workflow execution operations.
package execution

import (
	"context"
	"errors"
	"strings"

	"github.com/LerianStudio/flowker/api"
	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	libHTTP "github.com/LerianStudio/lib-commons/v5/commons/net/http"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CommandService defines the interface for execution command operations.
type CommandService interface {
	Execute(ctx context.Context, workflowID uuid.UUID, input *model.ExecuteWorkflowInput, idempotencyKey *string) (*model.WorkflowExecution, error)
}

// QueryService defines the interface for execution query operations.
type QueryService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error)
	GetResults(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error)
	List(ctx context.Context, filter query.ExecutionListFilter) (*query.ExecutionListResult, error)
}

// Handler handles HTTP requests for execution operations.
type Handler struct {
	cmdSvc   CommandService
	querySvc QueryService
}

// ErrExecutionHandlerNilDependency is returned when a required dependency is nil.
var ErrExecutionHandlerNilDependency = errors.New("execution handler: required dependency cannot be nil")

// NewHandler creates a new execution HTTP handler.
func NewHandler(cmdSvc CommandService, querySvc QueryService) (*Handler, error) {
	if cmdSvc == nil || querySvc == nil {
		return nil, ErrExecutionHandlerNilDependency
	}

	return &Handler{
		cmdSvc:   cmdSvc,
		querySvc: querySvc,
	}, nil
}

// RegisterRoutes registers execution routes on the given router.
// Execution routes are split:
//   - POST /workflows/:workflowId/executions -> under workflows group
//   - GET /executions/:id -> under executions group
//   - GET /executions/:id/results -> under executions group
func (h *Handler) RegisterRoutes(router fiber.Router) {
	// Routes under /workflows/:workflowId/executions
	router.Post("/workflows/:workflowId/executions", h.Execute)

	// Routes under /executions
	executions := router.Group("/executions")
	executions.Get("/", h.List)
	executions.Get("/:id", h.GetStatus)
	executions.Get("/:id/results", h.GetResults)
}

// Execute handles POST /v1/workflows/:workflowId/executions
// @Summary      Execute a workflow
// @Description  Starts a new workflow execution. Returns immediately with pending status.
// @Tags         Executions
// @Accept       json
// @Produce      json
// @Param        workflowId      path    string                        true  "Workflow ID"  Format(uuid)
// @Param        Idempotency-Key header  string                        true  "Idempotency key"
// @Param        execution       body    model.ExecuteWorkflowInput    true  "Execution input"
// @Success      201             {object} model.ExecutionCreateOutput
// @Success      200             {object} model.ExecutionCreateOutput  "Idempotent request"
// @Failure      400             {object} api.ErrorResponse
// @Failure      404             {object} api.ErrorResponse
// @Failure      413             {object} api.ErrorResponse  "Payload too large (max 1 MB)"
// @Failure      422             {object} api.ErrorResponse
// @Failure      409             {object} api.ErrorResponse
// @Failure      500             {object} api.ErrorResponse
// @Router       /v1/workflows/{workflowId}/executions [post]
func (h *Handler) Execute(c *fiber.Ctx) error {
	// Validate input payload size (max 1 MB)
	const maxBodySize = 1 << 20 // 1 MB
	if len(c.Body()) > maxBodySize {
		return libHTTP.Respond(c, fiber.StatusRequestEntityTooLarge, api.ErrorResponse{
			Code: constant.ErrExecutionInputTooLarge.Error(), Title: "Payload Too Large", Message: "request body must not exceed 1 MB",
		})
	}

	workflowID, err := uuid.Parse(c.Params("workflowId"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid workflow ID"})
	}

	var input model.ExecuteWorkflowInput
	if err := c.BodyParser(&input); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidRequestBody.Error(), Title: "Bad Request", Message: "invalid request body"})
	}

	// Extract idempotency key from header (required)
	key := strings.TrimSpace(c.Get("Idempotency-Key"))
	if key == "" {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrMissingIdempotencyKey.Error(), Title: "Bad Request", Message: "Idempotency-Key header is required"})
	}

	idempotencyKey := &key

	execution, err := h.cmdSvc.Execute(c.Context(), workflowID, &input, idempotencyKey)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ExecutionCreateOutputFromDomain(execution)

	// If the execution is in a terminal state (completed/failed), it was an idempotent
	// return of an existing execution — use 200 OK. New executions return 201 Created.
	if execution.IsTerminal() {
		return libHTTP.Respond(c, fiber.StatusOK, output)
	}

	return libHTTP.Respond(c, fiber.StatusCreated, output)
}

// GetStatus handles GET /v1/executions/:id
// @Summary      Get execution status
// @Description  Retrieves the current status of a workflow execution
// @Tags         Executions
// @Produce      json
// @Param        id   path      string  true  "Execution ID"  Format(uuid)
// @Success      200  {object}  model.ExecutionStatusOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/executions/{id} [get]
func (h *Handler) GetStatus(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid execution ID"})
	}

	execution, err := h.querySvc.GetByID(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ExecutionStatusOutputFromDomain(execution)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// GetResults handles GET /v1/executions/:id/results
// @Summary      Get execution results
// @Description  Retrieves the results of a completed workflow execution
// @Tags         Executions
// @Produce      json
// @Param        id   path      string  true  "Execution ID"  Format(uuid)
// @Success      200  {object}  model.ExecutionResultsOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse  "Execution still in progress"
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/executions/{id}/results [get]
func (h *Handler) GetResults(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid execution ID"})
	}

	execution, err := h.querySvc.GetResults(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ExecutionResultsOutputFromDomain(execution)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// List handles GET /v1/executions
// @Summary      List executions
// @Description  Retrieves a paginated list of workflow executions with optional filters
// @Tags         Executions
// @Accept       json
// @Produce      json
// @Param        workflowId  query     string  false  "Filter by workflow ID"  Format(uuid)
// @Param        status      query     string  false  "Filter by status"       Enums(pending,running,completed,failed)
// @Param        limit       query     int     false  "Page size (1-100)"      default(10)  minimum(1)  maximum(100)
// @Param        cursor      query     string  false  "Pagination cursor"
// @Param        sortBy      query     string  false  "Sort field"             default(startedAt) Enums(startedAt,completedAt)
// @Param        sortOrder   query     string  false  "Sort direction"         default(DESC) Enums(ASC,DESC)
// @Success      200         {object}  model.ExecutionListOutput
// @Failure      400         {object}  api.ErrorResponse
// @Failure      500         {object}  api.ErrorResponse
// @Router       /v1/executions [get]
func (h *Handler) List(c *fiber.Ctx) error {
	// 1. Parse raw query parameters (no defaults applied yet)
	filter := query.ExecutionListFilter{
		Cursor: c.Query("cursor"),
	}

	// Only set sortBy/sortOrder if explicitly provided (cursor contains its own sort config)
	if sortBy := c.Query("sortBy"); sortBy != "" {
		filter.SortBy = sortBy
	}

	if sortOrder := c.Query("sortOrder"); sortOrder != "" {
		filter.SortOrder = sortOrder
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		filter.Limit = c.QueryInt("limit", constant.DefaultPaginationLimit)
	} else {
		filter.Limit = constant.DefaultPaginationLimit
	}

	if workflowIDStr := c.Query("workflowId"); workflowIDStr != "" {
		wfID, err := uuid.Parse(workflowIDStr)
		if err != nil {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid workflowId filter"})
		}

		filter.WorkflowID = &wfID
	}

	if statusStr := c.Query("status"); statusStr != "" {
		status := model.ExecutionStatus(statusStr)
		filter.Status = &status
	}

	// 2. Validate before applying defaults (fail-fast)
	if err := filter.Validate(); err != nil {
		var ve pkg.ValidationError
		if errors.As(err, &ve) {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: ve.Code, Title: ve.Title, Message: ve.Message})
		}

		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrBadRequest.Error(), Title: "Bad Request", Message: err.Error()})
	}

	// 3. Apply defaults after validation
	filter.ApplyDefaults()

	result, err := h.querySvc.List(c.Context(), filter)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ExecutionListOutputFromDomain(result.Items, result.NextCursor, result.HasMore)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// handleError converts domain errors to appropriate HTTP responses.
func (h *Handler) handleError(c *fiber.Ctx, err error) error {
	var ve pkg.ValidationError
	if errors.As(err, &ve) {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: ve.Code, Title: ve.Title, Message: ve.Message})
	}

	switch {
	case errors.Is(err, constant.ErrExecutionNotFound):
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrExecutionNotFound.Error(), Title: "Not Found", Message: "execution not found"})

	case errors.Is(err, constant.ErrWorkflowNotFound):
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrWorkflowNotFound.Error(), Title: "Not Found", Message: "workflow not found"})

	case errors.Is(err, constant.ErrExecutionNotActive):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrExecutionNotActive.Error(), Title: "Unprocessable Entity", Message: "workflow is not active"})

	case errors.Is(err, constant.ErrExecutionInProgress):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrExecutionInProgress.Error(), Title: "Unprocessable Entity", Message: "execution is still in progress"})

	case errors.Is(err, constant.ErrExecutionDuplicate):
		return libHTTP.Respond(c, fiber.StatusConflict, api.ErrorResponse{Code: constant.ErrExecutionDuplicate.Error(), Title: "Conflict", Message: "duplicate idempotency key"})

	case errors.Is(err, constant.ErrConflictStateChanged):
		return libHTTP.Respond(c, fiber.StatusConflict, api.ErrorResponse{Code: constant.ErrConflictStateChanged.Error(), Title: "Conflict", Message: "resource state changed concurrently; retry with latest version"})

	case errors.Is(err, constant.ErrExecutionTimeout):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrExecutionTimeout.Error(), Title: "Unprocessable Entity", Message: "execution timeout exceeded"})

	default:
		return libHTTP.Respond(c, fiber.StatusInternalServerError, api.ErrorResponse{Code: constant.ErrInternalServer.Error(), Title: "Internal Server Error", Message: "internal server error"})
	}
}
