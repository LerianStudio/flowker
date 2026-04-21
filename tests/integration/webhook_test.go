// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

// =============================================================================
// Webhook Integration Tests — Dynamic Webhook Routes
// =============================================================================
//
// These tests verify that the dynamic webhook routing feature works correctly
// end-to-end with a running Flowker instance. When a workflow with a webhook
// trigger (triggerType: "webhook", path: "/some/path", method: "POST") is
// activated, the corresponding endpoint becomes live under /v1/webhooks/*.
//
// Test scenarios:
//   1. Create + activate workflow with webhook trigger -> call webhook -> verify execution
//   2. Call webhook for non-existent path -> 404
//   3. Deactivate workflow -> webhook returns 404
//   4. Webhook with verify_token -> valid token succeeds, invalid/missing fails
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

// ─── Webhook test helpers ──────────────────────────────────────────────────

// webhookWorkflowPayload creates a workflow payload with a webhook trigger node
// and a single executor node. The webhook trigger includes triggerType, path,
// method, and optional verify_token fields in the data map.
func webhookWorkflowPayload(name, providerConfigID, webhookPath, webhookMethod string, verifyToken string) map[string]any {
	triggerData := map[string]any{
		"triggerId":   "webhook",
		"triggerType": "webhook",
		"path":        webhookPath,
		"method":      webhookMethod,
	}
	if verifyToken != "" {
		triggerData["verify_token"] = verifyToken
	}

	return map[string]any{
		"name": name,
		"nodes": []map[string]any{
			{
				"id":       "trigger-webhook",
				"type":     "trigger",
				"data":     triggerData,
				"position": map[string]any{"x": 0, "y": 0},
			},
			{
				"id":   "exec-node",
				"type": "executor",
				"name": "Webhook Executor",
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
				},
				"position": map[string]any{"x": 200, "y": 0},
			},
		},
		"edges": []map[string]any{
			{"id": "e1", "source": "trigger-webhook", "target": "exec-node"},
		},
	}
}

// createAndActivateWebhookWorkflow creates a workflow with a webhook trigger
// and activates it. Returns the workflow ID.
func createAndActivateWebhookWorkflow(
	t *testing.T,
	client *http.Client,
	name, providerConfigID, webhookPath, webhookMethod, verifyToken string,
) string {
	t.Helper()

	payload := webhookWorkflowPayload(name, providerConfigID, webhookPath, webhookMethod, verifyToken)
	body, _ := json.Marshal(payload)

	resp, err := client.Post(baseURL()+"/v1/workflows", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err, "create workflow")

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("create workflow: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var cResp createWorkflowResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&cResp))
	resp.Body.Close()

	require.NotEmpty(t, cResp.WorkflowID)

	// Register cleanup early, before activation, so resources are cleaned up
	// even if activation or subsequent test steps fail.
	t.Cleanup(func() {
		// Best-effort deactivate (may already be deactivated or still in draft).
		deactResp, deactErr := client.Post(
			fmt.Sprintf("%s/v1/workflows/%s/deactivate", baseURL(), cResp.WorkflowID),
			"application/json", nil,
		)
		if deactErr == nil {
			deactResp.Body.Close()
		}

		// Best-effort delete.
		delReq, _ := http.NewRequest(http.MethodDelete, baseURL()+"/v1/workflows/"+cResp.WorkflowID, nil)
		delResp, delErr := client.Do(delReq)
		if delErr == nil {
			delResp.Body.Close()
		}
	})

	// Activate
	resp, err = client.Post(
		fmt.Sprintf("%s/v1/workflows/%s/activate", baseURL(), cResp.WorkflowID),
		"application/json", nil,
	)
	require.NoError(t, err, "activate workflow")

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("activate workflow: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	resp.Body.Close()

	return cResp.WorkflowID
}

// webhookDeactivateWorkflow deactivates the given workflow.
func webhookDeactivateWorkflow(t *testing.T, client *http.Client, workflowID string) {
	t.Helper()

	resp, err := client.Post(
		fmt.Sprintf("%s/v1/workflows/%s/deactivate", baseURL(), workflowID),
		"application/json", nil,
	)
	require.NoError(t, err, "deactivate workflow")
	require.Equal(t, http.StatusOK, resp.StatusCode, "deactivate should succeed")
	resp.Body.Close()
}

// ─── Test: Create + Activate + Call Webhook -> Verify Execution ────────────

func TestWebhook_CreateActivateCallWebhook(t *testing.T) {
	client := httpClient()

	// Start a mock server for the executor endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/health" {
			json.NewEncoder(w).Encode(map[string]any{"status": "healthy"})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"result":   "ok",
			"document": "received",
		})
	}))
	defer mockServer.Close()

	// Seed provider config pointing to the mock
	pcID := seedProviderConfig(t, "webhook-test-provider", "tracer", map[string]any{
		"base_url": mockServer.URL + "/v1/kyc",
		"api_key":  "test-key",
	})
	defer seedDeleteProviderConfig(t, pcID)

	// Create and activate workflow with webhook trigger
	wfID := createAndActivateWebhookWorkflow(t, client,
		"wf-webhook-basic", pcID,
		"/test/hook", "POST", "",
	)

	// Call the webhook endpoint
	webhookPayload := map[string]any{
		"document": "12345678900",
		"amount":   1000,
	}
	body, _ := json.Marshal(webhookPayload)
	resp, err := client.Post(baseURL()+"/v1/webhooks/test/hook", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err, "call webhook endpoint")

	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	// Assert 201 Created (execution started asynchronously)
	require.Equal(t, http.StatusCreated, resp.StatusCode,
		"webhook should return 201, got %d, body: %s", resp.StatusCode, string(bodyBytes))

	// Parse response and verify execution ID
	var execResp executionCreateResp
	require.NoError(t, json.Unmarshal(bodyBytes, &execResp))
	assert.NotEmpty(t, execResp.ExecutionID, "response should contain execution ID")
	assert.Equal(t, wfID, execResp.WorkflowID, "response workflow ID should match")

	// Verify X-Webhook-Workflow-ID header
	webhookWorkflowID := resp.Header.Get("X-Webhook-Workflow-ID")
	assert.Equal(t, wfID, webhookWorkflowID, "X-Webhook-Workflow-ID header should be set")

	// Verify X-Webhook-Execution-ID header
	webhookExecID := resp.Header.Get("X-Webhook-Execution-ID")
	assert.NotEmpty(t, webhookExecID, "X-Webhook-Execution-ID header should be set")
	assert.Equal(t, execResp.ExecutionID, webhookExecID, "execution ID in header should match response body")

	// Verify execution was created by polling its status
	deadline := time.Now().Add(30 * time.Second)
	reachedTerminal := false

	for time.Now().Before(deadline) {
		resp, err = client.Get(fmt.Sprintf("%s/v1/executions/%s", baseURL(), execResp.ExecutionID))
		require.NoError(t, err)

		var status executionStatusResp
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&status))
		resp.Body.Close()

		if status.Status == "completed" || status.Status == "failed" {
			reachedTerminal = true
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	require.True(t, reachedTerminal,
		"execution %s did not reach a terminal status (completed/failed) within 30s", execResp.ExecutionID)
}

