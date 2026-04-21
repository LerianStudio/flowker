// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

// =============================================================================
// Workflow Execution Integration Test — Full Flow
// =============================================================================
//
// This file tests the FULL workflow execution engine against real httptest
// mock servers, validating that the entire pipeline works:
//
//   Trigger → Executor A (KYC) → Conditional (risk score) → Executor B (AML) → Action (set_output)
//                                      ↓ (false branch)
//                                   Action (set_output: rejected)
//
// ─── MOCK SERVER ENDPOINTS ───────────────────────────────────────────────────
//
//   POST /v1/kyc/validate
//     Receives: {"document": "12345678900", "fullName": "JOHN DOE"}
//       (CPF with dots/dash removed via remove_characters, name uppercased via to_uppercase)
//     Returns:  {"customerId": "CUST-001", "riskScore": 25, "status": "approved"}
//
//   POST /v1/aml/check
//     Receives: {"customerId": "CUST-001", "transactionAmount": 1500.50}
//       (customerId mapped from KYC output, amount from workflow input)
//     Returns:  {"amlStatus": "cleared", "referenceId": "AML-REF-9876"}
//
//   POST /v1/kyc/validate  (high-risk variant)
//     Receives: {"document": "99999999999", "fullName": "RISKY PERSON"}
//     Returns:  {"customerId": "CUST-999", "riskScore": 85, "status": "review"}
//
//   POST /v1/aml/check  (failure variant)
//     Returns:  HTTP 500  (triggers executor failure scenario)
//
// ─── WORKFLOW GRAPH ──────────────────────────────────────────────────────────
//
//   [trigger-entry]
//       │
//       ▼
//   [kyc-executor]          executor node, calls POST /v1/kyc/validate
//       │                   inputMapping:  workflow.customer.cpf     → document  (remove_characters: ".-")
//       │                                  workflow.customer.name    → fullName  (to_uppercase)
//       │                   outputMapping: customerId                → result.customerId
//       │                                  riskScore                 → result.riskScore
//       ▼
//   [risk-check]            conditional node
//       │                   condition: "kyc-executor.result.riskScore < 50"
//       ├── true ──▶ [aml-executor]   executor node, calls POST /v1/aml/check
//       │                   inputMapping:  kyc-executor.result.customerId   → customerId
//       │                                  workflow.transaction.amount      → transactionAmount
//       │                   outputMapping: amlStatus                        → result.amlStatus
//       │                                  referenceId                      → result.referenceId
//       │                   │
//       │                   ▼
//       │           [approve-action]   action node: set_output {decision: "approved"}
//       │
//       └── false ──▶ [reject-action]  action node: set_output {decision: "rejected", reason: "high risk score"}
//
// ─── WORKFLOW CONTEXT (wfCtx) EVOLUTION (low-risk scenario) ──────────────────
//
//   After trigger:
//     wfCtx["workflow"] = {customer: {cpf: "123.456.789-00", name: "John Doe"}, transaction: {amount: 1500.50}}
//
//   After KYC executor (input transformed, output mapped):
//     wfCtx["kyc-executor"] = {result: {customerId: "CUST-001", riskScore: 25}}
//
//   After risk-check conditional (riskScore 25 < 50 → true):
//     step output = {condition: "kyc-executor.result.riskScore < 50", result: true, branchTaken: "true"}
//
//   After AML executor (input mapped from KYC output + workflow input):
//     wfCtx["aml-executor"] = {result: {amlStatus: "cleared", referenceId: "AML-REF-9876"}}
//
//   After approve-action:
//     wfCtx["output"] = {decision: "approved"}
//
//   Final output: {decision: "approved"}
//
// ─── TEST SCENARIOS ──────────────────────────────────────────────────────────
//
//   1. Happy path (low risk):   CPF 123.456.789-00, score=25 → approved
//   2. Conditional false path:  CPF 999.999.999-99, score=85 → rejected
//   3. Executor failure:        AML returns 500 → execution failed
//   4. Input transformation:    Verify remove_characters and to_uppercase applied
//   5. Data flow between nodes: Verify KYC output used as AML input
//   6. Step details:            Verify step count, names, statuses in results
//
// =============================================================================

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Response types for results endpoint (extended) ─────────────────────────

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

// requestRecord captures what the mock server received.
type requestRecord struct {
	Method string
	Path   string
	Body   map[string]any
}

// mockRecorder thread-safe request recorder.
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

// clear resets the recorder. Call this after executor config lifecycle setup
// to exclude connectivity/e2e test calls from execution assertions.
func (r *mockRecorder) clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests = nil
}

// ─── Mock server factory ────────────────────────────────────────────────────

