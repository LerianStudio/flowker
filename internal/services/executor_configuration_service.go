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

// ExecutorConfigurationService is a facade that combines executor configuration commands and queries.
type ExecutorConfigurationService struct {
	createCmd           *command.CreateExecutorConfigCommand
	updateCmd           *command.UpdateExecutorConfigCommand
	markConfigured      *command.MarkConfiguredCommand
	markTested          *command.MarkTestedCommand
	testConnectivityCmd *command.TestExecutorConnectivityCommand
	activateCmd         *command.ActivateExecutorConfigCommand
	disableCmd          *command.DisableExecutorConfigCommand
	enableCmd           *command.EnableExecutorConfigCommand
	deleteCmd           *command.DeleteExecutorConfigCommand
	getQuery            *query.GetExecutorConfigQuery
	getByNameQ          *query.GetExecutorConfigByNameQuery
	listQuery           *query.ListExecutorConfigsQuery
	existsQuery         *query.ExistsExecutorConfigQuery
}

// NewExecutorConfigurationService creates a new ExecutorConfigurationService facade.
// Returns error if any required dependency is nil.
func NewExecutorConfigurationService(
	createCmd *command.CreateExecutorConfigCommand,
	updateCmd *command.UpdateExecutorConfigCommand,
	markConfigured *command.MarkConfiguredCommand,
	markTested *command.MarkTestedCommand,
	testConnectivityCmd *command.TestExecutorConnectivityCommand,
	activateCmd *command.ActivateExecutorConfigCommand,
	disableCmd *command.DisableExecutorConfigCommand,
	enableCmd *command.EnableExecutorConfigCommand,
	deleteCmd *command.DeleteExecutorConfigCommand,
	getQuery *query.GetExecutorConfigQuery,
	getByNameQ *query.GetExecutorConfigByNameQuery,
	listQuery *query.ListExecutorConfigsQuery,
	existsQuery *query.ExistsExecutorConfigQuery,
) (*ExecutorConfigurationService, error) {
	if createCmd == nil || updateCmd == nil || markConfigured == nil ||
		markTested == nil || testConnectivityCmd == nil || activateCmd == nil ||
		disableCmd == nil || enableCmd == nil || deleteCmd == nil ||
		getQuery == nil || getByNameQ == nil || listQuery == nil || existsQuery == nil {
		return nil, ErrExecutorConfigServiceNilDependency
	}

	return &ExecutorConfigurationService{
		createCmd:           createCmd,
		updateCmd:           updateCmd,
		markConfigured:      markConfigured,
		markTested:          markTested,
		testConnectivityCmd: testConnectivityCmd,
		activateCmd:         activateCmd,
		disableCmd:          disableCmd,
		enableCmd:           enableCmd,
		deleteCmd:           deleteCmd,
		getQuery:            getQuery,
		getByNameQ:          getByNameQ,
		listQuery:           listQuery,
		existsQuery:         existsQuery,
	}, nil
}

// Create creates a new executor configuration.
func (s *ExecutorConfigurationService) Create(ctx context.Context, input *model.CreateExecutorConfigurationInput) (*model.ExecutorConfiguration, error) {
	return s.createCmd.Execute(ctx, input)
}

// Update updates an existing executor configuration.
func (s *ExecutorConfigurationService) Update(ctx context.Context, id uuid.UUID, input *model.UpdateExecutorConfigurationInput) (*model.ExecutorConfiguration, error) {
	return s.updateCmd.Execute(ctx, id, input)
}

// MarkConfigured transitions an executor configuration to configured status.
func (s *ExecutorConfigurationService) MarkConfigured(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error) {
	return s.markConfigured.Execute(ctx, id)
}

// MarkTested transitions an executor configuration to tested status.
func (s *ExecutorConfigurationService) MarkTested(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error) {
	return s.markTested.Execute(ctx, id)
}

// TestConnectivity tests executor connectivity and returns detailed results.
func (s *ExecutorConfigurationService) TestConnectivity(ctx context.Context, id uuid.UUID) (*model.ExecutorTestResult, error) {
	return s.testConnectivityCmd.Execute(ctx, id)
}

// Activate transitions an executor configuration to active status.
func (s *ExecutorConfigurationService) Activate(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error) {
	return s.activateCmd.Execute(ctx, id)
}

// Disable transitions an executor configuration to disabled status.
func (s *ExecutorConfigurationService) Disable(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error) {
	return s.disableCmd.Execute(ctx, id)
}

// Enable re-enables a disabled executor configuration.
func (s *ExecutorConfigurationService) Enable(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error) {
	return s.enableCmd.Execute(ctx, id)
}

// Delete removes an executor configuration.
func (s *ExecutorConfigurationService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.deleteCmd.Execute(ctx, id)
}

// GetByID retrieves an executor configuration by its ID.
func (s *ExecutorConfigurationService) GetByID(ctx context.Context, id uuid.UUID) (*model.ExecutorConfiguration, error) {
	return s.getQuery.Execute(ctx, id)
}

// GetByName retrieves an executor configuration by its name.
func (s *ExecutorConfigurationService) GetByName(ctx context.Context, name string) (*model.ExecutorConfiguration, error) {
	return s.getByNameQ.Execute(ctx, name)
}

// List retrieves executor configurations with optional filtering and pagination.
func (s *ExecutorConfigurationService) List(ctx context.Context, filter query.ExecutorConfigListFilter) (*query.ExecutorConfigListResult, error) {
	return s.listQuery.Execute(ctx, filter)
}

// Exists checks if an executor configuration with the given ID exists.
func (s *ExecutorConfigurationService) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.existsQuery.Execute(ctx, id)
}

// ExistsByName checks if an executor configuration with the given name exists.
func (s *ExecutorConfigurationService) ExistsByName(ctx context.Context, name string) (bool, error) {
	return s.existsQuery.ExecuteByName(ctx, name)
}
