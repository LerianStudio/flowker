// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package services

import (
	"testing"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/webhook"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPopulateRegistryFromWorkflows(t *testing.T) {
	registry := webhook.NewRegistry()

	wfActive := model.NewWorkflowFromDB(
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		"active-webhook-workflow",
		nil,
		model.WorkflowStatusActive,
		[]model.WorkflowNode{
			model.NewWorkflowNodeFromDB("trigger-1", model.NodeTypeTrigger, nil, model.Position{}, map[string]any{
				"triggerType":  "webhook",
				"path":         "/payment/notify",
				"method":       "POST",
				"verify_token": "tok-123",
			}),
		},
		nil,
		nil,
		time.Now(), time.Now(),
	)

	wfDraft := model.NewWorkflowFromDB(
		uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		"draft-webhook-workflow",
		nil,
		model.WorkflowStatusDraft,
		[]model.WorkflowNode{
			model.NewWorkflowNodeFromDB("trigger-1", model.NodeTypeTrigger, nil, model.Position{}, map[string]any{
				"triggerType": "webhook",
				"path":        "/draft-hook",
				"method":      "POST",
			}),
		},
		nil,
		nil,
		time.Now(), time.Now(),
	)

	wfNoWebhook := model.NewWorkflowFromDB(
		uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		"active-no-webhook",
		nil,
		model.WorkflowStatusActive,
		[]model.WorkflowNode{
			model.NewWorkflowNodeFromDB("trigger-1", model.NodeTypeTrigger, nil, model.Position{}, map[string]any{
				"triggerType": "manual",
			}),
		},
		nil,
		nil,
		time.Now(), time.Now(),
	)

	workflows := []*model.Workflow{wfActive, wfDraft, wfNoWebhook}
	registered := PopulateRegistryFromWorkflows(registry, workflows)

	assert.Equal(t, 1, registered)
	assert.Equal(t, 1, registry.Count())

	route, ok := registry.Resolve("POST", "/payment/notify")
	require.True(t, ok)
	assert.Equal(t, wfActive.ID(), route.WorkflowID)
	assert.Equal(t, "tok-123", route.VerifyToken)

	// Draft workflow should not be registered
	_, ok = registry.Resolve("POST", "/draft-hook")
	assert.False(t, ok)
}

func TestPopulateRegistryFromWorkflows_MissingPathOrMethod(t *testing.T) {
	registry := webhook.NewRegistry()

	wf := model.NewWorkflowFromDB(
		uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		"incomplete-webhook",
		nil,
		model.WorkflowStatusActive,
		[]model.WorkflowNode{
			model.NewWorkflowNodeFromDB("trigger-1", model.NodeTypeTrigger, nil, model.Position{}, map[string]any{
				"triggerType": "webhook",
				"path":        "/only-path",
				// missing method
			}),
		},
		nil,
		nil,
		time.Now(), time.Now(),
	)

	registered := PopulateRegistryFromWorkflows(registry, []*model.Workflow{wf})
	assert.Equal(t, 0, registered)
	assert.Equal(t, 0, registry.Count())
}
