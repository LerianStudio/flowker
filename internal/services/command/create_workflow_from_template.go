// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/model"
)

// CreateWorkflowFromTemplateCommand handles workflow creation from a template.
type CreateWorkflowFromTemplateCommand struct {
	catalog   executor.Catalog
	createCmd *CreateWorkflowCommand
}

// NewCreateWorkflowFromTemplateCommand creates a new CreateWorkflowFromTemplateCommand.
// Returns error if required dependencies are nil.
func NewCreateWorkflowFromTemplateCommand(
	catalog executor.Catalog,
	createCmd *CreateWorkflowCommand,
) (*CreateWorkflowFromTemplateCommand, error) {
	if catalog == nil {
		return nil, ErrCreateWorkflowFromTemplateNilCatalog
	}

	if createCmd == nil {
		return nil, ErrCreateWorkflowFromTemplateNilCreateCmd
	}

	return &CreateWorkflowFromTemplateCommand{
		catalog:   catalog,
		createCmd: createCmd,
	}, nil
}

// Execute creates a new workflow from a template.
func (c *CreateWorkflowFromTemplateCommand) Execute(ctx context.Context, input *model.CreateWorkflowFromTemplateInput) (*model.Workflow, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if input == nil {
		return nil, ErrCreateWorkflowFromTemplateNilInput
	}

	templateID := executor.TemplateID(input.TemplateID)

	tmpl, err := c.catalog.GetTemplate(templateID)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if err := tmpl.ValidateParams(input.Params); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	result, err := tmpl.Build(input.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to build workflow from template %s: %w", templateID, err)
	}

	workflowInput, ok := result.(*model.CreateWorkflowInput)
	if !ok {
		return nil, fmt.Errorf("template %s returned unexpected type: expected *model.CreateWorkflowInput", templateID)
	}

	return c.createCmd.Execute(ctx, workflowInput)
}