// newMockExecutorServer creates an httptest server that simulates KYC and AML providers.
// The failAML flag makes the AML endpoint return HTTP 500 to test failure scenarios.
func newMockExecutorServer(t *testing.T, recorder *mockRecorder, failAML bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body
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
		case "/v1/kyc/validate":
			// Respond based on the document received
			document, _ := body["document"].(string)

			switch document {
			case "99999999999": // High-risk customer
				json.NewEncoder(w).Encode(map[string]any{
					"customerId": "CUST-999",
					"riskScore":  85,
					"status":     "review",
				})
			default: // Normal customer
				json.NewEncoder(w).Encode(map[string]any{
					"customerId": "CUST-001",
					"riskScore":  25,
					"status":     "approved",
				})
			}

		case "/v1/aml/check":
			if failAML {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]any{
					"error": "internal server error",
				})
				return
			}

			json.NewEncoder(w).Encode(map[string]any{
				"amlStatus":   "cleared",
				"referenceId": "AML-REF-9876",
			})

		case "/health":
			json.NewEncoder(w).Encode(map[string]any{"status": "healthy"})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "not found"})
		}
	}))
}

// ─── Test helpers ───────────────────────────────────────────────────────────

// setupExecutorConfig seeds an executor config directly via MongoDB with "active" status,
// bypassing deprecated HTTP routes (POST /v1/executors, lifecycle transitions).
func setupExecutorConfig(t *testing.T, _ *http.Client, name, mockURL string, endpoints []map[string]any) string {
	t.Helper()

	return seedExecutorConfig(t, name, mockURL, endpoints)
}

// createWorkflowForFlow creates a workflow with the given payload and activates it.
func createWorkflowForFlow(t *testing.T, client *http.Client, payload map[string]any) string {
	t.Helper()

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

	// Activate
	resp, err = client.Post(
		fmt.Sprintf("%s/v1/workflows/%s/activate", baseURL(), cResp.WorkflowID),
		"application/json", nil,
	)
	require.NoError(t, err, "activate workflow")
	require.Equal(t, http.StatusOK, resp.StatusCode, "activate should succeed")
	resp.Body.Close()

	return cResp.WorkflowID
}

// executeWorkflow triggers a workflow execution and returns the executionID.
func executeWorkflow(t *testing.T, client *http.Client, wfID string, inputData map[string]any) string {
	t.Helper()

	payload := map[string]any{"inputData": inputData}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost,
		fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), wfID),
		bytes.NewBuffer(body),
	)
	require.NoError(t, err, "build execute request")

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", uuid.NewString())

	resp, err := client.Do(req)
	require.NoError(t, err, "execute workflow")

	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("execute workflow: status=%d body=%s", resp.StatusCode, string(bodyBytes))
	}

	var execResp executionCreateResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&execResp))
	resp.Body.Close()

	require.NotEmpty(t, execResp.ExecutionID)
	assert.Equal(t, "running", execResp.Status)

	return execResp.ExecutionID
}

// pollExecutionStatus polls until execution reaches a terminal status or timeout.
func pollExecutionStatus(t *testing.T, client *http.Client, execID string, timeout time.Duration) executionStatusResp {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var status executionStatusResp

	for time.Now().Before(deadline) {
		resp, err := client.Get(fmt.Sprintf("%s/v1/executions/%s", baseURL(), execID))
		require.NoError(t, err, "poll execution status")
		require.Equal(t, http.StatusOK, resp.StatusCode)

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&status))
		resp.Body.Close()

		if status.Status == "completed" || status.Status == "failed" {
			return status
		}

		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("execution %s did not reach terminal status within %s (last status: %s)", execID, timeout, status.Status)

	return status
}

// getExecutionResults fetches the full results of a completed/failed execution.
func getExecutionResults(t *testing.T, client *http.Client, execID string) flowExecutionResultsResp {
	t.Helper()

	resp, err := client.Get(fmt.Sprintf("%s/v1/executions/%s/results", baseURL(), execID))
	require.NoError(t, err, "get execution results")
	require.Equal(t, http.StatusOK, resp.StatusCode, "results should be available")

	var results flowExecutionResultsResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&results))
	resp.Body.Close()

	return results
}

