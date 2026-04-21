// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package executor

import "context"

// Runner defines the contract for executing an executor's action.
// Each executor has a corresponding runner that performs the actual work.
type Runner interface {
	// ExecutorID returns the ID of the executor this runner handles.
	ExecutorID() ID

	// Execute performs the executor's action with the given input.
	// Returns the execution result or an error.
	Execute(ctx context.Context, input ExecutionInput) (ExecutionResult, error)
}
