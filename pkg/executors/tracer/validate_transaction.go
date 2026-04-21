// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package tracer

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
)

// ValidateTransactionID is the executor ID for the validate-transaction operation.
const ValidateTransactionID executor.ID = "tracer.validate-transaction"

// newValidateTransactionExecutor creates the executor for POST /v1/validations.
// This is the core Tracer operation: validates a financial transaction against
// active compliance rules (CEL expressions) and financial limits (daily, weekly,
// monthly, per-transaction), returning a decision (ALLOW, DENY, REVIEW).
func newValidateTransactionExecutor() (*base.Executor, error) {
	exec, err := base.NewExecutor(
		ValidateTransactionID,
		"Validate Transaction",
		"Tracer",
		"v1",
		ProviderID,
		validateTransactionSchema,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return exec, nil
}

// validateTransactionSchema defines the input for POST /v1/validations.
// Matches the Tracer API's ValidationRequest format.
const validateTransactionSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "requestId": {
      "type": "string",
      "format": "uuid",
      "description": "Unique request identifier for idempotency"
    },
    "transactionType": {
      "type": "string",
      "enum": ["CARD", "WIRE", "PIX", "CRYPTO"],
      "description": "Type of financial transaction"
    },
    "subType": {
      "type": "string",
      "maxLength": 50,
      "description": "Transaction sub-type for more granular classification"
    },
    "amount": {
      "type": "string",
      "pattern": "^[0-9]+(\\.[0-9]+)?$",
      "description": "Transaction amount as decimal string (e.g., '1000.00')"
    },
    "currency": {
      "type": "string",
      "minLength": 3,
      "maxLength": 3,
      "pattern": "^[A-Z]{3}$",
      "description": "ISO 4217 currency code (e.g., BRL, USD)"
    },
    "transactionTimestamp": {
      "type": "string",
      "format": "date-time",
      "description": "Transaction timestamp in RFC3339 format"
    },
    "account": {
      "type": "object",
      "description": "Account context for the transaction",
      "required": ["accountId", "type", "status"],
      "properties": {
        "accountId": {
          "type": "string",
          "format": "uuid",
          "description": "Account identifier"
        },
        "type": {
          "type": "string",
          "enum": ["checking", "savings", "credit"],
          "description": "Account type"
        },
        "status": {
          "type": "string",
          "enum": ["active", "suspended", "closed"],
          "description": "Account status"
        },
        "metadata": {
          "type": "object",
          "description": "Additional account metadata"
        }
      }
    },
    "segment": {
      "type": "object",
      "description": "Optional segment context",
      "properties": {
        "segmentId": {
          "type": "string",
          "format": "uuid"
        },
        "name": {
          "type": "string"
        },
        "metadata": {
          "type": "object"
        }
      }
    },
    "portfolio": {
      "type": "object",
      "description": "Optional portfolio context",
      "properties": {
        "portfolioId": {
          "type": "string",
          "format": "uuid"
        },
        "name": {
          "type": "string"
        },
        "metadata": {
          "type": "object"
        }
      }
    },
    "merchant": {
      "type": "object",
      "description": "Optional merchant context",
      "properties": {
        "merchantId": {
          "type": "string",
          "format": "uuid",
          "description": "Merchant identifier"
        },
        "name": {
          "type": "string",
          "description": "Merchant name"
        },
        "category": {
          "type": "string",
          "minLength": 4,
          "maxLength": 4,
          "description": "Merchant Category Code (MCC)"
        },
        "country": {
          "type": "string",
          "minLength": 2,
          "maxLength": 2,
          "description": "ISO 3166-1 alpha-2 country code"
        },
        "metadata": {
          "type": "object"
        }
      }
    },
    "metadata": {
      "type": "object",
      "maxProperties": 50,
      "propertyNames": {
        "maxLength": 64
      },
      "description": "Additional transaction metadata (max 50 entries, max 64-char keys)"
    }
  },
  "required": ["requestId", "transactionType", "amount", "currency", "transactionTimestamp", "account"]
}`
