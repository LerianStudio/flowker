// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package workflow_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	httpWorkflow "github.com/LerianStudio/flowker/internal/adapters/http/in/workflow"
	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//go:generate go run go.uber.org/mock/mockgen@latest -source=handler.go -destination=mock_test.go -package=workflow_test

func createTestApp(t *testing.T, ctrl *gomock.Controller) (*fiber.App, *MockCommandService, *MockQueryService) {
	t.Helper()

	mockCmdSvc := NewMockCommandService(ctrl)
	mockQuerySvc := NewMockQueryService(ctrl)

	app := fiber.New()
	handler, err := httpWorkflow.NewHandler(mockCmdSvc, mockQuerySvc)
	require.NoError(t, err)

	handler.RegisterRoutes(app.Group("/v1"))

	return app, mockCmdSvc, mockQuerySvc
}

func createTestWorkflow(t *testing.T, name string) *model.Workflow {
	t.Helper()

	triggerNode, err := model.NewWorkflowNode("trigger-1", model.NodeTypeTrigger, nil, model.Position{X: 100, Y: 50}, map[string]any{"triggerType": "http"})
	require.NoError(t, err)

	nodes := []model.WorkflowNode{
		triggerNode,
	}

	w, err := model.NewWorkflow(name, nil, nodes, nil)
	require.NoError(t, err)

	return w
}

