// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package audit contains the HTTP handler for audit trail operations.
package audit

import (
	"context"
	"errors"
	"strconv"
	"time"

	api "github.com/LerianStudio/flowker/api"
	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	libHTTP "github.com/LerianStudio/lib-commons/v5/commons/net/http"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// QueryService defines the interface for audit query operations.
type QueryService interface {
	SearchLogs(ctx context.Context, filter query.AuditListFilter) ([]*model.AuditEntry, string, bool, error)
	GetByID(ctx context.Context, eventID uuid.UUID) (*model.AuditEntry, error)
	VerifyHashChain(ctx context.Context, eventID uuid.UUID) (*model.HashChainVerificationOutput, error)
}

// Handler handles HTTP requests for audit trail operations.
type Handler struct {
	querySvc QueryService
}

// ErrAuditHandlerNilDependency is returned when a required dependency is nil.
var ErrAuditHandlerNilDependency = errors.New("audit handler: required dependency cannot be nil")

// NewHandler creates a new audit HTTP handler.
// Returns error if required dependencies are nil.
func NewHandler(querySvc QueryService) (*Handler, error) {
	if querySvc == nil {
		return nil, ErrAuditHandlerNilDependency
	}

	return &Handler{
		querySvc: querySvc,
	}, nil
}

// RegisterRoutes registers all audit routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	auditLogs := router.Group("/audit-events")

	auditLogs.Get("/", h.SearchLogs)
	auditLogs.Get("/:id", h.GetByID)
	auditLogs.Get("/:id/verify", h.VerifyHashChain)
}

// SearchLogs handles GET /v1/audit-events
// @Summary      Search audit logs
// @Description  Retrieves a paginated list of audit log entries with optional filtering
// @Tags         Audit
// @Produce      json
// @Param        eventType     query     string  false  "Filter by event type"
// @Param        action        query     string  false  "Filter by action"
// @Param        result        query     string  false  "Filter by result"  Enums(SUCCESS, FAILED)
// @Param        resourceType  query     string  false  "Filter by resource type"  Enums(workflow, execution, provider_config)
// @Param        resourceId    query     string  false  "Filter by resource ID"  Format(uuid)
// @Param        dateFrom      query     string  false  "Start date filter (RFC3339)"  Format(date-time)
// @Param        dateTo        query     string  false  "End date filter (RFC3339)"  Format(date-time)
// @Param        limit         query     int     false  "Number of items per page"  default(20)  minimum(1)  maximum(100)
// @Param        cursor        query     string  false  "Pagination cursor"
// @Param        sortOrder     query     string  false  "Sort order"  Enums(ASC, DESC)  default(DESC)
// @Success      200           {object}  model.AuditEntryListOutput
// @Failure      400           {object}  api.ErrorResponse
// @Failure      500           {object}  api.ErrorResponse
// @Router       /v1/audit-events [get]
func (h *Handler) SearchLogs(c *fiber.Ctx) error {
	filter := query.AuditListFilter{
		Cursor:    c.Query("cursor"),
		SortOrder: c.Query("sortOrder", "DESC"),
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
				Code:    constant.ErrInvalidQueryParameter.Error(),
				Title:   "Bad Request",
				Message: "invalid limit parameter, expected integer",
			})
		}

		filter.Limit = limit
	} else {
		filter.Limit = 20
	}

	// Parse optional string filters
	if eventType := c.Query("eventType"); eventType != "" {
		filter.EventType = &eventType
	}

	if action := c.Query("action"); action != "" {
		filter.Action = &action
	}

	if result := c.Query("result"); result != "" {
		filter.Result = &result
	}

	if resourceType := c.Query("resourceType"); resourceType != "" {
		filter.ResourceType = &resourceType
	}

	if resourceIDStr := c.Query("resourceId"); resourceIDStr != "" {
		resourceID, err := uuid.Parse(resourceIDStr)
		if err != nil {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
				Code:    constant.ErrInvalidID.Error(),
				Title:   "Bad Request",
				Message: "invalid resourceId format, expected UUID",
			})
		}

		filter.ResourceID = &resourceID
	}

	if dateFromStr := c.Query("dateFrom"); dateFromStr != "" {
		t, err := time.Parse(time.RFC3339, dateFromStr)
		if err != nil {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
				Code:    constant.ErrInvalidDateFormat.Error(),
				Title:   "Bad Request",
				Message: "invalid dateFrom format, expected RFC3339",
			})
		}

		filter.DateFrom = &t
	}

	if dateToStr := c.Query("dateTo"); dateToStr != "" {
		t, err := time.Parse(time.RFC3339, dateToStr)
		if err != nil {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
				Code:    constant.ErrInvalidDateFormat.Error(),
				Title:   "Bad Request",
				Message: "invalid dateTo format, expected RFC3339",
			})
		}

		filter.DateTo = &t
	}

	if filter.DateFrom != nil && filter.DateTo != nil && filter.DateFrom.After(*filter.DateTo) {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
			Code:    constant.ErrInvalidDateRange.Error(),
			Title:   "Bad Request",
			Message: "dateFrom must be before dateTo",
		})
	}

	entries, nextCursor, hasMore, err := h.querySvc.SearchLogs(c.Context(), filter)
	if err != nil {
		logger, _, _, _ := libCommons.NewTrackingFromContext(c.Context())
		logger.Log(c.Context(), libLog.LevelError, "Failed to search audit logs", libLog.Any("error.message", err.Error()))

		return libHTTP.Respond(c, fiber.StatusInternalServerError, api.ErrorResponse{
			Code:    constant.ErrInternalServer.Error(),
			Title:   "Internal Server Error",
			Message: "internal server error",
		})
	}

	output := model.AuditEntryListOutputFromDomain(entries, nextCursor, hasMore)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// GetByID handles GET /v1/audit-events/:id
