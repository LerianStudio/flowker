// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package workflow contains the HTTP handler for workflow operations.
package workflow

import (
	"context"
	"errors"

	"github.com/LerianStudio/flowker/api"
	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	libHTTP "github.com/LerianStudio/lib-commons/v5/commons/net/http"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CommandService defines the interface for workflow command operations.
type CommandService interface {
	Create(ctx context.Context, input *model.CreateWorkflowInput) (*model.Workflow, error)
	CreateFromTemplate(ctx context.Context, input *model.CreateWorkflowFromTemplateInput) (*model.Workflow, error)
	Update(ctx context.Context, id uuid.UUID, input *model.UpdateWorkflowInput) (*model.Workflow, error)
	Clone(ctx context.Context, id uuid.UUID, input *model.CloneWorkflowInput) (*model.Workflow, error)
	Activate(ctx context.Context, id uuid.UUID) (*model.Workflow, error)
	Deactivate(ctx context.Context, id uuid.UUID) (*model.Workflow, error)
	MoveToDraft(ctx context.Context, id uuid.UUID) (*model.Workflow, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// QueryService defines the interface for workflow query operations.
type QueryService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error)
	GetByName(ctx context.Context, name string) (*model.Workflow, error)
	List(ctx context.Context, filter query.WorkflowListFilter) (*query.WorkflowListResult, error)
}

// Handler handles HTTP requests for workflow operations.
type Handler struct {
	cmdSvc   CommandService
	querySvc QueryService
}

// ErrWorkflowHandlerNilDependency is returned when a required dependency is nil.
var ErrWorkflowHandlerNilDependency = errors.New("workflow handler: required dependency cannot be nil")

// NewHandler creates a new workflow HTTP handler.
// Returns error if required dependencies are nil.
func NewHandler(cmdSvc CommandService, querySvc QueryService) (*Handler, error) {
	if cmdSvc == nil || querySvc == nil {
		return nil, ErrWorkflowHandlerNilDependency
	}

	return &Handler{
		cmdSvc:   cmdSvc,
		querySvc: querySvc,
	}, nil
}

// RegisterRoutes registers all workflow routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	workflows := router.Group("/workflows")

	workflows.Post("/", h.Create)
	workflows.Post("/from-template", h.CreateFromTemplate)
	workflows.Get("/", h.List)
	workflows.Get("/:id", h.GetByID)
	workflows.Put("/:id", h.Update)
	workflows.Delete("/:id", h.Delete)
	workflows.Post("/:id/clone", h.Clone)
	workflows.Post("/:id/activate", h.Activate)
	workflows.Post("/:id/deactivate", h.Deactivate)
	workflows.Post("/:id/draft", h.MoveToDraft)
}

// Create handles POST /v1/workflows
// @Summary      Create a new workflow
// @Description  Creates a new workflow definition in draft status
// @Tags         Workflows
// @Accept       json
// @Produce      json
// @Param        workflow  body      model.CreateWorkflowInput  true  "Workflow definition"
// @Success      201       {object}  model.WorkflowCreateOutput
// @Failure      400       {object}  api.ErrorResponse
// @Failure      409       {object}  api.ErrorResponse
// @Failure      500       {object}  api.ErrorResponse
// @Router       /v1/workflows [post]
func (h *Handler) Create(c *fiber.Ctx) error {
	var input model.CreateWorkflowInput
	if err := c.BodyParser(&input); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidRequestBody.Error(), Title: "Bad Request", Message: "invalid request body"})
	}

	workflow, err := h.cmdSvc.Create(c.Context(), &input)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.WorkflowCreateOutputFromDomain(workflow)

	return libHTTP.Respond(c, fiber.StatusCreated, output)
}

// CreateFromTemplate handles POST /v1/workflows/from-template
// @Summary      Create workflow from template
// @Description  Creates a new workflow from a registered template with provided parameters
// @Tags         Workflows
// @Accept       json
// @Produce      json
// @Param        input  body      model.CreateWorkflowFromTemplateInput  true  "Template ID and parameters"
// @Success      201    {object}  model.WorkflowCreateOutput
// @Failure      400    {object}  api.ErrorResponse
// @Failure      404    {object}  api.ErrorResponse
// @Failure      409    {object}  api.ErrorResponse
// @Failure      500    {object}  api.ErrorResponse
// @Router       /v1/workflows/from-template [post]
func (h *Handler) CreateFromTemplate(c *fiber.Ctx) error {
	var input model.CreateWorkflowFromTemplateInput
	if err := c.BodyParser(&input); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidRequestBody.Error(), Title: "Bad Request", Message: "invalid request body"})
	}

	workflow, err := h.cmdSvc.CreateFromTemplate(c.Context(), &input)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.WorkflowCreateOutputFromDomain(workflow)

	return libHTTP.Respond(c, fiber.StatusCreated, output)
}

