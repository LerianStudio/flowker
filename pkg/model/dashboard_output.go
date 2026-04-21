// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

// WorkflowSummaryOutput represents the response for the workflow dashboard summary.
type WorkflowSummaryOutput struct {
	Total    int64               `json:"total" example:"15"`
	Active   int64               `json:"active" example:"8"`
	ByStatus []StatusCountOutput `json:"byStatus"`
}

// StatusCountOutput represents a count grouped by status.
type StatusCountOutput struct {
	Status string `json:"status" example:"active"`
	Count  int64  `json:"count" example:"8"`
}

// ExecutionSummaryOutput represents the response for the execution dashboard summary.
type ExecutionSummaryOutput struct {
	Total     int64 `json:"total" example:"150"`
	Completed int64 `json:"completed" example:"120"`
	Failed    int64 `json:"failed" example:"25"`
	Pending   int64 `json:"pending" example:"3"`
	Running   int64 `json:"running" example:"2"`
}
