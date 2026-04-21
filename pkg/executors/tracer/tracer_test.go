// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package tracer_test

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

func TestTracerValidateTransaction_Success(t *testing.T) {
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
		assert.Equal(t, "/v1/validations", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := map[string]any{
			"validationId":     "550e8400-e29b-41d4-a716-446655440000",
			"requestId":        "660e8400-e29b-41d4-a716-446655440001",
			"decision":         "ALLOW",
			"matchedRuleIds":   []string{},
			"evaluatedRuleIds": []string{"770e8400-e29b-41d4-a716-446655440002"},
			"reason":           "",
			"processingTimeMs": 42,
			"evaluatedAt":      "2026-03-18T12:00:00Z",
		}

		err = json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	catalog := executor.NewCatalog()
	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	runner, err := catalog.GetRunner("tracer.validate-transaction")
	require.NoError(t, err)

	requestBody := map[string]any{
		"requestId":            "660e8400-e29b-41d4-a716-446655440001",
		"transactionType":      "PIX",
		"amount":               "1000.00",
		"currency":             "BRL",
		"transactionTimestamp": "2026-03-18T12:00:00Z",
		"account": map[string]any{
			"accountId": "880e8400-e29b-41d4-a716-446655440003",
			"type":      "checking",
			"status":    "active",
		},
	}

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "POST",
			"url":    mockServer.URL + "/v1/validations",
			"headers": map[string]any{
				"X-API-Key":    "test-api-key",
				"Content-Type": "application/json",
			},
			"body": requestBody,
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)
	assert.Empty(t, result.Error)

	// Assert mock server received the correct request body
	assert.Equal(t, "660e8400-e29b-41d4-a716-446655440001", receivedBody["requestId"])
	assert.Equal(t, "PIX", receivedBody["transactionType"])
	assert.Equal(t, "1000.00", receivedBody["amount"])
	assert.Equal(t, "BRL", receivedBody["currency"])

	account, ok := receivedBody["account"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "880e8400-e29b-41d4-a716-446655440003", account["accountId"])

	// Assert X-API-Key header was received
	assert.Equal(t, "test-api-key", receivedHeaders.Get("X-API-Key"))

	// Assert response body contains decision: ALLOW
	body, ok := result.Data["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ALLOW", body["decision"])
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", body["validationId"])
}

func TestTracerValidateTransaction_Denied(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := map[string]any{
			"validationId":     "550e8400-e29b-41d4-a716-446655440010",
			"requestId":        "660e8400-e29b-41d4-a716-446655440011",
			"decision":         "DENY",
			"matchedRuleIds":   []string{"rule-high-risk-amount"},
			"evaluatedRuleIds": []string{"rule-high-risk-amount", "rule-country-block"},
			"reason":           "Transaction amount exceeds daily limit",
			"processingTimeMs": 15,
			"evaluatedAt":      "2026-03-18T12:00:00Z",
		}

		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	catalog := executor.NewCatalog()
	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	runner, err := catalog.GetRunner("tracer.validate-transaction")
	require.NoError(t, err)

	requestBody := map[string]any{
		"requestId":            "660e8400-e29b-41d4-a716-446655440011",
		"transactionType":      "WIRE",
		"amount":               "999999.99",
		"currency":             "USD",
		"transactionTimestamp": "2026-03-18T12:00:00Z",
		"account": map[string]any{
			"accountId": "880e8400-e29b-41d4-a716-446655440013",
			"type":      "checking",
			"status":    "active",
		},
	}

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "POST",
			"url":    mockServer.URL + "/v1/validations",
			"headers": map[string]any{
				"X-API-Key":    "test-api-key",
				"Content-Type": "application/json",
			},
			"body": requestBody,
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusSuccess, result.Status)

	body, ok := result.Data["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "DENY", body["decision"])
	assert.Equal(t, "Transaction amount exceeds daily limit", body["reason"])

	matchedRules, ok := body["matchedRuleIds"].([]any)
	require.True(t, ok)
	assert.Len(t, matchedRules, 1)
	assert.Equal(t, "rule-high-risk-amount", matchedRules[0])
}

