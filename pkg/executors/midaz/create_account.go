// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package midaz

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
)

// CreateAccountID is the executor ID for the create-account operation.
const CreateAccountID executor.ID = "midaz.create-account"

// newCreateAccountExecutor creates the executor for
// POST /v1/organizations/{org_id}/ledgers/{ledger_id}/accounts.
// Creates a new account in the Midaz ledger with the specified asset code,
// type, and optional attributes like alias, parent account, and metadata.
func newCreateAccountExecutor() (*base.Executor, error) {
	exec, err := base.NewExecutor(
		CreateAccountID,
		"Create Account",
		"Midaz",
		"v1",
		ProviderID,
		createAccountSchema,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return exec, nil
}

// createAccountSchema defines the input for
// POST /v1/organizations/{org_id}/ledgers/{ledger_id}/accounts.
// Matches the Midaz API's CreateAccountInput format.
const createAccountSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "name": {
      "type": "string",
      "maxLength": 256,
      "description": "Account name"
    },
    "assetCode": {
      "type": "string",
      "maxLength": 100,
      "description": "Asset code (e.g., BRL, USD)"
    },
    "type": {
      "type": "string",
      "maxLength": 256,
      "description": "Account type"
    },
    "alias": {
      "type": "string",
      "maxLength": 100,
      "description": "Account alias (e.g., @user123)"
    },
    "parentAccountId": {
      "type": "string",
      "format": "uuid",
      "description": "Parent account ID for hierarchical account structures"
    },
    "portfolioId": {
      "type": "string",
      "format": "uuid",
      "description": "Portfolio ID to associate the account with"
    },
    "segmentId": {
      "type": "string",
      "format": "uuid",
      "description": "Segment ID to associate the account with"
    },
    "entityId": {
      "type": "string",
      "maxLength": 256,
      "description": "External entity identifier"
    },
    "status": {
      "type": "object",
      "description": "Account status",
      "properties": {
        "code": {
          "type": "string",
          "description": "Status code"
        },
        "description": {
          "type": "string",
          "description": "Status description"
        }
      }
    },
    "metadata": {
      "type": "object",
      "description": "Custom metadata key-value pairs"
    }
  },
  "required": ["assetCode", "type"]
}`
