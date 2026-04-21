// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errorResp struct {
	Code    string `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func TestCatalogErrors(t *testing.T) {
	client := httpClient()

	// executor not found
	resp, err := client.Get(baseURL() + "/v1/catalog/executors/unknown")
	require.NoError(t, err, "executor not found request failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	var er errorResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0200", er.Code)

	// trigger not found
	resp, err = client.Get(baseURL() + "/v1/catalog/triggers/unknown")
	require.NoError(t, err, "trigger not found request failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	er = errorResp{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0210", er.Code)

	// validate executor with invalid body
	resp, err = client.Post(baseURL()+"/v1/catalog/executors/tracer.validate-transaction/validate", "application/json", bytes.NewBufferString("invalid"))
	require.NoError(t, err, "validate invalid body failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	er = errorResp{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0001", er.Code)

	// validate executor with missing required fields
	bodyInvalid := map[string]any{"config": map[string]any{"unknown_field": true}}
	buf, _ := json.Marshal(bodyInvalid)
	resp, err = client.Post(baseURL()+"/v1/catalog/executors/tracer.validate-transaction/validate", "application/json", bytes.NewBuffer(buf))
	require.NoError(t, err, "validate invalid config failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	er = errorResp{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0201", er.Code)
}

func TestWorkflowErrors(t *testing.T) {
	client := httpClient()

	// Seed an active provider configuration for workflow tests
	pcID := seedProviderConfig(t, "tracer-errors-test", "tracer", map[string]any{"base_url": "https://example.com", "api_key": "test-key"})
	defer seedDeleteProviderConfig(t, pcID)

	// invalid UUID
	resp, err := client.Get(baseURL() + "/v1/workflows/not-a-uuid")
	require.NoError(t, err, "invalid uuid request failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	var er errorResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0002", er.Code)

	// not found
	resp, err = client.Get(baseURL() + "/v1/workflows/" + uuid.New().String())
	require.NoError(t, err, "not found request failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	er = errorResp{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0100", er.Code)

	// duplicate name
	name := "wf-dup"
	id1 := createWorkflow(t, client, name, pcID)
	defer deleteWorkflow(t, client, id1)

	body, _ := json.Marshal(minimalWorkflowPayload(name, pcID))
	resp, err = client.Post(baseURL()+"/v1/workflows", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err, "duplicate create failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	er = errorResp{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0101", er.Code)

	// invalid status transition (activate twice)
	id2 := createWorkflow(t, client, "wf-activate-twice", pcID)
	defer deleteWorkflow(t, client, id2)
	resp, err = client.Post(baseURL()+"/v1/workflows/"+id2+"/activate", "application/json", nil)
	require.NoError(t, err, "first activate failed")
	resp.Body.Close()
	resp, err = client.Post(baseURL()+"/v1/workflows/"+id2+"/activate", "application/json", nil)
	require.NoError(t, err, "second activate failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	er = errorResp{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0102", er.Code)

	// delete active workflow (not allowed)
	id3 := createWorkflow(t, client, "wf-delete-active", pcID)
	resp, err = client.Post(baseURL()+"/v1/workflows/"+id3+"/activate", "application/json", nil)
	require.NoError(t, err, "activate before delete failed")
	resp.Body.Close()
	req, _ := http.NewRequest(http.MethodDelete, baseURL()+"/v1/workflows/"+id3, nil)
	resp, err = client.Do(req)
	require.NoError(t, err, "delete active failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	er = errorResp{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0103", er.Code)

	// executor not found in workflow (executorId="unknown" fails before providerConfigId check)
	badPayload := minimalWorkflowPayload("wf-bad-provider", pcID)
	nodes := badPayload["nodes"].([]map[string]any)
	nodes[1]["data"] = map[string]any{"executorId": "unknown"}
	badPayload["nodes"] = nodes
	body, _ = json.Marshal(badPayload)
	resp, err = client.Post(baseURL()+"/v1/workflows", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err, "create with bad provider failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	er = errorResp{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0104", er.Code)
}

func TestFaultInjection(t *testing.T) {
	client := httpClient()

	req, _ := http.NewRequest(http.MethodGet, baseURL()+"/health", nil)
	req.Header.Set("X-Test-Fault-Injection", "timeout")
	resp, err := client.Do(req)
	require.NoError(t, err, "fault timeout request failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)
	var er errorResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0800", er.Code)

	req, _ = http.NewRequest(http.MethodGet, baseURL()+"/health", nil)
	req.Header.Set("X-Test-Fault-Injection", "unavailable")
	resp, err = client.Do(req)
	require.NoError(t, err, "fault unavailable request failed")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	er = errorResp{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
	assert.Equal(t, "FLK-0801", er.Code)
}

// helpers

func createWorkflow(t *testing.T, client *http.Client, name, providerConfigID string) string {
	t.Helper()

	payload := minimalWorkflowPayload(name, providerConfigID)
	body, _ := json.Marshal(payload)
	resp, err := client.Post(baseURL()+"/v1/workflows", "application/json", bytes.NewBuffer(body))
	require.NoError(t, err, "create workflow helper failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var cr createWorkflowResp
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&cr))
	resp.Body.Close()

	return cr.WorkflowID
}

func deleteWorkflow(t *testing.T, client *http.Client, id string) {
	t.Helper()

	req, _ := http.NewRequest(http.MethodDelete, baseURL()+"/v1/workflows/"+id, nil)
	resp, err := client.Do(req)
	require.NoError(t, err, "delete helper failed")
	resp.Body.Close()
}
