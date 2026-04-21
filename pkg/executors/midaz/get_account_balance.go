// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package midaz

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
)

// GetAccountBalanceID is the executor ID for the get-account-balance operation.
const GetAccountBalanceID executor.ID = "midaz.get-account-balance"

// newGetAccountBalanceExecutor creates the executor for
// GET /v1/organizations/{org_id}/ledgers/{ledger_id}/accounts/{account_id}/balances.
// Retrieves the balance information for a specific account, including available
// and on-hold amounts, with cursor-based pagination support.
func newGetAccountBalanceExecutor() (*base.Executor, error) {
	exec, err := base.NewExecutor(
		GetAccountBalanceID,
		"Get Account Balance",
		"Midaz",
		"v1",
		ProviderID,
		getAccountBalanceSchema,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return exec, nil
}

// getAccountBalanceSchema defines the query parameters for
// GET /v1/organizations/{org_id}/ledgers/{ledger_id}/accounts/{account_id}/balances.
const getAccountBalanceSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "accountId": {
      "type": "string",
      "format": "uuid",
      "description": "Account ID to retrieve balances for"
    },
    "limit": {
      "type": "integer",
      "minimum": 1,
      "maximum": 100,
      "default": 10,
      "description": "Number of items per page"
    },
    "cursor": {
      "type": "string",
      "description": "Pagination cursor for next page"
    }
  },
  "required": ["accountId"]
}`
