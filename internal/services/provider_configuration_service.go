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

// ProviderConfigurationService is a facade that combines provider configuration commands and queries.
type ProviderConfigurationService struct {
	createCmd           *command.CreateProviderConfigCommand
	updateCmd           *command.UpdateProviderConfigCommand
	deleteCmd           *command.DeleteProviderConfigCommand
	disableCmd          *command.DisableProviderConfigCommand
	enableCmd           *command.EnableProviderConfigCommand
	testConnectivityCmd *command.TestProviderConfigConnectivityCommand
	getQuery            *query.GetProviderConfigByIDQuery
	listQuery           *query.ListProviderConfigsQuery
}

// NewProviderConfigurationService creates a new ProviderConfigurationService facade.
// Returns error if any required dependency is nil.
func NewProviderConfigurationService(
	createCmd *command.CreateProviderConfigCommand,
	updateCmd *command.UpdateProviderConfigCommand,
	deleteCmd *command.DeleteProviderConfigCommand,
	disableCmd *command.DisableProviderConfigCommand,
	enableCmd *command.EnableProviderConfigCommand,
	testConnectivityCmd *command.TestProviderConfigConnectivityCommand,
	getQuery *query.GetProviderConfigByIDQuery,
	listQuery *query.ListProviderConfigsQuery,
) (*ProviderConfigurationService, error) {
	if createCmd == nil || updateCmd == nil || deleteCmd == nil ||
		disableCmd == nil || enableCmd == nil || testConnectivityCmd == nil ||
		getQuery == nil || listQuery == nil {
		return nil, ErrProviderConfigServiceNilDependency
	}

	return &ProviderConfigurationService{
		createCmd:           createCmd,
		updateCmd:           updateCmd,
		deleteCmd:           deleteCmd,
		disableCmd:          disableCmd,
		enableCmd:           enableCmd,
		testConnectivityCmd: testConnectivityCmd,
		getQuery:            getQuery,
		listQuery:           listQuery,
	}, nil
}

// Create creates a new provider configuration.
func (s *ProviderConfigurationService) Create(ctx context.Context, input *model.CreateProviderConfigurationInput) (*model.ProviderConfiguration, error) {
	return s.createCmd.Execute(ctx, input)
}

// Update updates an existing provider configuration.
func (s *ProviderConfigurationService) Update(ctx context.Context, id uuid.UUID, input *model.UpdateProviderConfigurationInput) (*model.ProviderConfiguration, error) {
	return s.updateCmd.Execute(ctx, id, input)
}

// Delete removes a provider configuration.
func (s *ProviderConfigurationService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.deleteCmd.Execute(ctx, id)
}

// Disable transitions a provider configuration to disabled status.
func (s *ProviderConfigurationService) Disable(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error) {
	return s.disableCmd.Execute(ctx, id)
}

// Enable re-enables a disabled provider configuration.
func (s *ProviderConfigurationService) Enable(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error) {
	return s.enableCmd.Execute(ctx, id)
}

// TestConnectivity tests provider configuration connectivity and returns detailed results.
func (s *ProviderConfigurationService) TestConnectivity(ctx context.Context, id uuid.UUID) (*model.ProviderConfigTestResult, error) {
	return s.testConnectivityCmd.Execute(ctx, id)
}

// GetByID retrieves a provider configuration by its ID.
func (s *ProviderConfigurationService) GetByID(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error) {
	return s.getQuery.Execute(ctx, id)
}

// List retrieves provider configurations with optional filtering and pagination.
func (s *ProviderConfigurationService) List(ctx context.Context, filter query.ProviderConfigListFilter) (*query.ProviderConfigListResult, error) {
	return s.listQuery.Execute(ctx, filter)
}
