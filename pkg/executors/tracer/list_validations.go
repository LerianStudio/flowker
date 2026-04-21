// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package tracer

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
)

// ListValidationsID is the executor ID for the list-validations operation.
const ListValidationsID executor.ID = "tracer.list-validations"

// newListValidationsExecutor creates the executor for GET /v1/validations.
// Retrieves past transaction validation results with pagination and filtering
// by decision, transaction type, and account.
func newListValidationsExecutor() (*base.Executor, error) {
	exec, err := base.NewExecutor(
		ListValidationsID,
		"List Validations",
		"Tracer",
		"v1",
		ProviderID,
		listValidationsSchema,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return exec, nil
}

// listValidationsSchema defines the query parameters for GET /v1/validations.
const listValidationsSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "decision": {
      "type": "string",
      "enum": ["ALLOW", "DENY", "REVIEW"],
      "description": "Filter by validation decision"
    },
    "transactionType": {
      "type": "string",
      "enum": ["CARD", "WIRE", "PIX", "CRYPTO"],
      "description": "Filter by transaction type"
    },
    "accountId": {
      "type": "string",
      "format": "uuid",
      "description": "Filter by account ID"
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
    },
    "sortBy": {
      "type": "string",
      "enum": ["createdAt"],
      "default": "createdAt",
      "description": "Sort field"
    },
    "sortOrder": {
      "type": "string",
      "enum": ["ASC", "DESC"],
      "default": "DESC",
      "description": "Sort direction"
    }
  },
  "additionalProperties": false
}`
