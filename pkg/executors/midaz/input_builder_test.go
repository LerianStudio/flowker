// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package midaz_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executors"
	"github.com/LerianStudio/flowker/pkg/executors/midaz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseProviderConfig(txBaseURL, onbBaseURL string) map[string]any {
	return map[string]any{
		"transaction_base_url": txBaseURL,
		"onboarding_base_url":  onbBaseURL,
		"organization_id":      "org-001",
		"ledger_id":            "ledger-001",
		"auth": map[string]any{
			"issuer_url":    "https://auth.example.com",
			"client_id":     "my-client",
			"client_secret": "my-secret",
		},
	}
}

func TestBuildInput_CreateTransaction(t *testing.T) {
	config := baseProviderConfig("https://tx.example.com", "https://onb.example.com")

	body, err := json.Marshal(map[string]any{
		"send": map[string]any{"asset": "BRL", "value": "100.00"},
	})
	require.NoError(t, err)

	input, err := midaz.BuildInput(config, "midaz.create-transaction", nil, body)
	require.NoError(t, err)

	assert.Equal(t, "POST", input.Config["method"])
	assert.Equal(t, "https://tx.example.com/v1/organizations/org-001/ledgers/ledger-001/transactions/json", input.Config["url"])
	assert.NotNil(t, input.Config["auth"])
	assert.NotNil(t, input.Config["body"])
}

func TestBuildInput_CreateAccount(t *testing.T) {
	config := baseProviderConfig("https://tx.example.com", "https://onb.example.com")

	body, err := json.Marshal(map[string]any{
		"assetCode": "BRL",
		"type":      "deposit",
	})
	require.NoError(t, err)

	input, err := midaz.BuildInput(config, "midaz.create-account", nil, body)
	require.NoError(t, err)

	assert.Equal(t, "POST", input.Config["method"])
	assert.Equal(t, "https://onb.example.com/v1/organizations/org-001/ledgers/ledger-001/accounts", input.Config["url"])
	assert.NotNil(t, input.Config["body"])
}

func TestBuildInput_GetAccount(t *testing.T) {
	config := baseProviderConfig("https://tx.example.com", "https://onb.example.com")
	nodeData := map[string]any{
		"accountId": "acc-123",
	}

	input, err := midaz.BuildInput(config, "midaz.get-account", nodeData, nil)
	require.NoError(t, err)

	assert.Equal(t, "GET", input.Config["method"])
	assert.Equal(t, "https://onb.example.com/v1/organizations/org-001/ledgers/ledger-001/accounts/acc-123", input.Config["url"])
	// GET requests should not have body
	assert.Nil(t, input.Config["body"])
}

func TestBuildInput_GetAccountBalance(t *testing.T) {
	config := baseProviderConfig("https://tx.example.com", "https://onb.example.com")
	nodeData := map[string]any{
		"accountId": "acc-456",
	}

	input, err := midaz.BuildInput(config, "midaz.get-account-balance", nodeData, nil)
	require.NoError(t, err)

	assert.Equal(t, "GET", input.Config["method"])
	assert.Equal(t, "https://tx.example.com/v1/organizations/org-001/ledgers/ledger-001/accounts/acc-456/balances", input.Config["url"])
	assert.Nil(t, input.Config["body"])
}

func TestBuildInput_AuthTranslation(t *testing.T) {
	config := baseProviderConfig("https://tx.example.com", "https://onb.example.com")

	input, err := midaz.BuildInput(config, "midaz.create-transaction", nil, nil)
	require.NoError(t, err)

	authMap, ok := input.Config["auth"].(map[string]any)
	require.True(t, ok, "auth must be a map")
	assert.Equal(t, "oidc_client_credentials", authMap["type"])

	authConfig, ok := authMap["config"].(map[string]any)
	require.True(t, ok, "auth config must be a map")
	assert.Equal(t, "https://auth.example.com", authConfig["issuer_url"])
	assert.Equal(t, "my-client", authConfig["client_id"])
	assert.Equal(t, "my-secret", authConfig["client_secret"])
}

func TestBuildInput_MissingOrgID(t *testing.T) {
	config := map[string]any{
		"transaction_base_url": "https://tx.example.com",
		"ledger_id":            "ledger-001",
	}

	_, err := midaz.BuildInput(config, "midaz.create-transaction", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required path parameter")
	assert.Contains(t, err.Error(), "organization_id")
}

func TestBuildInput_MissingBaseURL(t *testing.T) {
	config := map[string]any{
		"organization_id": "org-001",
		"ledger_id":       "ledger-001",
	}

	_, err := midaz.BuildInput(config, "midaz.create-transaction", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required field")
	assert.Contains(t, err.Error(), "transaction_base_url")
}

func TestBuildInput_UnknownExecutor(t *testing.T) {
	config := baseProviderConfig("https://tx.example.com", "https://onb.example.com")

	_, err := midaz.BuildInput(config, "midaz.unknown-executor", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown Midaz executor")
}

func TestBuildInput_WithRequestBody(t *testing.T) {
	config := baseProviderConfig("https://tx.example.com", "https://onb.example.com")

	requestBody := map[string]any{
		"description": "Test transaction",
		"send": map[string]any{
			"asset": "USD",
			"value": "250.00",
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	input, err := midaz.BuildInput(config, "midaz.create-transaction", nil, bodyBytes)
	require.NoError(t, err)

	assert.Equal(t, "POST", input.Config["method"])
	assert.NotNil(t, input.Config["body"], "POST requests with body should include body in config")

	// Verify body is a parsed object (not raw bytes)
	bodyMap, ok := input.Config["body"].(map[string]any)
	require.True(t, ok, "body should be parsed into a map")
	assert.Equal(t, "Test transaction", bodyMap["description"])
}

func TestBuildInput_FullFlow(t *testing.T) {
	var receivedMethod string
	var receivedPath string
	var receivedBody map[string]any

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path

		if r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			defer r.Body.Close()

			if len(bodyBytes) > 0 {
				err = json.Unmarshal(bodyBytes, &receivedBody)
				require.NoError(t, err)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		resp := map[string]any{
			"id":     "tx-001",
			"status": "APPROVED",
		}

		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	// Build provider config using mock server URL
	config := map[string]any{
		"transaction_base_url": mockServer.URL,
		"onboarding_base_url":  mockServer.URL,
		"organization_id":      "org-test",
		"ledger_id":            "ledger-test",
	}

	requestBody := map[string]any{
		"send": map[string]any{
			"asset": "BRL",
			"value": "500.00",
		},
	}

	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	// Build input via BuildInput
	input, err := midaz.BuildInput(config, "midaz.create-transaction", nil, bodyBytes)
	require.NoError(t, err)

	// Get runner from catalog
	catalog := executor.NewCatalog()
	err = executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	runner, err := catalog.GetRunner("midaz.create-transaction")
	require.NoError(t, err)

	// Execute via HTTP runner
	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)

	// Verify the mock server received the correct request
	assert.Equal(t, "POST", receivedMethod)
	assert.Equal(t, "/v1/organizations/org-test/ledgers/ledger-test/transactions/json", receivedPath)
	assert.NotNil(t, receivedBody)

	send, ok := receivedBody["send"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BRL", send["asset"])
	assert.Equal(t, "500.00", send["value"])

	// Verify response
	body, ok := result.Data["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "tx-001", body["id"])
}
