// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ─── Server connection ──────────────────────────────────────────────────────

// baseURL returns the Flowker API base URL.
// Uses the serverAddr set by TestMain (auto-bootstrapped or from E2E_BASE_URL).
func baseURL() string {
	return fmt.Sprintf("http://%s", serverAddr)
}

// httpClient returns an HTTP client with a sensible timeout for E2E tests.
func httpClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// ─── Shared response types ──────────────────────────────────────────────────
//
// These mirror the JSON shapes returned by the Flowker API.
// They are duplicated here (rather than imported) because the e2e package
// is intentionally decoupled from the integration test package.

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

type createWorkflowResp struct {
	WorkflowID string `json:"workflowId"`
	Status     string `json:"status"`
	Version    string `json:"version"`
}

type nodeOutput struct {
	ID   string         `json:"id"`
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

type edgeOutput struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type workflowOutput struct {
	ID     string         `json:"id"`
	Name   string         `json:"name"`
	Status string         `json:"status"`
	Nodes  []nodeOutput   `json:"nodes"`
	Edges  []edgeOutput   `json:"edges"`
	Meta   map[string]any `json:"metadata"`
}

type listWorkflowsResp struct {
	Items []workflowOutput `json:"items"`
}

type executorSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category"`
	Version  string `json:"version"`
}

type executorDetail struct {
	executorSummary
	Schema string `json:"schema"`
}

type triggerSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type triggerDetail struct {
	triggerSummary
	Schema string `json:"schema"`
}

type validationResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error"`
}

// ─── Execution results ──────────────────────────────────────────────────────

type flowStepResultResp struct {
	StepNumber   int            `json:"stepNumber"`
	StepName     string         `json:"stepName"`
	NodeID       string         `json:"nodeId"`
	Status       string         `json:"status"`
	Output       map[string]any `json:"output,omitempty"`
	ErrorMessage *string        `json:"errorMessage,omitempty"`
	DurationMs   int64          `json:"durationMs"`
}

type flowExecutionResultsResp struct {
	ExecutionID string               `json:"executionId"`
	WorkflowID  string               `json:"workflowId"`
	Status      string               `json:"status"`
	StepResults []flowStepResultResp `json:"stepResults"`
	FinalOutput map[string]any       `json:"finalOutput,omitempty"`
	CompletedAt *string              `json:"completedAt,omitempty"`
}

// ─── Mock server recording ──────────────────────────────────────────────────

type requestRecord struct {
	Method string
	Path   string
	Body   map[string]any
}

type mockRecorder struct {
	mu       sync.Mutex
	requests []requestRecord
}

func (r *mockRecorder) record(method, path string, body map[string]any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests = append(r.requests, requestRecord{
		Method: method,
		Path:   path,
		Body:   body,
	})
}

func (r *mockRecorder) getByPath(path string) []requestRecord {
	r.mu.Lock()
	defer r.mu.Unlock()

	var result []requestRecord
	for _, req := range r.requests {
		if req.Path == path {
			result = append(result, req)
		}
	}

	return result
}

func (r *mockRecorder) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests = nil
}

// ─── Polling helpers ────────────────────────────────────────────────────────

func pollExecutionStatus(t *testing.T, client *http.Client, execID string, timeout time.Duration) executionStatusResp {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var status executionStatusResp

	for time.Now().Before(deadline) {
		resp, err := client.Get(fmt.Sprintf("%s/v1/executions/%s", baseURL(), execID))
		require.NoError(t, err, "poll execution status")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		err = json.NewDecoder(resp.Body).Decode(&status)
		resp.Body.Close()
		require.NoError(t, err, "decode execution status")

		if status.Status == "completed" || status.Status == "failed" {
			return status
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("execution %s did not reach terminal status within %s (last status: %s)", execID, timeout, status.Status)

	return status
}

func getExecutionResults(t *testing.T, client *http.Client, execID string) flowExecutionResultsResp {
	t.Helper()

	resp, err := client.Get(fmt.Sprintf("%s/v1/executions/%s/results", baseURL(), execID))
	require.NoError(t, err, "get execution results")
	require.Equal(t, http.StatusOK, resp.StatusCode, "results should be available")

	var results flowExecutionResultsResp
	err = json.NewDecoder(resp.Body).Decode(&results)
	resp.Body.Close()
	require.NoError(t, err, "decode execution results")

	return results
}