// ─── Test: Call Webhook for Non-Existent Path -> 404 ───────────────────────

func TestWebhook_NonExistentPath_Returns404(t *testing.T) {
	client := httpClient()

	resp, err := client.Post(baseURL()+"/v1/webhooks/does-not-exist", "application/json", nil)
	require.NoError(t, err, "call non-existent webhook")
	resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "non-existent webhook path should return 404")
}

// ─── Test: Deactivate Workflow -> Webhook Returns 404 ──────────────────────

func TestWebhook_DeactivateWorkflow_Returns404(t *testing.T) {
	client := httpClient()

	// Start a mock server for the executor endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/health" {
			json.NewEncoder(w).Encode(map[string]any{"status": "healthy"})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{"result": "ok"})
	}))
	defer mockServer.Close()

	// Seed provider config
	pcID := seedProviderConfig(t, "webhook-deact-provider", "tracer", map[string]any{
		"base_url": mockServer.URL + "/v1/kyc",
		"api_key":  "test-key",
	})
	defer seedDeleteProviderConfig(t, pcID)

	// Create and activate workflow with webhook trigger
	wfID := createAndActivateWebhookWorkflow(t, client,
		"wf-webhook-deact", pcID,
		"/test/deactivate-hook", "POST", "",
	)

	// Verify the webhook works while active
	resp, err := client.Post(baseURL()+"/v1/webhooks/test/deactivate-hook", "application/json",
		bytes.NewBuffer([]byte(`{"test": true}`)))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode, "webhook should work while active")
	resp.Body.Close()

	// Deactivate the workflow
	webhookDeactivateWorkflow(t, client, wfID)

	// Now the webhook should return 404
	resp, err = client.Post(baseURL()+"/v1/webhooks/test/deactivate-hook", "application/json",
		bytes.NewBuffer([]byte(`{"test": true}`)))
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode,
		"webhook should return 404 after workflow deactivation")
}

// ─── Test: Webhook with verify_token ───────────────────────────────────────

func TestWebhook_VerifyToken(t *testing.T) {
	client := httpClient()

	// Start a mock server for the executor endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/health" {
			json.NewEncoder(w).Encode(map[string]any{"status": "healthy"})
			return
		}

		json.NewEncoder(w).Encode(map[string]any{"result": "ok"})
	}))
	defer mockServer.Close()

	// Seed provider config
	pcID := seedProviderConfig(t, "webhook-token-provider", "tracer", map[string]any{
		"base_url": mockServer.URL + "/v1/kyc",
		"api_key":  "test-key",
	})
	defer seedDeleteProviderConfig(t, pcID)

	secret := "my-secret-token" //nolint:gosec // test token, not a real secret

	// Create and activate workflow with webhook trigger and verify_token
	// Cleanup is handled by t.Cleanup registered in createAndActivateWebhookWorkflow.
	createAndActivateWebhookWorkflow(t, client,
		"wf-webhook-token", pcID,
		"/test/token-hook", "POST", secret,
	)

	webhookURL := baseURL() + "/v1/webhooks/test/token-hook"
	payload := []byte(`{"document": "12345"}`)

	t.Run("valid_token_succeeds", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Token", secret)

		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode,
			"webhook with valid token should return 201")
	})

	t.Run("invalid_token_returns_401", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Webhook-Token", "wrong-token")

		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"webhook with invalid token should return 401")
	})

	t.Run("missing_token_returns_401", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(payload))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		// No X-Webhook-Token header

		resp, err := client.Do(req)
		require.NoError(t, err)
		resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode,
			"webhook without token should return 401 when verify_token is configured")
	})
}
