// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package executors_test

import (
	"testing"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterDefaults_TracerProviderRegistered(t *testing.T) {
	catalog := executor.NewCatalog()

	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	provider, err := catalog.GetProvider("tracer")
	require.NoError(t, err)
	assert.Equal(t, "Tracer", provider.Name())
	assert.Equal(t, "v1", provider.Version())
}

func TestRegisterDefaults_TracerValidateTransactionExecutor(t *testing.T) {
	catalog := executor.NewCatalog()

	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	exec, err := catalog.GetExecutor("tracer.validate-transaction")
	require.NoError(t, err)
	assert.Equal(t, "Tracer", exec.Category())
	assert.Equal(t, executor.ProviderID("tracer"), exec.ProviderID())
	assert.Equal(t, "Validate Transaction", exec.Name())
}

func TestRegisterDefaults_TracerListValidationsExecutor(t *testing.T) {
	catalog := executor.NewCatalog()

	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	exec, err := catalog.GetExecutor("tracer.list-validations")
	require.NoError(t, err)
	assert.Equal(t, "Tracer", exec.Category())
	assert.Equal(t, executor.ProviderID("tracer"), exec.ProviderID())
	assert.Equal(t, "List Validations", exec.Name())
}

func TestRegisterDefaults_TracerRunnersRegistered(t *testing.T) {
	catalog := executor.NewCatalog()

	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	runner1, err := catalog.GetRunner("tracer.validate-transaction")
	require.NoError(t, err)
	assert.NotNil(t, runner1)

	runner2, err := catalog.GetRunner("tracer.list-validations")
	require.NoError(t, err)
	assert.NotNil(t, runner2)
}

func TestRegisterDefaults_AllProvidersCount(t *testing.T) {
	catalog := executor.NewCatalog()

	err := executors.RegisterDefaults(catalog)
	require.NoError(t, err)

	providers := catalog.ListProviders()
	assert.Len(t, providers, 2)
}
