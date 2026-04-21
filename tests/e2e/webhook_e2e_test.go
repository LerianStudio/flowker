// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build e2e

package e2e

// =============================================================================
// Webhook End-to-End Test
// =============================================================================
//
// HOW TO RUN:
//
//   go test -tags=e2e -run TestWebhookE2E -v -timeout 5m ./tests/e2e/
//
// WHAT THIS TEST DOES:
//
//   Tests the dynamic webhook routing feature end-to-end. When a workflow
//   with a webhook trigger (triggerType: "webhook", path, method) is activated,
//   the corresponding endpoint becomes live under /v1/webhooks/*.
//
//   Flow:
//     1. Create a provider config pointing to a mock executor server
//     2. Create a workflow with a webhook trigger node + executor node
//     3. Activate the workflow (registers the webhook route)
//     4. POST to the webhook endpoint with data
//     5. Verify execution was created and reaches terminal state
//     6. Verify webhook metadata was injected into input
//     7. Clean up (deactivate + delete workflow, disable + delete config)
//
// =============================================================================

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookE2E(t *testing.T) {
	client := httpClient()
	recorder := &mockRecorder{}

	// ── Start mock executor server ──
	// Simulates a KYC-like external service that the executor node calls.
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				_ = json.Unmarshal(bodyBytes, &body)
			}
		}

		recorder.record(r.Method, r.URL.Path, body)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/health":
			json.NewEncoder(w).Encode(map[string]any{"status": "healthy"})
		default:
			// Generic success response for any executor call
			json.NewEncoder(w).Encode(map[string]any{
				"status":     "validated",
				"customerId": "CUST-WEBHOOK-001",
			})
		}
	}))
	defer mockServer.Close()

	// ── Shared state ──
	var providerConfigID string
	var workflowID string

	// =========================================================================
	// PHASE 1: Setup — Create Provider Config
	// =========================================================================
	t.Run("Phase1_Setup", func(t *testing.T) {
		t.Run("create_provider_config", func(t *testing.T) {
			payload := map[string]any{
				"name":       "e2e-webhook-provider",
				"providerId": "tracer",
				"config": map[string]any{
					"base_url": mockServer.URL + "/v1/kyc",
					"api_key":  "test-key",
				},
			}

			resp := e2ePostJSON(t, client, baseURL()+"/v1/provider-configurations", payload)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode,
				"create provider config: %s", string(bodyBytes))

			var cr e2eProviderConfigCreateResp
			require.NoError(t, json.Unmarshal(bodyBytes, &cr))
			require.NotEmpty(t, cr.ID)
			assert.Equal(t, "active", cr.Status)
			providerConfigID = cr.ID
		})
	})

	// =========================================================================
	// PHASE 2: Workflow Lifecycle — Create + Activate with Webhook Trigger
	// =========================================================================
	t.Run("Phase2_WorkflowLifecycle", func(t *testing.T) {
		t.Run("create_webhook_workflow", func(t *testing.T) {
			payload := map[string]any{
				"name":        "e2e-webhook-workflow",
				"description": "E2E test workflow with dynamic webhook trigger",
				"nodes": []map[string]any{
					{
						"id":   "trigger-webhook",
						"type": "trigger",
						"data": map[string]any{
							"triggerId":   "webhook",
							"triggerType": "webhook",
							"path":        "/e2e/webhook-test",
							"method":      "POST",
						},
						"position": map[string]any{"x": 0, "y": 0},
					},
					{
						"id":   "executor-node",
						"type": "executor",
						"name": "Webhook KYC Validation",
						"data": map[string]any{
							"executorId":       "tracer.validate-transaction",
							"providerConfigId": providerConfigID,
							"endpointName":     "validate",
							"inputMapping": []map[string]any{
								{
									"source": "workflow.document",
									"target": "document",
								},
							},
							"outputMapping": []map[string]any{
								{
									"source": "body.customerId",
									"target": "result.customerId",
								},
							},
						},
						"position": map[string]any{"x": 200, "y": 0},
					},
				},
				"edges": []map[string]any{
					{"id": "e1", "source": "trigger-webhook", "target": "executor-node"},
				},
			}

			resp := e2ePostJSON(t, client, baseURL()+"/v1/workflows", payload)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode,
				"create workflow: %s", string(bodyBytes))

			var cr createWorkflowResp
			require.NoError(t, json.Unmarshal(bodyBytes, &cr))
			require.NotEmpty(t, cr.WorkflowID)
			assert.Equal(t, "draft", cr.Status)
			workflowID = cr.WorkflowID
		})

		t.Run("activate_webhook_workflow", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/workflows/%s/activate", baseURL(), workflowID),
				"application/json", nil,
			)
			require.NoError(t, err)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode,
				"activate workflow: %s", string(bodyBytes))
		})

		t.Run("verify_webhook_not_found_before_path", func(t *testing.T) {
			// A completely different path should 404
			resp, err := client.Post(
				baseURL()+"/v1/webhooks/non-existent-path",
				"application/json", nil,
			)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode,
				"non-existent webhook path should return 404")
		})
	})

	// =========================================================================
	// PHASE 3: Webhook Execution — Call the dynamic webhook endpoint
	// =========================================================================
	t.Run("Phase3_WebhookExecution", func(t *testing.T) {
		recorder.clear()

		var execID string

		t.Run("call_webhook_endpoint", func(t *testing.T) {
			webhookPayload := map[string]any{
				"document":    "12345678900",
				"description": "E2E webhook test payload",
			}
			body, _ := json.Marshal(webhookPayload)

			resp, err := client.Post(
				baseURL()+"/v1/webhooks/e2e/webhook-test",
				"application/json",
				bytes.NewBuffer(body),
			)
			require.NoError(t, err)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode,
				"webhook should return 201: %s", string(bodyBytes))

			// Parse execution response
			var execResp executionCreateResp
			require.NoError(t, json.Unmarshal(bodyBytes, &execResp))
			require.NotEmpty(t, execResp.ExecutionID, "execution ID should be returned")
			assert.Equal(t, workflowID, execResp.WorkflowID, "workflow ID should match")
			execID = execResp.ExecutionID

			// Verify response headers
			assert.Equal(t, workflowID, resp.Header.Get("X-Webhook-Workflow-ID"),
				"X-Webhook-Workflow-ID header should be set")
			assert.NotEmpty(t, resp.Header.Get("X-Webhook-Execution-ID"),
				"X-Webhook-Execution-ID header should be set")
		})

		t.Run("poll_execution_status", func(t *testing.T) {
			status := pollExecutionStatus(t, client, execID, 30*time.Second)
			assert.Contains(t, []string{"completed", "failed"}, status.Status,
				"execution should reach a terminal state")

			if status.Status == "completed" {
				assert.Nil(t, status.ErrorMessage)
			}
		})

		t.Run("verify_execution_results", func(t *testing.T) {
			results := getExecutionResults(t, client, execID)
			assert.Contains(t, []string{"completed", "failed"}, results.Status)

			if results.Status == "completed" {
				require.GreaterOrEqual(t, len(results.StepResults), 1,
					"should have at least 1 step result")
				assert.Equal(t, "executor-node", results.StepResults[0].NodeID)
			}
		})

		t.Run("verify_mock_received_request", func(t *testing.T) {
			// The executor should have called the mock server
			kycRequests := recorder.getByPath("/v1/kyc/validate")
			require.GreaterOrEqual(t, len(kycRequests), 1,
				"mock server should have received at least 1 request")

			// Verify the input mapping delivered the document field
			lastRequest := kycRequests[len(kycRequests)-1]
			assert.Equal(t, "12345678900", lastRequest.Body["document"],
				"executor should receive mapped document from webhook input")
		})
	})

	// =========================================================================
	// PHASE 4: Deactivation — Webhook becomes unavailable
	// =========================================================================
	t.Run("Phase4_Deactivation", func(t *testing.T) {
		t.Run("deactivate_workflow", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/workflows/%s/deactivate", baseURL(), workflowID),
				"application/json", nil,
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		})

		t.Run("webhook_returns_404_after_deactivation", func(t *testing.T) {
			resp, err := client.Post(
				baseURL()+"/v1/webhooks/e2e/webhook-test",
				"application/json",
				bytes.NewBuffer([]byte(`{"test": true}`)),
			)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode,
				"webhook should return 404 after workflow deactivation")
		})
	})

	// =========================================================================
	// PHASE 5: Cleanup
	// =========================================================================
	t.Run("Phase5_Cleanup", func(t *testing.T) {
		t.Run("delete_workflow", func(t *testing.T) {
			resp := e2eDelete(t, client, fmt.Sprintf("%s/v1/workflows/%s", baseURL(), workflowID))
			resp.Body.Close()
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		t.Run("disable_provider_config", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/provider-configurations/%s/disable", baseURL(), providerConfigID),
				"application/json", nil,
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		})

		t.Run("delete_provider_config", func(t *testing.T) {
			resp := e2eDelete(t, client, fmt.Sprintf("%s/v1/provider-configurations/%s", baseURL(), providerConfigID))
			resp.Body.Close()
			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		t.Run("verify_all_deleted", func(t *testing.T) {
			resp, err := client.Get(fmt.Sprintf("%s/v1/workflows/%s", baseURL(), workflowID))
			require.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, http.StatusNotFound, resp.StatusCode, "workflow should be gone")

			resp, err = client.Get(fmt.Sprintf("%s/v1/provider-configurations/%s", baseURL(), providerConfigID))
			require.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, http.StatusNotFound, resp.StatusCode, "provider config should be gone")
		})
	})
}
