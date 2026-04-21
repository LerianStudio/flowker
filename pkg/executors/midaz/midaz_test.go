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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMidazCreateTransaction_Success(t *testing.T) {
	var receivedBody map[string]any
	var receivedHeaders http.Header

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		err = json.Unmarshal(body, &receivedBody)
		require.NoError(t, err)

		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/organizations/org-001/ledgers/ledger-001/transactions/json", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		resp := map[string]any{
			"id":          "tx-550e8400-e29b-41d4-a716-446655440000",
			"description": "Payment for services",
			"status": map[string]any{
				"code":        "APPROVED",
				"description": "Transaction approved",
			},
			"amount":    "1500.00",
			"assetCode": "BRL",
			"createdAt": "2026-03-18T12:00:00Z",
		}

		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	catalog := executor.NewCatalog()
	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	runner, err := catalog.GetRunner("midaz.create-transaction")
	require.NoError(t, err)

	requestBody := map[string]any{
		"description": "Payment for services",
		"code":        "PAY-001",
		"pending":     false,
		"send": map[string]any{
			"asset": "BRL",
			"value": "1500.00",
			"source": map[string]any{
				"from": []map[string]any{
					{
						"accountAlias": "@user123",
					},
				},
			},
			"distribute": map[string]any{
				"to": []map[string]any{
					{
						"accountAlias": "@merchant456",
					},
				},
			},
		},
	}

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "POST",
			"url":    mockServer.URL + "/v1/organizations/org-001/ledgers/ledger-001/transactions/json",
			"headers": map[string]any{
				"Authorization": "Bearer test-token",
				"Content-Type":  "application/json",
			},
			"body": requestBody,
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Empty(t, result.Error)

	// Assert mock server received the correct request body
	send, ok := receivedBody["send"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "BRL", send["asset"])
	assert.Equal(t, "1500.00", send["value"])

	source, ok := send["source"].(map[string]any)
	require.True(t, ok)
	from, ok := source["from"].([]any)
	require.True(t, ok)
	assert.Len(t, from, 1)

	distribute, ok := send["distribute"].(map[string]any)
	require.True(t, ok)
	to, ok := distribute["to"].([]any)
	require.True(t, ok)
	assert.Len(t, to, 1)

	// Assert Authorization Bearer header was received
	assert.Equal(t, "Bearer test-token", receivedHeaders.Get("Authorization"))

	// Assert response body contains transaction data
	body, ok := result.Data["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "tx-550e8400-e29b-41d4-a716-446655440000", body["id"])
	assert.Equal(t, "1500.00", body["amount"])
}

func TestMidazGetAccountBalance_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/organizations/org-001/ledgers/ledger-001/accounts/acc-001/balances", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := map[string]any{
			"items": []map[string]any{
				{
					"id":        "bal-001",
					"accountId": "acc-001",
					"assetCode": "BRL",
					"available": "10000.00",
					"onHold":    "500.00",
					"scale":     2,
				},
			},
			"nextCursor": "",
			"total":      1,
		}

		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	catalog := executor.NewCatalog()
	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	runner, err := catalog.GetRunner("midaz.get-account-balance")
	require.NoError(t, err)

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    mockServer.URL + "/v1/organizations/org-001/ledgers/ledger-001/accounts/acc-001/balances",
			"headers": map[string]any{
				"Authorization": "Bearer test-token",
			},
			"query": map[string]any{
				"limit": "10",
			},
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Empty(t, result.Error)

	body, ok := result.Data["body"].(map[string]any)
	require.True(t, ok)

	items, ok := body["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)

	firstItem, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "10000.00", firstItem["available"])
	assert.Equal(t, "500.00", firstItem["onHold"])
}

func TestMidazCreateAccount_Success(t *testing.T) {
	var receivedBody map[string]any
	var receivedHeaders http.Header

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		err = json.Unmarshal(body, &receivedBody)
		require.NoError(t, err)

		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/organizations/org-001/ledgers/ledger-001/accounts", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		resp := map[string]any{
			"id":        "acc-new-001",
			"name":      "User Checking Account",
			"assetCode": "BRL",
			"type":      "deposit",
			"alias":     "@user789",
			"status": map[string]any{
				"code":        "active",
				"description": "Account is active",
			},
			"createdAt": "2026-03-18T12:00:00Z",
		}

		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	catalog := executor.NewCatalog()
	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	runner, err := catalog.GetRunner("midaz.create-account")
	require.NoError(t, err)

	requestBody := map[string]any{
		"name":      "User Checking Account",
		"assetCode": "BRL",
		"type":      "deposit",
		"alias":     "@user789",
	}

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "POST",
			"url":    mockServer.URL + "/v1/organizations/org-001/ledgers/ledger-001/accounts",
			"headers": map[string]any{
				"Authorization": "Bearer test-token",
				"Content-Type":  "application/json",
			},
			"body": requestBody,
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Empty(t, result.Error)

	// Assert mock server received correct body fields
	assert.Equal(t, "BRL", receivedBody["assetCode"])
	assert.Equal(t, "deposit", receivedBody["type"])

	// Assert Authorization Bearer header was received
	assert.Equal(t, "Bearer test-token", receivedHeaders.Get("Authorization"))

	// Assert response body contains account data
	body, ok := result.Data["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "acc-new-001", body["id"])
	assert.Equal(t, "BRL", body["assetCode"])
}

