// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package model_test

import (
	"errors"
	"testing"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ExecutorCallDetails tests ---

func TestNewExecutorCallDetails(t *testing.T) {
	d := model.NewExecutorCallDetails("config-1", "validate", "POST", "https://api.example.com/v1", 200, 150)

	assert.Equal(t, "config-1", d.ExecutorConfigID())
	assert.Equal(t, "validate", d.EndpointName())
	assert.Equal(t, "POST", d.Method())
	assert.Equal(t, "https://api.example.com/v1", d.URL())
	assert.Equal(t, 200, d.StatusCode())
	assert.Equal(t, int64(150), d.DurationMs())
}

// --- ExecutionStep tests ---

func TestNewExecutionStep(t *testing.T) {
	input := map[string]any{"key": "value"}
	step := model.NewExecutionStep(1, "KYC Check", "node-kyc", input)

	assert.NotEqual(t, uuid.Nil, step.StepID())
	assert.Equal(t, 1, step.StepNumber())
	assert.Equal(t, "KYC Check", step.StepName())
	assert.Equal(t, "node-kyc", step.NodeID())
	assert.Equal(t, model.StepStatusRunning, step.Status())
	assert.Equal(t, input, step.InputData())
	assert.Nil(t, step.OutputData())
	assert.Nil(t, step.ErrorMessage())
	assert.Nil(t, step.ExecutorCallDetails())
	assert.False(t, step.StartedAt().IsZero())
	assert.Nil(t, step.CompletedAt())
	assert.Equal(t, int64(0), step.Duration())
	assert.Equal(t, 1, step.AttemptNumber())
}

func TestExecutionStep_MarkCompleted(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)
	output := map[string]any{"result": "ok"}

	err := step.MarkCompleted(output)

	require.NoError(t, err)
	assert.Equal(t, model.StepStatusCompleted, step.Status())
	assert.Equal(t, output, step.OutputData())
	assert.NotNil(t, step.CompletedAt())
	assert.GreaterOrEqual(t, step.Duration(), int64(0))
}

func TestExecutionStep_MarkFailed(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)

	err := step.MarkFailed("connection refused")

	require.NoError(t, err)
	assert.Equal(t, model.StepStatusFailed, step.Status())
	require.NotNil(t, step.ErrorMessage())
	assert.Equal(t, "connection refused", *step.ErrorMessage())
	assert.NotNil(t, step.CompletedAt())
}

func TestExecutionStep_MarkSkipped(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)

	err := step.MarkSkipped()

	require.NoError(t, err)
	assert.Equal(t, model.StepStatusSkipped, step.Status())
	assert.NotNil(t, step.CompletedAt())
}

func TestExecutionStep_SetExecutorCallDetails(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)
	details := model.NewExecutorCallDetails("cfg-1", "ep", "GET", "https://api.test.com", 200, 50)

	step.SetExecutorCallDetails(details)

	require.NotNil(t, step.ExecutorCallDetails())
	assert.Equal(t, "cfg-1", step.ExecutorCallDetails().ExecutorConfigID())
}

func TestExecutionStep_SetAttemptNumber(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)
	assert.Equal(t, 1, step.AttemptNumber())

	step.SetAttemptNumber(3)
	assert.Equal(t, 3, step.AttemptNumber())
}

func TestNewExecutionStepFromDB(t *testing.T) {
	stepID := uuid.New()
	now := time.Now().UTC()
	completed := now.Add(100 * time.Millisecond)
	errMsg := "timeout"
	details := model.NewExecutorCallDetails("cfg-1", "ep", "POST", "https://api.test.com", 500, 1000)

	step := model.NewExecutionStepFromDB(
		stepID, 2, "AML Check", "node-aml",
		model.StepStatusFailed,
		map[string]any{"in": "data"},
		map[string]any{"out": "data"},
		&errMsg,
		&details,
		now, &completed, 100, 3,
	)

	assert.Equal(t, stepID, step.StepID())
	assert.Equal(t, 2, step.StepNumber())
	assert.Equal(t, "AML Check", step.StepName())
	assert.Equal(t, "node-aml", step.NodeID())
	assert.Equal(t, model.StepStatusFailed, step.Status())
	assert.Equal(t, map[string]any{"in": "data"}, step.InputData())
	assert.Equal(t, map[string]any{"out": "data"}, step.OutputData())
	require.NotNil(t, step.ErrorMessage())
	assert.Equal(t, "timeout", *step.ErrorMessage())
	require.NotNil(t, step.ExecutorCallDetails())
	assert.Equal(t, now, step.StartedAt())
	require.NotNil(t, step.CompletedAt())
	assert.Equal(t, completed, *step.CompletedAt())
	assert.Equal(t, int64(100), step.Duration())
	assert.Equal(t, 3, step.AttemptNumber())
}