func TestHandler_Create(t *testing.T) {
	t.Run("creates workflow successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		workflow := createTestWorkflow(t, "Test Workflow")

		mockCmdSvc.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(workflow, nil)

		body := `{
			"name": "Test Workflow",
			"nodes": [{"id": "trigger-1", "type": "trigger", "position": {"x": 100, "y": 50}}]
		}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result model.WorkflowCreateOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, workflow.ID(), result.ID)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 409 for duplicate name", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		mockCmdSvc.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			Return(nil, constant.ErrWorkflowDuplicateName)

		body := `{
			"name": "Duplicate Workflow",
			"nodes": [{"id": "trigger-1", "type": "trigger", "position": {"x": 100, "y": 50}}]
		}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestHandler_GetByID(t *testing.T) {
	t.Run("returns workflow by ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		workflow := createTestWorkflow(t, "Test Workflow")

		mockQuerySvc.EXPECT().
			GetByID(gomock.Any(), workflow.ID()).
			Return(workflow, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/workflows/"+workflow.ID().String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.WorkflowOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, workflow.ID(), result.ID)
		assert.Equal(t, "Test Workflow", result.Name)
	})

	t.Run("returns 404 for non-existent workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		nonExistentID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockQuerySvc.EXPECT().
			GetByID(gomock.Any(), nonExistentID).
			Return(nil, constant.ErrWorkflowNotFound)

		req := httptest.NewRequest(http.MethodGet, "/v1/workflows/"+nonExistentID.String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		req := httptest.NewRequest(http.MethodGet, "/v1/workflows/invalid-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestHandler_List(t *testing.T) {
	t.Run("returns list of workflows", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		w1 := createTestWorkflow(t, "Workflow 1")
		w2 := createTestWorkflow(t, "Workflow 2")

		mockQuerySvc.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(&query.WorkflowListResult{
				Items:      []*model.Workflow{w1, w2},
				NextCursor: "",
				HasMore:    false,
			}, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.WorkflowListOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Len(t, result.Items, 2)
	})

	t.Run("returns empty list", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		mockQuerySvc.EXPECT().
			List(gomock.Any(), gomock.Any()).
			Return(&query.WorkflowListResult{
				Items:      []*model.Workflow{},
				NextCursor: "",
				HasMore:    false,
			}, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.WorkflowListOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Empty(t, result.Items)
	})

	t.Run("passes query parameters to filter", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		var capturedFilter query.WorkflowListFilter

		mockQuerySvc.EXPECT().
			List(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, filter query.WorkflowListFilter) (*query.WorkflowListResult, error) {
				capturedFilter = filter

				return &query.WorkflowListResult{
					Items:      []*model.Workflow{},
					NextCursor: "",
					HasMore:    false,
				}, nil
			})

		req := httptest.NewRequest(http.MethodGet, "/v1/workflows?status=active&limit=5&sortBy=name&sortOrder=ASC", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		require.NotNil(t, capturedFilter.Status)
		assert.Equal(t, model.WorkflowStatusActive, *capturedFilter.Status)
		assert.Equal(t, 5, capturedFilter.Limit)
		assert.Equal(t, "name", capturedFilter.SortBy)
		assert.Equal(t, "ASC", capturedFilter.SortOrder)
	})
}

func TestHandler_Update(t *testing.T) {
	t.Run("updates workflow successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		workflow := createTestWorkflow(t, "Updated Workflow")

		mockCmdSvc.EXPECT().
			Update(gomock.Any(), workflow.ID(), gomock.Any()).
			Return(workflow, nil)

		body := `{
			"name": "Updated Workflow",
			"nodes": [{"id": "trigger-1", "type": "trigger", "position": {"x": 100, "y": 50}}]
		}`
		req := httptest.NewRequest(http.MethodPut, "/v1/workflows/"+workflow.ID().String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		nonExistentID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Update(gomock.Any(), nonExistentID, gomock.Any()).
			Return(nil, constant.ErrWorkflowNotFound)

		body := `{
			"name": "Updated Workflow",
			"nodes": [{"id": "trigger-1", "type": "trigger", "position": {"x": 100, "y": 50}}]
		}`
		req := httptest.NewRequest(http.MethodPut, "/v1/workflows/"+nonExistentID.String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 422 for non-draft workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		nonDraftID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Update(gomock.Any(), nonDraftID, gomock.Any()).
			Return(nil, constant.ErrWorkflowCannotModify)

		body := `{
			"name": "Updated Workflow",
			"nodes": [{"id": "trigger-1", "type": "trigger", "position": {"x": 100, "y": 50}}]
		}`
		req := httptest.NewRequest(http.MethodPut, "/v1/workflows/"+nonDraftID.String(), bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})
}

func TestHandler_Delete(t *testing.T) {
	t.Run("deletes workflow successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		deleteID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Delete(gomock.Any(), deleteID).
			Return(nil)

		req := httptest.NewRequest(http.MethodDelete, "/v1/workflows/"+deleteID.String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		nonExistentID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Delete(gomock.Any(), nonExistentID).
			Return(constant.ErrWorkflowNotFound)

		req := httptest.NewRequest(http.MethodDelete, "/v1/workflows/"+nonExistentID.String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 422 for active workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		activeID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Delete(gomock.Any(), activeID).
			Return(constant.ErrWorkflowCannotModify)

		req := httptest.NewRequest(http.MethodDelete, "/v1/workflows/"+activeID.String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})
}

func TestHandler_Clone(t *testing.T) {
	t.Run("clones workflow successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		cloned := createTestWorkflow(t, "Cloned Workflow")
		sourceID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Clone(gomock.Any(), sourceID, gomock.Any()).
			Return(cloned, nil)

		body := `{"name": "Cloned Workflow"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+sourceID.String()+"/clone", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result model.WorkflowCreateOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, cloned.ID(), result.ID)
	})

	t.Run("returns 404 for non-existent source workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		sourceID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Clone(gomock.Any(), sourceID, gomock.Any()).
			Return(nil, constant.ErrWorkflowNotFound)

		body := `{"name": "Cloned Workflow"}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+sourceID.String()+"/clone", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestHandler_Activate(t *testing.T) {
	t.Run("activates workflow successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		workflow := createTestWorkflow(t, "Test Workflow")
		_ = workflow.Activate()

		mockCmdSvc.EXPECT().
			Activate(gomock.Any(), workflow.ID()).
			Return(workflow, nil)

		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflow.ID().String()+"/activate", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(respBody), "active")
	})

	t.Run("returns 404 for non-existent workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		nonExistentID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Activate(gomock.Any(), nonExistentID).
			Return(nil, constant.ErrWorkflowNotFound)

		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+nonExistentID.String()+"/activate", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 422 for already active workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		alreadyActiveID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Activate(gomock.Any(), alreadyActiveID).
			Return(nil, constant.ErrWorkflowInvalidStatus)

		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+alreadyActiveID.String()+"/activate", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})
}

func TestHandler_Deactivate(t *testing.T) {
	t.Run("deactivates workflow successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		workflow := createTestWorkflow(t, "Test Workflow")
		_ = workflow.Activate()
		_ = workflow.Deactivate()

		mockCmdSvc.EXPECT().
			Deactivate(gomock.Any(), workflow.ID()).
			Return(workflow, nil)

		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflow.ID().String()+"/deactivate", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(respBody), "inactive")
	})

	t.Run("returns 404 for non-existent workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		nonExistentID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Deactivate(gomock.Any(), nonExistentID).
			Return(nil, constant.ErrWorkflowNotFound)

		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+nonExistentID.String()+"/deactivate", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 422 for non-active workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		nonActiveID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")

		mockCmdSvc.EXPECT().
			Deactivate(gomock.Any(), nonActiveID).
			Return(nil, constant.ErrWorkflowInvalidStatus)

		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+nonActiveID.String()+"/deactivate", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})
}