// buildFullFlowWorkflowPayload creates the multi-step workflow payload.
// kycProviderConfigID and amlProviderConfigID are the IDs of the provider configurations.
func buildFullFlowWorkflowPayload(name, kycProviderConfigID, amlProviderConfigID string) map[string]any {
	return map[string]any{
		"name": name,
		"nodes": []map[string]any{
			// ── Trigger node ──
			{
				"id":       "trigger-entry",
				"type":     "trigger",
				"data":     map[string]any{"triggerId": "webhook"},
				"position": map[string]any{"x": 0, "y": 0},
			},
			// ── KYC Executor node ──
			// Calls POST /v1/kyc/validate with transformed input
			{
				"id":   "kyc-executor",
				"type": "executor",
				"name": "KYC Validation",
				"data": map[string]any{
					"executorId":       "tracer.validate-transaction",
					"providerConfigId": kycProviderConfigID,
					"endpointName":     "validate",
					"inputMapping": []map[string]any{
						{
							"source":   "workflow.customer.cpf",
							"target":   "document",
							"required": true,
							"transformation": map[string]any{
								"type":   "remove_characters",
								"config": map[string]any{"characters": ".-"},
							},
						},
						{
							"source": "workflow.customer.name",
							"target": "fullName",
							"transformation": map[string]any{
								"type":   "to_uppercase",
								"config": map[string]any{},
							},
						},
					},
					"outputMapping": []map[string]any{
						{
							"source": "body.customerId",
							"target": "result.customerId",
						},
						{
							"source": "body.riskScore",
							"target": "result.riskScore",
						},
					},
				},
				"position": map[string]any{"x": 200, "y": 0},
			},
			// ── Risk Check conditional node ──
			// Evaluates: KYC riskScore < 50
			{
				"id":   "risk-check",
				"type": "conditional",
				"name": "Risk Assessment",
				"data": map[string]any{
					"condition": "kyc-executor.result.riskScore < 50",
				},
				"position": map[string]any{"x": 400, "y": 0},
			},
			// ── AML Executor node (true branch) ──
			// Calls POST /v1/aml/check with data from KYC output + workflow input
			{
				"id":   "aml-executor",
				"type": "executor",
				"name": "AML Check",
				"data": map[string]any{
					"executorId":       "tracer.validate-transaction",
					"providerConfigId": amlProviderConfigID,
					"endpointName":     "check",
					"inputMapping": []map[string]any{
						{
							"source":   "kyc-executor.result.customerId",
							"target":   "customerId",
							"required": true,
						},
						{
							"source":   "workflow.transaction.amount",
							"target":   "transactionAmount",
							"required": true,
						},
					},
					"outputMapping": []map[string]any{
						{
							"source": "body.amlStatus",
							"target": "result.amlStatus",
						},
						{
							"source": "body.referenceId",
							"target": "result.referenceId",
						},
					},
				},
				"position": map[string]any{"x": 600, "y": -100},
			},
			// ── Approve action (after AML) ──
			{
				"id":   "approve-action",
				"type": "action",
				"name": "Approve Transaction",
				"data": map[string]any{
					"actionType": "set_output",
					"output": map[string]any{
						"decision": "approved",
					},
				},
				"position": map[string]any{"x": 800, "y": -100},
			},
			// ── Reject action (false branch from conditional) ──
			{
				"id":   "reject-action",
				"type": "action",
				"name": "Reject Transaction",
				"data": map[string]any{
					"actionType": "set_output",
					"output": map[string]any{
						"decision": "rejected",
						"reason":   "high risk score",
					},
				},
				"position": map[string]any{"x": 600, "y": 100},
			},
		},
		"edges": []map[string]any{
			// trigger → KYC executor
			{"id": "e1", "source": "trigger-entry", "target": "kyc-executor"},
			// KYC executor → risk-check conditional
			{"id": "e2", "source": "kyc-executor", "target": "risk-check"},
			// risk-check TRUE → AML executor
			{"id": "e3", "source": "risk-check", "target": "aml-executor", "sourceHandle": "true"},
			// risk-check FALSE → reject action
			{"id": "e4", "source": "risk-check", "target": "reject-action", "sourceHandle": "false"},
			// AML executor → approve action
			{"id": "e5", "source": "aml-executor", "target": "approve-action"},
		},
	}
}

// =============================================================================
// TEST CASES
// =============================================================================

