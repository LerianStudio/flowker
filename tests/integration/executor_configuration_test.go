// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type createExecutorConfigResp struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type executorConfigOutput struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    *string        `json:"description"`
	BaseURL        string         `json:"baseUrl"`
	Status         string         `json:"status"`
	Endpoints      []endpointOut  `json:"endpoints"`
	Authentication authOut        `json:"authentication"`
	Metadata       map[string]any `json:"metadata"`
}

type endpointOut struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Method  string `json:"method"`
	Timeout int    `json:"timeout"`
}

type authOut struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config"`
}

type listExecutorConfigsResp struct {
	Items []executorConfigOutput `json:"items"`
}

// executorTestResultResp represents the test result output
type executorTestResultResp struct {
	ExecutorConfigID string            `json:"executorConfigId"`
	OverallStatus    string            `json:"overallStatus"`
	DurationMs       int64             `json:"durationMs"`
	Stages           []stageTestResult `json:"stages"`
	Summary          string            `json:"summary"`
}

type stageTestResult struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	DurationMs int64          `json:"durationMs"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
	Error      *string        `json:"error,omitempty"`
}

func minimalExecutorConfigPayload(name string) map[string]any {
	return map[string]any{
		"name":        name,
		"description": "Test executor configuration",
		"baseUrl":     "https://api.example.com",
		"endpoints": []map[string]any{
			{
				"name":    "validate",
				"path":    "/v1/validate",
				"method":  "POST",
				"timeout": 30,
			},
		},
		"authentication": map[string]any{
			"type": "none",
		},
	}
}

func TestExecutorConfigurationCRUD(t *testing.T) {
	t.Skip("POST /v1/executors removed; re-enable once config-only seeding is available (PR2)")
}

func TestExecutorConfigurationLifecycle(t *testing.T) {
	t.Skip("lifecycle routes removed; re-enable once config-only API is available (PR2)")
}

func TestExecutorConnectivityTest(t *testing.T) {
	t.Skip("POST /v1/executors and lifecycle routes removed; re-enable in PR2")
}

func TestExecutorConfigurationErrors(t *testing.T) {
	client := httpClient()

	t.Run("invalid UUID returns 400", func(t *testing.T) {
		resp, err := client.Get(baseURL() + "/v1/executors/not-a-uuid")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var er errorResp
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
		assert.Equal(t, "FLK-0002", er.Code)
	})

	t.Run("not found returns 404", func(t *testing.T) {
		resp, err := client.Get(baseURL() + "/v1/executors/" + uuid.New().String())
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var er errorResp
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&er))
		assert.Equal(t, "FLK-0250", er.Code)
	})

	t.Run("duplicate name", func(t *testing.T) {
		t.Skip("POST /v1/executors removed; re-enable in PR2")
	})

	t.Run("invalid status transition", func(t *testing.T) {
		t.Skip("lifecycle routes removed; re-enable in PR2")
	})

	t.Run("delete active executor", func(t *testing.T) {
		t.Skip("lifecycle routes removed; re-enable in PR2")
	})

	t.Run("update active executor", func(t *testing.T) {
		t.Skip("lifecycle routes removed; re-enable in PR2")
	})
}

func TestExecutorConfigurationWithAuthentication(t *testing.T) {
	t.Skip("POST /v1/executors removed; re-enable in PR2")
}

func TestExecutorConfigurationListFiltering(t *testing.T) {
	t.Skip("POST /v1/executors removed; re-enable in PR2")
}

// helpers

func getExecutorConfig(t *testing.T, client *http.Client, id string) executorConfigOutput {
	t.Helper()

	resp, err := client.Get(fmt.Sprintf("%s/v1/executors/%s", baseURL(), id))
	require.NoError(t, err, "get executor config helper failed")
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var out executorConfigOutput
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	resp.Body.Close()

	return out
}

func deleteExecutorConfig(t *testing.T, client *http.Client, id string) {
	t.Helper()

	// Just attempt DELETE directly; best-effort cleanup.
	req, _ := http.NewRequest(http.MethodDelete, baseURL()+"/v1/executors/"+id, nil)
	resp, err := client.Do(req)
	require.NoError(t, err, "delete executor config helper failed")
	assert.Contains(t, []int{http.StatusNoContent, http.StatusNotFound}, resp.StatusCode,
		"unexpected status in cleanup delete")
	resp.Body.Close()
}
