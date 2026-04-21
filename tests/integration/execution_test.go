// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

type executionCreateResp struct {
	ExecutionID string `json:"executionId"`
	WorkflowID  string `json:"workflowId"`
	Status      string `json:"status"`
}

type executionStatusResp struct {
	ExecutionID       string  `json:"executionId"`
	WorkflowID        string  `json:"workflowId"`
	Status            string  `json:"status"`
	CurrentStepNumber int     `json:"currentStepNumber"`
	TotalSteps        int     `json:"totalSteps"`
	ErrorMessage      *string `json:"errorMessage,omitempty"`
}

type executionResultsResp struct {
	ExecutionID string           `json:"executionId"`
	Status      string           `json:"status"`
	StepResults []stepResultResp `json:"stepResults"`
	FinalOutput map[string]any   `json:"finalOutput,omitempty"`
}

type stepResultResp struct {
	StepNumber int    `json:"stepNumber"`
	StepName   string `json:"stepName"`
	Status     string `json:"status"`
}

// createAndActivateWorkflow creates a minimal workflow and activates it.
// Seeds its own provider configuration and returns the workflow ID.
// The caller should call seedDeleteProviderConfig with the returned pcID.
func createAndActivateWorkflow(t *testing.T, name string) (workflowID string, providerConfigCleanup func()) {
	t.Helper()

	pcID := seedProviderConfig(t, "tracer-exec-"+name, "tracer", map[string]any{"base_url": "https://example.com", "api_key": "test-key"})

	client := httpClient()
	payload := minimalWorkflowPayload(name, pcID)
	body, _ := json.Marshal(payload)

	// Create
	resp, err := client.Post(baseURL()+"/v1/workflows", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", resp.StatusCode)
	}

	var cResp createWorkflowResp
	json.NewDecoder(resp.Body).Decode(&cResp)
	resp.Body.Close()

	// Activate
	resp, err = client.Post(fmt.Sprintf("%s/v1/workflows/%s/activate", baseURL(), cResp.WorkflowID), "application/json", nil)
	if err != nil {
		t.Fatalf("activate workflow: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("activate status = %d", resp.StatusCode)
	}
	resp.Body.Close()

	return cResp.WorkflowID, func() { seedDeleteProviderConfig(t, pcID) }
}

func TestExecutionCreate(t *testing.T) {
	client := httpClient()
	wfID, pcCleanup := createAndActivateWorkflow(t, "wf-exec-create")
	defer pcCleanup()

	// Execute workflow
	execPayload := map[string]any{
		"inputData": map[string]any{
			"cpf":    "12345678900",
			"amount": 1000,
		},
	}
	body, _ := json.Marshal(execPayload)

	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), wfID),
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		bodyBytes := make([]byte, 1024)
		n, _ := resp.Body.Read(bodyBytes)
		t.Fatalf("create execution status = %d, body = %s", resp.StatusCode, string(bodyBytes[:n]))
	}

	var execResp executionCreateResp
	json.NewDecoder(resp.Body).Decode(&execResp)
	resp.Body.Close()

	if execResp.ExecutionID == "" {
		t.Fatal("expected execution ID")
	}
	if execResp.WorkflowID != wfID {
		t.Fatalf("expected workflow ID %s, got %s", wfID, execResp.WorkflowID)
	}
	if execResp.Status != "running" {
		t.Fatalf("expected status running, got %s", execResp.Status)
	}
}

func TestExecutionGetStatus(t *testing.T) {
	client := httpClient()
	wfID, pcCleanup := createAndActivateWorkflow(t, "wf-exec-status")
	defer pcCleanup()

	// Execute
	execPayload := map[string]any{
		"inputData": map[string]any{"test": true},
	}
	body, _ := json.Marshal(execPayload)

	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), wfID),
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}

	var execResp executionCreateResp
	json.NewDecoder(resp.Body).Decode(&execResp)
	resp.Body.Close()

	// Wait for execution to finish (executor node will fail since there's no real executor)
	time.Sleep(2 * time.Second)

	// Get status
	resp, err = client.Get(fmt.Sprintf("%s/v1/executions/%s", baseURL(), execResp.ExecutionID))
	if err != nil {
		t.Fatalf("get execution status: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status code = %d", resp.StatusCode)
	}

	var statusResp executionStatusResp
	json.NewDecoder(resp.Body).Decode(&statusResp)
	resp.Body.Close()

	if statusResp.ExecutionID != execResp.ExecutionID {
		t.Fatalf("execution ID mismatch: %s != %s", statusResp.ExecutionID, execResp.ExecutionID)
	}

	// Status should be either running, completed, or failed
	validStatuses := map[string]bool{"pending": true, "running": true, "completed": true, "failed": true}
	if !validStatuses[statusResp.Status] {
		t.Fatalf("unexpected status: %s", statusResp.Status)
	}
}

