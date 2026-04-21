// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package executor

import "net/http"

// LogEmitter is a function that emits log entries during execution.
type LogEmitter func(level, message string, fields map[string]any)

// ExecutionInput contains all data needed to execute an executor action.
type ExecutionInput struct {
	// Config contains the executor configuration from the workflow node.
	// This has been validated against the executor's JSON Schema.
	Config map[string]any

	// Context contains data from the workflow execution context.
	// This includes outputs from previous nodes accessible via expressions.
	Context map[string]any

	// Credentials contains resolved credential values if the node references credentials.
	Credentials map[string]any

	// HTTPClient is an injectable HTTP client for making external requests.
	// If nil, http.DefaultClient should be used.
	HTTPClient *http.Client

	// WorkflowID is the ID of the workflow being executed.
	WorkflowID string

	// ExecutionID is the unique ID for this execution run.
	ExecutionID string

	// NodeID is the ID of the current node being executed.
	NodeID string

	// Emit is a function to emit log entries during execution.
	Emit LogEmitter
}

// ExecutionStatus represents the status of an execution result.
type ExecutionStatus string

const (
	// ExecutionStatusSuccess indicates the execution completed successfully.
	ExecutionStatusSuccess ExecutionStatus = "success"

	// ExecutionStatusError indicates the execution failed with an error.
	ExecutionStatusError ExecutionStatus = "error"
)

// ExecutionResult contains the output from an executor execution.
type ExecutionResult struct {
	// Data contains the output data from the execution.
	// This data is available to subsequent nodes via context expressions.
	Data map[string]any

	// Status indicates whether the execution succeeded or failed.
	Status ExecutionStatus

	// Error contains the error message if Status is ExecutionStatusError.
	Error string
}

// NewSuccessResult creates a successful execution result with the given data.
func NewSuccessResult(data map[string]any) ExecutionResult {
	return ExecutionResult{
		Data:   data,
		Status: ExecutionStatusSuccess,
	}
}

// NewErrorResult creates a failed execution result with the given error message.
func NewErrorResult(err string) ExecutionResult {
	return ExecutionResult{
		Status: ExecutionStatusError,
		Error:  err,
	}
}
