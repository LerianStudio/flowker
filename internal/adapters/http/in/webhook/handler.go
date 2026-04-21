// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package webhook contains the HTTP handler for dynamic webhook endpoints.
// Webhook routes are resolved at runtime from an in-memory registry that
// maps HTTP method+path pairs to active workflow triggers.
package webhook

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"

	"github.com/LerianStudio/flowker/api"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/webhook"
	libHTTP "github.com/LerianStudio/lib-commons/v4/commons/net/http"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// ExecuteService defines the interface needed to trigger workflow execution.
// This matches the ExecuteWorkflowCommand's Execute signature.
type ExecuteService interface {
	Execute(ctx context.Context, workflowID uuid.UUID, input *model.ExecuteWorkflowInput, idempotencyKey *string) (*model.WorkflowExecution, error)
}

// Handler handles incoming webhook HTTP requests by resolving them against
// the webhook registry and triggering the corresponding workflow execution.
type Handler struct {
	registry   *webhook.Registry
	executeSvc ExecuteService
}

// NewHandler creates a new webhook handler.
// Returns an error if required dependencies are nil.
func NewHandler(registry *webhook.Registry, executeSvc ExecuteService) (*Handler, error) {
	if registry == nil {
		return nil, ErrWebhookHandlerNilRegistry
	}

	if executeSvc == nil {
		return nil, ErrWebhookHandlerNilExecuteService
	}

	return &Handler{
		registry:   registry,
		executeSvc: executeSvc,
	}, nil
}

// RegisterRoutes registers the catch-all webhook route on the given router group.
// The router group should be mounted at /v1/webhooks.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.All("/*", h.HandleWebhook)
}

// maxWebhookBodySize is the maximum allowed request body size for webhook requests (1MB).
const maxWebhookBodySize = 1 * 1024 * 1024