func TestExecutionStep_InputDataDefensiveCopy(t *testing.T) {
	input := map[string]any{"key": "original"}
	step := model.NewExecutionStep(1, "step", "node-1", input)

	// Modify returned copy
	returned := step.InputData()
	returned["key"] = "modified"

	// Original should be unchanged
	assert.Equal(t, "original", step.InputData()["key"])
}

// --- WorkflowExecution tests ---

func TestNewWorkflowExecution(t *testing.T) {
	workflowID := uuid.New()
	input := map[string]any{"cpf": "123"}
	key := "idem-key-1"

	exec := model.NewWorkflowExecution(workflowID, input, &key, 3)

	assert.NotEqual(t, uuid.Nil, exec.ExecutionID())
	assert.Equal(t, workflowID, exec.WorkflowID())
	assert.Equal(t, model.ExecutionStatusPending, exec.Status())
	assert.Equal(t, input, exec.InputData())
	assert.Nil(t, exec.OutputData())
	assert.Nil(t, exec.ErrorMessage())
	assert.Equal(t, 0, exec.CurrentStepNumber())
	assert.Equal(t, 3, exec.TotalSteps())
	assert.Empty(t, exec.Steps())
	require.NotNil(t, exec.IdempotencyKey())
	assert.Equal(t, "idem-key-1", *exec.IdempotencyKey())
	assert.False(t, exec.StartedAt().IsZero())
	assert.Nil(t, exec.CompletedAt())
	assert.False(t, exec.IsTerminal())
}

func TestNewWorkflowExecution_NilIdempotencyKey(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)

	assert.Nil(t, exec.IdempotencyKey())
}

func TestWorkflowExecution_MarkRunning(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)

	err := exec.MarkRunning()

	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusRunning, exec.Status())
	assert.False(t, exec.IsTerminal())
}

func TestWorkflowExecution_MarkCompleted(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)
	require.NoError(t, exec.MarkRunning())
	output := map[string]any{"result": "approved"}

	err := exec.MarkCompleted(output)

	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusCompleted, exec.Status())
	assert.Equal(t, output, exec.OutputData())
	assert.NotNil(t, exec.CompletedAt())
	assert.True(t, exec.IsTerminal())
}

func TestWorkflowExecution_MarkFailed(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)
	require.NoError(t, exec.MarkRunning())

	err := exec.MarkFailed("node failed: timeout")

	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusFailed, exec.Status())
	require.NotNil(t, exec.ErrorMessage())
	assert.Equal(t, "node failed: timeout", *exec.ErrorMessage())
	assert.NotNil(t, exec.CompletedAt())
	assert.True(t, exec.IsTerminal())
}

func TestWorkflowExecution_AddStep(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 3)
	step1 := model.NewExecutionStep(1, "step-1", "node-1", nil)
	step2 := model.NewExecutionStep(2, "step-2", "node-2", nil)

	exec.AddStep(step1)
	assert.Len(t, exec.Steps(), 1)
	assert.Equal(t, 1, exec.CurrentStepNumber())

	exec.AddStep(step2)
	assert.Len(t, exec.Steps(), 2)
	assert.Equal(t, 2, exec.CurrentStepNumber())
}

func TestWorkflowExecution_SetCurrentStep(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 5)

	exec.SetCurrentStep(3)

	assert.Equal(t, 3, exec.CurrentStepNumber())
}

func TestWorkflowExecution_LastCompletedStepNumber(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 3)

	// No steps
	assert.Equal(t, 0, exec.LastCompletedStepNumber())

	// Add completed step
	step1 := model.NewExecutionStep(1, "step-1", "node-1", nil)
	_ = step1.MarkCompleted(nil)
	exec.AddStep(step1)
	assert.Equal(t, 1, exec.LastCompletedStepNumber())

	// Add failed step
	step2 := model.NewExecutionStep(2, "step-2", "node-2", nil)
	_ = step2.MarkFailed("error")
	exec.AddStep(step2)
	// Last completed is still 1
	assert.Equal(t, 1, exec.LastCompletedStepNumber())

	// Add another completed step
	step3 := model.NewExecutionStep(3, "step-3", "node-3", nil)
	_ = step3.MarkCompleted(nil)
	exec.AddStep(step3)
	assert.Equal(t, 3, exec.LastCompletedStepNumber())
}

func TestWorkflowExecution_StepsDefensiveCopy(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)
	step := model.NewExecutionStep(1, "step-1", "node-1", nil)
	exec.AddStep(step)

	steps := exec.Steps()
	assert.Len(t, steps, 1)

	// Modifying the returned slice should not affect the original
	steps = append(steps, model.NewExecutionStep(2, "step-2", "node-2", nil))
	assert.Len(t, exec.Steps(), 1)
}

