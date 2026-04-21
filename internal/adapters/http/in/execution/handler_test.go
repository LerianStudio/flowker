// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package execution_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/api"
	httpExecution "github.com/LerianStudio/flowker/internal/adapters/http/in/execution"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func createTestApp(t *testing.T, ctrl *gomock.Controller) (*fiber.App, *MockCommandService, *MockQueryService) {
	t.Helper()

	mockCmdSvc := NewMockCommandService(ctrl)
	mockQuerySvc := NewMockQueryService(ctrl)

	app := fiber.New()
	handler, err := httpExecution.NewHandler(mockCmdSvc, mockQuerySvc)
	require.NoError(t, err)

	handler.RegisterRoutes(app.Group("/v1"))

	return app, mockCmdSvc, mockQuerySvc
}

func createTestExecution(workflowID uuid.UUID) *model.WorkflowExecution {
	return model.NewWorkflowExecution(workflowID, map[string]any{"cpf": "123"}, nil, 2)
}

func TestNewHandler_NilDependencies(t *testing.T) {
	t.Run("nil command service", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockQuerySvc := NewMockQueryService(ctrl)

		handler, err := httpExecution.NewHandler(nil, mockQuerySvc)
		require.Error(t, err)
		assert.Nil(t, handler)
	})

	t.Run("nil query service", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockCmdSvc := NewMockCommandService(ctrl)

		handler, err := httpExecution.NewHandler(mockCmdSvc, nil)
		require.Error(t, err)
		assert.Nil(t, handler)
	})
}

func TestHandler_Execute(t *testing.T) {
	t.Run("creates execution successfully", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()
		exec := createTestExecution(workflowID)

		mockCmdSvc.EXPECT().
			Execute(gomock.Any(), workflowID, gomock.Any(), gomock.Any()).
			Return(exec, nil)

		body := `{"inputData": {"cpf": "12345678900"}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var result model.ExecutionCreateOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, exec.ExecutionID(), result.ExecutionID)
		assert.Equal(t, "pending", result.Status)
	})

	t.Run("returns 200 for idempotent request", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()
		exec := createTestExecution(workflowID)
		_ = exec.MarkRunning()
		_ = exec.MarkCompleted(map[string]any{"result": "ok"})

		mockCmdSvc.EXPECT().
			Execute(gomock.Any(), workflowID, gomock.Any(), gomock.Any()).
			Return(exec, nil)

		body := `{"inputData": {"cpf": "123"}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "key-123")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("returns 400 for missing idempotency key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()
		body := `{"inputData": {"cpf": "123"}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 400 for empty idempotency key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()
		body := `{"inputData": {"cpf": "123"}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 400 for whitespace-only idempotency key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()
		body := `{"inputData": {"cpf": "123"}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "   ")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 413 for payload exceeding 1 MB", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()
		// Create a body slightly over 1 MB
		largeBody := make([]byte, 1<<20+1)
		for i := range largeBody {
			largeBody[i] = 'a'
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewReader(largeBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)

		var errResp api.ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Equal(t, constant.ErrExecutionInputTooLarge.Error(), errResp.Code)
		assert.Equal(t, "Payload Too Large", errResp.Title)
	})

	t.Run("accepts payload at exactly 1 MB", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()
		// Create exactly 1 MB body — will pass size check but fail JSON parsing
		exactBody := make([]byte, 1<<20)
		for i := range exactBody {
			exactBody[i] = 'a'
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewReader(exactBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())

		resp, err := app.Test(req)
		require.NoError(t, err)
		// Passes size check, but fails JSON parsing → 400
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 400 for invalid workflow ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		body := `{"inputData": {"cpf": "123"}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/invalid-uuid/executions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewBufferString("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()

		mockCmdSvc.EXPECT().
			Execute(gomock.Any(), workflowID, gomock.Any(), gomock.Any()).
			Return(nil, constant.ErrWorkflowNotFound)

		body := `{"inputData": {"cpf": "123"}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 422 for non-active workflow", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()

		mockCmdSvc.EXPECT().
			Execute(gomock.Any(), workflowID, gomock.Any(), gomock.Any()).
			Return(nil, constant.ErrExecutionNotActive)

		body := `{"inputData": {"cpf": "123"}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", uuid.NewString())

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})

	t.Run("returns 409 for duplicate idempotency key", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, mockCmdSvc, _ := createTestApp(t, ctrl)

		workflowID := uuid.New()

		mockCmdSvc.EXPECT().
			Execute(gomock.Any(), workflowID, gomock.Any(), gomock.Any()).
			Return(nil, constant.ErrExecutionDuplicate)

		body := `{"inputData": {"cpf": "123"}}`
		req := httptest.NewRequest(http.MethodPost, "/v1/workflows/"+workflowID.String()+"/executions", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "duplicate-key")

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})
}

func TestHandler_GetStatus(t *testing.T) {
	t.Run("returns execution status", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		exec := createTestExecution(uuid.New())
		_ = exec.MarkRunning()

		mockQuerySvc.EXPECT().
			GetByID(gomock.Any(), exec.ExecutionID()).
			Return(exec, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/executions/"+exec.ExecutionID().String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.ExecutionStatusOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, exec.ExecutionID(), result.ExecutionID)
		assert.Equal(t, "running", result.Status)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		req := httptest.NewRequest(http.MethodGet, "/v1/executions/invalid-uuid", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent execution", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		nonExistentID := uuid.New()

		mockQuerySvc.EXPECT().
			GetByID(gomock.Any(), nonExistentID).
			Return(nil, constant.ErrExecutionNotFound)

		req := httptest.NewRequest(http.MethodGet, "/v1/executions/"+nonExistentID.String(), nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

func TestHandler_GetResults(t *testing.T) {
	t.Run("returns execution results", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		exec := createTestExecution(uuid.New())
		_ = exec.MarkRunning()
		step := model.NewExecutionStep(1, "KYC", "node-kyc", nil)
		_ = step.MarkCompleted(map[string]any{"approved": true})
		exec.AddStep(step)
		_ = exec.MarkCompleted(map[string]any{"final": "result"})

		mockQuerySvc.EXPECT().
			GetResults(gomock.Any(), exec.ExecutionID()).
			Return(exec, nil)

		req := httptest.NewRequest(http.MethodGet, "/v1/executions/"+exec.ExecutionID().String()+"/results", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var result model.ExecutionResultsOutput
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
		assert.Equal(t, exec.ExecutionID(), result.ExecutionID)
		assert.Equal(t, "completed", result.Status)
		assert.Len(t, result.StepResults, 1)
	})

	t.Run("returns 400 for invalid UUID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, _ := createTestApp(t, ctrl)

		req := httptest.NewRequest(http.MethodGet, "/v1/executions/invalid-uuid/results", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 404 for non-existent execution", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		nonExistentID := uuid.New()

		mockQuerySvc.EXPECT().
			GetResults(gomock.Any(), nonExistentID).
			Return(nil, constant.ErrExecutionNotFound)

		req := httptest.NewRequest(http.MethodGet, "/v1/executions/"+nonExistentID.String()+"/results", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("returns 422 for execution in progress", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		app, _, mockQuerySvc := createTestApp(t, ctrl)

		execID := uuid.New()

		mockQuerySvc.EXPECT().
			GetResults(gomock.Any(), execID).
			Return(nil, constant.ErrExecutionInProgress)

		req := httptest.NewRequest(http.MethodGet, "/v1/executions/"+execID.String()+"/results", nil)

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	})
}