func TestTracerListValidations_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/validations", r.URL.Path)
		assert.Equal(t, "ALLOW", r.URL.Query().Get("decision"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		resp := map[string]any{
			"items": []map[string]any{
				{
					"validationId": "aaa-111",
					"decision":     "ALLOW",
					"amount":       "500.00",
					"currency":     "BRL",
				},
				{
					"validationId": "bbb-222",
					"decision":     "ALLOW",
					"amount":       "250.00",
					"currency":     "BRL",
				},
			},
			"nextCursor": "cursor-abc",
			"total":      2,
		}

		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	catalog := executor.NewCatalog()
	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	runner, err := catalog.GetRunner("tracer.list-validations")
	require.NoError(t, err)

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "GET",
			"url":    mockServer.URL + "/v1/validations",
			"headers": map[string]any{
				"X-API-Key": "test-api-key",
			},
			"query": map[string]any{
				"decision": "ALLOW",
				"limit":    "10",
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
	assert.Len(t, items, 2)

	firstItem, ok := items[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ALLOW", firstItem["decision"])
}

func TestTracerValidateTransaction_AuthFailure(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "valid-key" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)

			resp := map[string]any{
				"error":   "unauthorized",
				"message": "Invalid or missing API key",
			}

			err := json.NewEncoder(w).Encode(resp)
			require.NoError(t, err)

			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	catalog := executor.NewCatalog()
	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	runner, err := catalog.GetRunner("tracer.validate-transaction")
	require.NoError(t, err)

	requestBody := map[string]any{
		"requestId":            "660e8400-e29b-41d4-a716-446655440021",
		"transactionType":      "CARD",
		"amount":               "100.00",
		"currency":             "USD",
		"transactionTimestamp": "2026-03-18T12:00:00Z",
		"account": map[string]any{
			"accountId": "880e8400-e29b-41d4-a716-446655440023",
			"type":      "checking",
			"status":    "active",
		},
	}

	input := executor.ExecutionInput{
		Config: map[string]any{
			"method": "POST",
			"url":    mockServer.URL + "/v1/validations",
			"headers": map[string]any{
				"X-API-Key":    "wrong-key",
				"Content-Type": "application/json",
			},
			"body": requestBody,
		},
	}

	result, err := runner.Execute(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, executor.ExecutionStatusError, result.Status)
	assert.Contains(t, result.Error, "401")

	// Verify response data includes the error body
	body, ok := result.Data["body"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "unauthorized", body["error"])
}

func TestTracerProviderSchema_Valid(t *testing.T) {
	catalog := executor.NewCatalog()
	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	// Validate validate-transaction executor schema
	exec, err := catalog.GetExecutor("tracer.validate-transaction")
	require.NoError(t, err)

	schema := exec.Schema()
	assert.NotEmpty(t, schema)

	var parsed map[string]any
	err = json.Unmarshal([]byte(schema), &parsed)
	require.NoError(t, err, "validate-transaction schema must be valid JSON")
	assert.Equal(t, "object", parsed["type"])
	assert.Contains(t, parsed, "properties")

	props, ok := parsed["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, props, "requestId")
	assert.Contains(t, props, "transactionType")
	assert.Contains(t, props, "amount")
	assert.Contains(t, props, "currency")
	assert.Contains(t, props, "account")

	// Assert required fields
	validateRequired, ok := parsed["required"].([]any)
	require.True(t, ok, "validate-transaction schema must have required array")
	assert.Contains(t, validateRequired, "requestId")
	assert.Contains(t, validateRequired, "transactionType")
	assert.Contains(t, validateRequired, "amount")
	assert.Contains(t, validateRequired, "currency")
	assert.Contains(t, validateRequired, "account")

	// Validate list-validations executor schema
	listExec, err := catalog.GetExecutor("tracer.list-validations")
	require.NoError(t, err)

	listSchema := listExec.Schema()
	assert.NotEmpty(t, listSchema)

	var listParsed map[string]any
	err = json.Unmarshal([]byte(listSchema), &listParsed)
	require.NoError(t, err, "list-validations schema must be valid JSON")
	assert.Equal(t, "object", listParsed["type"])

	listProps, ok := listParsed["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, listProps, "decision")
	assert.Contains(t, listProps, "transactionType")
	assert.Contains(t, listProps, "limit")

	// Validate provider config schema
	provider, err := catalog.GetProvider("tracer")
	require.NoError(t, err)

	providerSchema := provider.ConfigSchema()
	assert.NotEmpty(t, providerSchema)

	var providerParsed map[string]any
	err = json.Unmarshal([]byte(providerSchema), &providerParsed)
	require.NoError(t, err, "provider config schema must be valid JSON")
	assert.Equal(t, "object", providerParsed["type"])

	providerProps, ok := providerParsed["properties"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, providerProps, "base_url")
	assert.Contains(t, providerProps, "api_key")

	// Assert provider required fields
	providerRequired, ok := providerParsed["required"].([]any)
	require.True(t, ok, "provider schema must have required array")
	assert.Contains(t, providerRequired, "base_url")
	assert.Contains(t, providerRequired, "api_key")
}