func TestWorkflowExecution_InputDataDefensiveCopy(t *testing.T) {
	input := map[string]any{"key": "original"}
	exec := model.NewWorkflowExecution(uuid.New(), input, nil, 1)

	returned := exec.InputData()
	returned["key"] = "modified"

	assert.Equal(t, "original", exec.InputData()["key"])
}

func TestNewWorkflowExecutionFromDB(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()
	now := time.Now().UTC()
	completed := now.Add(5 * time.Second)
	errMsg := "failure"
	key := "idem-123"

	step := model.NewExecutionStepFromDB(
		uuid.New(), 1, "step-1", "node-1",
		model.StepStatusCompleted, nil, nil, nil, nil,
		now, nil, 0, 1,
	)

	exec := model.NewWorkflowExecutionFromDB(
		execID, wfID,
		model.ExecutionStatusFailed,
		map[string]any{"in": "data"},
		map[string]any{"out": "data"},
		&errMsg, 1, 2,
		[]model.ExecutionStep{step},
		&key, now, &completed,
	)

	assert.Equal(t, execID, exec.ExecutionID())
	assert.Equal(t, wfID, exec.WorkflowID())
	assert.Equal(t, model.ExecutionStatusFailed, exec.Status())
	assert.Equal(t, map[string]any{"in": "data"}, exec.InputData())
	assert.Equal(t, map[string]any{"out": "data"}, exec.OutputData())
	require.NotNil(t, exec.ErrorMessage())
	assert.Equal(t, "failure", *exec.ErrorMessage())
	assert.Equal(t, 1, exec.CurrentStepNumber())
	assert.Equal(t, 2, exec.TotalSteps())
	assert.Len(t, exec.Steps(), 1)
	require.NotNil(t, exec.IdempotencyKey())
	assert.Equal(t, "idem-123", *exec.IdempotencyKey())
	assert.Equal(t, now, exec.StartedAt())
	require.NotNil(t, exec.CompletedAt())
	assert.Equal(t, completed, *exec.CompletedAt())
	assert.True(t, exec.IsTerminal())
}

func TestWorkflowExecution_StepsNil(t *testing.T) {
	exec := model.NewWorkflowExecutionFromDB(
		uuid.New(), uuid.New(),
		model.ExecutionStatusPending,
		nil, nil, nil, 0, 0, nil, nil,
		time.Now().UTC(), nil,
	)

	// FromDB creates a copy of the steps slice, so even nil input becomes an empty slice
	assert.Empty(t, exec.Steps())
}

// --- Output DTO tests ---

func TestExecutionCreateOutputFromDomain(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), map[string]any{"k": "v"}, nil, 2)

	output := model.ExecutionCreateOutputFromDomain(exec)

	assert.Equal(t, exec.ExecutionID(), output.ExecutionID)
	assert.Equal(t, exec.WorkflowID(), output.WorkflowID)
	assert.Equal(t, "pending", output.Status)
	assert.Equal(t, exec.StartedAt(), output.StartedAt)
}

func TestExecutionStatusOutputFromDomain(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 5)
	_ = exec.MarkRunning()
	exec.SetCurrentStep(3)

	output := model.ExecutionStatusOutputFromDomain(exec)

	assert.Equal(t, exec.ExecutionID(), output.ExecutionID)
	assert.Equal(t, "running", output.Status)
	assert.Equal(t, 3, output.CurrentStepNumber)
	assert.Equal(t, 5, output.TotalSteps)
	assert.Nil(t, output.CompletedAt)
	assert.Nil(t, output.ErrorMessage)
}

func TestExecutionResultsOutputFromDomain(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 2)
	_ = exec.MarkRunning()

	step1 := model.NewExecutionStep(1, "KYC", "node-kyc", nil)
	_ = step1.MarkCompleted(map[string]any{"approved": true})
	exec.AddStep(step1)

	step2 := model.NewExecutionStep(2, "AML", "node-aml", nil)
	_ = step2.MarkCompleted(map[string]any{"risk": "low"})
	exec.AddStep(step2)

	_ = exec.MarkCompleted(map[string]any{"final": "result"})

	output := model.ExecutionResultsOutputFromDomain(exec)

	assert.Equal(t, exec.ExecutionID(), output.ExecutionID)
	assert.Equal(t, "completed", output.Status)
	require.Len(t, output.StepResults, 2)
	assert.Equal(t, "KYC", output.StepResults[0].StepName)
	assert.Equal(t, "node-kyc", output.StepResults[0].NodeID)
	assert.Equal(t, "completed", output.StepResults[0].Status)
	assert.Equal(t, map[string]any{"final": "result"}, output.FinalOutput)
	assert.NotNil(t, output.CompletedAt)
}

