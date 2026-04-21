// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package executorconfiguration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	httpExecutorConfig "github.com/LerianStudio/flowker/internal/adapters/http/in/executor_configuration"
	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//go:generate go run go.uber.org/mock/mockgen@latest -source=handler.go -destination=mock_test.go -package=executorconfiguration_test

func createTestApp(t *testing.T, ctrl *gomock.Controller) (*fiber.App, *MockCommandService, *MockQueryService) {
	t.Helper()

	mockCmdSvc := NewMockCommandService(ctrl)
	mockQuerySvc := NewMockQueryService(ctrl)

	app := fiber.New()
	handler, err := httpExecutorConfig.NewHandler(mockCmdSvc, mockQuerySvc)
	require.NoError(t, err)

	handler.RegisterRoutes(app.Group("/v1"))

	return app, mockCmdSvc, mockQuerySvc
}

func createTestExecutorConfig(name string) *model.ExecutorConfiguration {
	endpoint, _ := model.NewExecutorEndpoint("validate", "/validate", "POST", 30)
	auth, _ := model.NewExecutorAuthentication("api_key", map[string]any{"key": "test"})
	config, _ := model.NewExecutorConfiguration(
		name,
		nil,
		"https://api.example.com",
		[]model.ExecutorEndpoint{*endpoint},
		*auth,
	)

	return config
}

func TestHandler_GetByID(t *testing.T) {
	t.Run("returns executor configuration by ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		config := createTestExecutorConfig("Test Executor")

		mockQuerySvc.EXPECT().
			GetByID(gomock.Any(), config.ID()).
			Return(config, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/executors/"+config.ID().String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.ExecutorConfigurationOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, config.ID(), result.ID)
		assert.Equal(t, "Test Executor", result.Name)
	})

	t.Run("returns 404 for non-existent executor configuration", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		nonExistentID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

		mockQuerySvc.EXPECT().
			GetByID(gomock.Any(), nonExistentID).
			Return(nil, constant.ErrExecutorConfigNotFound)

		req := httptest.NewRequest(http.MethodGet, "/v1/executors/"+nonExistentID.String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		req := httptest.NewRequest(http.MethodGet, "/v1/executors/invalid-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("returns list of executor configurations", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		c1 := createTestExecutorConfig("Executor 1")
		c2 := createTestExecutorConfig("Executor 2")

		mockQuerySvc.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(&query.ExecutorConfigListResult{
				Items:      []*model.ExecutorConfiguration{c1, c2},
				NextCursor: "",
				HasMore:    false,
			}, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/executors/", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.ExecutorConfigurationListOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
	})

	t.Run("returns empty list", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		mockQuerySvc.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(&query.ExecutorConfigListResult{
				Items:      []*model.ExecutorConfiguration{},
				NextCursor: "",
				HasMore:    false,
			}, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/executors/", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.ExecutorConfigurationListOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Empty(t, result.Items)
	})

	t.Run("passes query parameters to filter", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		var capturedFilter query.ExecutorConfigListFilter

		mockQuerySvc.EXPECT().
			List(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, filter query.ExecutorConfigListFilter) (*query.ExecutorConfigListResult, error) {
				capturedFilter = filter

				return &query.ExecutorConfigListResult{
					Items:      []*model.ExecutorConfiguration{},
					NextCursor: "",
					HasMore:    false,
				}, nil
			})

		req := httptest.NewRequest(http.MethodGet, "/v1/executors/?status=active&limit=5&sortBy=name&sortOrder=ASC", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		require.NotNil(t, capturedFilter.Status)
		assert.Equal(t, model.ExecutorConfigurationStatusActive, *capturedFilter.Status)
		assert.Equal(t, 5, capturedFilter.Limit)
		assert.Equal(t, "name", capturedFilter.SortBy)
		assert.Equal(t, "ASC", capturedFilter.SortOrder)
	})
}

func TestHandler_Update(t *testing.T) {
	t.Run("updates executor configuration successfully via PATCH", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		config := createTestExecutorConfig("Updated Executor")

		mockCmdSvc.EXPECT().
			Update(gomock.Any(), config.ID(), gomock.Any()).
			Return(config, nil)

		body := `{
			"name": "Updated Executor",
			"baseUrl": "https://api.example.com",
			"endpoints": [{"name": "validate", "path": "/validate", "method": "POST"}],
			"authentication": {"type": "api_key"}
		}`
		req := httptest.NewRequest(http.MethodPatch, "/v1/executors/"+config.ID().String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent executor configuration", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		nonExistentID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

		mockCmdSvc.EXPECT().
			Update(gomock.Any(), nonExistentID, gomock.Any()).
			Return(nil, constant.ErrExecutorConfigNotFound)

		body := `{
			"name": "Updated Executor",
			"baseUrl": "https://api.example.com",
			"endpoints": [{"name": "validate", "path": "/validate", "method": "POST"}],
			"authentication": {"type": "api_key"}
		}`
		req := httptest.NewRequest(http.MethodPatch, "/v1/executors/"+nonExistentID.String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 400 for malformed JSON body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		id := uuid.New()
		req := httptest.NewRequest(http.MethodPatch, "/v1/executors/"+id.String(), bytes.NewBufferString(`{invalid json`))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 422 for active executor configuration", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		activeID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

		mockCmdSvc.EXPECT().
			Update(gomock.Any(), activeID, gomock.Any()).
			Return(nil, constant.ErrExecutorConfigCannotModify)

		body := `{
			"name": "Updated Executor",
			"baseUrl": "https://api.example.com",
			"endpoints": [{"name": "validate", "path": "/validate", "method": "POST"}],
			"authentication": {"type": "api_key"}
		}`
		req := httptest.NewRequest(http.MethodPatch, "/v1/executors/"+activeID.String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Run("deletes executor configuration successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		deleteID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

		mockCmdSvc.EXPECT().
			Delete(gomock.Any(), deleteID).
			Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/v1/executors/"+deleteID.String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent executor configuration", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		nonExistentID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

		mockCmdSvc.EXPECT().
			Delete(gomock.Any(), nonExistentID).
			Return(constant.ErrExecutorConfigNotFound)

		req := httptest.NewRequest(http.MethodDelete, "/v1/executors/"+nonExistentID.String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 422 for active executor configuration", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		activeID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

		mockCmdSvc.EXPECT().
			Delete(gomock.Any(), activeID).
			Return(constant.ErrExecutorConfigCannotModify)

		req := httptest.NewRequest(http.MethodDelete, "/v1/executors/"+activeID.String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})
}
