// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package midaz

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
)

// CreateTransactionID is the executor ID for the create-transaction operation.
const CreateTransactionID executor.ID = "midaz.create-transaction"

// newCreateTransactionExecutor creates the executor for
// POST /v1/organizations/{org_id}/ledgers/{ledger_id}/transactions/json.
// Creates a financial transaction in the Midaz ledger with source and destination
// account distributions, supporting pending transactions that require explicit commit.
func newCreateTransactionExecutor() (*base.Executor, error) {
	exec, err := base.NewExecutor(
		CreateTransactionID,
		"Create Transaction",
		"Midaz",
		"v1",
		ProviderID,
		createTransactionSchema,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return exec, nil
}

// createTransactionSchema defines the input for
// POST /v1/organizations/{org_id}/ledgers/{ledger_id}/transactions/json.
// Matches the Midaz API's CreateTransactionInput format.
const createTransactionSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "description": {
      "type": "string",
      "maxLength": 256,
      "description": "Transaction description"
    },
    "code": {
      "type": "string",
      "maxLength": 100,
      "description": "Transaction code"
    },
    "pending": {
      "type": "boolean",
      "default": false,
      "description": "Create as pending transaction that requires explicit commit"
    },
    "chartOfAccountsGroupName": {
      "type": "string",
      "description": "Chart of accounts group name"
    },
    "metadata": {
      "type": "object",
      "description": "Custom metadata key-value pairs"
    },
    "send": {
      "type": "object",
      "description": "Transaction send block defining asset, value, sources, and destinations",
      "required": ["asset", "value"],
      "properties": {
        "asset": {
          "type": "string",
          "description": "Asset code (e.g., BRL, USD)"
        },
        "value": {
          "type": "string",
          "pattern": "^[0-9]+(\\.[0-9]+)?$",
          "description": "Transaction amount as decimal string"
        },
        "source": {
          "type": "object",
          "description": "Source accounts for the transaction",
          "properties": {
            "from": {
              "type": "array",
              "description": "List of source account entries",
              "items": {
                "type": "object",
                "required": ["accountAlias"],
                "properties": {
                  "accountAlias": {
                    "type": "string",
                    "description": "Account alias (e.g., @user123)"
                  },
                  "amount": {
                    "type": "object",
                    "description": "Specific amount from this source",
                    "properties": {
                      "asset": {
                        "type": "string",
                        "description": "Asset code"
                      },
                      "value": {
                        "type": "string",
                        "pattern": "^[0-9]+(\\.[0-9]+)?$",
                        "description": "Amount as decimal string"
                      }
                    }
                  },
                  "share": {
                    "type": "object",
                    "description": "Proportional share from this source"
                  },
                  "description": {
                    "type": "string",
                    "description": "Description for this source entry"
                  },
                  "metadata": {
                    "type": "object",
                    "description": "Metadata for this source entry"
                  }
                }
              }
            }
          }
        },
        "distribute": {
          "type": "object",
          "description": "Destination accounts for the transaction",
          "properties": {
            "to": {
              "type": "array",
              "description": "List of destination account entries",
              "items": {
                "type": "object",
                "required": ["accountAlias"],
                "properties": {
                  "accountAlias": {
                    "type": "string",
                    "description": "Account alias (e.g., @merchant456)"
                  },
                  "amount": {
                    "type": "object",
                    "description": "Specific amount to this destination",
                    "properties": {
                      "asset": {
                        "type": "string",
                        "description": "Asset code"
                      },
                      "value": {
                        "type": "string",
                        "pattern": "^[0-9]+(\\.[0-9]+)?$",
                        "description": "Amount as decimal string"
                      }
                    }
                  },
                  "share": {
                    "type": "object",
                    "description": "Proportional share to this destination"
                  },
                  "description": {
                    "type": "string",
                    "description": "Description for this destination entry"
                  },
                  "metadata": {
                    "type": "object",
                    "description": "Metadata for this destination entry"
                  }
                }
              }
            }
          }
        }
      }
    }
  },
  "required": ["send"]
}`
