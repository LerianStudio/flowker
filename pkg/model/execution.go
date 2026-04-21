// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrInvalidExecutionTransition indicates an attempt to transition to a status
	// that is not reachable from the current status.
	ErrInvalidExecutionTransition = errors.New("invalid execution state transition")
	// ErrExecutionAlreadyTerminal indicates the execution is already in a terminal
	// state (completed or failed) and cannot be modified further.
	ErrExecutionAlreadyTerminal = errors.New("execution is already in a terminal state")
	// ErrInvalidStepTransition indicates an attempt to transition a step to a
	// status that is not reachable from the current status.
	ErrInvalidStepTransition = errors.New("invalid step state transition")
)

// ExecutionStatus represents the status of a workflow execution.
type ExecutionStatus string

const (
	// ExecutionStatusPending indicates the execution has been created but not yet started.
	ExecutionStatusPending ExecutionStatus = "pending"
	// ExecutionStatusRunning indicates the execution is currently in progress.
	ExecutionStatusRunning ExecutionStatus = "running"
	// ExecutionStatusCompleted indicates the execution finished successfully.
	ExecutionStatusCompleted ExecutionStatus = "completed"
	// ExecutionStatusFailed indicates the execution finished with an error.
	ExecutionStatusFailed ExecutionStatus = "failed"
)

// IsValid checks if the execution status is one of the allowed enum values.
func (s ExecutionStatus) IsValid() bool {
	switch s {
	case ExecutionStatusPending, ExecutionStatusRunning, ExecutionStatusCompleted, ExecutionStatusFailed:
		return true
	default:
		return false
	}
}

// StepStatus represents the status of an execution step.
type StepStatus string

const (
	// StepStatusCompleted indicates the step finished successfully.
	StepStatusCompleted StepStatus = "completed"
	// StepStatusFailed indicates the step finished with an error.
	StepStatusFailed StepStatus = "failed"
	// StepStatusSkipped indicates the step was skipped (e.g., conditional branch not taken).
	StepStatusSkipped StepStatus = "skipped"
	// StepStatusRunning indicates the step is currently being executed.
	StepStatusRunning StepStatus = "running"
)

// ExecutorCallDetails holds details about the executor call for observability.
type ExecutorCallDetails struct {
	executorConfigID string
	endpointName     string
	method           string
	url              string
	statusCode       int
	durationMs       int64
}

// NewExecutorCallDetails creates a new ExecutorCallDetails.
func NewExecutorCallDetails(executorConfigID, endpointName, method, url string, statusCode int, durationMs int64) ExecutorCallDetails {
	return ExecutorCallDetails{
		executorConfigID: executorConfigID,
		endpointName:     endpointName,
		method:           method,
		url:              url,
		statusCode:       statusCode,
		durationMs:       durationMs,
	}
}

// ExecutorConfigID returns the executor configuration ID.
func (d ExecutorCallDetails) ExecutorConfigID() string { return d.executorConfigID }

// EndpointName returns the endpoint name.
func (d ExecutorCallDetails) EndpointName() string { return d.endpointName }

// Method returns the HTTP method.
func (d ExecutorCallDetails) Method() string { return d.method }

// URL returns the called URL.
func (d ExecutorCallDetails) URL() string { return d.url }

// StatusCode returns the HTTP status code.
func (d ExecutorCallDetails) StatusCode() int { return d.statusCode }

// DurationMs returns the call duration in milliseconds.
func (d ExecutorCallDetails) DurationMs() int64 { return d.durationMs }

// ExecutionStep represents a single step within a workflow execution.
type ExecutionStep struct {
	stepID              uuid.UUID
	stepNumber          int
	stepName            string
	nodeID              string
	status              StepStatus
	inputData           map[string]any
	outputData          map[string]any
	errorMessage        *string
	executorCallDetails *ExecutorCallDetails
	startedAt           time.Time
	completedAt         *time.Time
	duration            int64
	attemptNumber       int
}

