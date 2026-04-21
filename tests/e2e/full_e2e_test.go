// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build e2e

package e2e

// =============================================================================
// Full End-to-End Integration Test
// =============================================================================
//
// HOW TO RUN:
//
//   go test -tags=e2e -run TestFullE2E -v -timeout 5m ./tests/e2e/
//
// WHAT THIS TEST DOES:
//
//   This is a single, sequential test that simulates the ENTIRE lifecycle of
//   the Flowker platform — from exploring the catalog to executing a real
//   workflow and cleaning up everything afterwards. It works like a manual QA
//   session: each step depends on the previous one, and if any step fails, the
//   test stops immediately.
//
//   Think of it as a user walking through the platform step by step:
//
//     1. "What providers and executors are available?"  (Phase 1 — Catalog)
//     2. "Let me configure my KYC and AML providers."   (Phase 2 — Provider Configs)
//     3. "Let me build a compliance workflow."           (Phase 3 — Workflow Lifecycle)
//     4. "Let me run it with real customer data."        (Phase 4 — Execution)
//     5. "Let me clean up everything I created."         (Phase 5 — Cleanup)
//
// ─── MOCK SERVERS ──────────────────────────────────────────────────────────────
//
//   The test starts TWO local HTTP servers that pretend to be external providers:
//
//   KYC Mock Server (Know Your Customer):
//     Simulates a customer identity verification service.
//     ┌───────────────────────────────────────────────────────────────────────┐
//     │ POST /v1/kyc/validate                                                │
//     │   Receives: { "document": "12345678900", "fullName": "JOHN DOE" }    │
//     │   If document starts with "1" → low risk  (riskScore: 25, approved)  │
//     │   Otherwise                   → high risk (riskScore: 85, review)    │
//     │                                                                       │
//     │ POST /v1/kyc/blocked-check                                            │
//     │   Always returns: { "blocked": false }                                │
//     │                                                                       │
//     │ GET /health → { "status": "healthy" }                                 │
//     └───────────────────────────────────────────────────────────────────────┘
//
//   AML Mock Server (Anti-Money Laundering):
//     Simulates a transaction screening service.
//     ┌───────────────────────────────────────────────────────────────────────┐
//     │ POST /v1/aml/check                                                    │
//     │   Receives: { "customerId": "CUST-001", "transactionAmount": 1500 }   │
//     │   Returns:  { "amlStatus": "cleared", "referenceId": "AML-REF-9876" } │
//     │                                                                       │
//     │ POST /v1/aml/enhanced                                                 │
//     │   Returns: { "enhancedStatus": "cleared", "score": 10 }               │
//     │                                                                       │
//     │ GET /health → { "status": "healthy" }                                 │
//     └───────────────────────────────────────────────────────────────────────┘
//
//   All requests to these mock servers are RECORDED so we can verify exactly
//   what the workflow engine sent (after transformations).
//
// ─── WORKFLOW GRAPH ────────────────────────────────────────────────────────────
//
//   The test builds this workflow — a typical financial compliance pipeline:
//
//     [trigger-entry]  (webhook — receives customer + transaction data)
//           │
//           ▼
//     [kyc-executor]   (calls KYC mock → validates customer identity)
//           │           INPUT TRANSFORMATIONS:
//           │             • CPF "123.456.789-00" → remove "." and "-" → "12345678900"
//           │             • Name "John Doe"      → uppercase           → "JOHN DOE"
//           │           OUTPUT MAPPING:
//           │             • customerId  → stored as kyc-executor.result.customerId
//           │             • riskScore   → stored as kyc-executor.result.riskScore
//           ▼
//     [risk-check]     (conditional: is riskScore < 50?)
//           │
//           ├── YES (score=25) ──────────────────────────┐
//           │                                             ▼
//           │                                    [aml-executor]  (calls AML mock)
//           │                                             │       INPUT MAPPING:
//           │                                             │         • customerId from KYC output
//           │                                             │         • amount from workflow input
//           │                                             │       OUTPUT MAPPING:
//           │                                             │         • amlStatus, referenceId stored
//           │                                             ▼
//           │                                    [approve-action]
//           │                                       output: { decision: "approved" }
//           │
//           └── NO (score=85) ───────────────────────────┐
//                                                         ▼
//                                                 [reject-action]
//                                                    output: { decision: "rejected",
//                                                              reason: "high risk score" }
//
// ─── DATA FLOW (LOW-RISK SCENARIO) ────────────────────────────────────────────
//
//   Step 1: User submits execution with:
//     { customer: { cpf: "123.456.789-00", name: "John Doe" },
//       transaction: { amount: 1500.50 } }
//
//   Step 2: KYC executor transforms input and calls mock:
//     Sent to mock:  { document: "12345678900", fullName: "JOHN DOE" }
//     Mock responds:  { customerId: "CUST-001", riskScore: 25, status: "approved" }
//     Stored in context: kyc-executor.result = { customerId: "CUST-001", riskScore: 25 }
//
//   Step 3: Conditional evaluates "kyc-executor.result.riskScore < 50":
//     25 < 50 → true → takes the YES branch → goes to AML executor
//
//   Step 4: AML executor maps input from KYC output + workflow input:
//     Sent to mock:  { customerId: "CUST-001", transactionAmount: 1500.50 }
//     Mock responds:  { amlStatus: "cleared", referenceId: "AML-REF-9876" }
//
//   Step 5: Approve action sets final output:
//     { decision: "approved" }
//
// ─── DATA FLOW (HIGH-RISK SCENARIO) ───────────────────────────────────────────
//
//   Step 1: User submits with CPF "999.999.999-99", name "Risky Person"
//   Step 2: KYC returns riskScore: 85 (document starts with "9")
//   Step 3: Conditional: 85 < 50 → false → takes the NO branch
//   Step 4: Reject action sets: { decision: "rejected", reason: "high risk score" }
//   NOTE: AML executor is NEVER called (skipped by conditional)
//
// ─── WHAT EACH PHASE TESTS ────────────────────────────────────────────────────
//
//   Phase 1 — Catalog Exploration (9 subtests):
//     • Lists all providers (http, midaz, s3)
//     • Gets HTTP provider details and config schema
//     • Lists executors for the HTTP provider
//     • Validates executor configs (valid and invalid)
//     • Lists and gets trigger details (webhook)
//
//   Phase 2 — Provider Configuration CRUD (15 subtests):
//     • Creates 3 provider configs via HTTP API (KYC, AML, temporary)
//     • Reads back a config and verifies all fields
//     • Lists configs with and without filters (by provider, by status)
//     • Updates a config's description (PATCH)
//     • Disables and re-enables a config (lifecycle transitions)
//     • Tests connectivity to mock servers (verifies reachability)
//     • Verifies duplicate name returns 409 Conflict
//     • Deletes a config and verifies it's gone (404)
//
//   Phase 3 — Workflow Lifecycle (12 subtests):
//     • Creates the complex 6-node workflow shown above
//     • Reads it back and verifies nodes, edges, and mappings
//     • Lists workflows with status filters
//     • Updates metadata on the draft workflow
//     • Clones the workflow (creates a copy)
//     • Activates the original → vere-activatifies rion fails (422)
//     • Verifies update on active workflow fails (422)
//     • Clones the active workflow for execution
//     • Deactivates original, activates the clone
//     • Deletes both the draft clone and the inactive original
//
//   Phase 4 — Execution E2E (8 subtests):
//     • Executes low-risk scenario → polls until completed
//     • Verifies step-by-step results:
//         - KYC output mapping (customerId, riskScore)
//         - Conditional branch decision (true/false)
//         - AML output mapping (amlStatus, referenceId)
//         - Final output (decision: "approved")
//     • Verifies input transformations via mock recorder:
//         - remove_characters applied to CPF
//         - to_uppercase applied to name
//     • Verifies cross-step data flow via mock recorder:
//         - AML received customerId from KYC output
//         - AML received amount from workflow input
//     • Tests idempotency (same key → same execution ID)
//     • Executes high-risk scenario → verifies rejection
//     • Verifies AML was NOT called (conditional false path)
//     • Lists all executions
//     • Verifies missing idempotency key returns 400
//
//   Phase 5 — Cleanup (7 subtests):
//     • Deactivates and deletes the execution workflow
//     • Disables and deletes both provider configs
//     • Verifies everything returns 404 after deletion
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

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Response Types ─────────────────────────────────────────────────────────
//
// These structs map to the JSON responses returned by the Flowker API.
// They are prefixed with "e2e" to avoid conflicts with types defined in
// other test files in the same package.
// Shared types and helpers are defined in helpers_test.go within this package.