// @Summary      Get audit entry by ID
// @Description  Retrieves a single audit log entry by its event ID
// @Tags         Audit
// @Produce      json
// @Param        id   path      string  true  "Audit event ID"  Format(uuid)
// @Success      200  {object}  model.AuditEntryOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/audit-events/{id} [get]
func (h *Handler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
			Code:    constant.ErrInvalidID.Error(),
			Title:   "Bad Request",
			Message: "invalid audit event ID",
		})
	}

	entry, err := h.querySvc.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, constant.ErrAuditEntryNotFound) {
			return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{
				Code:    constant.ErrAuditEntryNotFound.Error(),
				Title:   "Not Found",
				Message: "audit entry not found",
			})
		}

		logger, _, _, _ := libCommons.NewTrackingFromContext(c.Context())
		logger.Log(c.Context(), libLog.LevelError, "Failed to get audit entry", libLog.Any("error.message", err.Error()))

		return libHTTP.Respond(c, fiber.StatusInternalServerError, api.ErrorResponse{
			Code:    constant.ErrInternalServer.Error(),
			Title:   "Internal Server Error",
			Message: "internal server error",
		})
	}

	output := model.AuditEntryOutputFromDomain(entry)

	return libHTTP.Respond(c, fiber.StatusOK, output)
}

// VerifyHashChain handles GET /v1/audit-events/:id/verify
// @Summary      Verify audit hash chain
// @Description  Verifies the hash chain integrity up to the specified audit entry
// @Tags         Audit
// @Produce      json
// @Param        id   path      string  true  "Audit event ID"  Format(uuid)
// @Success      200  {object}  model.HashChainVerificationOutput
// @Failure      400  {object}  api.ErrorResponse
// @Failure      404  {object}  api.ErrorResponse
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/audit-events/{id}/verify [get]
func (h *Handler) VerifyHashChain(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
			Code:    constant.ErrInvalidID.Error(),
			Title:   "Bad Request",
			Message: "invalid audit event ID",
		})
	}

	result, err := h.querySvc.VerifyHashChain(c.Context(), id)
	if err != nil {
		if errors.Is(err, constant.ErrAuditEntryNotFound) {
			return libHTTP.Respond(c, fiber.StatusNotFound, api.ErrorResponse{
				Code:    constant.ErrAuditEntryNotFound.Error(),
				Title:   "Not Found",
				Message: "audit entry not found",
			})
		}

		logger, _, _, _ := libCommons.NewTrackingFromContext(c.Context())
		logger.Log(c.Context(), libLog.LevelError, "Failed to verify audit hash chain", libLog.Any("error.message", err.Error()))

		return libHTTP.Respond(c, fiber.StatusInternalServerError, api.ErrorResponse{
			Code:    constant.ErrInternalServer.Error(),
			Title:   "Internal Server Error",
			Message: "internal server error",
		})
	}

	return libHTTP.Respond(c, fiber.StatusOK, result)
}
