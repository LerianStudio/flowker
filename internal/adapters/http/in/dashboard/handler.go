// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package dashboard contains the HTTP handler for dashboard operations.
package dashboard

import (
	"context"
	"errors"
	"time"

	"github.com/LerianStudio/flowker/api"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	libHTTP "github.com/LerianStudio/lib-commons/v4/commons/net/http"
	"github.com/gofiber/fiber/v2"
)

// validExecutionStatuses defines the allowed status values for the execution summary filter.
var validExecutionStatuses = map[string]bool{
	string(model.ExecutionStatusPending):   true,
	string(model.ExecutionStatusRunning):   true,
	string(model.ExecutionStatusCompleted): true,
	string(model.ExecutionStatusFailed):    true,
}

// QueryService defines the interface for dashboard query operations.
type QueryService interface {
	WorkflowSummary(ctx context.Context) (*model.WorkflowSummaryOutput, error)
	ExecutionSummary(ctx context.Context, startTime, endTime *time.Time, status *string) (*model.ExecutionSummaryOutput, error)
}

// Handler handles HTTP requests for dashboard operations.
type Handler struct {
	querySvc QueryService
}

// ErrDashboardHandlerNilDependency is returned when a required dependency is nil.
var ErrDashboardHandlerNilDependency = errors.New("dashboard handler: required dependency cannot be nil")

// NewHandler creates a new dashboard HTTP handler.
func NewHandler(querySvc QueryService) (*Handler, error) {
	if querySvc == nil {
		return nil, ErrDashboardHandlerNilDependency
	}

	return &Handler{
		querySvc: querySvc,
	}, nil
}

// RegisterRoutes registers dashboard routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	dashboards := router.Group("/dashboards")
	dashboards.Get("/workflows/summary", h.WorkflowSummary)
	dashboards.Get("/executions", h.ExecutionSummary)
}

// WorkflowSummary handles GET /v1/dashboards/workflows/summary
// @Summary      Get workflow summary
// @Description  Retrieves an aggregated summary of all workflows grouped by status
// @Tags         Dashboard
// @Produce      json
// @Success      200  {object}  model.WorkflowSummaryOutput
// @Failure      500  {object}  api.ErrorResponse
// @Router       /v1/dashboards/workflows/summary [get]
func (h *Handler) WorkflowSummary(c *fiber.Ctx) error {
	result, err := h.querySvc.WorkflowSummary(c.Context())
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusInternalServerError, api.ErrorResponse{Code: constant.ErrInternalServer.Error(), Title: "Internal Server Error", Message: "internal server error"})
	}

	return libHTTP.Respond(c, fiber.StatusOK, result)
}

// ExecutionSummary handles GET /v1/dashboards/executions
// @Summary      Get execution summary
// @Description  Retrieves an aggregated summary of executions with optional time range and status filters
// @Tags         Dashboard
// @Produce      json
// @Param        startTime  query     string  false  "Start time filter (RFC3339)" Format(date-time)
// @Param        endTime    query     string  false  "End time filter (RFC3339)" Format(date-time)
// @Param        status      query     string  false  "Status filter"  Enums(pending,running,completed,failed)
// @Success      200         {object}  model.ExecutionSummaryOutput
// @Failure      400         {object}  api.ErrorResponse
// @Failure      500         {object}  api.ErrorResponse
// @Router       /v1/dashboards/executions [get]
func (h *Handler) ExecutionSummary(c *fiber.Ctx) error {
	var startTime, endTime *time.Time

	var status *string

	if startTimeStr := c.Query("startTime"); startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
				Code:    constant.ErrBadRequest.Error(),
				Title:   "Bad Request",
				Message: "invalid startTime format, expected RFC3339",
			})
		}

		startTime = &t
	}

	if endTimeStr := c.Query("endTime"); endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
				Code:    constant.ErrBadRequest.Error(),
				Title:   "Bad Request",
				Message: "invalid endTime format, expected RFC3339",
			})
		}

		endTime = &t
	}

	if startTime != nil && endTime != nil && startTime.After(*endTime) {
		return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
			Code:    constant.ErrBadRequest.Error(),
			Title:   "Bad Request",
			Message: "startTime must be before endTime",
		})
	}

	if statusStr := c.Query("status"); statusStr != "" {
		if !validExecutionStatuses[statusStr] {
			return libHTTP.Respond(c, fiber.StatusBadRequest, api.ErrorResponse{
				Code:    constant.ErrBadRequest.Error(),
				Title:   "Bad Request",
				Message: "invalid status, must be one of: pending, running, completed, failed",
			})
		}

		status = &statusStr
	}

	result, err := h.querySvc.ExecutionSummary(c.Context(), startTime, endTime, status)
	if err != nil {
		return libHTTP.Respond(c, fiber.StatusInternalServerError, api.ErrorResponse{Code: constant.ErrInternalServer.Error(), Title: "Internal Server Error", Message: "internal server error"})
	}

	return libHTTP.Respond(c, fiber.StatusOK, result)
}