// GetByID handles GET /v1/workflows/:id
// @Summary      Get workflow by ID
// @Description  Retrieves a workflow definition by its ID
// @Tags         Workflows
// @Produce      json
// @Param        id   path      string  true  "Workflow ID"  Format(uuid)
// @Success      200  {object}  model.WorkflowOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/workflows/{id} [get]
func (h *Handler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid workflow ID"})
	}

	workflow, err := h.querySvc.GetByID(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.WorkflowOutputFromDomain(workflow)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// List handles GET /v1/workflows
// @Summary      List workflows
// @Description  Retrieves a paginated list of workflows with optional filtering
// @Tags         Workflows
// @Produce      json
// @Param        status     query     string  false  "Filter by status"  Enums(draft, active, inactive)
// @Param        limit      query     int     false  "Number of items per page"  default(10)  minimum(1)  maximum(100)
// @Param        cursor     query     string  false  "Pagination cursor"
// @Param        sortBy     query     string  false  "Sort field"  Enums(createdAt, updatedAt, name)  default(createdAt)
// @Param        sortOrder  query     string  false  "Sort order"  Enums(ASC, DESC)  default(DESC)
// @Success      200        {object}  model.WorkflowListOutput
// @Failure      400        {object}  api.ErrorResponse
// @Failure      500        {object}  api.ErrorResponse
// @Router       /v1/workflows [get]
func (h *Handler) List(c *fiber.Ctx) error {
	filter := query.WorkflowListFilter{
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
		status := model.WorkflowStatus(statusStr)
		filter.Status = &status
	}

	result, err := h.querySvc.List(c.Context(), filter)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.WorkflowListOutputFromDomain(result.Items, result.NextCursor, result.HasMore)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// Update handles PUT /v1/workflows/:id
// @Summary      Update a workflow
// @Description  Updates an existing workflow definition (only draft workflows can be updated)
// @Tags         Workflows
// @Accept       json
// @Produce      json
// @Param        id        path      string                     true  "Workflow ID"  Format(uuid)
// @Param        workflow  body      model.UpdateWorkflowInput  true  "Updated workflow definition"
// @Success      200       {object}  model.WorkflowOutput
// @Failure      400       {object}  api.ErrorResponse
// @Failure      404       {object}  api.ErrorResponse
// @Failure      422       {object}  api.ErrorResponse
// @Failure      409       {object}  api.ErrorResponse
// @Failure      500       {object}  api.ErrorResponse
// @Router       /v1/workflows/{id} [put]
func (h *Handler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid workflow ID"})
	}

	var input model.UpdateWorkflowInput
	if err := c.BodyParser(&input); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidRequestBody.Error(), Title: "Bad Request", Message: "invalid request body"})
	}

	workflow, err := h.cmdSvc.Update(c.Context(), id, &input)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.WorkflowOutputFromDomain(workflow)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// Delete handles DELETE /v1/workflows/:id
// @Summary      Delete a workflow
// @Description  Deletes a workflow definition (only draft and inactive workflows can be deleted)
// @Tags         Workflows
// @Param        id   path  string  true  "Workflow ID"  Format(uuid)
// @Success      204  "No Content"
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/workflows/{id} [delete]
func (h *Handler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid workflow ID"})
	}

	if err := h.cmdSvc.Delete(c.Context(), id); err != nil {
		return h.handleError(c, err)
	}

	return libHTTP.RespondStatus(c, fiber.StatusNoContent)
}

// Clone handles POST /v1/workflows/:id/clone
// @Summary      Clone a workflow
// @Description  Creates a copy of an existing workflow with a new name
// @Tags         Workflows
// @Accept       json
// @Produce      json
// @Param        id     path      string                    true  "Source workflow ID"  Format(uuid)
// @Param        clone  body      model.CloneWorkflowInput  true  "Clone parameters"
// @Success      201    {object}  model.WorkflowCreateOutput
// @Failure      400    {object}  api.ErrorResponse
// @Failure      404    {object}  api.ErrorResponse
// @Failure      409    {object}  api.ErrorResponse
// @Failure      500    {object}  api.ErrorResponse
// @Router       /v1/workflows/{id}/clone [post]
func (h *Handler) Clone(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid workflow ID"})
	}

	var input model.CloneWorkflowInput
	if err := c.BodyParser(&input); err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidRequestBody.Error(), Title: "Bad Request", Message: "invalid request body"})
	}

	workflow, err := h.cmdSvc.Clone(c.Context(), id, &input)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.WorkflowCreateOutputFromDomain(workflow)

	return libHTTP.Respond(c, fiber.StatusCreated, output)
}

