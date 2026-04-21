// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package midaz

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
)

// GetAccountID is the executor ID for the get-account operation.
const GetAccountID executor.ID = "midaz.get-account"

// newGetAccountExecutor creates the executor for
// GET /v1/organizations/{org_id}/ledgers/{ledger_id}/accounts/{account_id}.
// Retrieves the details of a specific account by its ID.
func newGetAccountExecutor() (*base.Executor, error) {
	exec, err := base.NewExecutor(
		GetAccountID,
		"Get Account",
		"Midaz",
		"v1",
		ProviderID,
		getAccountSchema,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return exec, nil
}

// getAccountSchema defines the input for
// GET /v1/organizations/{org_id}/ledgers/{ledger_id}/accounts/{account_id}.
const getAccountSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "accountId": {
      "type": "string",
      "format": "uuid",
      "description": "Account ID to retrieve"
    }
  },
  "required": ["accountId"]
}`