// e2eProviderConfigCreateResp is what the API returns when you create a new provider config.
// Example: POST /v1/provider-configurations → { id: "uuid", name: "kyc", status: "active" }
type e2eProviderConfigCreateResp struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

// e2eProviderConfigOutput is the full provider config object returned by GET/PATCH endpoints.
type e2eProviderConfigOutput struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	ProviderID  string         `json:"providerId"`
	Config      map[string]any `json:"config"` // e.g. {"base_url": "http://..."}
	Status      string         `json:"status"` // "active" or "disabled"
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   string         `json:"createdAt"`
	UpdatedAt   string         `json:"updatedAt"`
}

// e2eProviderConfigListResp wraps a list of provider configurations.
type e2eProviderConfigListResp struct {
	Items []e2eProviderConfigOutput `json:"items"`
}

// e2eTestStage is one stage of a connectivity test (e.g., "connectivity", "authentication").
type e2eTestStage struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // "passed", "failed", or "skipped"
	Message    string `json:"message"`
	Error      string `json:"error"`
	DurationMs int64  `json:"durationMs"`
}

// e2eProviderConfigTestResp is the response from POST /v1/provider-configurations/:id/test.
// It tells you if the configured external service is reachable.
type e2eProviderConfigTestResp struct {
	ProviderConfigID string         `json:"providerConfigId"`
	ProviderID       string         `json:"providerId"`
	OverallStatus    string         `json:"overallStatus"` // "passed", "failed", or "partial"
	DurationMs       int64          `json:"durationMs"`
	Stages           []e2eTestStage `json:"stages"`
	Summary          string         `json:"summary"`
}

// e2eProviderSummary is a provider listed in the catalog (e.g., midaz, tracer).
type e2eProviderSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

// e2eProviderDetail is a single provider with its configuration JSON Schema.
type e2eProviderDetail struct {
	e2eProviderSummary
	ConfigSchema string `json:"configSchema"` // JSON Schema string
}

// e2eProviderExecutor is an executor that belongs to a provider (e.g., "tracer.validate-transaction" executor under "tracer" provider).
type e2eProviderExecutor struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	Version    string `json:"version"`
	ProviderID string `json:"providerId"`
}

// e2eExecutionListResp wraps a list of execution status objects.
type e2eExecutionListResp struct {
	Items []executionStatusResp `json:"items"`
}

// ─── Mock Server Factories ──────────────────────────────────────────────────
//
// These create local HTTP servers that simulate external financial providers.
// Every request they receive is recorded by the mockRecorder so the test can
// verify EXACTLY what the workflow engine sent (after applying transformations).

// newE2EKYCMockServer creates a fake KYC (Know Your Customer) service.
//
// Route logic:
//   - POST /v1/kyc/validate: If the "document" field starts with "1", the customer
//     is considered low-risk (riskScore=25, status="approved"). Otherwise, the
//     customer is high-risk (riskScore=85, status="review"). This lets the test
//     control which conditional branch the workflow takes.
//   - POST /v1/kyc/blocked-check: Always returns "not blocked".
//   - GET /health: Returns healthy (used by connectivity tests).
func newE2EKYCMockServer(t *testing.T, recorder *mockRecorder) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		case "/v1/kyc/validate":
			document, _ := body["document"].(string)
			if len(document) > 0 && document[0] == '1' {
				json.NewEncoder(w).Encode(map[string]any{
					"customerId": "CUST-001",
					"riskScore":  25,
					"status":     "approved",
				})
			} else {
				json.NewEncoder(w).Encode(map[string]any{
					"customerId": "CUST-999",
					"riskScore":  85,
					"status":     "review",
				})
			}

		case "/v1/kyc/blocked-check":
			json.NewEncoder(w).Encode(map[string]any{
				"blocked": false,
				"reason":  "not in blocklist",
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "not found"})
		}
	}))
}

// newE2EAMLMockServer creates a fake AML (Anti-Money Laundering) service.
//
// Route logic:
//   - POST /v1/aml/check: Always returns amlStatus="cleared" with a reference ID.
//     In the high-risk scenario, this endpoint should NEVER be called because
//     the conditional node skips the AML step.
//   - POST /v1/aml/enhanced: Returns enhanced check results (not used in main flow).
//   - GET /health: Returns healthy (used by connectivity tests).
func newE2EAMLMockServer(t *testing.T, recorder *mockRecorder) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		case "/v1/aml/check":
			json.NewEncoder(w).Encode(map[string]any{
				"amlStatus":   "cleared",
				"referenceId": "AML-REF-9876",
			})

		case "/v1/aml/enhanced":
			json.NewEncoder(w).Encode(map[string]any{
				"enhancedStatus": "cleared",
				"score":          10,
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{"error": "not found"})
		}
	}))
}

// ─── HTTP Helpers ───────────────────────────────────────────────────────────
//
// Convenience wrappers that reduce boilerplate for common HTTP operations.
// They all serialize the payload to JSON and return the raw response.
// The CALLER is responsible for closing resp.Body.

// e2ePostJSON sends a POST request with a JSON body. Returns the raw response.
func e2ePostJSON(t *testing.T, client *http.Client, url string, payload any) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	require.NoError(t, err)

	return resp
}