func TestExecutionGetStatus_NotFound(t *testing.T) {
	client := httpClient()
	nonExistentID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	resp, err := client.Get(fmt.Sprintf("%s/v1/executions/%s", baseURL(), nonExistentID))
	if err != nil {
		t.Fatalf("get execution status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestExecutionGetStatus_InvalidID(t *testing.T) {
	client := httpClient()

	resp, err := client.Get(fmt.Sprintf("%s/v1/executions/%s", baseURL(), "not-a-uuid"))
	if err != nil {
		t.Fatalf("get execution status: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestExecutionGetResults_NotFound(t *testing.T) {
	client := httpClient()
	nonExistentID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	resp, err := client.Get(fmt.Sprintf("%s/v1/executions/%s/results", baseURL(), nonExistentID))
	if err != nil {
		t.Fatalf("get execution results: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestExecutionCreate_WorkflowNotFound(t *testing.T) {
	client := httpClient()
	nonExistentWfID := "cccccccc-cccc-cccc-cccc-cccccccccccc"

	execPayload := map[string]any{
		"inputData": map[string]any{"test": true},
	}
	body, _ := json.Marshal(execPayload)

	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), nonExistentWfID),
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestExecutionCreate_WorkflowNotActive(t *testing.T) {
	client := httpClient()

	// Seed provider config
	pcID := seedProviderConfig(t, "tracer-exec-not-active", "tracer", map[string]any{"base_url": "https://example.com", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, pcID)

	// Create workflow but do NOT activate it
	payload := minimalWorkflowPayload("wf-exec-not-active", pcID)
	body, _ := json.Marshal(payload)

	resp, err := client.Post(baseURL()+"/v1/workflows", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	var cResp createWorkflowResp
	json.NewDecoder(resp.Body).Decode(&cResp)
	resp.Body.Close()

	// Try to execute (should fail - workflow is draft, not active)
	execPayload := map[string]any{
		"inputData": map[string]any{"test": true},
	}
	body, _ = json.Marshal(execPayload)

	execReq, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), cResp.WorkflowID),
		bytes.NewBuffer(body),
	)
	execReq.Header.Set("Content-Type", "application/json")
	execReq.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err = client.Do(execReq)
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		bodyBytes := make([]byte, 1024)
		n, _ := resp.Body.Read(bodyBytes)
		t.Fatalf("expected 422, got %d, body = %s", resp.StatusCode, string(bodyBytes[:n]))
	}
}

func TestExecutionCreate_InvalidWorkflowID(t *testing.T) {
	client := httpClient()

	execPayload := map[string]any{
		"inputData": map[string]any{"test": true},
	}
	body, _ := json.Marshal(execPayload)

	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), "not-a-uuid"),
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestExecutionCreate_MissingIdempotencyKey(t *testing.T) {
	client := httpClient()
	wfID, pcCleanup := createAndActivateWorkflow(t, "wf-exec-no-idem")
	defer pcCleanup()

	execPayload := map[string]any{
		"inputData": map[string]any{"test": true},
	}
	body, _ := json.Marshal(execPayload)

	// POST without Idempotency-Key header
	resp, err := client.Post(
		fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), wfID),
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		t.Fatalf("create execution: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestExecutionIdempotency(t *testing.T) {
	client := httpClient()
	wfID, pcCleanup := createAndActivateWorkflow(t, "wf-exec-idempotency")
	defer pcCleanup()

	execPayload := map[string]any{
		"inputData": map[string]any{"cpf": "123"},
	}
	body, _ := json.Marshal(execPayload)
	idempotencyKey := "idem-test-123" //nolint:gosec // not a credential, test idempotency key

	// First call
	req, _ := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), wfID),
		bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("first execution: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		bodyBytes := make([]byte, 1024)
		n, _ := resp.Body.Read(bodyBytes)
		t.Fatalf("first execution status = %d, body = %s", resp.StatusCode, string(bodyBytes[:n]))
	}

	var firstResp executionCreateResp
	json.NewDecoder(resp.Body).Decode(&firstResp)
	resp.Body.Close()

	// Second call with same idempotency key
	body, _ = json.Marshal(execPayload)
	req, _ = http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), wfID),
		bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("second execution: %v", err)
	}

	// Should return 200 (idempotent) with the same execution ID
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes := make([]byte, 1024)
		n, _ := resp.Body.Read(bodyBytes)
		t.Fatalf("second execution status = %d, body = %s", resp.StatusCode, string(bodyBytes[:n]))
	}

	var secondResp executionCreateResp
	json.NewDecoder(resp.Body).Decode(&secondResp)
	resp.Body.Close()

	if firstResp.ExecutionID != secondResp.ExecutionID {
		t.Fatalf("idempotent call should return same execution ID: %s != %s", firstResp.ExecutionID, secondResp.ExecutionID)
	}
}