func TestMidazGetAccount_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/organizations/org-001/ledgers/ledger-001/accounts/acc-001", r.URL.Path)
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := map[string]any{
			"id":        "acc-001",
			"name":      "Main Account",
			"assetCode": "BRL",
			"type":      "deposit",
			"alias":     "@main",
			"status": map[string]any{
				"code":        "active",
				"description": "Account is active",
			},
			"createdAt": "2026-03-18T10:00:00Z",
			"updatedAt": "2026-03-18T12:00:00Z",
		}

		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	catalog := executor.NewCatalog()
	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	runner, err := catalog.GetRunner("midaz.get-account")
	require.NoError(t, err)

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    mockServer.URL + "/v1/organizations/org-001/ledgers/ledger-001/accounts/acc-001",
			"headers": map[string]any{
				"Authorization": "Bearer test-token",
			},
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Empty(t, result.Error)

	body, ok := result.Data["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "acc-001", body["id"])
	assert.Equal(t, "Main Account", body["name"])
	assert.Equal(t, "BRL", body["assetCode"])
	assert.Equal(t, "deposit", body["type"])
}

func TestMidazProviderSchema_Valid(t *testing.T) {
	catalog := executor.NewCatalog()
	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	// Validate create-transaction executor schema
	createTxExec, err := catalog.GetExecutor("midaz.create-transaction")
	require.NoError(t, err)

	createTxSchema := createTxExec.Schema()
	assert.NotEmpty(t, createTxSchema)

	var createTxParsed map[string]any
	err = json.Unmarshal([]byte(createTxSchema), &createTxParsed)
	require.NoError(t, err, "create-transaction schema must be valid JSON")
	assert.Equal(t, "object", createTxParsed["type"])
	assert.Contains(t, createTxParsed, "properties")

	createTxProps, ok := createTxParsed["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, createTxProps, "send")
	assert.Contains(t, createTxProps, "description")
	assert.Contains(t, createTxProps, "code")
	assert.Contains(t, createTxProps, "pending")
	assert.Contains(t, createTxProps, "metadata")

	createTxRequired, ok := createTxParsed["required"].([]any)
	require.True(t, ok, "create-transaction schema must have required array")
	assert.Contains(t, createTxRequired, "send")

	// Validate get-account-balance executor schema
	getBalanceExec, err := catalog.GetExecutor("midaz.get-account-balance")
	require.NoError(t, err)

	getBalanceSchema := getBalanceExec.Schema()
	assert.NotEmpty(t, getBalanceSchema)

	var getBalanceParsed map[string]any
	err = json.Unmarshal([]byte(getBalanceSchema), &getBalanceParsed)
	require.NoError(t, err, "get-account-balance schema must be valid JSON")
	assert.Equal(t, "object", getBalanceParsed["type"])

	getBalanceProps, ok := getBalanceParsed["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, getBalanceProps, "accountId")
	assert.Contains(t, getBalanceProps, "limit")
	assert.Contains(t, getBalanceProps, "cursor")

	getBalanceRequired, ok := getBalanceParsed["required"].([]any)
	require.True(t, ok, "get-account-balance schema must have required array")
	assert.Contains(t, getBalanceRequired, "accountId")

	// Validate create-account executor schema
	createAccExec, err := catalog.GetExecutor("midaz.create-account")
	require.NoError(t, err)

	createAccSchema := createAccExec.Schema()
	assert.NotEmpty(t, createAccSchema)

	var createAccParsed map[string]any
	err = json.Unmarshal([]byte(createAccSchema), &createAccParsed)
	require.NoError(t, err, "create-account schema must be valid JSON")
	assert.Equal(t, "object", createAccParsed["type"])

	createAccProps, ok := createAccParsed["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, createAccProps, "assetCode")
	assert.Contains(t, createAccProps, "type")
	assert.Contains(t, createAccProps, "name")
	assert.Contains(t, createAccProps, "alias")

	createAccRequired, ok := createAccParsed["required"].([]any)
	require.True(t, ok, "create-account schema must have required array")
	assert.Contains(t, createAccRequired, "assetCode")
	assert.Contains(t, createAccRequired, "type")

	// Validate get-account executor schema
	getAccExec, err := catalog.GetExecutor("midaz.get-account")
	require.NoError(t, err)

	getAccSchema := getAccExec.Schema()
	assert.NotEmpty(t, getAccSchema)

	var getAccParsed map[string]any
	err = json.Unmarshal([]byte(getAccSchema), &getAccParsed)
	require.NoError(t, err, "get-account schema must be valid JSON")
	assert.Equal(t, "object", getAccParsed["type"])

	getAccProps, ok := getAccParsed["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, getAccProps, "accountId")

	getAccRequired, ok := getAccParsed["required"].([]any)
	require.True(t, ok, "get-account schema must have required array")
	assert.Contains(t, getAccRequired, "accountId")

	// Validate provider config schema
	provider, err := catalog.GetProvider("midaz")
	require.NoError(t, err)

	providerSchema := provider.ConfigSchema()
	assert.NotEmpty(t, providerSchema)

	var providerParsed map[string]any
	err = json.Unmarshal([]byte(providerSchema), &providerParsed)
	require.NoError(t, err, "provider config schema must be valid JSON")
	assert.Equal(t, "object", providerParsed["type"])

	providerProps, ok := providerParsed["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, providerProps, "onboarding_base_url")
	assert.Contains(t, providerProps, "transaction_base_url")
	assert.Contains(t, providerProps, "organization_id")
	assert.Contains(t, providerProps, "ledger_id")
	assert.Contains(t, providerProps, "auth")

	providerRequired, ok := providerParsed["required"].([]any)
	require.True(t, ok, "provider schema must have required array")
	assert.Contains(t, providerRequired, "onboarding_base_url")
	assert.Contains(t, providerRequired, "transaction_base_url")
	assert.Contains(t, providerRequired, "organization_id")
	assert.Contains(t, providerRequired, "ledger_id")
}