// NewExecutionStep creates a new ExecutionStep.
func NewExecutionStep(stepNumber int, stepName, nodeID string, inputData map[string]any) ExecutionStep {
	return ExecutionStep{
		stepID:        uuid.New(),
		stepNumber:    stepNumber,
		stepName:      stepName,
		nodeID:        nodeID,
		status:        StepStatusRunning,
		inputData:     cloneAnyMap(inputData),
		startedAt:     time.Now().UTC(),
		attemptNumber: 1,
	}
}

// NewExecutionStepFromDB reconstructs an ExecutionStep from database values.
func NewExecutionStepFromDB(
	stepID uuid.UUID,
	stepNumber int,
	stepName, nodeID string,
	status StepStatus,
	inputData, outputData map[string]any,
	errorMessage *string,
	executorCallDetails *ExecutorCallDetails,
	startedAt time.Time,
	completedAt *time.Time,
	duration int64,
	attemptNumber int,
) ExecutionStep {
	return ExecutionStep{
		stepID:              stepID,
		stepNumber:          stepNumber,
		stepName:            stepName,
		nodeID:              nodeID,
		status:              status,
		inputData:           cloneAnyMap(inputData),
		outputData:          cloneAnyMap(outputData),
		errorMessage:        errorMessage,
		executorCallDetails: executorCallDetails,
		startedAt:           startedAt,
		completedAt:         completedAt,
		duration:            duration,
		attemptNumber:       attemptNumber,
	}
}

// StepID returns the step's unique identifier.
func (s ExecutionStep) StepID() uuid.UUID { return s.stepID }

// StepNumber returns the step's sequence number.
func (s ExecutionStep) StepNumber() int { return s.stepNumber }

// StepName returns the step's display name.
func (s ExecutionStep) StepName() string { return s.stepName }

// NodeID returns the workflow node ID this step executed.
func (s ExecutionStep) NodeID() string { return s.nodeID }

// Status returns the step's status.
func (s ExecutionStep) Status() StepStatus { return s.status }

// InputData returns a copy of the step's input data.
func (s ExecutionStep) InputData() map[string]any { return cloneAnyMap(s.inputData) }

// OutputData returns a copy of the step's output data.
func (s ExecutionStep) OutputData() map[string]any { return cloneAnyMap(s.outputData) }

// ErrorMessage returns the step's error message.
func (s ExecutionStep) ErrorMessage() *string { return s.errorMessage }

// ExecutorCallDetails returns the executor call details.
func (s ExecutionStep) ExecutorCallDetails() *ExecutorCallDetails { return s.executorCallDetails }

// StartedAt returns when the step started.
func (s ExecutionStep) StartedAt() time.Time { return s.startedAt }

// CompletedAt returns when the step completed.
func (s ExecutionStep) CompletedAt() *time.Time { return s.completedAt }

// Duration returns the step duration in milliseconds.
func (s ExecutionStep) Duration() int64 { return s.duration }

// AttemptNumber returns the attempt number for this step.
func (s ExecutionStep) AttemptNumber() int { return s.attemptNumber }

// isStepTerminal returns true if the step status is terminal (completed, failed, or skipped).
func isStepTerminal(status StepStatus) bool {
	return status == StepStatusCompleted || status == StepStatusFailed || status == StepStatusSkipped
}

// MarkCompleted marks the step as completed with output data.
func (s *ExecutionStep) MarkCompleted(outputData map[string]any) error {
	if isStepTerminal(s.status) {
		return fmt.Errorf("%w: cannot complete step in %s state", ErrInvalidStepTransition, s.status)
	}

	s.status = StepStatusCompleted
	s.outputData = cloneAnyMap(outputData)
	now := time.Now().UTC()
	s.completedAt = &now
	s.duration = now.Sub(s.startedAt).Milliseconds()

	return nil
}

