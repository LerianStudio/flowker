// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"time"

	"github.com/google/uuid"
)

// ExecutorTestResultOutput represents the API response for an executor test result.
type ExecutorTestResultOutput struct {
	ExecutorConfigID uuid.UUID               `json:"executorConfigId" swaggertype:"string" format:"uuid"`
	OverallStatus    string                  `json:"overallStatus"`
	DurationMs       int64                   `json:"durationMs"`
	Stages           []StageTestResultOutput `json:"stages"`
	Summary          string                  `json:"summary"`
	StartedAt        time.Time               `json:"startedAt"`
	CompletedAt      *time.Time              `json:"completedAt,omitempty"`
}

// StageTestResultOutput represents the API response for a single test stage.
type StageTestResultOutput struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	DurationMs int64          `json:"durationMs"`
	Message    string         `json:"message,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
	Error      *string        `json:"error,omitempty"`
}

// ExecutorTestResultOutputFromDomain converts an ExecutorTestResult to its output representation.
func ExecutorTestResultOutputFromDomain(result *ExecutorTestResult) ExecutorTestResultOutput {
	if result == nil {
		return ExecutorTestResultOutput{}
	}

	stages := make([]StageTestResultOutput, len(result.Stages()))
	for i, stage := range result.Stages() {
		stages[i] = StageTestResultOutputFromDomain(stage)
	}

	return ExecutorTestResultOutput{
		ExecutorConfigID: result.ExecutorConfigID(),
		OverallStatus:    string(result.OverallStatus()),
		DurationMs:       result.DurationMs(),
		Stages:           stages,
		Summary:          result.Summary(),
		StartedAt:        result.StartedAt(),
		CompletedAt:      result.CompletedAt(),
	}
}

// StageTestResultOutputFromDomain converts a StageTestResult to its output representation.
func StageTestResultOutputFromDomain(stage StageTestResult) StageTestResultOutput {
	return StageTestResultOutput{
		Name:       string(stage.Name()),
		Status:     string(stage.Status()),
		DurationMs: stage.DurationMs(),
		Message:    stage.Message(),
		Details:    stage.Details(),
		Error:      stage.Error(),
	}
}
