// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package tracer provides the Lerian Tracer provider for transaction validation
// against compliance rules and financial limits.
package tracer

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
	httpExecutor "github.com/LerianStudio/flowker/pkg/executors/http"
)

// ProviderID is the unique identifier for the Tracer provider.
const ProviderID executor.ProviderID = "tracer"

// Register registers the Tracer provider with all its executors into the given catalog.
func Register(catalog executor.Catalog) error {
	if catalog == nil {
		return nil
	}

	provider, err := base.NewProvider(
		ProviderID,
		"Tracer",
		"Lerian Tracer provider for transaction validation against compliance rules and financial limits",
		"v1",
		providerConfigSchema,
	)
	if err != nil {
		return fmt.Errorf("failed to create Tracer provider: %w", err)
	}

	validateTxExec, err := newValidateTransactionExecutor()
	if err != nil {
		return fmt.Errorf("failed to create Tracer validate-transaction executor: %w", err)
	}

	listValidationsExec, err := newListValidationsExecutor()
	if err != nil {
		return fmt.Errorf("failed to create Tracer list-validations executor: %w", err)
	}

	return catalog.RegisterProvider(provider, []executor.ExecutorRegistration{
		{
			Executor: validateTxExec,
			Runner:   httpExecutor.NewRunner(),
		},
		{
			Executor: listValidationsExec,
			Runner:   httpExecutor.NewRunner(),
		},
	})
}

// providerConfigSchema defines the connection and authentication settings for a Tracer instance.
// Authentication uses API Key via the X-API-Key header, which is the primary auth method
// supported by the Tracer API.
const providerConfigSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "base_url": {
      "type": "string",
      "format": "uri",
      "description": "Tracer API base URL (e.g., https://tracer.lerian.studio)"
    },
    "api_key": {
      "type": "string",
      "minLength": 1,
      "description": "API key for authenticating with Tracer via X-API-Key header"
    }
  },
  "required": ["base_url", "api_key"]
}`