// MarkFailed marks the step as failed with an error message.
func (s *ExecutionStep) MarkFailed(errMsg string) error {
	if isStepTerminal(s.status) {
		return fmt.Errorf("%w: cannot fail step in %s state", ErrInvalidStepTransition, s.status)
	}

	s.status = StepStatusFailed
	s.errorMessage = &errMsg
	now := time.Now().UTC()
	s.completedAt = &now
	s.duration = now.Sub(s.startedAt).Milliseconds()

	return nil
}

// MarkSkipped marks the step as skipped.
func (s *ExecutionStep) MarkSkipped() error {
	if isStepTerminal(s.status) {
		return fmt.Errorf("%w: cannot skip step in %s state", ErrInvalidStepTransition, s.status)
	}

	s.status = StepStatusSkipped
	now := time.Now().UTC()
	s.completedAt = &now
	s.duration = now.Sub(s.startedAt).Milliseconds()

	return nil
}

// SetExecutorCallDetails sets the executor call details.
func (s *ExecutionStep) SetExecutorCallDetails(details ExecutorCallDetails) {
	s.executorCallDetails = &details
}

// SetAttemptNumber sets the attempt number.
func (s *ExecutionStep) SetAttemptNumber(n int) {
	s.attemptNumber = n
}

// WorkflowExecution represents a single execution of a workflow (Rich Domain Model).
type WorkflowExecution struct {
	executionID       uuid.UUID
	workflowID        uuid.UUID
	status            ExecutionStatus
	inputData         map[string]any
	outputData        map[string]any
	errorMessage      *string
	currentStepNumber int
	totalSteps        int
	steps             []ExecutionStep
	idempotencyKey    *string
	startedAt         time.Time
	completedAt       *time.Time
}

// NewWorkflowExecution creates a new WorkflowExecution in pending status.
func NewWorkflowExecution(workflowID uuid.UUID, inputData map[string]any, idempotencyKey *string, totalSteps int) *WorkflowExecution {
	return &WorkflowExecution{
		executionID:       uuid.New(),
		workflowID:        workflowID,
		status:            ExecutionStatusPending,
		inputData:         cloneAnyMap(inputData),
		currentStepNumber: 0,
		totalSteps:        totalSteps,
		steps:             make([]ExecutionStep, 0),
		idempotencyKey:    idempotencyKey,
		startedAt:         time.Now().UTC(),
	}
}

// NewWorkflowExecutionFromDB reconstructs a WorkflowExecution from database values.
func NewWorkflowExecutionFromDB(
	executionID, workflowID uuid.UUID,
	status ExecutionStatus,
	inputData, outputData map[string]any,
	errorMessage *string,
	currentStepNumber, totalSteps int,
	steps []ExecutionStep,
	idempotencyKey *string,
	startedAt time.Time,
	completedAt *time.Time,
) *WorkflowExecution {
	stepsCopy := make([]ExecutionStep, len(steps))
	copy(stepsCopy, steps)

	return &WorkflowExecution{
		executionID:       executionID,
		workflowID:        workflowID,
		status:            status,
		inputData:         cloneAnyMap(inputData),
		outputData:        cloneAnyMap(outputData),
		errorMessage:      errorMessage,
		currentStepNumber: currentStepNumber,
		totalSteps:        totalSteps,
		steps:             stepsCopy,
		idempotencyKey:    idempotencyKey,
		startedAt:         startedAt,
		completedAt:       completedAt,
	}
}

// ExecutionID returns the execution's unique identifier.
func (e *WorkflowExecution) ExecutionID() uuid.UUID { return e.executionID }

// WorkflowID returns the workflow being executed.
func (e *WorkflowExecution) WorkflowID() uuid.UUID { return e.workflowID }

// Status returns the execution's current status.
func (e *WorkflowExecution) Status() ExecutionStatus { return e.status }

// InputData returns a copy of the execution's input data.
func (e *WorkflowExecution) InputData() map[string]any { return cloneAnyMap(e.inputData) }

// OutputData returns a copy of the execution's output data.
func (e *WorkflowExecution) OutputData() map[string]any { return cloneAnyMap(e.outputData) }

