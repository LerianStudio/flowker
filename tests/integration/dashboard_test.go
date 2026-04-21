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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type workflowSummaryResp struct {
	Total    int64             `json:"total"`
	Active   int64             `json:"active"`
	ByStatus []statusCountResp `json:"byStatus"`
}

type statusCountResp struct {
	Status string `json:"status"`
	Count  int64  `json:"count"`
}

type executionSummaryResp struct {
	Total     int64 `json:"total"`
	Completed int64 `json:"completed"`
	Failed    int64 `json:"failed"`
	Pending   int64 `json:"pending"`
	Running   int64 `json:"running"`
}

type dashboardErrorResp struct {
	Code    string `json:"code"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

func TestDashboardWorkflowSummary(t *testing.T) {
	client := httpClient()

	resp, err := client.Get(baseURL() + "/v1/dashboards/workflows/summary")
	require.NoError(t, err, "GET workflow summary")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK")

	var result workflowSummaryResp
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "decode workflow summary response")

	assert.GreaterOrEqual(t, result.Total, int64(0), "total should be >= 0")
	assert.NotNil(t, result.ByStatus, "byStatus should not be nil")
}

func TestDashboardExecutionSummary(t *testing.T) {
	client := httpClient()

	t.Run("no filters returns 200", func(t *testing.T) {
		resp, err := client.Get(baseURL() + "/v1/dashboards/executions")
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK")

		var result executionSummaryResp
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err, "decode execution summary response")

		assert.GreaterOrEqual(t, result.Total, int64(0))
		assert.GreaterOrEqual(t, result.Completed, int64(0))
		assert.GreaterOrEqual(t, result.Failed, int64(0))
		assert.GreaterOrEqual(t, result.Pending, int64(0))
		assert.GreaterOrEqual(t, result.Running, int64(0))
	})

	t.Run("with startTime and endTime returns 200", func(t *testing.T) {
		url := fmt.Sprintf("%s/v1/dashboards/executions?startTime=2026-01-01T00:00:00Z&endTime=2026-12-31T23:59:59Z", baseURL())

		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK with time filters")

		var result executionSummaryResp
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
	})

	t.Run("with status=completed returns 200", func(t *testing.T) {
		url := fmt.Sprintf("%s/v1/dashboards/executions?status=completed", baseURL())

		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode, "expected 200 OK with status filter")

		var result executionSummaryResp
		err = json.NewDecoder(resp.Body).Decode(&result)
		require.NoError(t, err)
	})

	t.Run("invalid startTime returns 400", func(t *testing.T) {
		url := fmt.Sprintf("%s/v1/dashboards/executions?startTime=not-a-date", baseURL())

		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected 400 for invalid startTime")

		var errResp dashboardErrorResp
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.NotEmpty(t, errResp.Message)
	})

	t.Run("startTime after endTime returns 400", func(t *testing.T) {
		url := fmt.Sprintf("%s/v1/dashboards/executions?startTime=2026-06-01T00:00:00Z&endTime=2026-01-01T00:00:00Z", baseURL())

		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected 400 when startTime > endTime")

		var errResp dashboardErrorResp
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Message, "startTime must be before endTime")
	})

	t.Run("invalid status returns 400", func(t *testing.T) {
		url := fmt.Sprintf("%s/v1/dashboards/executions?status=invalid", baseURL())

		resp, err := client.Get(url)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusBadRequest, resp.StatusCode, "expected 400 for invalid status")

		var errResp dashboardErrorResp
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp.Message, "invalid status")
	})
}
