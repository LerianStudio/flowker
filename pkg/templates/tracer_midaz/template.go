// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package tracer_midaz provides the tracer-midaz-validation workflow template.
// This template creates a workflow that validates transactions via Tracer
// and creates transactions in Midaz when approved.
package tracer_midaz

import (
	"errors"
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
	"github.com/LerianStudio/flowker/pkg/model"
)

// ErrNilCatalog is returned when Register is called with a nil catalog.
var ErrNilCatalog = errors.New("catalog cannot be nil")

// TemplateID is the unique identifier for the tracer-midaz-validation template.
const TemplateID executor.TemplateID = "tracer-midaz-validation"

// Register registers the tracer-midaz-validation template into the given catalog.
func Register(catalog executor.Catalog) error {
	if catalog == nil {
		return ErrNilCatalog
	}

	tmpl, err := base.NewTemplate(
		TemplateID,
		"Tracer Validation + Midaz Transaction",
		"Receives a webhook request, validates the transaction via Tracer, and creates a transaction in Midaz if approved",
		"v1",
		"Compliance",
		paramSchema,
		build,
		[]executor.ProviderConfigField{
			{ParamName: "tracerProviderConfigId", ProviderID: "tracer"},
			{ParamName: "midazProviderConfigId", ProviderID: "midaz"},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create tracer-midaz-validation template: %w", err)
	}

	return catalog.RegisterTemplate(tmpl)
}

// build generates a complete CreateWorkflowInput from the given parameters.
func build(params map[string]any) (any, error) {
	workflowName, ok := params["workflowName"].(string)
	if !ok || workflowName == "" {
		return nil, fmt.Errorf("workflowName is required")
	}

	tracerProviderConfigID, ok := params["tracerProviderConfigId"].(string)
	if !ok || tracerProviderConfigID == "" {
		return nil, fmt.Errorf("tracerProviderConfigId is required")
	}

	midazProviderConfigID, ok := params["midazProviderConfigId"].(string)
	if !ok || midazProviderConfigID == "" {
		return nil, fmt.Errorf("midazProviderConfigId is required")
	}

	webhookPath := "/validate-and-transact"
	if p, ok := params["webhookPath"].(string); ok && p != "" {
		webhookPath = p
	}

	webhookMethod := "POST"
	if m, ok := params["webhookMethod"].(string); ok && m != "" {
		webhookMethod = m
	}

	var description *string
	if d, ok := params["workflowDescription"].(string); ok && d != "" {
		description = &d
	}

	// Build node names
	webhookTriggerName := "Webhook Trigger"
	tracerValidateName := "Tracer Validate Transaction"
	decisionCheckName := "Decision Check"
	midazCreateTxName := "Midaz Create Transaction"

	nodes := []model.WorkflowNodeInput{
		{
			ID:       "webhook-trigger",
			Type:     "trigger",
			Name:     &webhookTriggerName,
			Position: model.PositionInput{X: 250, Y: 0},
			Data: map[string]any{
				"triggerType": "webhook",
				"path":        webhookPath,
				"method":      webhookMethod,
			},
		},
		{
			ID:       "tracer-validate",
			Type:     "executor",
			Name:     &tracerValidateName,
			Position: model.PositionInput{X: 250, Y: 150},
			Data: map[string]any{
				"executorId":       "tracer.validate-transaction",
				"providerConfigId": tracerProviderConfigID,
				"method":           "POST",
				"path":             "/v1/validations",
				"inputMapping": []map[string]any{
					{
						"source": "workflow.requestId",
						"target": "requestId",
					},
					{
						"source": "workflow.transactionType",
						"target": "transactionType",
					},
					{
						"source": "workflow.amount",
						"target": "amount",
					},
					{
						"source": "workflow.currency",
						"target": "currency",
					},
					{
						"source": "workflow.transactionTimestamp",
						"target": "transactionTimestamp",
					},
					{
						"source": "workflow.account",
						"target": "account",
					},
				},
			},
		},
		{
			ID:       "decision-check",
			Type:     "conditional",
			Name:     &decisionCheckName,
			Position: model.PositionInput{X: 250, Y: 300},
			Data: map[string]any{
				"condition": `tracer-validate.body.decision == "ALLOW"`,
			},
		},
		{
			ID:       "midaz-create-tx",
			Type:     "executor",
			Name:     &midazCreateTxName,
			Position: model.PositionInput{X: 100, Y: 450},
			Data: map[string]any{
				"executorId":       "midaz.create-transaction",
				"providerConfigId": midazProviderConfigID,
				"inputMapping": []map[string]any{
					{
						"source": "workflow.transaction",
						"target": "send",
					},
					{
						"source": "workflow.transactionDescription",
						"target": "description",
					},
				},
			},
		},
	}

	trueCondition := "true"
	trueLabel := "Approved"

	edges := []model.WorkflowEdgeInput{
		{
			ID:     "webhook-to-tracer",
			Source: "webhook-trigger",
			Target: "tracer-validate",
		},
		{
			ID:     "tracer-to-decision",
			Source: "tracer-validate",
			Target: "decision-check",
		},
		{
			ID:           "decision-to-midaz",
			Source:       "decision-check",
			Target:       "midaz-create-tx",
			SourceHandle: &trueCondition,
			Label:        &trueLabel,
		},
	}

	return &model.CreateWorkflowInput{
		Name:        workflowName,
		Description: description,
		Nodes:       nodes,
		Edges:       edges,
		Metadata: map[string]any{
			"templateId":      string(TemplateID),
			"templateVersion": "v1",
		},
	}, nil
}

// paramSchema defines the JSON Schema for template parameters.
const paramSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["workflowName", "tracerProviderConfigId", "midazProviderConfigId"],
  "properties": {
    "workflowName": {
      "type": "string",
      "minLength": 1,
      "maxLength": 100,
      "description": "Name for the created workflow"
    },
    "workflowDescription": {
      "type": "string",
      "maxLength": 500,
      "description": "Optional description"
    },
    "tracerProviderConfigId": {
      "type": "string",
      "format": "uuid",
      "description": "Provider configuration ID for Tracer (transaction validation)"
    },
    "midazProviderConfigId": {
      "type": "string",
      "format": "uuid",
      "description": "Provider configuration ID for Midaz (ledger transaction creation)"
    },
    "webhookPath": {
      "type": "string",
      "default": "/validate-and-transact",
      "description": "Webhook endpoint path"
    },
    "webhookMethod": {
      "type": "string",
      "enum": ["POST"],
      "default": "POST"
    }
  }
}`
