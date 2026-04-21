// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package dashboard_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpDashboard "github.com/LerianStudio/flowker/internal/adapters/http/in/dashboard"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func createTestApp(t *testing.T, ctrl *gomock.Controller) (*fiber.App, *MockQueryService) {
	t.Helper()

	mockQuerySvc := NewMockQueryService(ctrl)

	app := fiber.New()
	handler, err := httpDashboard.NewHandler(mockQuerySvc)
	require.NoError(t, err)

	handler.RegisterRoutes(app.Group("/v1"))

	return app, mockQuerySvc
}

func TestNewHandler_NilDependency(t *testing.T) {
	t.Run("nil query service", func(t *testing.T) {
		handler, err := httpDashboard.NewHandler(nil)
		require.Error(t, err)
		assert.Nil(t, handler)
		assert.ErrorIs(t, err, httpDashboard.ErrDashboardHandlerNilDependency)
	})
}

func TestHandler_WorkflowSummary(t *testing.T) {
	t.Run("returns workflow summary successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockQuerySvc := createTestApp(t, ctrl)

		expected := &model.WorkflowSummaryOutput{
			Total:  5,
			Active: 3,
			ByStatus: []model.StatusCountOutput{
				{Status: "active", Count: 3},
				{Status: "draft", Count: 2},
			},
		}

		mockQuerySvc.EXPECT().
			WorkflowSummary(gomock.Any()).
			Return(expected, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/dashboards/workflows/summary", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.WorkflowSummaryOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, int64(5), result.Total)
		assert.Equal(t, int64(3), result.Active)
		assert.Len(t, result.ByStatus, 2)
	})
}

func TestHandler_ExecutionSummary(t *testing.T) {
	t.Run("returns execution summary with no filters", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockQuerySvc := createTestApp(t, ctrl)

		expected := &model.ExecutionSummaryOutput{
			Total:     10,
			Completed: 6,
			Failed:    2,
			Pending:   1,
			Running:   1,
		}

		mockQuerySvc.EXPECT().
			ExecutionSummary(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(expected, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/dashboards/executions", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.ExecutionSummaryOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, int64(10), result.Total)
		assert.Equal(t, int64(6), result.Completed)
	})

	t.Run("returns execution summary with all filters", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockQuerySvc := createTestApp(t, ctrl)

		expected := &model.ExecutionSummaryOutput{
			Total:     3,
			Completed: 3,
		}

		mockQuerySvc.EXPECT().
			ExecutionSummary(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(expected, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/dashboards/executions?startTime=2026-01-01T00:00:00Z&endTime=2026-01-31T23:59:59Z&status=completed", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.ExecutionSummaryOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, int64(3), result.Total)
	})

	t.Run("returns 400 for invalid startTime format", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _ := createTestApp(t, ctrl)

		req := httptest.NewRequest(http.MethodGet, "/v1/dashboards/executions?startTime=not-a-date", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 400 for startTime after endTime", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _ := createTestApp(t, ctrl)

		req := httptest.NewRequest(http.MethodGet, "/v1/dashboards/executions?startTime=2026-02-01T00:00:00Z&endTime=2026-01-01T00:00:00Z", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 400 for invalid status", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _ := createTestApp(t, ctrl)

		req := httptest.NewRequest(http.MethodGet, "/v1/dashboards/executions?status=invalid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
