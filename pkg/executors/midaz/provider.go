// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package midaz provides the Lerian Midaz provider for core banking ledger
// operations including transactions, accounts, and balance queries.
package midaz

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
	httpExecutor "github.com/LerianStudio/flowker/pkg/executors/http"
)

// ProviderID is the unique identifier for the Midaz provider.
const ProviderID executor.ProviderID = "midaz"

// midazProvider wraps base.Provider and implements executor.InputBuilder
// to provide Midaz-specific URL routing and auth translation.
type midazProvider struct {
	*base.Provider
}

// BuildInput implements executor.InputBuilder for Midaz-specific execution input.
func (p *midazProvider) BuildInput(providerConfig map[string]any, executorID executor.ID, nodeData map[string]any, requestBody []byte) (executor.ExecutionInput, error) {
	return BuildInput(providerConfig, executorID, nodeData, requestBody)
}

// Compile-time check that midazProvider implements both Provider and InputBuilder.
var _ executor.Provider = (*midazProvider)(nil)
var _ executor.InputBuilder = (*midazProvider)(nil)

// Register registers the Midaz provider with all its executors into the given catalog.
func Register(catalog executor.Catalog) error {
	if catalog == nil {
		return nil
	}

	baseProvider, err := base.NewProvider(
		ProviderID,
		"Midaz",
		"Lerian Midaz core banking ledger provider for managing transactions, accounts, and balances",
		"v1",
		providerConfigSchema,
	)
	if err != nil {
		return fmt.Errorf("failed to create Midaz provider: %w", err)
	}

	provider := &midazProvider{Provider: baseProvider}

	createTxExec, err := newCreateTransactionExecutor()
	if err != nil {
		return fmt.Errorf("failed to create Midaz create-transaction executor: %w", err)
	}

	getBalanceExec, err := newGetAccountBalanceExecutor()
	if err != nil {
		return fmt.Errorf("failed to create Midaz get-account-balance executor: %w", err)
	}

	createAccountExec, err := newCreateAccountExecutor()
	if err != nil {
		return fmt.Errorf("failed to create Midaz create-account executor: %w", err)
	}

	getAccountExec, err := newGetAccountExecutor()
	if err != nil {
		return fmt.Errorf("failed to create Midaz get-account executor: %w", err)
	}

	return catalog.RegisterProvider(provider, []executor.ExecutorRegistration{
		{
			Executor: createTxExec,
			Runner:   httpExecutor.NewRunner(),
		},
		{
			Executor: getBalanceExec,
			Runner:   httpExecutor.NewRunner(),
		},
		{
			Executor: createAccountExec,
			Runner:   httpExecutor.NewRunner(),
		},
		{
			Executor: getAccountExec,
			Runner:   httpExecutor.NewRunner(),
		},
	})
}

// providerConfigSchema defines the connection and authentication settings for a Midaz instance.
// Midaz uses JWT Bearer Token authentication via OIDC (Casdoor) with client_credentials flow.
const providerConfigSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "onboarding_base_url": {
      "type": "string",
      "format": "uri",
      "description": "Midaz Onboarding component base URL (port 3000) for organizations, ledgers, assets, and accounts"
    },
    "transaction_base_url": {
      "type": "string",
      "format": "uri",
      "description": "Midaz Transaction component base URL (port 3001) for transactions, operations, and balances"
    },
    "organization_id": {
      "type": "string",
      "format": "uuid",
      "description": "Default organization ID for the nested URL path context"
    },
    "ledger_id": {
      "type": "string",
      "format": "uuid",
      "description": "Default ledger ID for the nested URL path context"
    },
    "auth": {
      "type": "object",
      "description": "OIDC authentication configuration for JWT Bearer Token via client_credentials flow",
      "properties": {
        "issuer_url": {
          "type": "string",
          "format": "uri",
          "description": "OIDC issuer URL for token exchange (e.g., Casdoor)"
        },
        "client_id": {
          "type": "string",
          "minLength": 1,
          "description": "OAuth2 client ID for client_credentials flow"
        },
        "client_secret": {
          "type": "string",
          "minLength": 1,
          "description": "OAuth2 client secret for client_credentials flow"
        },
        "scopes": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "OAuth2 scopes to request during token exchange"
        }
      },
      "required": ["issuer_url", "client_id", "client_secret"]
    }
  },
  "required": ["onboarding_base_url", "transaction_base_url", "organization_id", "ledger_id"]
}`