// HandleWebhook is the catch-all handler for all webhook requests.
// It resolves the request method+path against the registry and triggers
// the corresponding workflow execution.
//
// Routing note: at runtime the handler is mounted as a wildcard
// (router.All("/*", ...)), so it accepts every HTTP method and paths
// with any number of slash-separated segments. Swagger 2.0 cannot
// express either of those: the documented contract below advertises
// POST and a single-segment {path} parameter, which is the canonical
// shape used by external webhook providers. GET/PUT/DELETE/PATCH and
// nested paths also reach HandleWebhook when a matching route is
// registered, but are not separately documented here. When migrating
// the spec to OpenAPI 3.0 this annotation can be extended accordingly.
//
// @Summary      Trigger a webhook
// @Description  Receives a webhook callback and triggers the associated workflow execution. Runtime also accepts GET/PUT/DELETE/PATCH and nested-segment paths that Swagger 2.0 cannot express.
// @Tags         Webhooks
// @Accept       json
// @Produce      json
// @Param        path               path    string  true   "Webhook path registered by a workflow (Swagger 2.0 single segment; runtime accepts nested paths)"
// @Param        Idempotency-Key    header  string  false  "Optional idempotency key"
// @Param        X-API-Key          header  string  false  "Infrastructure API key enforced by AuthGuard when API-key auth is enabled"
// @Param        X-Webhook-Token    header  string  false  "Per-webhook verification token (validated by the handler after AuthGuard)"
// @Param        body               body    object  false  "Webhook payload"
// @Success      200  {object}  model.ExecutionCreateOutput  "Workflow execution already terminal (sync response)"
// @Success      201  {object}  model.ExecutionCreateOutput  "Workflow execution created"
// @Failure      400  {object}  api.ErrorResponse  "Invalid request body"
// @Failure      401  {object}  api.ErrorResponse  "Unauthorized (AuthGuard API-key failure or invalid/missing X-Webhook-Token)"
// @Failure      404  {object}  api.ErrorResponse  "No webhook registered for this path/method, or workflow not found"
// @Failure      409  {object}  api.ErrorResponse  "Duplicate idempotency key"
// @Failure      413  {object}  api.ErrorResponse  "Webhook payload exceeds 1MB limit"
// @Failure      422  {object}  api.ErrorResponse  "Workflow is not active"
// @Failure      500  {object}  api.ErrorResponse  "Internal server error"
// @Router       /v1/webhooks/{path} [post]
func (h *Handler) HandleWebhook(c *fiber.Ctx) error {
	// Enforce body size limit
	if len(c.Body()) > maxWebhookBodySize {
		return libHTTP.Respond(c, fiber.StatusRequestEntityTooLarge, api.ErrorResponse{
			Code:    constant.ErrWebhookPayloadTooLarge.Error(),
			Title:   "Payload Too Large",
			Message: "webhook request body exceeds 1MB limit",
		})
	}

	// Extract the path from the catch-all parameter
	webhookPath := c.Params("*")
	method := c.Method()

	// Resolve the route from the registry
	route, ok := h.registry.Resolve(method, webhookPath)
	if !ok {
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{
			Code:    constant.ErrWebhookRouteNotFound.Error(),
			Title:   "Not Found",
			Message: "no webhook registered for this path and method",
		})
	}

	// Verify token if configured
	if route.VerifyToken != "" {
		token := c.Get("X-Webhook-Token")
		if subtle.ConstantTimeCompare([]byte(token), []byte(route.VerifyToken)) != 1 {
			return libHTTP.Respond(c, fiber.StatusUnauthorized, api.ErrorResponse{
				Code:    constant.ErrWebhookTokenInvalid.Error(),
				Title:   "Unauthorized",
				Message: "invalid or missing webhook verification token",
			})
		}
	}

	// Parse request body as input data
	var inputData map[string]any
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&inputData); err != nil {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
				Code:    constant.ErrInvalidRequestBody.Error(),
				Title:   "Bad Request",
				Message: "invalid request body",
			})
		}
	}

	if inputData == nil {
		inputData = make(map[string]any)
	}

	// Add webhook metadata to input
	inputData["_webhook"] = map[string]any{
		"method":    method,
		"path":      webhookPath,
		"headers":   extractHeaders(c),
		"query":     extractQueryParams(c),
		"remote_ip": c.IP(),
	}

	input := &model.ExecuteWorkflowInput{
		InputData: inputData,
	}

	// Extract optional idempotency key
	var idempotencyKey *string

	if key := strings.TrimSpace(c.Get("Idempotency-Key")); key != "" {
		idempotencyKey = &key
	}

	execution, err := h.executeSvc.Execute(c.Context(), route.WorkflowID, input, idempotencyKey)
	if err != nil {
		return h.handleError(c, err)
	}

	output := model.ExecutionCreateOutputFromDomain(execution)

	// Set debugging headers
	c.Set("X-Webhook-Workflow-ID", route.WorkflowID.String())
	c.Set("X-Webhook-Execution-ID", execution.ExecutionID().String())

	if execution.IsTerminal() {
		return libHTTP.Respond(c, fiber.StatusOK, output)
	}

	return libHTTP.Respond(c, fiber.StatusCreated, output)
}

// extractHeaders returns a map of request headers.
func extractHeaders(c *fiber.Ctx) map[string]string {
	headers := make(map[string]string)

	for key, value := range c.Request().Header.All() {
		k := string(key)
		// Skip sensitive headers
		lk := strings.ToLower(k)
		if lk == "authorization" || lk == "x-api-key" || lk == "x-webhook-token" {
			continue
		}

		headers[k] = string(value)
	}

	return headers
}

// extractQueryParams returns a map of query string parameters.
func extractQueryParams(c *fiber.Ctx) map[string]string {
	params := make(map[string]string)

	for key, value := range c.Request().URI().QueryArgs().All() {
		params[string(key)] = string(value)
	}

	return params
}

// handleError converts domain errors to appropriate HTTP responses.
func (h *Handler) handleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, constant.ErrWorkflowNotFound):
		return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{
			Code: constant.ErrWorkflowNotFound.Error(), Title: "Not Found", Message: "workflow not found",
		})

	case errors.Is(err, constant.ErrExecutionNotActive):
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, api.ErrorResponse{
			Code: constant.ErrExecutionNotActive.Error(), Title: "Unprocessable Entity", Message: "workflow is not active",
		})

	case errors.Is(err, constant.ErrExecutionDuplicate):
		return libHTTP.Respond(c, fiber.StatusConflict, api.ErrorResponse{
			Code: constant.ErrExecutionDuplicate.Error(), Title: "Conflict", Message: "duplicate idempotency key",
		})

	default:
		return libHTTP.Respond(c, fiber.StatusInternalServerError, api.ErrorResponse{
			Code: constant.ErrInternalServer.Error(), Title: "Internal Server Error", Message: "internal server error",
		})
	}
}
