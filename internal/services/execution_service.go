// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package services

import (
	"context"

	"github.com/LerianStudio/flowker/internal/services/command"
	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// ExecutionService is a facade that combines execution commands and queries.
type ExecutionService struct {
	executeCmd      *command.ExecuteWorkflowCommand
	getQuery        *query.GetExecutionQuery
	getResultsQuery *query.GetExecutionResultsQuery
	listQuery       *query.ListExecutionsQuery
}

// NewExecutionService creates a new ExecutionService facade.
func NewExecutionService(
	executeCmd *command.ExecuteWorkflowCommand,
	getQuery *query.GetExecutionQuery,
	getResultsQuery *query.GetExecutionResultsQuery,
	listQuery *query.ListExecutionsQuery,
) (*ExecutionService, error) {
	if executeCmd == nil || getQuery == nil || getResultsQuery == nil || listQuery == nil {
		return nil, ErrExecutionServiceNilDependency
	}

	return &ExecutionService{
		executeCmd:      executeCmd,
		getQuery:        getQuery,
		getResultsQuery: getResultsQuery,
		listQuery:       listQuery,
	}, nil
}

// Execute starts a workflow execution.
func (s *ExecutionService) Execute(ctx context.Context, workflowID uuid.UUID, input *model.ExecuteWorkflowInput, idempotencyKey *string) (*model.WorkflowExecution, error) {
	return s.executeCmd.Execute(ctx, workflowID, input, idempotencyKey)
}

// GetByID retrieves an execution by its ID.
func (s *ExecutionService) GetByID(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error) {
	return s.getQuery.Execute(ctx, id)
}

// GetResults retrieves execution results by execution ID.
func (s *ExecutionService) GetResults(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error) {
	return s.getResultsQuery.Execute(ctx, id)
}

// List retrieves executions with filtering and pagination.
func (s *ExecutionService) List(ctx context.Context, filter query.ExecutionListFilter) (*query.ExecutionListResult, error) {
	return s.listQuery.Execute(ctx, filter)
}

// RecoverIncompleteExecutions recovers executions interrupted by restart.
func (s *ExecutionService) RecoverIncompleteExecutions(ctx context.Context) error {
	return s.executeCmd.RecoverIncompleteExecutions(ctx)
}