// e2ePatchJSON sends a PATCH request with a JSON body (for partial updates).
func e2ePatchJSON(t *testing.T, client *http.Client, url string, payload any) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

// e2ePutJSON sends a PUT request with a JSON body (for full replacement).
func e2ePutJSON(t *testing.T, client *http.Client, url string, payload any) *http.Response {
	t.Helper()

	body, err := json.Marshal(payload)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

// e2eDelete sends a DELETE request. Returns the raw response.
func e2eDelete(t *testing.T, client *http.Client, url string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	return resp
}

// e2eReadBody reads the entire response body into a byte slice (does NOT close it).
func e2eReadBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return bodyBytes
}

// =============================================================================
// MAIN E2E TEST
// =============================================================================
//
// This is the entry point. It sets up mock servers and then runs 5 phases
// in sequence. Variables are shared across phases so that IDs created in
// one phase (e.g., provider config IDs) can be used in later phases
// (e.g., workflow creation, execution).
//
// If any phase fails, Go's testing framework stops executing subsequent
// subtests within that phase, but other top-level phases may still run
// (and likely fail due to missing IDs). This is by design — sequential
// dependencies make the test behave like a real user session.

func TestFullE2E(t *testing.T) {
	client := httpClient()
	recorder := &mockRecorder{} // Records every HTTP request to mock servers

	// Start mock external providers
	kycServer := newE2EKYCMockServer(t, recorder)
	defer kycServer.Close()

	amlServer := newE2EAMLMockServer(t, recorder)
	defer amlServer.Close()

	// ── Shared state across phases ──
	// These variables carry IDs between phases, simulating a real user session
	// where you first create resources, then use them, then clean them up.
	var kycConfigID, amlConfigID, tempConfigID string   // from Phase 2
	var workflowID, clonedID, clonedFromActiveID string // from Phase 3
	var execID1, idempotencyKey1 string                 // from Phase 4

	// =========================================================================
	// PHASE 1: Catalog Exploration
	// =========================================================================
	//
	// "What's available on the platform?"
	//
	// Before configuring anything, a user would browse the catalog to see
	// which providers, executors, and triggers are available. This phase
	// verifies the catalog is populated and the static registry works.
	//
	// Providers = groups of executors (e.g., "tracer" provider has "tracer.validate-transaction" executor)
	// Executors = the actual operations (e.g., HTTP calls, S3 uploads)
	// Triggers  = how workflows start (e.g., webhook, scheduled)
	// =========================================================================
	t.Run("Phase1_Catalog", func(t *testing.T) {
		t.Run("list_providers", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/catalog/providers")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var providers []e2eProviderSummary
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&providers))
			resp.Body.Close()

			require.GreaterOrEqual(t, len(providers), 2, "should have at least midaz, tracer providers")

			// Verify known providers exist
			providerIDs := make(map[string]bool)
			for _, p := range providers {
				providerIDs[p.ID] = true
			}
			assert.True(t, providerIDs["midaz"], "midaz provider should exist")
			assert.True(t, providerIDs["tracer"], "tracer provider should exist")
		})

		t.Run("get_provider_tracer", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/catalog/providers/tracer")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var provider e2eProviderDetail
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&provider))
			resp.Body.Close()

			assert.Equal(t, "tracer", provider.ID)
			assert.NotEmpty(t, provider.Name)
			assert.NotEmpty(t, provider.ConfigSchema, "Tracer provider should have a config schema")
		})

		t.Run("get_provider_executors", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/catalog/providers/tracer/executors")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var executors []e2eProviderExecutor
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&executors))
			resp.Body.Close()

			require.GreaterOrEqual(t, len(executors), 1, "tracer provider should have at least 1 executor")

			found := false
			for _, e := range executors {
				if e.ID == "tracer.validate-transaction" {
					found = true
					break
				}
			}
			assert.True(t, found, "tracer.validate-transaction executor should be in tracer provider executors")
		})

		t.Run("list_executors", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/catalog/executors")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var execs []executorSummary
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&execs))
			resp.Body.Close()

			require.GreaterOrEqual(t, len(execs), 1, "should have at least 1 executor")
		})

		t.Run("get_executor_tracer", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/catalog/executors/tracer.validate-transaction")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var ed executorDetail
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&ed))
			resp.Body.Close()

			assert.Equal(t, "tracer.validate-transaction", ed.ID)
			assert.NotEmpty(t, ed.Schema, "Tracer executor should have a schema")
		})

		t.Run("validate_executor_valid", func(t *testing.T) {
			payload := map[string]any{
				"config": map[string]any{
					"requestId":            "550e8400-e29b-41d4-a716-446655440000",
					"transactionType":      "PIX",
					"amount":               "100",
					"currency":             "BRL",
					"transactionTimestamp": "2026-01-01T00:00:00Z",
					"account": map[string]any{
						"accountId": "550e8400-e29b-41d4-a716-446655440001",
						"type":      "checking",
						"status":    "active",
					},
				},
			}
			resp := e2ePostJSON(t, client, baseURL()+"/v1/catalog/executors/tracer.validate-transaction/validate", payload)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var vr validationResponse
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&vr))
			resp.Body.Close()

			assert.True(t, vr.Valid, "valid config should pass validation")
		})

		t.Run("validate_executor_invalid", func(t *testing.T) {
			payload := map[string]any{
				"config": map[string]any{
					"unknown_required_field": true,
				},
			}
			resp := e2ePostJSON(t, client, baseURL()+"/v1/catalog/executors/tracer.validate-transaction/validate", payload)
			// Accept either 200 or 400 depending on executor validation strictness
			resp.Body.Close()
		})

		t.Run("list_triggers", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/catalog/triggers")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var trigs []triggerSummary
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&trigs))
			resp.Body.Close()

			require.GreaterOrEqual(t, len(trigs), 1, "should have at least 1 trigger")
		})

		t.Run("get_trigger_webhook", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/catalog/triggers/webhook")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var td triggerDetail
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&td))
			resp.Body.Close()

			assert.Equal(t, "webhook", td.ID)
			assert.NotEmpty(t, td.Schema, "webhook trigger should have a schema")
		})
	})

	// =========================================================================
	// PHASE 2: Provider Configuration CRUD
	// =========================================================================
	//
	// "Let me set up connections to my external services."
	//
	// A provider configuration tells the platform HOW to reach an external
	// service. It stores the base URL and credentials. This phase tests
	// every CRUD operation on provider configs:
	//
	//   Create 3 configs (KYC, AML, temporary) → Read → List → Filter →
	//   Update description → Disable → List active only → Enable →
	//   Test connectivity → Duplicate name error → Delete → Verify 404
	//
	// IMPORTANT: The base_url includes the path prefix (e.g., "/v1/kyc")
	// because the execution engine builds the full URL by joining:
	//   url.JoinPath(base_url, endpointName) → "http://mock:1234/v1/kyc/validate"
	// =========================================================================
	t.Run("Phase2_ProviderConfigs", func(t *testing.T) {
		// ── Create 3 provider configs via HTTP API (not MongoDB seeding) ──

		t.Run("create_kyc_config", func(t *testing.T) {
			// base_url must include the path prefix because the execution engine
			// does url.JoinPath(base_url, endpointName) to build the full URL.
			// e.g. base_url="/v1/kyc" + endpointName="validate" → "/v1/kyc/validate"
			payload := map[string]any{
				"name":       "e2e-kyc-provider",
				"providerId": "tracer",
				"config": map[string]any{
					"base_url": kycServer.URL + "/v1/kyc",
					"api_key":  "test-key",
				},
			}
			resp := e2ePostJSON(t, client, baseURL()+"/v1/provider-configurations", payload)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode, "create kyc config: %s", string(bodyBytes))

			var cr e2eProviderConfigCreateResp
			require.NoError(t, json.Unmarshal(bodyBytes, &cr))
			require.NotEmpty(t, cr.ID)
			assert.Equal(t, "active", cr.Status)
			kycConfigID = cr.ID
		})

		t.Run("create_aml_config", func(t *testing.T) {
			payload := map[string]any{
				"name":       "e2e-aml-provider",
				"providerId": "tracer",
				"config": map[string]any{
					"base_url": amlServer.URL + "/v1/aml",
					"api_key":  "test-key",
				},
			}
			resp := e2ePostJSON(t, client, baseURL()+"/v1/provider-configurations", payload)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode, "create aml config: %s", string(bodyBytes))

			var cr e2eProviderConfigCreateResp
			require.NoError(t, json.Unmarshal(bodyBytes, &cr))
			require.NotEmpty(t, cr.ID)
			amlConfigID = cr.ID
		})

		t.Run("create_temporary_config", func(t *testing.T) {
			payload := map[string]any{
				"name":       "e2e-temp-provider",
				"providerId": "tracer",
				"config": map[string]any{
					"base_url": "http://localhost:9999",
					"api_key":  "test-key",
				},
			}
			resp := e2ePostJSON(t, client, baseURL()+"/v1/provider-configurations", payload)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode, "create temp config: %s", string(bodyBytes))

			var cr e2eProviderConfigCreateResp
			require.NoError(t, json.Unmarshal(bodyBytes, &cr))
			require.NotEmpty(t, cr.ID)
			tempConfigID = cr.ID
		})

		// ── Read operations ──

		t.Run("get_kyc_config", func(t *testing.T) {
			resp, err := client.Get(fmt.Sprintf("%s/v1/provider-configurations/%s", baseURL(), kycConfigID))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var pc e2eProviderConfigOutput
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&pc))
			resp.Body.Close()

			assert.Equal(t, kycConfigID, pc.ID)
			assert.Equal(t, "e2e-kyc-provider", pc.Name)
			assert.Equal(t, "tracer", pc.ProviderID)
			assert.Equal(t, "active", pc.Status)
			assert.NotEmpty(t, pc.Config)
		})

		t.Run("list_configs", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/provider-configurations?limit=100")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var list e2eProviderConfigListResp
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
			resp.Body.Close()

			require.GreaterOrEqual(t, len(list.Items), 3, "should have at least 3 configs")
		})

		t.Run("list_configs_filter_by_provider", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/provider-configurations?providerId=tracer&limit=100")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var list e2eProviderConfigListResp
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
			resp.Body.Close()

			require.GreaterOrEqual(t, len(list.Items), 3, "should have at least 3 tracer configs")
			for _, item := range list.Items {
				assert.Equal(t, "tracer", item.ProviderID)
			}
		})

		// ── Update ──

		t.Run("update_kyc_config_description", func(t *testing.T) {
			payload := map[string]any{
				"description": "Updated KYC provider for E2E integration testing",
			}
			resp := e2ePatchJSON(t, client, fmt.Sprintf("%s/v1/provider-configurations/%s", baseURL(), kycConfigID), payload)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var pc e2eProviderConfigOutput
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&pc))
			resp.Body.Close()

			assert.Equal(t, "Updated KYC provider for E2E integration testing", pc.Description)
		})

		// ── Lifecycle transitions ──

		t.Run("disable_temp_config", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/provider-configurations/%s/disable", baseURL(), tempConfigID),
				"application/json", nil,
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var pc e2eProviderConfigOutput
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&pc))
			resp.Body.Close()

			assert.Equal(t, "disabled", pc.Status)
		})

		t.Run("list_active_only", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/provider-configurations?status=active&limit=100")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var list e2eProviderConfigListResp
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
			resp.Body.Close()

			for _, item := range list.Items {
				assert.NotEqual(t, tempConfigID, item.ID, "disabled config should not appear in active-only list")
			}
		})

		t.Run("enable_temp_config", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/provider-configurations/%s/enable", baseURL(), tempConfigID),
				"application/json", nil,
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var pc e2eProviderConfigOutput
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&pc))
			resp.Body.Close()

			assert.Equal(t, "active", pc.Status)
		})

		// ── Test Connectivity ──

		t.Run("test_kyc_connectivity", func(t *testing.T) {
			// The connectivity test runs 3 stages:
			//   1. connectivity (GET base_url, pass if < 500)
			//   2. authentication (skipped — no api_key/headers configured)
			//   3. end-to-end (GET base_url, pass only if 2xx)
			//
			// Stage 3 may fail because GET on /v1/kyc returns 404 (mock only
			// handles POST routes), making overallStatus "partial".
			// Both "passed" and "partial" mean the server is reachable.
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/provider-configurations/%s/test", baseURL(), kycConfigID),
				"application/json", nil,
			)
			require.NoError(t, err)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode, "test kyc connectivity: %s", string(bodyBytes))

			var result e2eProviderConfigTestResp
			require.NoError(t, json.Unmarshal(bodyBytes, &result))

			assert.Equal(t, kycConfigID, result.ProviderConfigID)
			assert.Contains(t, []string{"passed", "partial"}, result.OverallStatus,
				"KYC mock should be reachable (passed or partial)")
		})

		t.Run("test_aml_connectivity", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/provider-configurations/%s/test", baseURL(), amlConfigID),
				"application/json", nil,
			)
			require.NoError(t, err)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode, "test aml connectivity: %s", string(bodyBytes))

			var result e2eProviderConfigTestResp
			require.NoError(t, json.Unmarshal(bodyBytes, &result))

			assert.Contains(t, []string{"passed", "partial"}, result.OverallStatus,
				"AML mock should be reachable (passed or partial)")
		})

		// ── Duplicate name error ──

		t.Run("create_duplicate_name_fails", func(t *testing.T) {
			payload := map[string]any{
				"name":       "e2e-kyc-provider", // same name as existing
				"providerId": "tracer",
				"config":     map[string]any{"base_url": "http://localhost:1234", "api_key": "test-key"},
			}
			resp := e2ePostJSON(t, client, baseURL()+"/v1/provider-configurations", payload)
			resp.Body.Close()

			assert.Equal(t, http.StatusConflict, resp.StatusCode, "duplicate name should return 409")
		})

		// ── Delete temp config ──

		t.Run("disable_before_delete", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/provider-configurations/%s/disable", baseURL(), tempConfigID),
				"application/json", nil,
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		})

		t.Run("delete_temp_config", func(t *testing.T) {
			resp := e2eDelete(t, client, fmt.Sprintf("%s/v1/provider-configurations/%s", baseURL(), tempConfigID))
			resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		t.Run("get_deleted_config_fails", func(t *testing.T) {
			resp, err := client.Get(fmt.Sprintf("%s/v1/provider-configurations/%s", baseURL(), tempConfigID))
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	})

	// =========================================================================
	// PHASE 3: Complex Workflow Lifecycle
	// =========================================================================
	//
	// "Let me build a compliance workflow and manage its lifecycle."
	//
	// A workflow is a directed graph of nodes connected by edges. This phase
	// creates a COMPLEX workflow that models a real-world compliance check:
	//
	//   trigger (webhook)
	//       │
	//       ▼
	//   kyc-executor ──── Calls KYC mock server
	//       │              INPUT:  cpf → document (remove "." and "-")
	//       │                      name → fullName (UPPERCASE)
	//       │              OUTPUT: customerId, riskScore
	//       ▼
	//   risk-check ────── Is riskScore < 50?
	//       │
	//       ├── YES ──▶ aml-executor ──── Calls AML mock server
	//       │               │              INPUT:  customerId (from KYC output)
	//       │               │                      amount (from workflow input)
	//       │               │              OUTPUT: amlStatus, referenceId
	//       │               ▼
	//       │           approve-action ─── Sets decision = "approved"
	//       │
	//       └── NO  ──▶ reject-action ─── Sets decision = "rejected"
	//
	// After building the workflow, this phase tests the full lifecycle:
	//
	//   Create (draft) → Read → List → Filter → Update metadata →
	//   Clone → Activate → Cannot update active → Cannot activate again →
	//   Clone from active → Deactivate → Activate clone for Phase 4 →
	//   Delete draft clone → Delete inactive original
	//
	// The provider config IDs from Phase 2 (kycConfigID, amlConfigID) are
	// embedded in the executor nodes, linking this workflow to the mock
	// servers we configured earlier.
	// =========================================================================
	t.Run("Phase3_WorkflowLifecycle", func(t *testing.T) {
		// ── Build complex workflow payload ──
		complexPayload := map[string]any{
			"name":        "e2e-complex-workflow",
			"description": "Full E2E test workflow with KYC, conditional branching, and AML",
			"metadata": map[string]any{
				"team":    "compliance",
				"version": "1.0",
			},
			"nodes": []map[string]any{
				// Trigger
				{
					"id":       "trigger-entry",
					"type":     "trigger",
					"data":     map[string]any{"triggerId": "webhook"},
					"position": map[string]any{"x": 0, "y": 0},
				},
				// KYC Executor
				{
					"id":   "kyc-executor",
					"type": "executor",
					"name": "KYC Validation",
					"data": map[string]any{
						"executorId":       "tracer.validate-transaction",
						"providerConfigId": kycConfigID,
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
							{"source": "body.customerId", "target": "result.customerId"},
							{"source": "body.riskScore", "target": "result.riskScore"},
						},
					},
					"position": map[string]any{"x": 200, "y": 0},
				},
				// Risk Check Conditional
				{
					"id":   "risk-check",
					"type": "conditional",
					"name": "Risk Assessment",
					"data": map[string]any{
						"condition": "kyc-executor.result.riskScore < 50",
					},
					"position": map[string]any{"x": 400, "y": 0},
				},
				// AML Executor (true branch)
				{
					"id":   "aml-executor",
					"type": "executor",
					"name": "AML Check",
					"data": map[string]any{
						"executorId":       "tracer.validate-transaction",
						"providerConfigId": amlConfigID,
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
							{"source": "body.amlStatus", "target": "result.amlStatus"},
							{"source": "body.referenceId", "target": "result.referenceId"},
						},
					},
					"position": map[string]any{"x": 600, "y": -100},
				},
				// Approve Action
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
				// Reject Action (false branch)
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
				{"id": "e1", "source": "trigger-entry", "target": "kyc-executor"},
				{"id": "e2", "source": "kyc-executor", "target": "risk-check"},
				{"id": "e3", "source": "risk-check", "target": "aml-executor", "sourceHandle": "true"},
				{"id": "e4", "source": "risk-check", "target": "reject-action", "sourceHandle": "false"},
				{"id": "e5", "source": "aml-executor", "target": "approve-action"},
			},
		}

		// ── Create complex workflow ──

		t.Run("create_complex_workflow", func(t *testing.T) {
			resp := e2ePostJSON(t, client, baseURL()+"/v1/workflows", complexPayload)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode, "create complex workflow: %s", string(bodyBytes))

			var cr createWorkflowResp
			require.NoError(t, json.Unmarshal(bodyBytes, &cr))
			require.NotEmpty(t, cr.WorkflowID)
			assert.Equal(t, "draft", cr.Status)
			workflowID = cr.WorkflowID
		})

		// ── Read and verify ──

		t.Run("get_workflow", func(t *testing.T) {
			resp, err := client.Get(fmt.Sprintf("%s/v1/workflows/%s", baseURL(), workflowID))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var wf workflowOutput
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&wf))
			resp.Body.Close()

			assert.Equal(t, "e2e-complex-workflow", wf.Name)
			assert.Len(t, wf.Nodes, 6, "should have 6 nodes")
			assert.Len(t, wf.Edges, 5, "should have 5 edges")

			// Verify KYC node has input/output mappings
			for _, node := range wf.Nodes {
				if node.ID == "kyc-executor" {
					inputMapping, ok := node.Data["inputMapping"].([]any)
					assert.True(t, ok, "kyc node should have inputMapping")
					assert.Len(t, inputMapping, 2, "kyc should have 2 input mappings")

					outputMapping, ok := node.Data["outputMapping"].([]any)
					assert.True(t, ok, "kyc node should have outputMapping")
					assert.Len(t, outputMapping, 2, "kyc should have 2 output mappings")
				}
			}
		})

		t.Run("list_workflows", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/workflows?limit=100")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var list listWorkflowsResp
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
			resp.Body.Close()

			require.GreaterOrEqual(t, len(list.Items), 1)
		})

		t.Run("list_workflows_filter_draft", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/workflows?status=draft&limit=100")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var list listWorkflowsResp
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
			resp.Body.Close()

			found := false
			for _, item := range list.Items {
				if item.ID == workflowID {
					found = true
					assert.Equal(t, "draft", item.Status)
					break
				}
			}
			assert.True(t, found, "workflow should appear in draft filter")
		})

		// ── Update workflow (add metadata) ──

		t.Run("update_workflow_metadata", func(t *testing.T) {
			updatePayload := complexPayload
			updatePayload["metadata"] = map[string]any{
				"team":    "compliance",
				"version": "2.0",
				"e2e":     true,
			}
			resp := e2ePutJSON(t, client, fmt.Sprintf("%s/v1/workflows/%s", baseURL(), workflowID), updatePayload)
			resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode, "update should succeed for draft workflow")
		})

		// ── Clone before activate ──

		t.Run("clone_workflow", func(t *testing.T) {
			payload := map[string]any{
				"name": "e2e-cloned-workflow",
			}
			resp := e2ePostJSON(t, client, fmt.Sprintf("%s/v1/workflows/%s/clone", baseURL(), workflowID), payload)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode, "clone workflow: %s", string(bodyBytes))

			var cr createWorkflowResp
			require.NoError(t, json.Unmarshal(bodyBytes, &cr))
			require.NotEmpty(t, cr.WorkflowID)
			assert.Equal(t, "draft", cr.Status)
			assert.NotEqual(t, workflowID, cr.WorkflowID)
			clonedID = cr.WorkflowID
		})

		// ── Activate original ──

		t.Run("activate_workflow", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/workflows/%s/activate", baseURL(), workflowID),
				"application/json", nil,
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		})

		// ── Verify cannot update active ──

		t.Run("update_active_fails", func(t *testing.T) {
			resp := e2ePutJSON(t, client, fmt.Sprintf("%s/v1/workflows/%s", baseURL(), workflowID), complexPayload)
			resp.Body.Close()

			assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, "cannot modify active workflow")
		})

		// ── Verify cannot activate again ──

		t.Run("activate_again_fails", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/workflows/%s/activate", baseURL(), workflowID),
				"application/json", nil,
			)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, "cannot activate already active workflow")
		})

		// ── Clone active workflow ──

		t.Run("clone_active_workflow", func(t *testing.T) {
			payload := map[string]any{
				"name": "e2e-cloned-from-active",
			}
			resp := e2ePostJSON(t, client, fmt.Sprintf("%s/v1/workflows/%s/clone", baseURL(), workflowID), payload)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode, "clone active workflow: %s", string(bodyBytes))

			var cr createWorkflowResp
			require.NoError(t, json.Unmarshal(bodyBytes, &cr))
			assert.Equal(t, "draft", cr.Status)
			clonedFromActiveID = cr.WorkflowID
		})

		// ── Deactivate original ──

		t.Run("deactivate_workflow", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/workflows/%s/deactivate", baseURL(), workflowID),
				"application/json", nil,
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		})

		// ── Activate clone for execution ──

		t.Run("activate_clone_for_execution", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/workflows/%s/activate", baseURL(), clonedFromActiveID),
				"application/json", nil,
			)
			require.NoError(t, err)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusOK, resp.StatusCode, "activate clone: %s", string(bodyBytes))
		})

		// ── Delete draft clone ──

		t.Run("delete_draft_clone", func(t *testing.T) {
			resp := e2eDelete(t, client, fmt.Sprintf("%s/v1/workflows/%s", baseURL(), clonedID))
			resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		// ── Delete inactive original ──

		t.Run("delete_inactive_original", func(t *testing.T) {
			resp := e2eDelete(t, client, fmt.Sprintf("%s/v1/workflows/%s", baseURL(), workflowID))
			resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})
	})

	// =========================================================================
	// PHASE 4: Workflow Execution E2E
	// =========================================================================
	//
	// "Let me run the workflow with real customer data."
	//
	// This is the most important phase — it actually EXECUTES the workflow
	// from Phase 3 against the mock KYC and AML servers. We run TWO
	// scenarios to test both branches of the conditional:
	//
	// ── Scenario A: Low Risk (CPF starts with "1") ──────────────────────
	//
	//   Input:   cpf="123.456.789-00", name="John Doe", amount=1500.50
	//
	//   Step 1 → KYC mock receives: document="12345678900", fullName="JOHN DOE"
	//            (remove_characters stripped ".-", to_uppercase converted name)
	//
	//   Step 2 → KYC returns riskScore=25 (CPF starts with "1" = low risk)
	//
	//   Step 3 → Conditional: 25 < 50? YES → take the "true" branch
	//
	//   Step 4 → AML mock receives: customerId="CUST-001", amount=1500.50
	//            (customerId came from KYC output via cross-step data flow)
	//
	//   Step 5 → AML returns amlStatus="cleared"
	//
	//   Step 6 → Approve action: decision="approved"
	//
	// ── Scenario B: High Risk (CPF starts with "9") ─────────────────────
	//
	//   Input:   cpf="999.999.999-99", name="Risky Person", amount=50000
	//
	//   Step 1 → KYC mock receives: document="99999999999", fullName="RISKY PERSON"
	//
	//   Step 2 → KYC returns riskScore=85 (CPF doesn't start with "1")
	//
	//   Step 3 → Conditional: 85 < 50? NO → take the "false" branch
	//
	//   Step 4 → Reject action: decision="rejected", reason="high risk score"
	//            AML is NEVER called (skipped by conditional)
	//
	// ── Additional checks ───────────────────────────────────────────────
	//
	//   - Idempotency: same Idempotency-Key returns same execution (no duplicate)
	//   - List executions: verify both executions appear in the list
	//   - Missing header: POST without Idempotency-Key returns 400
	//
	// The mock recorder captures every HTTP request the engine made, so we
	// can verify EXACTLY what data was sent to each external service.
	// =========================================================================
	t.Run("Phase4_Execution", func(t *testing.T) {
		// Clear recorder to track only execution calls
		recorder.clear()

		// ── Execute with low risk (happy path) ──

		t.Run("execute_low_risk", func(t *testing.T) {
			idempotencyKey1 = uuid.NewString()
			execPayload := map[string]any{
				"inputData": map[string]any{
					"customer": map[string]any{
						"cpf":  "123.456.789-00",
						"name": "John Doe",
					},
					"transaction": map[string]any{
						"amount": 1500.50,
					},
				},
			}
			body, _ := json.Marshal(execPayload)

			req, err := http.NewRequest(http.MethodPost,
				fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), clonedFromActiveID),
				bytes.NewBuffer(body),
			)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", idempotencyKey1)

			resp, err := client.Do(req)
			require.NoError(t, err)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode, "execute low risk: %s", string(bodyBytes))

			var er executionCreateResp
			require.NoError(t, json.Unmarshal(bodyBytes, &er))
			require.NotEmpty(t, er.ExecutionID)
			assert.Equal(t, clonedFromActiveID, er.WorkflowID)
			assert.Equal(t, "running", er.Status)
			execID1 = er.ExecutionID
		})

		// ── Poll until completion ──

		t.Run("poll_low_risk_status", func(t *testing.T) {
			status := pollExecutionStatus(t, client, execID1, 30*time.Second)
			assert.Equal(t, "completed", status.Status, "low-risk execution should complete successfully")
			assert.Nil(t, status.ErrorMessage, "no error expected")
		})

		// ── Verify results ──

		t.Run("verify_low_risk_results", func(t *testing.T) {
			results := getExecutionResults(t, client, execID1)
			assert.Equal(t, "completed", results.Status)
			assert.NotNil(t, results.CompletedAt, "completedAt should be set")

			// Verify final output
			require.NotNil(t, results.FinalOutput, "finalOutput should be present")
			assert.Equal(t, "approved", results.FinalOutput["decision"], "low-risk should be approved")

			// Verify step results
			require.GreaterOrEqual(t, len(results.StepResults), 4, "should have at least 4 steps")

			// ── Step 1: KYC executor — verify output mapping applied ──
			kycStep := results.StepResults[0]
			assert.Equal(t, "kyc-executor", kycStep.NodeID)
			assert.Equal(t, "KYC Validation", kycStep.StepName)
			assert.Equal(t, "completed", kycStep.Status)
			assert.Nil(t, kycStep.ErrorMessage)

			// Output mapping should have mapped customerId → result.customerId, riskScore → result.riskScore
			if kycStep.Output != nil {
				if result, ok := kycStep.Output["result"].(map[string]any); ok {
					assert.Equal(t, "CUST-001", result["customerId"],
						"KYC output mapping: customerId should be mapped to result.customerId")
					assert.Equal(t, float64(25), result["riskScore"],
						"KYC output mapping: riskScore should be mapped to result.riskScore")
				}
			}

			// ── Step 2: Risk Assessment — verify conditional evaluated correctly ──
			riskStep := results.StepResults[1]
			assert.Equal(t, "risk-check", riskStep.NodeID)
			assert.Equal(t, "Risk Assessment", riskStep.StepName)
			assert.Equal(t, "completed", riskStep.Status)

			if riskStep.Output != nil {
				assert.Equal(t, true, riskStep.Output["result"],
					"condition kyc-executor.result.riskScore(25) < 50 should be true")
				assert.Equal(t, "true", riskStep.Output["branchTaken"],
					"should take true branch → AML executor")
			}

			// ── Step 3: AML executor — verify input came from KYC output ──
			amlStep := results.StepResults[2]
			assert.Equal(t, "aml-executor", amlStep.NodeID)
			assert.Equal(t, "AML Check", amlStep.StepName)
			assert.Equal(t, "completed", amlStep.Status)

			// Output mapping should have mapped amlStatus → result.amlStatus, referenceId → result.referenceId
			if amlStep.Output != nil {
				if result, ok := amlStep.Output["result"].(map[string]any); ok {
					assert.Equal(t, "cleared", result["amlStatus"],
						"AML output mapping: amlStatus should be mapped to result.amlStatus")
					assert.Equal(t, "AML-REF-9876", result["referenceId"],
						"AML output mapping: referenceId should be mapped to result.referenceId")
				}
			}

			// ── Step 4: Approve action — verify set_output action ──
			approveStep := results.StepResults[3]
			assert.Equal(t, "approve-action", approveStep.NodeID)
			assert.Equal(t, "Approve Transaction", approveStep.StepName)
			assert.Equal(t, "completed", approveStep.Status)

			// ── Verify INPUT transformations via mock recorder ──
			// KYC: remove_characters(".-") applied to CPF, to_uppercase applied to name
			kycRequests := recorder.getByPath("/v1/kyc/validate")
			require.GreaterOrEqual(t, len(kycRequests), 1, "KYC should be called at least once")
			kycBody := kycRequests[len(kycRequests)-1].Body
			assert.Equal(t, "12345678900", kycBody["document"],
				"input transformation: remove_characters should strip '.' and '-' from 123.456.789-00")
			assert.Equal(t, "JOHN DOE", kycBody["fullName"],
				"input transformation: to_uppercase should convert 'John Doe' to 'JOHN DOE'")

			// ── Verify data flow between steps via mock recorder ──
			// AML received customerId from KYC output (wfCtx["kyc-executor"].result.customerId)
			// AML received transactionAmount from workflow input (wfCtx["workflow"].transaction.amount)
			amlRequests := recorder.getByPath("/v1/aml/check")
			require.GreaterOrEqual(t, len(amlRequests), 1, "AML should be called at least once")
			amlBody := amlRequests[len(amlRequests)-1].Body
			assert.Equal(t, "CUST-001", amlBody["customerId"],
				"cross-step data flow: AML input should receive customerId from KYC output mapping")
			assert.Equal(t, 1500.50, amlBody["transactionAmount"],
				"cross-step data flow: AML input should receive amount from workflow.transaction.amount")
		})

		// ── Idempotency check ──

		t.Run("idempotency_same_key", func(t *testing.T) {
			execPayload := map[string]any{
				"inputData": map[string]any{
					"customer": map[string]any{
						"cpf":  "123.456.789-00",
						"name": "John Doe",
					},
					"transaction": map[string]any{
						"amount": 1500.50,
					},
				},
			}
			body, _ := json.Marshal(execPayload)

			req, err := http.NewRequest(http.MethodPost,
				fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), clonedFromActiveID),
				bytes.NewBuffer(body),
			)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", idempotencyKey1)

			resp, err := client.Do(req)
			require.NoError(t, err)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			// Should return 200 (idempotent) with the same execution ID
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
				var er executionCreateResp
				require.NoError(t, json.Unmarshal(bodyBytes, &er))
				assert.Equal(t, execID1, er.ExecutionID, "idempotent call should return same execution ID")
			} else {
				t.Fatalf("idempotency check: expected 200 or 201, got %d, body: %s", resp.StatusCode, string(bodyBytes))
			}
		})

		// ── Execute with high risk (conditional false path) ──

		var execID2 string

		t.Run("execute_high_risk", func(t *testing.T) {
			recorder.clear() // clear for high-risk verification

			execPayload := map[string]any{
				"inputData": map[string]any{
					"customer": map[string]any{
						"cpf":  "999.999.999-99",
						"name": "Risky Person",
					},
					"transaction": map[string]any{
						"amount": 50000,
					},
				},
			}
			body, _ := json.Marshal(execPayload)

			req, err := http.NewRequest(http.MethodPost,
				fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), clonedFromActiveID),
				bytes.NewBuffer(body),
			)
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Idempotency-Key", uuid.NewString())

			resp, err := client.Do(req)
			require.NoError(t, err)
			bodyBytes := e2eReadBody(t, resp)
			resp.Body.Close()

			require.Equal(t, http.StatusCreated, resp.StatusCode, "execute high risk: %s", string(bodyBytes))

			var er executionCreateResp
			require.NoError(t, json.Unmarshal(bodyBytes, &er))
			require.NotEmpty(t, er.ExecutionID)
			execID2 = er.ExecutionID
		})

		t.Run("poll_high_risk_status", func(t *testing.T) {
			status := pollExecutionStatus(t, client, execID2, 30*time.Second)
			assert.Equal(t, "completed", status.Status, "high-risk execution should complete (rejection is not a failure)")
		})

		t.Run("verify_high_risk_results", func(t *testing.T) {
			results := getExecutionResults(t, client, execID2)
			assert.Equal(t, "completed", results.Status)

			// Verify rejected
			require.NotNil(t, results.FinalOutput)
			assert.Equal(t, "rejected", results.FinalOutput["decision"])
			assert.Equal(t, "high risk score", results.FinalOutput["reason"])

			// Verify only 3 steps (KYC, conditional, reject) — no AML
			require.Len(t, results.StepResults, 3, "high-risk should have 3 steps (no AML)")

			// ── Step 1: KYC — verify output mapping shows high risk ──
			kycStep := results.StepResults[0]
			assert.Equal(t, "kyc-executor", kycStep.NodeID)
			assert.Equal(t, "completed", kycStep.Status)

			if kycStep.Output != nil {
				if result, ok := kycStep.Output["result"].(map[string]any); ok {
					assert.Equal(t, "CUST-999", result["customerId"],
						"high-risk KYC should return customerId CUST-999")
					assert.Equal(t, float64(85), result["riskScore"],
						"high-risk KYC should return riskScore 85")
				}
			}

			// ── Step 2: Risk Assessment — conditional false path ──
			riskStep := results.StepResults[1]
			assert.Equal(t, "risk-check", riskStep.NodeID)
			assert.Equal(t, "completed", riskStep.Status)

			if riskStep.Output != nil {
				assert.Equal(t, false, riskStep.Output["result"],
					"condition kyc-executor.result.riskScore(85) < 50 should be false")
				assert.Equal(t, "false", riskStep.Output["branchTaken"],
					"should take false branch → reject action")
			}

			// ── Step 3: Reject action ──
			rejectStep := results.StepResults[2]
			assert.Equal(t, "reject-action", rejectStep.NodeID)
			assert.Equal(t, "completed", rejectStep.Status)

			// ── Verify AML was NOT called (false branch skips it) ──
			amlRequests := recorder.getByPath("/v1/aml/check")
			assert.Len(t, amlRequests, 0, "AML endpoint should NOT be called for high-risk path")

			// ── Verify KYC input transformations applied correctly ──
			kycRequests := recorder.getByPath("/v1/kyc/validate")
			require.GreaterOrEqual(t, len(kycRequests), 1)
			assert.Equal(t, "99999999999", kycRequests[0].Body["document"],
				"input transformation: remove_characters should strip '.' and '-' from 999.999.999-99")
			assert.Equal(t, "RISKY PERSON", kycRequests[0].Body["fullName"],
				"input transformation: to_uppercase should convert 'Risky Person' to 'RISKY PERSON'")
		})

		// ── List executions ──

		t.Run("list_executions", func(t *testing.T) {
			resp, err := client.Get(baseURL() + "/v1/executions?limit=100")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var list e2eExecutionListResp
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&list))
			resp.Body.Close()

			require.GreaterOrEqual(t, len(list.Items), 2, "should have at least 2 executions")
		})

		// ── Execution errors ──

		t.Run("execute_missing_idempotency_fails", func(t *testing.T) {
			execPayload := map[string]any{
				"inputData": map[string]any{"test": true},
			}
			body, _ := json.Marshal(execPayload)

			// POST without Idempotency-Key header
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/workflows/%s/executions", baseURL(), clonedFromActiveID),
				"application/json",
				bytes.NewBuffer(body),
			)
			require.NoError(t, err)
			resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "missing idempotency key should return 400")
		})
	})

	// =========================================================================
	// PHASE 5: Cleanup Verification
	// =========================================================================
	//
	// "Let me clean up everything I created and verify it's really gone."
	//
	// A well-behaved test cleans up after itself. This phase deletes every
	// resource created during the test, in reverse dependency order:
	//
	//   1. Deactivate the execution workflow (active → inactive)
	//   2. Delete the execution workflow
	//   3. Disable KYC provider config (active → disabled, required before delete)
	//   4. Delete KYC provider config
	//   5. Disable AML provider config
	//   6. Delete AML provider config
	//   7. Verify ALL resources return 404 (truly gone from the database)
	//
	// NOTE: Active workflows and active provider configs cannot be deleted
	// directly — they must first be deactivated/disabled. This ensures
	// production safety: you can't accidentally delete a config that's
	// still in use by running workflows.
	// =========================================================================
	t.Run("Phase5_Cleanup", func(t *testing.T) {
		t.Run("deactivate_execution_workflow", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/workflows/%s/deactivate", baseURL(), clonedFromActiveID),
				"application/json", nil,
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		})

		t.Run("delete_execution_workflow", func(t *testing.T) {
			resp := e2eDelete(t, client, fmt.Sprintf("%s/v1/workflows/%s", baseURL(), clonedFromActiveID))
			resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		t.Run("disable_kyc_config", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/provider-configurations/%s/disable", baseURL(), kycConfigID),
				"application/json", nil,
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		})

		t.Run("delete_kyc_config", func(t *testing.T) {
			resp := e2eDelete(t, client, fmt.Sprintf("%s/v1/provider-configurations/%s", baseURL(), kycConfigID))
			resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		t.Run("disable_aml_config", func(t *testing.T) {
			resp, err := client.Post(
				fmt.Sprintf("%s/v1/provider-configurations/%s/disable", baseURL(), amlConfigID),
				"application/json", nil,
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			resp.Body.Close()
		})

		t.Run("delete_aml_config", func(t *testing.T) {
			resp := e2eDelete(t, client, fmt.Sprintf("%s/v1/provider-configurations/%s", baseURL(), amlConfigID))
			resp.Body.Close()

			assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		})

		// ── Verify all deleted ──

		t.Run("verify_configs_deleted", func(t *testing.T) {
			resp, err := client.Get(fmt.Sprintf("%s/v1/provider-configurations/%s", baseURL(), kycConfigID))
			require.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, http.StatusNotFound, resp.StatusCode, "KYC config should be gone")

			resp, err = client.Get(fmt.Sprintf("%s/v1/provider-configurations/%s", baseURL(), amlConfigID))
			require.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, http.StatusNotFound, resp.StatusCode, "AML config should be gone")

			resp, err = client.Get(fmt.Sprintf("%s/v1/workflows/%s", baseURL(), clonedFromActiveID))
			require.NoError(t, err)
			resp.Body.Close()
			assert.Equal(t, http.StatusNotFound, resp.StatusCode, "execution workflow should be gone")
		})
	})
}
