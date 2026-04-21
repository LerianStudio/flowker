// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/webhook"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecuteService is a test double for ExecuteService.
type mockExecuteService struct {
	executeFn func(ctx context.Context, workflowID uuid.UUID, input *model.ExecuteWorkflowInput, idempotencyKey *string) (*model.WorkflowExecution, error)
}

func (m *mockExecuteService) Execute(ctx context.Context, workflowID uuid.UUID, input *model.ExecuteWorkflowInput, idempotencyKey *string) (*model.WorkflowExecution, error) {
	return m.executeFn(ctx, workflowID, input, idempotencyKey)
}

// newRunningExecution creates a test WorkflowExecution in running status.
func newRunningExecution(executionID, workflowID uuid.UUID, inputData map[string]any) *model.WorkflowExecution {
	return model.NewWorkflowExecutionFromDB(
		executionID, workflowID, model.ExecutionStatusRunning,
		inputData, nil, nil,
		0, 1,
		nil, nil,
		time.Now(), nil,
	)
}

func TestNewHandler_NilRegistry(t *testing.T) {
	h, err := NewHandler(nil, &mockExecuteService{})
	require.Nil(t, h)
	require.ErrorIs(t, err, ErrWebhookHandlerNilRegistry)
}

func TestNewHandler_NilExecuteService(t *testing.T) {
	h, err := NewHandler(webhook.NewRegistry(), nil)
	require.Nil(t, h)
	require.ErrorIs(t, err, ErrWebhookHandlerNilExecuteService)
}

func TestNewHandler_Success(t *testing.T) {
	h, err := NewHandler(webhook.NewRegistry(), &mockExecuteService{})
	require.NoError(t, err)
	require.NotNil(t, h)
}

func TestHandleWebhook_RouteNotFound(t *testing.T) {
	registry := webhook.NewRegistry()
	handler, err := NewHandler(registry, &mockExecuteService{})
	require.NoError(t, err)

	app := fiber.New()
	webhooks := app.Group("/v1/webhooks")
	handler.RegisterRoutes(webhooks)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/unknown", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestHandleWebhook_Success(t *testing.T) {
	registry := webhook.NewRegistry()
	wfID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	execID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	err := registry.Register(webhook.Route{
		WorkflowID: wfID,
		Path:       "/payment/callback",
		Method:     "POST",
	})
	require.NoError(t, err)

	executeSvc := &mockExecuteService{
		executeFn: func(_ context.Context, workflowID uuid.UUID, input *model.ExecuteWorkflowInput, _ *string) (*model.WorkflowExecution, error) {
			assert.Equal(t, wfID, workflowID)
			assert.NotNil(t, input.InputData["_webhook"])

			return newRunningExecution(execID, workflowID, input.InputData), nil
		},
	}

	handler, err := NewHandler(registry, executeSvc)
	require.NoError(t, err)

	app := fiber.New()
	webhooks := app.Group("/v1/webhooks")
	handler.RegisterRoutes(webhooks)

	body, _ := json.Marshal(map[string]any{"amount": 100})
	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/payment/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)

	// Check debugging headers
	assert.Equal(t, wfID.String(), resp.Header.Get("X-Webhook-Workflow-ID"))
	assert.Equal(t, execID.String(), resp.Header.Get("X-Webhook-Execution-ID"))
}

