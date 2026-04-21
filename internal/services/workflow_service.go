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

// WorkflowService is a facade that combines workflow commands and queries.
type WorkflowService struct {
	createCmd             *command.CreateWorkflowCommand
	createFromTemplateCmd *command.CreateWorkflowFromTemplateCommand
	updateCmd             *command.UpdateWorkflowCommand
	cloneCmd              *command.CloneWorkflowCommand
	activateCmd           *command.ActivateWorkflowCommand
	deactivateCmd         *command.DeactivateWorkflowCommand
	moveToDraftCmd        *command.MoveToDraftWorkflowCommand
	deleteCmd             *command.DeleteWorkflowCommand
	getQuery              *query.GetWorkflowQuery
	getByNameQ            *query.GetWorkflowByNameQuery
	listQuery             *query.ListWorkflowsQuery
}

// NewWorkflowService creates a new WorkflowService facade.
// Returns error if any required dependency is nil.
func NewWorkflowService(
	createCmd *command.CreateWorkflowCommand,
	createFromTemplateCmd *command.CreateWorkflowFromTemplateCommand,
	updateCmd *command.UpdateWorkflowCommand,
	cloneCmd *command.CloneWorkflowCommand,
	activateCmd *command.ActivateWorkflowCommand,
	deactivateCmd *command.DeactivateWorkflowCommand,
	moveToDraftCmd *command.MoveToDraftWorkflowCommand,
	deleteCmd *command.DeleteWorkflowCommand,
	getQuery *query.GetWorkflowQuery,
	getByNameQ *query.GetWorkflowByNameQuery,
	listQuery *query.ListWorkflowsQuery,
) (*WorkflowService, error) {
	if createCmd == nil || createFromTemplateCmd == nil || updateCmd == nil || cloneCmd == nil ||
		activateCmd == nil || deactivateCmd == nil || moveToDraftCmd == nil || deleteCmd == nil ||
		getQuery == nil || getByNameQ == nil || listQuery == nil {
		return nil, ErrWorkflowServiceNilDependency
	}

	return &WorkflowService{
		createCmd:             createCmd,
		createFromTemplateCmd: createFromTemplateCmd,
		updateCmd:             updateCmd,
		cloneCmd:              cloneCmd,
		activateCmd:           activateCmd,
		deactivateCmd:         deactivateCmd,
		moveToDraftCmd:        moveToDraftCmd,
		deleteCmd:             deleteCmd,
		getQuery:              getQuery,
		getByNameQ:            getByNameQ,
		listQuery:             listQuery,
	}, nil
}

// Create creates a new workflow.
func (s *WorkflowService) Create(ctx context.Context, input *model.CreateWorkflowInput) (*model.Workflow, error) {
	return s.createCmd.Execute(ctx, input)
}

// CreateFromTemplate creates a new workflow from a registered template.
func (s *WorkflowService) CreateFromTemplate(ctx context.Context, input *model.CreateWorkflowFromTemplateInput) (*model.Workflow, error) {
	return s.createFromTemplateCmd.Execute(ctx, input)
}

// Update updates an existing workflow.
func (s *WorkflowService) Update(ctx context.Context, id uuid.UUID, input *model.UpdateWorkflowInput) (*model.Workflow, error) {
	return s.updateCmd.Execute(ctx, id, input)
}

// Clone creates a copy of an existing workflow.
func (s *WorkflowService) Clone(ctx context.Context, id uuid.UUID, input *model.CloneWorkflowInput) (*model.Workflow, error) {
	return s.cloneCmd.Execute(ctx, id, input)
}

// Activate transitions a workflow to active status.
func (s *WorkflowService) Activate(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	return s.activateCmd.Execute(ctx, id)
}

// Deactivate transitions a workflow to inactive status.
func (s *WorkflowService) Deactivate(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	return s.deactivateCmd.Execute(ctx, id)
}

// MoveToDraft transitions a workflow from inactive to draft status.
func (s *WorkflowService) MoveToDraft(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	return s.moveToDraftCmd.Execute(ctx, id)
}

// Delete removes a workflow.
func (s *WorkflowService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.deleteCmd.Execute(ctx, id)
}

// GetByID retrieves a workflow by its ID.
func (s *WorkflowService) GetByID(ctx context.Context, id uuid.UUID) (*model.Workflow, error) {
	return s.getQuery.Execute(ctx, id)
}

// GetByName retrieves a workflow by its name.
func (s *WorkflowService) GetByName(ctx context.Context, name string) (*model.Workflow, error) {
	return s.getByNameQ.Execute(ctx, name)
}

// List retrieves workflows with optional filtering and pagination.
func (s *WorkflowService) List(ctx context.Context, filter query.WorkflowListFilter) (*query.WorkflowListResult, error) {
	return s.listQuery.Execute(ctx, filter)
}