func TestExecutionFlow_WorkflowExecution_HappyPath_LowRisk(t *testing.T) {
	// ─── Scenario ────────────────────────────────────────────────────────
	// Customer: CPF 123.456.789-00, name "John Doe", transaction $1500.50
	// KYC returns riskScore=25 → conditional true → AML clears → approved
	//
	// Expected steps: KYC executor → Risk Assessment → AML executor → Approve
	// Expected final output: {decision: "approved"}
	// ─────────────────────────────────────────────────────────────────────
	client := httpClient()
	recorder := &mockRecorder{}

	// 1. Start mock executor server
	mockServer := newMockExecutorServer(t, recorder, false)
	defer mockServer.Close()

	// 2. Create provider configurations
	kycPCID := seedProviderConfig(t, "flow-kyc-low", "tracer", map[string]any{"base_url": mockServer.URL + "/v1/kyc", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, kycPCID)

	amlPCID := seedProviderConfig(t, "flow-aml-low", "tracer", map[string]any{"base_url": mockServer.URL + "/v1/aml", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, amlPCID)

	// 3. Create and activate workflow
	wfPayload := buildFullFlowWorkflowPayload("flow-wf-happy-low-risk", kycPCID, amlPCID)
	wfID := createWorkflowForFlow(t, client, wfPayload)

	// 4. Execute workflow
	// Clear recorder to exclude lifecycle calls (connectivity + e2e tests)
	recorder.clear()

	inputData := map[string]any{
		"customer": map[string]any{
			"cpf":  "123.456.789-00",
			"name": "John Doe",
		},
		"transaction": map[string]any{
			"amount": 1500.50,
		},
	}
	execID := executeWorkflow(t, client, wfID, inputData)

	// 5. Wait for completion
	status := pollExecutionStatus(t, client, execID, 30*time.Second)
	assert.Equal(t, "completed", status.Status, "execution should complete successfully")
	assert.Nil(t, status.ErrorMessage, "no error message expected")
	assert.Equal(t, 5, status.TotalSteps, "should have 5 executable nodes (2 executors + 1 conditional + 2 actions)")

	// 6. Get full results
	results := getExecutionResults(t, client, execID)
	assert.Equal(t, "completed", results.Status)
	assert.NotNil(t, results.CompletedAt, "completedAt should be set")

	// ── Verify final output ──
	require.NotNil(t, results.FinalOutput, "finalOutput should be present")
	assert.Equal(t, "approved", results.FinalOutput["decision"], "decision should be approved")

	// ── Verify step results ──
	require.Len(t, results.StepResults, 4, "should have 4 step results")

	// Step 1: KYC Executor
	kycStep := results.StepResults[0]
	assert.Equal(t, 1, kycStep.StepNumber)
	assert.Equal(t, "KYC Validation", kycStep.StepName)
	assert.Equal(t, "kyc-executor", kycStep.NodeID)
	assert.Equal(t, "completed", kycStep.Status)
	assert.Nil(t, kycStep.ErrorMessage)

	// Step 2: Risk Assessment (conditional)
	riskStep := results.StepResults[1]
	assert.Equal(t, 2, riskStep.StepNumber)
	assert.Equal(t, "Risk Assessment", riskStep.StepName)
	assert.Equal(t, "risk-check", riskStep.NodeID)
	assert.Equal(t, "completed", riskStep.Status)
	// Verify conditional output
	if riskStep.Output != nil {
		assert.Equal(t, true, riskStep.Output["result"], "condition should evaluate to true")
		assert.Equal(t, "true", riskStep.Output["branchTaken"], "should take true branch")
	}

	// Step 3: AML Executor
	amlStep := results.StepResults[2]
	assert.Equal(t, 3, amlStep.StepNumber)
	assert.Equal(t, "AML Check", amlStep.StepName)
	assert.Equal(t, "aml-executor", amlStep.NodeID)
	assert.Equal(t, "completed", amlStep.Status)

	// Step 4: Approve Action
	approveStep := results.StepResults[3]
	assert.Equal(t, 4, approveStep.StepNumber)
	assert.Equal(t, "Approve Transaction", approveStep.StepName)
	assert.Equal(t, "approve-action", approveStep.NodeID)
	assert.Equal(t, "completed", approveStep.Status)

	// ── Verify input transformations were applied correctly ──
	kycRequests := recorder.getByPath("/v1/kyc/validate")
	require.Len(t, kycRequests, 1, "KYC endpoint should be called exactly once")
	kycBody := kycRequests[0].Body

	// remove_characters should strip "." and "-" from CPF
	assert.Equal(t, "12345678900", kycBody["document"],
		"CPF should have dots and dash removed: 123.456.789-00 → 12345678900")

	// to_uppercase should uppercase the name
	assert.Equal(t, "JOHN DOE", kycBody["fullName"],
		"Name should be uppercased: John Doe → JOHN DOE")

	// ── Verify data flow between executors ──
	amlRequests := recorder.getByPath("/v1/aml/check")
	require.Len(t, amlRequests, 1, "AML endpoint should be called exactly once")
	amlBody := amlRequests[0].Body

	// customerId from KYC output should flow to AML input
	assert.Equal(t, "CUST-001", amlBody["customerId"],
		"AML should receive customerId from KYC output")

	// transaction amount from workflow input should flow to AML input
	assert.Equal(t, 1500.50, amlBody["transactionAmount"],
		"AML should receive transactionAmount from workflow input")
}

func TestExecutionFlow_WorkflowExecution_ConditionalFalsePath_HighRisk(t *testing.T) {
	// ─── Scenario ────────────────────────────────────────────────────────
	// Customer: CPF 999.999.999-99, name "Risky Person", transaction $50000
	// KYC returns riskScore=85 → conditional false → rejected
	//
	// Expected steps: KYC executor → Risk Assessment → Reject action
	// Expected final output: {decision: "rejected", reason: "high risk score"}
	// AML endpoint should NOT be called.
	// ─────────────────────────────────────────────────────────────────────
	client := httpClient()
	recorder := &mockRecorder{}

	mockServer := newMockExecutorServer(t, recorder, false)
	defer mockServer.Close()

	kycPCID := seedProviderConfig(t, "flow-kyc-high", "tracer", map[string]any{"base_url": mockServer.URL + "/v1/kyc", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, kycPCID)

	amlPCID := seedProviderConfig(t, "flow-aml-high", "tracer", map[string]any{"base_url": mockServer.URL + "/v1/aml", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, amlPCID)

	wfPayload := buildFullFlowWorkflowPayload("flow-wf-high-risk", kycPCID, amlPCID)
	wfID := createWorkflowForFlow(t, client, wfPayload)

	// Clear recorder to exclude lifecycle calls (connectivity + e2e tests)
	recorder.clear()

	inputData := map[string]any{
		"customer": map[string]any{
			"cpf":  "999.999.999-99",
			"name": "Risky Person",
		},
		"transaction": map[string]any{
			"amount": 50000,
		},
	}
	execID := executeWorkflow(t, client, wfID, inputData)

	status := pollExecutionStatus(t, client, execID, 30*time.Second)
	assert.Equal(t, "completed", status.Status, "execution should complete (reject is not a failure)")
	assert.Nil(t, status.ErrorMessage)

	results := getExecutionResults(t, client, execID)
	assert.Equal(t, "completed", results.Status)

	// ── Verify final output is rejection ──
	require.NotNil(t, results.FinalOutput)
	assert.Equal(t, "rejected", results.FinalOutput["decision"])
	assert.Equal(t, "high risk score", results.FinalOutput["reason"])

	// ── Verify only 3 steps executed (KYC, conditional, reject action) ──
	// AML executor should NOT run because conditional took false branch
	require.Len(t, results.StepResults, 3, "should have 3 steps (no AML)")

	// Step 1: KYC
	assert.Equal(t, "kyc-executor", results.StepResults[0].NodeID)
	assert.Equal(t, "completed", results.StepResults[0].Status)

	// Step 2: Risk Assessment
	riskStep := results.StepResults[1]
	assert.Equal(t, "risk-check", riskStep.NodeID)
	assert.Equal(t, "completed", riskStep.Status)
	if riskStep.Output != nil {
		assert.Equal(t, false, riskStep.Output["result"], "condition should evaluate to false (85 >= 50)")
		assert.Equal(t, "false", riskStep.Output["branchTaken"])
	}

	// Step 3: Reject Action
	assert.Equal(t, "reject-action", results.StepResults[2].NodeID)
	assert.Equal(t, "completed", results.StepResults[2].Status)

	// ── Verify AML was NOT called ──
	amlRequests := recorder.getByPath("/v1/aml/check")
	assert.Len(t, amlRequests, 0, "AML endpoint should NOT be called for high-risk path")

	// ── Verify KYC received correct transformed input ──
	kycRequests := recorder.getByPath("/v1/kyc/validate")
	require.Len(t, kycRequests, 1)
	assert.Equal(t, "99999999999", kycRequests[0].Body["document"],
		"CPF should have dots and dash removed: 999.999.999-99 → 99999999999")
	assert.Equal(t, "RISKY PERSON", kycRequests[0].Body["fullName"],
		"Name should be uppercased: Risky Person → RISKY PERSON")
}

func TestExecutionFlow_WorkflowExecution_ExecutorFailure(t *testing.T) {
	// ─── Scenario ────────────────────────────────────────────────────────
	// Customer: CPF 123.456.789-00, name "John Doe"
	// KYC succeeds with riskScore=25 → conditional true → AML returns 500
	// Execution should be marked as FAILED after retries exhausted.
	//
	// Note: The engine retries up to 5 times with exponential backoff (1s,2s,4s,8s,16s).
	// This test uses a 120s timeout to accommodate the retry delay.
	// ─────────────────────────────────────────────────────────────────────
	client := httpClient()
	recorder := &mockRecorder{}

	// Use toggleable mock: AML succeeds during lifecycle, fails during execution
	var failAML atomic.Bool // starts false (lifecycle succeeds)

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
		case "/v1/kyc/validate":
			json.NewEncoder(w).Encode(map[string]any{
				"customerId": "CUST-001",
				"riskScore":  25,
				"status":     "approved",
			})
		case "/v1/aml/check":
			if failAML.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]any{"error": "internal server error"})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"amlStatus":   "cleared",
				"referenceId": "AML-REF-9876",
			})
		case "/health":
			json.NewEncoder(w).Encode(map[string]any{"status": "healthy"})
		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "not found"})
		}
	}))
	defer mockServer.Close()

	kycPCID := seedProviderConfig(t, "flow-kyc-fail", "tracer", map[string]any{"base_url": mockServer.URL + "/v1/kyc", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, kycPCID)

	amlPCID := seedProviderConfig(t, "flow-aml-fail", "tracer", map[string]any{"base_url": mockServer.URL + "/v1/aml", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, amlPCID)

	// Enable AML failure mode after lifecycle setup
	failAML.Store(true)

	// Clear recorder to exclude lifecycle calls (connectivity + e2e tests)
	recorder.clear()

	wfPayload := buildFullFlowWorkflowPayload("flow-wf-exec-fail", kycPCID, amlPCID)
	wfID := createWorkflowForFlow(t, client, wfPayload)

	inputData := map[string]any{
		"customer": map[string]any{
			"cpf":  "123.456.789-00",
			"name": "John Doe",
		},
		"transaction": map[string]any{
			"amount": 1500.50,
		},
	}
	execID := executeWorkflow(t, client, wfID, inputData)

	// Longer timeout because of retries (5 attempts with exponential backoff: 1+2+4+8+16 = 31s)
	status := pollExecutionStatus(t, client, execID, 120*time.Second)
	assert.Equal(t, "failed", status.Status, "execution should fail when AML returns 500")
	require.NotNil(t, status.ErrorMessage, "error message should be present")
	assert.Contains(t, *status.ErrorMessage, "executor call failed",
		"error should mention executor call failure")

	results := getExecutionResults(t, client, execID)
	assert.Equal(t, "failed", results.Status)

	// ── KYC and conditional should succeed, AML step should be failed ──
	require.GreaterOrEqual(t, len(results.StepResults), 3, "should have at least KYC, conditional, and AML steps")

	// Find the AML step
	var amlStep *flowStepResultResp
	for i := range results.StepResults {
		if results.StepResults[i].NodeID == "aml-executor" {
			amlStep = &results.StepResults[i]
			break
		}
	}

	require.NotNil(t, amlStep, "AML step should exist in results")
	assert.Equal(t, "failed", amlStep.Status, "AML step should be failed")
	require.NotNil(t, amlStep.ErrorMessage, "AML step error message should be present")

	// ── Verify AML was retried multiple times ──
	amlRequests := recorder.getByPath("/v1/aml/check")
	assert.GreaterOrEqual(t, len(amlRequests), 2, "AML should have been retried at least twice")
	assert.LessOrEqual(t, len(amlRequests), 5, "AML should not exceed 5 retry attempts")
}

func TestExecutionFlow_WorkflowExecution_SimpleLinearFlow(t *testing.T) {
	// ─── Scenario ────────────────────────────────────────────────────────
	// Simple workflow: Trigger → Single executor (no conditionals)
	// Tests the most basic execution flow without branching.
	//
	// Workflow: trigger → executor (KYC validate)
	// ─────────────────────────────────────────────────────────────────────
	client := httpClient()
	recorder := &mockRecorder{}

	mockServer := newMockExecutorServer(t, recorder, false)
	defer mockServer.Close()

	pcID := seedProviderConfig(t, "flow-simple-exec", "tracer", map[string]any{"base_url": mockServer.URL + "/v1/kyc", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, pcID)

	// Simple linear workflow: trigger → executor
	wfPayload := map[string]any{
		"name": "flow-simple-linear",
		"nodes": []map[string]any{
			{
				"id":       "trigger-1",
				"type":     "trigger",
				"data":     map[string]any{"triggerId": "webhook"},
				"position": map[string]any{"x": 0, "y": 0},
			},
			{
				"id":   "exec-1",
				"type": "executor",
				"name": "Simple Validation",
				"data": map[string]any{
					"executorId":       "tracer.validate-transaction",
					"providerConfigId": pcID,
					"endpointName":     "validate",
					"inputMapping": []map[string]any{
						{
							"source": "workflow.cpf",
							"target": "document",
						},
					},
				},
				"position": map[string]any{"x": 200, "y": 0},
			},
		},
		"edges": []map[string]any{
			{"id": "e1", "source": "trigger-1", "target": "exec-1"},
		},
	}

	wfID := createWorkflowForFlow(t, client, wfPayload)

	// Clear recorder to exclude lifecycle calls (connectivity + e2e tests)
	recorder.clear()

	inputData := map[string]any{
		"cpf": "00011122233",
	}
	execID := executeWorkflow(t, client, wfID, inputData)

	status := pollExecutionStatus(t, client, execID, 30*time.Second)
	assert.Equal(t, "completed", status.Status)
	assert.Equal(t, 1, status.TotalSteps, "only 1 executable node")

	results := getExecutionResults(t, client, execID)
	require.Len(t, results.StepResults, 1)
	assert.Equal(t, "completed", results.StepResults[0].Status)
	assert.Equal(t, "exec-1", results.StepResults[0].NodeID)

	// Verify executor received the mapped input
	kycRequests := recorder.getByPath("/v1/kyc/validate")
	require.Len(t, kycRequests, 1)
	assert.Equal(t, "00011122233", kycRequests[0].Body["document"],
		"document should be mapped from workflow.cpf")
}

func TestExecutionFlow_WorkflowExecution_OutputMappingVerification(t *testing.T) {
	// ─── Scenario ────────────────────────────────────────────────────────
	// Tests that output mappings correctly transform executor response.
	// KYC returns: {customerId: "CUST-001", riskScore: 25, status: "approved"}
	// Output mapping maps: customerId → result.id, riskScore → result.score
	// The mapped output should appear in step results and be accessible by
	// downstream nodes.
	// ─────────────────────────────────────────────────────────────────────
	client := httpClient()
	recorder := &mockRecorder{}

	mockServer := newMockExecutorServer(t, recorder, false)
	defer mockServer.Close()

	pcID := seedProviderConfig(t, "flow-output-map", "tracer", map[string]any{"base_url": mockServer.URL + "/v1/kyc", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, pcID)

	wfPayload := map[string]any{
		"name": "flow-output-mapping",
		"nodes": []map[string]any{
			{
				"id":       "trigger",
				"type":     "trigger",
				"data":     map[string]any{"triggerId": "webhook"},
				"position": map[string]any{"x": 0, "y": 0},
			},
			{
				"id":   "kyc-node",
				"type": "executor",
				"name": "KYC with Output Mapping",
				"data": map[string]any{
					"executorId":       "tracer.validate-transaction",
					"providerConfigId": pcID,
					"endpointName":     "validate",
					"inputMapping": []map[string]any{
						{"source": "workflow.cpf", "target": "document"},
					},
					"outputMapping": []map[string]any{
						{"source": "body.customerId", "target": "result.id"},
						{"source": "body.riskScore", "target": "result.score"},
					},
				},
				"position": map[string]any{"x": 200, "y": 0},
			},
		},
		"edges": []map[string]any{
			{"id": "e1", "source": "trigger", "target": "kyc-node"},
		},
	}

	wfID := createWorkflowForFlow(t, client, wfPayload)

	execID := executeWorkflow(t, client, wfID, map[string]any{"cpf": "12345678900"})

	status := pollExecutionStatus(t, client, execID, 30*time.Second)
	assert.Equal(t, "completed", status.Status)

	results := getExecutionResults(t, client, execID)
	require.Len(t, results.StepResults, 1)

	kycOutput := results.StepResults[0].Output
	require.NotNil(t, kycOutput, "step output should be present")

	// Verify output mapping applied: raw response fields mapped to result.id and result.score
	if resultObj, ok := kycOutput["result"].(map[string]any); ok {
		assert.Equal(t, "CUST-001", resultObj["id"], "customerId should be mapped to result.id")
		// riskScore comes as float64 from JSON
		assert.Equal(t, float64(25), resultObj["score"], "riskScore should be mapped to result.score")
	} else {
		t.Errorf("expected output.result to be a map, got: %v", kycOutput)
	}
}

func TestExecutionFlow_WorkflowExecution_MultipleTransformations(t *testing.T) {
	// ─── Scenario ────────────────────────────────────────────────────────
	// Tests that multiple transformations are applied correctly in sequence.
	// Input: cpf="123.456.789-00", email="Test@Example.COM"
	// Transforms: cpf → remove_characters(".-"), email → to_lowercase
	// Expected: cpf="12345678900", email="test@example.com"
	// ─────────────────────────────────────────────────────────────────────
	client := httpClient()
	recorder := &mockRecorder{}

	mockServer := newMockExecutorServer(t, recorder, false)
	defer mockServer.Close()

	pcID := seedProviderConfig(t, "flow-multi-transform", "tracer", map[string]any{"base_url": mockServer.URL + "/v1/kyc", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, pcID)

	wfPayload := map[string]any{
		"name": "flow-multi-transforms",
		"nodes": []map[string]any{
			{
				"id":       "trigger",
				"type":     "trigger",
				"data":     map[string]any{"triggerId": "webhook"},
				"position": map[string]any{"x": 0, "y": 0},
			},
			{
				"id":   "exec-multi",
				"type": "executor",
				"name": "Multi-Transform Executor",
				"data": map[string]any{
					"executorId":       "tracer.validate-transaction",
					"providerConfigId": pcID,
					"endpointName":     "validate",
					"inputMapping": []map[string]any{
						{
							"source": "workflow.cpf",
							"target": "document",
							"transformation": map[string]any{
								"type":   "remove_characters",
								"config": map[string]any{"characters": ".-"},
							},
						},
						{
							"source": "workflow.email",
							"target": "emailAddress",
							"transformation": map[string]any{
								"type":   "to_lowercase",
								"config": map[string]any{},
							},
						},
						{
							"source": "workflow.name",
							"target": "fullName",
							"transformation": map[string]any{
								"type":   "to_uppercase",
								"config": map[string]any{},
							},
						},
					},
				},
				"position": map[string]any{"x": 200, "y": 0},
			},
		},
		"edges": []map[string]any{
			{"id": "e1", "source": "trigger", "target": "exec-multi"},
		},
	}

	wfID := createWorkflowForFlow(t, client, wfPayload)

	// Clear recorder to exclude lifecycle calls (connectivity + e2e tests)
	recorder.clear()

	inputData := map[string]any{
		"cpf":   "123.456.789-00",
		"email": "Test@Example.COM",
		"name":  "Maria Silva",
	}
	execID := executeWorkflow(t, client, wfID, inputData)

	status := pollExecutionStatus(t, client, execID, 30*time.Second)
	assert.Equal(t, "completed", status.Status)

	// Verify all 3 transformations applied
	kycRequests := recorder.getByPath("/v1/kyc/validate")
	require.Len(t, kycRequests, 1)
	body := kycRequests[0].Body

	assert.Equal(t, "12345678900", body["document"],
		"remove_characters should strip '.' and '-'")
	assert.Equal(t, "test@example.com", body["emailAddress"],
		"to_lowercase should lowercase email")
	assert.Equal(t, "MARIA SILVA", body["fullName"],
		"to_uppercase should uppercase name")
}

func TestExecutionFlow_WorkflowExecution_ResultsNotReady(t *testing.T) {
	// ─── Scenario ────────────────────────────────────────────────────────
	// Try to fetch results immediately after creating an execution
	// (before it completes). Should return 422 Unprocessable Entity.
	// ─────────────────────────────────────────────────────────────────────
	client := httpClient()
	recorder := &mockRecorder{}

	// Use a slow mock server to ensure execution is still running.
	// slowMode starts false so lifecycle tests (configure->test->activate) pass quickly.
	var slowMode atomic.Bool

	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err == nil {
				_ = json.Unmarshal(bodyBytes, &body)
			}
		}
		recorder.record(r.Method, r.URL.Path, body)

		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"status": "healthy"})
			return
		}

		// Only slow during execution, not during lifecycle setup
		if slowMode.Load() {
			time.Sleep(10 * time.Second)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"result": "ok"})
	}))
	defer slowServer.Close()

	pcID := seedProviderConfig(t, "flow-slow-exec", "tracer", map[string]any{"base_url": slowServer.URL + "/v1/kyc", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, pcID)

	wfPayload := map[string]any{
		"name": "flow-results-not-ready",
		"nodes": []map[string]any{
			{
				"id":       "trigger",
				"type":     "trigger",
				"data":     map[string]any{"triggerId": "webhook"},
				"position": map[string]any{"x": 0, "y": 0},
			},
			{
				"id":   "slow-exec",
				"type": "executor",
				"data": map[string]any{
					"executorId":       "tracer.validate-transaction",
					"providerConfigId": pcID,
					"endpointName":     "validate",
				},
				"position": map[string]any{"x": 200, "y": 0},
			},
		},
		"edges": []map[string]any{
			{"id": "e1", "source": "trigger", "target": "slow-exec"},
		},
	}

	wfID := createWorkflowForFlow(t, client, wfPayload)

	// Enable slow mode after lifecycle setup
	slowMode.Store(true)

	execID := executeWorkflow(t, client, wfID, map[string]any{"test": true})

	// Immediately try to get results (should fail with 422)
	time.Sleep(500 * time.Millisecond) // Small delay to ensure state machine started

	resp, err := client.Get(fmt.Sprintf("%s/v1/executions/%s/results", baseURL(), execID))
	require.NoError(t, err)

	// Either 422 (in progress) or 200 (if somehow already done) are acceptable
	if resp.StatusCode == http.StatusUnprocessableEntity {
		// Expected: execution still in progress
		t.Log("Correctly returned 422 for in-progress execution")
	} else if resp.StatusCode == http.StatusOK {
		// Execution completed very quickly (unlikely with slow server but possible)
		t.Log("Execution completed before we could check - still valid")
	} else {
		bodyBytes, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d, body: %s", resp.StatusCode, string(bodyBytes))
	}
	resp.Body.Close()
}