// Activate handles POST /v1/workflows/:id/activate
// @Summary      Activate a workflow
// @Description  Transitions a workflow from draft to active status
// @Tags         Workflows
// @Produce      json
// @Param        id   path      string  true  "Workflow ID"  Format(uuid)
// @Success      200  {object}  model.WorkflowOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/workflows/{id}/activate [post]
func (h *Handler) Activate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid workflow ID"})
	}

	workflow, err := h.cmdSvc.Activate(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.WorkflowOutputFromDomain(workflow)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// Deactivate handles POST /v1/workflows/:id/deactivate
// @Summary      Deactivate a workflow
// @Description  Transitions a workflow from active to inactive status
// @Tags         Workflows
// @Produce      json
// @Param        id   path      string  true  "Workflow ID"  Format(uuid)
// @Success      200  {object}  model.WorkflowOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/workflows/{id}/deactivate [post]
func (h *Handler) Deactivate(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid workflow ID"})
	}

	workflow, err := h.cmdSvc.Deactivate(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.WorkflowOutputFromDomain(workflow)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// MoveToDraft handles POST /v1/workflows/:id/draft
// @Summary      Move workflow to draft
// @Description  Transitions a workflow from inactive to draft status for editing
// @Tags         Workflows
// @Produce      json
// @Param        id   path      string  true  "Workflow ID"  Format(uuid)
// @Success      200  {object}  model.WorkflowOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      422  {object}  api.ErrorResponse
// @Failure      409  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/workflows/{id}/draft [post]
func (h *Handler) MoveToDraft(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: constant.ErrInvalidID.Error(), Title: "Bad Request", Message: "invalid workflow ID"})
	}

	workflow, err := h.cmdSvc.MoveToDraft(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.WorkflowOutputFromDomain(workflow)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// handleError converts domain errors to appropriate HTTP responses.
func (h *Handler) handleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, constant.ErrWorkflowNotFound):
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrWorkflowNotFound.Error(), Title: "Not Found", Message: "workflow not found"})

	case errors.Is(err, constant.ErrWorkflowDuplicateName):
		return libHTTP.Respond(c, fiber.StatusConflict, api.ErrorResponse{Code: constant.ErrWorkflowDuplicateName.Error(), Title: "Conflict", Message: "workflow name already exists"})

	case errors.Is(err, constant.ErrWorkflowInvalidStatus):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrWorkflowInvalidStatus.Error(), Title: "Unprocessable Entity", Message: "invalid workflow status transition"})

	case errors.Is(err, constant.ErrWorkflowCannotModify):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrWorkflowCannotModify.Error(), Title: "Unprocessable Entity", Message: "cannot modify non-draft workflow"})

	case errors.Is(err, constant.ErrWorkflowExecutorNotFound):
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrWorkflowExecutorNotFound.Error(), Title: "Not Found", Message: "referenced executor not found"})

	case errors.Is(err, constant.ErrTemplateNotFound):
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: constant.ErrTemplateNotFound.Error(), Title: "Not Found", Message: "template not found"})

	case errors.Is(err, constant.ErrWorkflowInvalidCondition):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrWorkflowInvalidCondition.Error(), Title: "Unprocessable Entity", Message: "invalid conditional expression"})

	case errors.Is(err, constant.ErrConflictStateChanged):
		return libHTTP.Respond(c, fiber.StatusConflict, api.ErrorResponse{Code: constant.ErrConflictStateChanged.Error(), Title: "Conflict", Message: "resource state changed concurrently; retry with latest version"})

	case errors.Is(err, model.ErrWorkflowNodesRequired):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrWorkflowNodesRequired.Error(), Title: "Unprocessable Entity", Message: "workflow must have at least one node to activate"})

	case errors.Is(err, model.ErrWorkflowNoTrigger):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{Code: constant.ErrWorkflowNoTrigger.Error(), Title: "Unprocessable Entity", Message: "workflow must have a trigger node to activate"})

	default:
		// Check for entity not found errors — only map to 404 for resource lookups
		// (template, workflow), not for validation errors (unknown executor in workflow definition)
		var entityNotFoundErr pkg.EntityNotFoundError
		if errors.As(err, &entityNotFoundErr) {
			if entityNotFoundErr.Code == constant.ErrTemplateNotFound.Error() ||
				entityNotFoundErr.Code == constant.ErrWorkflowNotFound.Error() {
				return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{Code: entityNotFoundErr.Code, Title: "Not Found", Message: entityNotFoundErr.Message})
			}

			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: entityNotFoundErr.Code, Title: "Bad Request", Message: entityNotFoundErr.Message})
		}

		// Check for validation errors (including transformation validation)
		var validationErr pkg.ValidationError
		if errors.As(err, &validationErr) {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{Code: validationErr.Code, Title: "Bad Request", Message: validationErr.Message})
		}

		return libHTTP.Respond(c, fiber.StatusInternalServerError, api.ErrorResponse{Code: constant.ErrInternalServer.Error(), Title: "Internal Server Error", Message: "internal server error"})
	}
}
