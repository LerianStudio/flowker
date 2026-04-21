// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package webhook provides a webhook trigger configuration validated via JSON Schema.
package webhook

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
)

// ID is the unique identifier for the webhook trigger.
const ID executor.TriggerID = "webhook"

// Version is the trigger version.
const Version = "v1"

// WebhookTrigger represents an HTTP webhook entrypoint.
type WebhookTrigger struct {
	*base.Trigger
}

// New creates a new webhook trigger with JSON Schema validation.
func New() (*WebhookTrigger, error) {
	t, err := base.NewTrigger(ID, "Webhook", Version, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook trigger: %w", err)
	}

	return &WebhookTrigger{Trigger: t}, nil
}

// Verify WebhookTrigger implements executor.Trigger.
var _ executor.Trigger = (*WebhookTrigger)(nil)

// schema defines the webhook trigger configuration contract.
// It keeps only essential fields for now; can be extended with header matching later.
const schema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["path", "method"],
  "properties": {
    "path": {
      "type": "string",
      "minLength": 1,
      "description": "HTTP path to receive the webhook (e.g., /hooks/order)"
    },
    "method": {
      "type": "string",
      "enum": ["GET", "POST", "PUT", "PATCH", "DELETE"],
      "description": "HTTP method the trigger will accept"
    },
    "verify_token": {
      "type": "string",
      "description": "Optional static token required in query/header for validation"
    }
  },
  "additionalProperties": false
}`
