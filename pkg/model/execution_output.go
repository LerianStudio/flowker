// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionCreateOutput is the response for POST /v1/workflows/{id}/executions.
type ExecutionCreateOutput struct {
	ExecutionID uuid.UUID `json:"executionId" swaggertype:"string" format:"uuid"`
	WorkflowID  uuid.UUID `json:"workflowId" swaggertype:"string" format:"uuid"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"startedAt"`
}

// ExecutionStatusOutput is the response for GET /v1/executions/{id}.
type ExecutionStatusOutput struct {
	ExecutionID       uuid.UUID  `json:"executionId" swaggertype:"string" format:"uuid"`
	WorkflowID        uuid.UUID  `json:"workflowId" swaggertype:"string" format:"uuid"`
	Status            string     `json:"status"`
	CurrentStepNumber int        `json:"currentStepNumber"`
	TotalSteps        int        `json:"totalSteps"`
	StartedAt         time.Time  `json:"startedAt"`
	CompletedAt       *time.Time `json:"completedAt,omitempty"`
	ErrorMessage      *string    `json:"errorMessage,omitempty"`
}

// ExecutionResultsOutput is the response for GET /v1/executions/{id}/results.
type ExecutionResultsOutput struct {
	ExecutionID uuid.UUID          `json:"executionId" swaggertype:"string" format:"uuid"`
	WorkflowID  uuid.UUID          `json:"workflowId" swaggertype:"string" format:"uuid"`
	Status      string             `json:"status"`
	StepResults []StepResultOutput `json:"stepResults"`
	FinalOutput map[string]any     `json:"finalOutput,omitempty"`
	StartedAt   time.Time          `json:"startedAt"`
	CompletedAt *time.Time         `json:"completedAt,omitempty"`
}

// StepResultOutput is the output for a single execution step.
type StepResultOutput struct {
	StepNumber   int            `json:"stepNumber"`
	StepName     string         `json:"stepName"`
	NodeID       string         `json:"nodeId"`
	Status       string         `json:"status"`
	Output       map[string]any `json:"output,omitempty"`
	ErrorMessage *string        `json:"errorMessage,omitempty"`
	ExecutedAt   time.Time      `json:"executedAt"`
	Duration     int64          `json:"durationMs"`
}

// ExecutionListOutput is the response for GET /v1/executions.
type ExecutionListOutput struct {
	Items      []ExecutionStatusOutput `json:"items"`
	NextCursor string                  `json:"nextCursor"`
	HasMore    bool                    `json:"hasMore"`
}

// ExecutionListOutputFromDomain converts a list of WorkflowExecution to ExecutionListOutput.
func ExecutionListOutputFromDomain(executions []*WorkflowExecution, nextCursor string, hasMore bool) ExecutionListOutput {
	items := make([]ExecutionStatusOutput, len(executions))
	for i, e := range executions {
		items[i] = ExecutionStatusOutputFromDomain(e)
	}

	return ExecutionListOutput{Items: items, NextCursor: nextCursor, HasMore: hasMore}
}

// ExecutionCreateOutputFromDomain converts a WorkflowExecution to ExecutionCreateOutput.
func ExecutionCreateOutputFromDomain(e *WorkflowExecution) ExecutionCreateOutput {
	return ExecutionCreateOutput{
		ExecutionID: e.ExecutionID(),
		WorkflowID:  e.WorkflowID(),
		Status:      string(e.Status()),
		StartedAt:   e.StartedAt(),
	}
}

// ExecutionStatusOutputFromDomain converts a WorkflowExecution to ExecutionStatusOutput.
func ExecutionStatusOutputFromDomain(e *WorkflowExecution) ExecutionStatusOutput {
	return ExecutionStatusOutput{
		ExecutionID:       e.ExecutionID(),
		WorkflowID:        e.WorkflowID(),
		Status:            string(e.Status()),
		CurrentStepNumber: e.CurrentStepNumber(),
		TotalSteps:        e.TotalSteps(),
		StartedAt:         e.StartedAt(),
		CompletedAt:       e.CompletedAt(),
		ErrorMessage:      e.ErrorMessage(),
	}
}

// ExecutionResultsOutputFromDomain converts a WorkflowExecution to ExecutionResultsOutput.
func ExecutionResultsOutputFromDomain(e *WorkflowExecution) ExecutionResultsOutput {
	steps := e.Steps()
	stepResults := make([]StepResultOutput, len(steps))

	for i, step := range steps {
		stepResults[i] = StepResultOutput{
			StepNumber:   step.StepNumber(),
			StepName:     step.StepName(),
			NodeID:       step.NodeID(),
			Status:       string(step.Status()),
			Output:       step.OutputData(),
			ErrorMessage: step.ErrorMessage(),
			ExecutedAt:   step.StartedAt(),
			Duration:     step.Duration(),
		}
	}

	return ExecutionResultsOutput{
		ExecutionID: e.ExecutionID(),
		WorkflowID:  e.WorkflowID(),
		Status:      string(e.Status()),
		StepResults: stepResults,
		FinalOutput: e.OutputData(),
		StartedAt:   e.StartedAt(),
		CompletedAt: e.CompletedAt(),
	}
}
