// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package midaz

import (
	"encoding/json"
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
)

// executorMeta holds the HTTP routing metadata for a Midaz executor.
type executorMeta struct {
	method       string
	baseURLKey   string
	pathTemplate string
	pathParams   []string // keys to extract: from providerConfig first, then nodeData
}

// executorRoutes maps each Midaz executor to its HTTP routing metadata.
var executorRoutes = map[executor.ID]executorMeta{
	CreateTransactionID: {"POST", "transaction_base_url", "/v1/organizations/%s/ledgers/%s/transactions/json", []string{"organization_id", "ledger_id"}},
	CreateAccountID:     {"POST", "onboarding_base_url", "/v1/organizations/%s/ledgers/%s/accounts", []string{"organization_id", "ledger_id"}},
	GetAccountID:        {"GET", "onboarding_base_url", "/v1/organizations/%s/ledgers/%s/accounts/%s", []string{"organization_id", "ledger_id", "accountId"}},
	GetAccountBalanceID: {"GET", "transaction_base_url", "/v1/organizations/%s/ledgers/%s/accounts/%s/balances", []string{"organization_id", "ledger_id", "accountId"}},
}

// BuildInput constructs an ExecutionInput for a Midaz executor.
// It selects the correct base URL, builds the path with org/ledger/account IDs,
// and translates the nested auth block into the format the HTTP runner expects.
func BuildInput(providerConfig map[string]any, executorID executor.ID, nodeData map[string]any, requestBody []byte) (executor.ExecutionInput, error) {
	route, ok := executorRoutes[executorID]
	if !ok {
		return executor.ExecutionInput{}, fmt.Errorf("unknown Midaz executor: %s", executorID)
	}

	// Extract base URL
	baseURL, ok := providerConfig[route.baseURLKey].(string)
	if !ok || baseURL == "" {
		return executor.ExecutionInput{}, fmt.Errorf("missing required field %q in provider config", route.baseURLKey)
	}

	// Extract path parameters
	params, err := resolvePathParams(route.pathParams, providerConfig, nodeData, requestBody, executorID)
	if err != nil {
		return executor.ExecutionInput{}, err
	}

	// Build URL
	path := fmt.Sprintf(route.pathTemplate, params...)
	fullURL := baseURL + path

	// Build auth config
	authConfig := buildMidazAuth(providerConfig)

	// Build ExecutionInput.Config
	config := map[string]any{
		"method": route.method,
		"url":    fullURL,
	}

	if authConfig != nil {
		config["auth"] = authConfig
	}

	if len(requestBody) > 0 && route.method == "POST" {
		var body any
		if err := json.Unmarshal(requestBody, &body); err == nil {
			config["body"] = body
		} else {
			config["body"] = string(requestBody)
		}
	}

	return executor.ExecutionInput{Config: config}, nil
}

// resolvePathParams extracts path parameter values from provider config, node data,
// or request body (parsed as JSON) in order of precedence.
func resolvePathParams(keys []string, providerConfig, nodeData map[string]any, requestBody []byte, executorID executor.ID) ([]any, error) {
	var bodyMap map[string]any
	if len(requestBody) > 0 {
		_ = json.Unmarshal(requestBody, &bodyMap) // best-effort parse
	}

	params := make([]any, 0, len(keys))

	for _, key := range keys {
		if val := extractStringParam(key, providerConfig, nodeData, bodyMap); val != "" {
			params = append(params, val)
			continue
		}

		return nil, fmt.Errorf("missing required path parameter %q for executor %s", key, executorID)
	}

	return params, nil
}

// extractStringParam tries to find a string value for the given key from multiple sources.
func extractStringParam(key string, sources ...map[string]any) string {
	for _, src := range sources {
		if val, ok := src[key].(string); ok && val != "" {
			return val
		}
	}

	return ""
}

// buildMidazAuth translates the nested Midaz auth block into the format
// that the HTTP auth factory (auth.NewFromConfig) expects.
func buildMidazAuth(providerConfig map[string]any) map[string]any {
	authBlock, ok := providerConfig["auth"].(map[string]any)
	if !ok || len(authBlock) == 0 {
		return nil
	}

	return map[string]any{
		"type":   "oidc_client_credentials",
		"config": authBlock,
	}
}