// ErrorMessage returns the execution's error message.
func (e *WorkflowExecution) ErrorMessage() *string { return e.errorMessage }

// CurrentStepNumber returns the current step number.
func (e *WorkflowExecution) CurrentStepNumber() int { return e.currentStepNumber }

// TotalSteps returns the total number of steps.
func (e *WorkflowExecution) TotalSteps() int { return e.totalSteps }

// Steps returns a copy of the execution's steps.
func (e *WorkflowExecution) Steps() []ExecutionStep {
	if e.steps == nil {
		return nil
	}

	result := make([]ExecutionStep, len(e.steps))
	copy(result, e.steps)

	return result
}

// IdempotencyKey returns the idempotency key.
func (e *WorkflowExecution) IdempotencyKey() *string { return e.idempotencyKey }

// StartedAt returns when the execution started.
func (e *WorkflowExecution) StartedAt() time.Time { return e.startedAt }

// CompletedAt returns when the execution completed.
func (e *WorkflowExecution) CompletedAt() *time.Time { return e.completedAt }

// IsTerminal returns true if the execution is in a terminal state.
func (e *WorkflowExecution) IsTerminal() bool {
	return e.status == ExecutionStatusCompleted || e.status == ExecutionStatusFailed
}

// MarkRunning transitions the execution from pending to running.
func (e *WorkflowExecution) MarkRunning() error {
	if e.status != ExecutionStatusPending {
		return fmt.Errorf("%w: cannot transition from %s to running", ErrInvalidExecutionTransition, e.status)
	}

	e.status = ExecutionStatusRunning

	return nil
}

// MarkCompleted transitions the execution to completed with output data.
func (e *WorkflowExecution) MarkCompleted(outputData map[string]any) error {
	if e.status != ExecutionStatusRunning {
		return fmt.Errorf("%w: cannot transition from %s to completed", ErrInvalidExecutionTransition, e.status)
	}

	e.status = ExecutionStatusCompleted
	e.outputData = cloneAnyMap(outputData)
	now := time.Now().UTC()
	e.completedAt = &now

	return nil
}

// MarkFailed transitions the execution to failed with an error message.
// Valid from any non-terminal state (pending or running) — execution can fail
// before it ever started running (e.g. "no trigger node found"). Only rejects
// re-entry from terminal states (completed/failed) to prevent overwrites.
func (e *WorkflowExecution) MarkFailed(errMsg string) error {
	if e.IsTerminal() {
		return fmt.Errorf("%w: cannot fail execution in %s state", ErrExecutionAlreadyTerminal, e.status)
	}

	e.status = ExecutionStatusFailed
	e.errorMessage = &errMsg
	now := time.Now().UTC()
	e.completedAt = &now

	return nil
}

// AddStep appends a step and updates the current step number.
func (e *WorkflowExecution) AddStep(step ExecutionStep) {
	e.steps = append(e.steps, step)
	e.currentStepNumber = step.StepNumber()
}

// SetCurrentStep updates the current step number.
func (e *WorkflowExecution) SetCurrentStep(n int) {
	e.currentStepNumber = n
}

// Snapshot returns a deep copy of the execution, safe to return to callers
// while the original is mutated by a background goroutine.
func (e *WorkflowExecution) Snapshot() *WorkflowExecution {
	return NewWorkflowExecutionFromDB(
		e.executionID,
		e.workflowID,
		e.status,
		cloneAnyMap(e.inputData),
		cloneAnyMap(e.outputData),
		e.errorMessage,
		e.currentStepNumber,
		e.totalSteps,
		e.steps,
		e.idempotencyKey,
		e.startedAt,
		e.completedAt,
	)
}

// LastCompletedStepNumber returns the step number of the last completed step,
// or 0 if no steps have been completed.
func (e *WorkflowExecution) LastCompletedStepNumber() int {
	last := 0
	for _, step := range e.steps {
		if step.Status() == StepStatusCompleted && step.StepNumber() > last {
			last = step.StepNumber()
		}
	}

	return last
}