func TestExecutionResultsOutputFromDomain_NoSteps(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 0)
	_ = exec.MarkRunning()
	_ = exec.MarkCompleted(nil)

	output := model.ExecutionResultsOutputFromDomain(exec)

	assert.Empty(t, output.StepResults)
}

// --- State transition guard tests (WorkflowExecution) ---

func TestMarkRunning_FromPending_Succeeds(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)
	assert.Equal(t, model.ExecutionStatusPending, exec.Status())

	err := exec.MarkRunning()

	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusRunning, exec.Status())
}

func TestMarkRunning_FromRunning_Fails(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)
	require.NoError(t, exec.MarkRunning())

	err := exec.MarkRunning()

	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrInvalidExecutionTransition))
	assert.Contains(t, err.Error(), "running")
}

func TestMarkRunning_FromCompleted_Fails(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)
	require.NoError(t, exec.MarkRunning())
	require.NoError(t, exec.MarkCompleted(nil))

	err := exec.MarkRunning()

	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrInvalidExecutionTransition))
	assert.Contains(t, err.Error(), "completed")
}

func TestMarkCompleted_FromRunning_Succeeds(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)
	require.NoError(t, exec.MarkRunning())

	err := exec.MarkCompleted(map[string]any{"result": "ok"})

	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusCompleted, exec.Status())
	assert.NotNil(t, exec.CompletedAt())
}

func TestMarkCompleted_FromPending_Fails(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)

	err := exec.MarkCompleted(nil)

	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrInvalidExecutionTransition))
	assert.Contains(t, err.Error(), "pending")
}

func TestMarkFailed_FromPending_Succeeds(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)

	err := exec.MarkFailed("something went wrong")

	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusFailed, exec.Status())
	require.NotNil(t, exec.ErrorMessage())
	assert.Equal(t, "something went wrong", *exec.ErrorMessage())
}

func TestMarkFailed_FromRunning_Succeeds(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)
	require.NoError(t, exec.MarkRunning())

	err := exec.MarkFailed("node timed out")

	require.NoError(t, err)
	assert.Equal(t, model.ExecutionStatusFailed, exec.Status())
	assert.NotNil(t, exec.CompletedAt())
}

func TestMarkFailed_FromCompleted_Fails(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)
	require.NoError(t, exec.MarkRunning())
	require.NoError(t, exec.MarkCompleted(nil))

	err := exec.MarkFailed("late failure")

	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrExecutionAlreadyTerminal))
	assert.Contains(t, err.Error(), "completed")
	// Status should remain completed
	assert.Equal(t, model.ExecutionStatusCompleted, exec.Status())
}

func TestMarkFailed_FromFailed_Fails(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 1)
	require.NoError(t, exec.MarkRunning())
	require.NoError(t, exec.MarkFailed("first failure"))

	err := exec.MarkFailed("second failure")

	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrExecutionAlreadyTerminal))
	assert.Contains(t, err.Error(), "failed")
	// Error message should remain the first one
	require.NotNil(t, exec.ErrorMessage())
	assert.Equal(t, "first failure", *exec.ErrorMessage())
}

// --- State transition guard tests (ExecutionStep) ---

func TestStepMarkCompleted_FromRunning_Succeeds(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)

	err := step.MarkCompleted(map[string]any{"result": "ok"})

	require.NoError(t, err)
	assert.Equal(t, model.StepStatusCompleted, step.Status())
}

func TestStepMarkCompleted_FromCompleted_Fails(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)
	require.NoError(t, step.MarkCompleted(nil))

	err := step.MarkCompleted(nil)

	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrInvalidStepTransition))
	assert.Contains(t, err.Error(), "completed")
}

func TestStepMarkFailed_FromRunning_Succeeds(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)

	err := step.MarkFailed("timeout")

	require.NoError(t, err)
	assert.Equal(t, model.StepStatusFailed, step.Status())
}

func TestStepMarkFailed_FromFailed_Fails(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)
	require.NoError(t, step.MarkFailed("first error"))

	err := step.MarkFailed("second error")

	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrInvalidStepTransition))
}

func TestStepMarkSkipped_FromRunning_Succeeds(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)

	err := step.MarkSkipped()

	require.NoError(t, err)
	assert.Equal(t, model.StepStatusSkipped, step.Status())
}

func TestStepMarkSkipped_FromCompleted_Fails(t *testing.T) {
	step := model.NewExecutionStep(1, "step", "node-1", nil)
	require.NoError(t, step.MarkCompleted(nil))

	err := step.MarkSkipped()

	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrInvalidStepTransition))
}