func TestHandleWebhook_TokenValidation_Valid(t *testing.T) {
	registry := webhook.NewRegistry()
	wfID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	err := registry.Register(webhook.Route{
		WorkflowID:  wfID,
		Path:        "/secure-hook",
		Method:      "POST",
		VerifyToken: "my-secret",
	})
	require.NoError(t, err)

	executeSvc := &mockExecuteService{
		executeFn: func(_ context.Context, _ uuid.UUID, _ *model.ExecuteWorkflowInput, _ *string) (*model.WorkflowExecution, error) {
			return newRunningExecution(uuid.New(), wfID, nil), nil
		},
	}

	handler, err := NewHandler(registry, executeSvc)
	require.NoError(t, err)

	app := fiber.New()
	webhooks := app.Group("/v1/webhooks")
	handler.RegisterRoutes(webhooks)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/secure-hook", nil)
	req.Header.Set("X-Webhook-Token", "my-secret")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestHandleWebhook_TokenValidation_Invalid(t *testing.T) {
	registry := webhook.NewRegistry()

	err := registry.Register(webhook.Route{
		WorkflowID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Path:        "/secure-hook",
		Method:      "POST",
		VerifyToken: "my-secret",
	})
	require.NoError(t, err)

	handler, err := NewHandler(registry, &mockExecuteService{})
	require.NoError(t, err)

	app := fiber.New()
	webhooks := app.Group("/v1/webhooks")
	handler.RegisterRoutes(webhooks)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/secure-hook", nil)
	req.Header.Set("X-Webhook-Token", "wrong-token")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestHandleWebhook_TokenValidation_Missing(t *testing.T) {
	registry := webhook.NewRegistry()

	err := registry.Register(webhook.Route{
		WorkflowID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Path:        "/secure-hook",
		Method:      "POST",
		VerifyToken: "my-secret",
	})
	require.NoError(t, err)

	handler, err := NewHandler(registry, &mockExecuteService{})
	require.NoError(t, err)

	app := fiber.New()
	webhooks := app.Group("/v1/webhooks")
	handler.RegisterRoutes(webhooks)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/secure-hook", nil)
	// No X-Webhook-Token header

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestHandleWebhook_EmptyBody(t *testing.T) {
	registry := webhook.NewRegistry()
	wfID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	err := registry.Register(webhook.Route{
		WorkflowID: wfID,
		Path:       "/no-body",
		Method:     "POST",
	})
	require.NoError(t, err)

	executeSvc := &mockExecuteService{
		executeFn: func(_ context.Context, _ uuid.UUID, input *model.ExecuteWorkflowInput, _ *string) (*model.WorkflowExecution, error) {
			// Input data should still exist with webhook metadata
			assert.NotNil(t, input.InputData["_webhook"])

			return newRunningExecution(uuid.New(), wfID, input.InputData), nil
		},
	}

	handler, err := NewHandler(registry, executeSvc)
	require.NoError(t, err)

	app := fiber.New()
	webhooks := app.Group("/v1/webhooks")
	handler.RegisterRoutes(webhooks)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/no-body", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestHandleWebhook_IdempotencyKey(t *testing.T) {
	registry := webhook.NewRegistry()
	wfID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	err := registry.Register(webhook.Route{
		WorkflowID: wfID,
		Path:       "/with-key",
		Method:     "POST",
	})
	require.NoError(t, err)

	executeSvc := &mockExecuteService{
		executeFn: func(_ context.Context, _ uuid.UUID, _ *model.ExecuteWorkflowInput, idempotencyKey *string) (*model.WorkflowExecution, error) {
			require.NotNil(t, idempotencyKey)
			assert.Equal(t, "my-idempotency-key", *idempotencyKey)

			return newRunningExecution(uuid.New(), wfID, nil), nil
		},
	}

	handler, err := NewHandler(registry, executeSvc)
	require.NoError(t, err)

	app := fiber.New()
	webhooks := app.Group("/v1/webhooks")
	handler.RegisterRoutes(webhooks)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/with-key", nil)
	req.Header.Set("Idempotency-Key", "my-idempotency-key")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestHandleWebhook_ExecuteError(t *testing.T) {
	registry := webhook.NewRegistry()

	err := registry.Register(webhook.Route{
		WorkflowID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Path:       "/error-hook",
		Method:     "POST",
	})
	require.NoError(t, err)

	executeSvc := &mockExecuteService{
		executeFn: func(_ context.Context, _ uuid.UUID, _ *model.ExecuteWorkflowInput, _ *string) (*model.WorkflowExecution, error) {
			return nil, errors.New("unexpected error")
		},
	}

	handler, err := NewHandler(registry, executeSvc)
	require.NoError(t, err)

	app := fiber.New()
	webhooks := app.Group("/v1/webhooks")
	handler.RegisterRoutes(webhooks)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/error-hook", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestHandleWebhook_WorkflowNotActive(t *testing.T) {
	registry := webhook.NewRegistry()

	err := registry.Register(webhook.Route{
		WorkflowID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Path:       "/inactive-hook",
		Method:     "POST",
	})
	require.NoError(t, err)

	executeSvc := &mockExecuteService{
		executeFn: func(_ context.Context, _ uuid.UUID, _ *model.ExecuteWorkflowInput, _ *string) (*model.WorkflowExecution, error) {
			return nil, constant.ErrExecutionNotActive
		},
	}

	handler, err := NewHandler(registry, executeSvc)
	require.NoError(t, err)

	app := fiber.New()
	webhooks := app.Group("/v1/webhooks")
	handler.RegisterRoutes(webhooks)

	req := httptest.NewRequest(http.MethodPost, "/v1/webhooks/inactive-hook", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, resp.StatusCode)
}

func TestHandleWebhook_GETMethod(t *testing.T) {
	registry := webhook.NewRegistry()
	wfID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	err := registry.Register(webhook.Route{
		WorkflowID: wfID,
		Path:       "/get-hook",
		Method:     "GET",
	})
	require.NoError(t, err)

	executeSvc := &mockExecuteService{
		executeFn: func(_ context.Context, _ uuid.UUID, _ *model.ExecuteWorkflowInput, _ *string) (*model.WorkflowExecution, error) {
			return newRunningExecution(uuid.New(), wfID, nil), nil
		},
	}

	handler, err := NewHandler(registry, executeSvc)
	require.NoError(t, err)

	app := fiber.New()
	webhooks := app.Group("/v1/webhooks")
	handler.RegisterRoutes(webhooks)

	req := httptest.NewRequest(http.MethodGet, "/v1/webhooks/get-hook", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}
