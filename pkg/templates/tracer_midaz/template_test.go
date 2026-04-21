// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package tracer_midaz_test

import (
	"testing"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/model"
	tracerMidaz "github.com/LerianStudio/flowker/pkg/templates/tracer_midaz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validParams() map[string]any {
	return map[string]any{
		"workflowName":           "test-workflow",
		"tracerProviderConfigId": "550e8400-e29b-41d4-a716-446655440000",
		"midazProviderConfigId":  "660e8400-e29b-41d4-a716-446655440001",
	}
}

func TestBuild_ValidParams(t *testing.T) {
	catalog := executor.NewCatalog()
	err := tracerMidaz.Register(catalog)
	require.NoError(t, err)

	tmpl, err := catalog.GetTemplate(tracerMidaz.TemplateID)
	require.NoError(t, err)

	params := validParams()
	params["webhookPath"] = "/custom-path"
	params["webhookMethod"] = "POST"
	params["workflowDescription"] = "A test workflow"

	result, err := tmpl.Build(params)
	require.NoError(t, err)
	require.NotNil(t, result)

	input, ok := result.(*model.CreateWorkflowInput)
	require.True(t, ok, "Build should return *model.CreateWorkflowInput")

	// Verify workflow name and description
	assert.Equal(t, "test-workflow", input.Name)
	require.NotNil(t, input.Description)
	assert.Equal(t, "A test workflow", *input.Description)

	// Verify 4 nodes (trigger, 2 executors, conditional — no action nodes)
	require.Len(t, input.Nodes, 4)

	// Verify node IDs
	nodeIDs := make(map[string]bool)
	for _, n := range input.Nodes {
		nodeIDs[n.ID] = true
	}
	assert.True(t, nodeIDs["webhook-trigger"])
	assert.True(t, nodeIDs["tracer-validate"])
	assert.True(t, nodeIDs["decision-check"])
	assert.True(t, nodeIDs["midaz-create-tx"])

	// Verify node types
	nodeTypes := make(map[string]string)
	for _, n := range input.Nodes {
		nodeTypes[n.ID] = n.Type
	}
	assert.Equal(t, "trigger", nodeTypes["webhook-trigger"])
	assert.Equal(t, "executor", nodeTypes["tracer-validate"])
	assert.Equal(t, "conditional", nodeTypes["decision-check"])
	assert.Equal(t, "executor", nodeTypes["midaz-create-tx"])

	// Verify webhook trigger data
	nodeData := make(map[string]map[string]any)
	for _, n := range input.Nodes {
		nodeData[n.ID] = n.Data
	}
	assert.Equal(t, "webhook", nodeData["webhook-trigger"]["triggerType"])
	assert.Equal(t, "/custom-path", nodeData["webhook-trigger"]["path"])
	assert.Equal(t, "POST", nodeData["webhook-trigger"]["method"])

	// Verify executor data
	assert.Equal(t, "tracer.validate-transaction", nodeData["tracer-validate"]["executorId"])
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", nodeData["tracer-validate"]["providerConfigId"])
	assert.Equal(t, "midaz.create-transaction", nodeData["midaz-create-tx"]["executorId"])
	assert.Equal(t, "660e8400-e29b-41d4-a716-446655440001", nodeData["midaz-create-tx"]["providerConfigId"])

	// Verify 3 edges (webhook→tracer, tracer→decision, decision→midaz)
	require.Len(t, input.Edges, 3)

	edgeMap := make(map[string]struct{ source, target string })
	for _, e := range input.Edges {
		edgeMap[e.ID] = struct{ source, target string }{e.Source, e.Target}
	}
	assert.Equal(t, "webhook-trigger", edgeMap["webhook-to-tracer"].source)
	assert.Equal(t, "tracer-validate", edgeMap["webhook-to-tracer"].target)
	assert.Equal(t, "tracer-validate", edgeMap["tracer-to-decision"].source)
	assert.Equal(t, "decision-check", edgeMap["tracer-to-decision"].target)
	assert.Equal(t, "decision-check", edgeMap["decision-to-midaz"].source)
	assert.Equal(t, "midaz-create-tx", edgeMap["decision-to-midaz"].target)

	// Verify metadata
	assert.Equal(t, "tracer-midaz-validation", input.Metadata["templateId"])
	assert.Equal(t, "v1", input.Metadata["templateVersion"])
}

func TestBuild_MissingRequiredParams(t *testing.T) {
	catalog := executor.NewCatalog()
	err := tracerMidaz.Register(catalog)
	require.NoError(t, err)

	tmpl, err := catalog.GetTemplate(tracerMidaz.TemplateID)
	require.NoError(t, err)

	// Missing all required params
	_, err = tmpl.Build(map[string]any{})
	assert.Error(t, err)

	// Missing midazProviderConfigId
	_, err = tmpl.Build(map[string]any{
		"workflowName":           "test",
		"tracerProviderConfigId": "550e8400-e29b-41d4-a716-446655440000",
	})
	assert.Error(t, err)
}

func TestValidateParams_Valid(t *testing.T) {
	catalog := executor.NewCatalog()
	err := tracerMidaz.Register(catalog)
	require.NoError(t, err)

	tmpl, err := catalog.GetTemplate(tracerMidaz.TemplateID)
	require.NoError(t, err)

	err = tmpl.ValidateParams(validParams())
	assert.NoError(t, err)
}

func TestValidateParams_Invalid(t *testing.T) {
	catalog := executor.NewCatalog()
	err := tracerMidaz.Register(catalog)
	require.NoError(t, err)

	tmpl, err := catalog.GetTemplate(tracerMidaz.TemplateID)
	require.NoError(t, err)

	// Missing required fields
	err = tmpl.ValidateParams(map[string]any{})
	assert.Error(t, err)

	// Invalid workflowName type
	err = tmpl.ValidateParams(map[string]any{
		"workflowName":           123,
		"tracerProviderConfigId": "550e8400-e29b-41d4-a716-446655440000",
		"midazProviderConfigId":  "660e8400-e29b-41d4-a716-446655440001",
	})
	assert.Error(t, err)
}

func TestRegister(t *testing.T) {
	catalog := executor.NewCatalog()
	err := tracerMidaz.Register(catalog)
	require.NoError(t, err)

	templates := catalog.ListTemplates()
	require.Len(t, templates, 1)
	assert.Equal(t, tracerMidaz.TemplateID, templates[0].ID())
	assert.Equal(t, "Tracer Validation + Midaz Transaction", templates[0].Name())
	assert.Equal(t, "Compliance", templates[0].Category())
	assert.Equal(t, "v1", templates[0].Version())

	// Verify duplicate registration fails
	err = tracerMidaz.Register(catalog)
	assert.Error(t, err)
}
