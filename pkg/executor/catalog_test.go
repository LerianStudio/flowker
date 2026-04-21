// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package executor_test

import (
	"context"
	"testing"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testProvider is a minimal implementation of executor.Provider for testing.
type testProvider struct {
	id           executor.ProviderID
	name         string
	description  string
	version      string
	configSchema string
}

func (p *testProvider) ID() executor.ProviderID { return p.id }
func (p *testProvider) Name() string            { return p.name }
func (p *testProvider) Description() string     { return p.description }
func (p *testProvider) Version() string         { return p.version }
func (p *testProvider) ConfigSchema() string    { return p.configSchema }

// testExecutor is a minimal implementation of executor.Executor for testing.
type testExecutor struct {
	id         executor.ID
	name       string
	category   string
	version    string
	providerID executor.ProviderID
	schema     string
}

func (e *testExecutor) ID() executor.ID                       { return e.id }
func (e *testExecutor) Name() string                          { return e.name }
func (e *testExecutor) Category() string                      { return e.category }
func (e *testExecutor) Version() string                       { return e.version }
func (e *testExecutor) ProviderID() executor.ProviderID       { return e.providerID }
func (e *testExecutor) Schema() string                        { return e.schema }
func (e *testExecutor) ValidateConfig(_ map[string]any) error { return nil }

// testRunner is a minimal implementation of executor.Runner for testing.
type testRunner struct {
	executorID executor.ID
}

func (r *testRunner) ExecutorID() executor.ID { return r.executorID }
func (r *testRunner) Execute(_ context.Context, _ executor.ExecutionInput) (executor.ExecutionResult, error) {
	return executor.ExecutionResult{}, nil
}

func newTestProvider(id executor.ProviderID, name string) *testProvider {
	return &testProvider{
		id:           id,
		name:         name,
		description:  "Test provider: " + name,
		version:      "v1",
		configSchema: `{"type":"object"}`,
	}
}

func newTestExecutor(id executor.ID, providerID executor.ProviderID) *testExecutor {
	return &testExecutor{
		id:         id,
		name:       string(id),
		category:   "Test",
		version:    "v1",
		providerID: providerID,
		schema:     `{"type":"object"}`,
	}
}

func newTestRunner(executorID executor.ID) *testRunner {
	return &testRunner{executorID: executorID}
}

func TestRegisterProvider(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(catalog *executor.InMemoryCatalog)
		provider  executor.Provider
		executors []executor.ExecutorRegistration
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "register provider with executors",
			provider: newTestProvider("http", "HTTP"),
			executors: []executor.ExecutorRegistration{
				{
					Executor: newTestExecutor("http.request", "http"),
					Runner:   newTestRunner("http.request"),
				},
			},
			wantErr: false,
		},
		{
			name:      "register provider without executors",
			provider:  newTestProvider("empty", "Empty"),
			executors: nil,
			wantErr:   false,
		},
		{
			name:     "register provider with multiple executors",
			provider: newTestProvider("s3", "S3"),
			executors: []executor.ExecutorRegistration{
				{
					Executor: newTestExecutor("s3.put-object", "s3"),
					Runner:   newTestRunner("s3.put-object"),
				},
				{
					Executor: newTestExecutor("s3.get-object", "s3"),
					Runner:   newTestRunner("s3.get-object"),
				},
			},
			wantErr: false,
		},
		{
			name: "register duplicate provider returns error",
			setup: func(catalog *executor.InMemoryCatalog) {
				_ = catalog.RegisterProvider(
					newTestProvider("http", "HTTP"),
					nil,
				)
			},
			provider:  newTestProvider("http", "HTTP"),
			executors: nil,
			wantErr:   true,
			errMsg:    "provider already registered: http",
		},
		{
			name:     "register nil provider returns error",
			provider: nil,
			wantErr:  true,
		},
		{
			name:     "register provider with nil executor returns error",
			provider: newTestProvider("broken", "Broken"),
			executors: []executor.ExecutorRegistration{
				{
					Executor: nil,
					Runner:   newTestRunner("broken.exec"),
				},
			},
			wantErr: true,
		},
		{
			name:     "register provider with nil runner returns error",
			provider: newTestProvider("broken2", "Broken2"),
			executors: []executor.ExecutorRegistration{
				{
					Executor: newTestExecutor("broken2.exec", "broken2"),
					Runner:   nil,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := executor.NewCatalog()

			if tt.setup != nil {
				tt.setup(catalog)
			}

			err := catalog.RegisterProvider(tt.provider, tt.executors)

			if tt.wantErr {
				require.Error(t, err, "expected error for test case: %s", tt.name)

				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}

				return
			}

			require.NoError(t, err, "unexpected error for test case: %s", tt.name)
		})
	}
}

func TestGetProvider(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(catalog *executor.InMemoryCatalog)
		id       executor.ProviderID
		wantErr  bool
		wantName string
	}{
		{
			name: "get existing provider",
			setup: func(catalog *executor.InMemoryCatalog) {
				_ = catalog.RegisterProvider(
					newTestProvider("http", "HTTP"),
					nil,
				)
			},
			id:       "http",
			wantErr:  false,
			wantName: "HTTP",
		},
		{
			name:    "get non-existing provider returns error",
			id:      "unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := executor.NewCatalog()

			if tt.setup != nil {
				tt.setup(catalog)
			}

			provider, err := catalog.GetProvider(tt.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, provider)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantName, provider.Name())
		})
	}
}

func TestListProviders(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(catalog *executor.InMemoryCatalog)
		wantCount int
		wantIDs   []executor.ProviderID
	}{
		{
			name:      "empty catalog returns empty list",
			wantCount: 0,
		},
		{
			name: "single provider",
			setup: func(catalog *executor.InMemoryCatalog) {
				_ = catalog.RegisterProvider(
					newTestProvider("http", "HTTP"),
					nil,
				)
			},
			wantCount: 1,
			wantIDs:   []executor.ProviderID{"http"},
		},
		{
			name: "multiple providers are sorted by ID",
			setup: func(catalog *executor.InMemoryCatalog) {
				_ = catalog.RegisterProvider(newTestProvider("s3", "S3"), nil)
				_ = catalog.RegisterProvider(newTestProvider("http", "HTTP"), nil)
				_ = catalog.RegisterProvider(newTestProvider("midaz", "Midaz"), nil)
			},
			wantCount: 3,
			wantIDs:   []executor.ProviderID{"http", "midaz", "s3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := executor.NewCatalog()

			if tt.setup != nil {
				tt.setup(catalog)
			}

			providers := catalog.ListProviders()
			assert.Len(t, providers, tt.wantCount)

			if tt.wantIDs != nil {
				gotIDs := make([]executor.ProviderID, len(providers))
				for i, p := range providers {
					gotIDs[i] = p.ID()
				}

				assert.Equal(t, tt.wantIDs, gotIDs, "providers should be sorted by ID")
			}
		})
	}
}

func TestGetProviderExecutors(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(catalog *executor.InMemoryCatalog)
		id        executor.ProviderID
		wantErr   bool
		wantCount int
		wantIDs   []executor.ID
	}{
		{
			name: "get executors for provider with one executor",
			setup: func(catalog *executor.InMemoryCatalog) {
				_ = catalog.RegisterProvider(
					newTestProvider("http", "HTTP"),
					[]executor.ExecutorRegistration{
						{
							Executor: newTestExecutor("http.request", "http"),
							Runner:   newTestRunner("http.request"),
						},
					},
				)
			},
			id:        "http",
			wantCount: 1,
			wantIDs:   []executor.ID{"http.request"},
		},
		{
			name: "get executors for provider with multiple executors sorted by ID",
			setup: func(catalog *executor.InMemoryCatalog) {
				_ = catalog.RegisterProvider(
					newTestProvider("s3", "S3"),
					[]executor.ExecutorRegistration{
						{
							Executor: newTestExecutor("s3.put-object", "s3"),
							Runner:   newTestRunner("s3.put-object"),
						},
						{
							Executor: newTestExecutor("s3.get-object", "s3"),
							Runner:   newTestRunner("s3.get-object"),
						},
					},
				)
			},
			id:        "s3",
			wantCount: 2,
			wantIDs:   []executor.ID{"s3.get-object", "s3.put-object"},
		},
		{
			name: "get executors for provider with no executors",
			setup: func(catalog *executor.InMemoryCatalog) {
				_ = catalog.RegisterProvider(
					newTestProvider("empty", "Empty"),
					nil,
				)
			},
			id:        "empty",
			wantCount: 0,
		},
		{
			name:    "get executors for non-existing provider returns error",
			id:      "unknown",
			wantErr: true,
		},
		{
			name: "executors from different providers are isolated",
			setup: func(catalog *executor.InMemoryCatalog) {
				_ = catalog.RegisterProvider(
					newTestProvider("http", "HTTP"),
					[]executor.ExecutorRegistration{
						{
							Executor: newTestExecutor("http.request", "http"),
							Runner:   newTestRunner("http.request"),
						},
					},
				)
				_ = catalog.RegisterProvider(
					newTestProvider("s3", "S3"),
					[]executor.ExecutorRegistration{
						{
							Executor: newTestExecutor("s3.put-object", "s3"),
							Runner:   newTestRunner("s3.put-object"),
						},
					},
				)
			},
			id:        "http",
			wantCount: 1,
			wantIDs:   []executor.ID{"http.request"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog := executor.NewCatalog()

			if tt.setup != nil {
				tt.setup(catalog)
			}

			executors, err := catalog.GetProviderExecutors(tt.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, executors)

				return
			}

			require.NoError(t, err)
			assert.Len(t, executors, tt.wantCount)

			if tt.wantIDs != nil {
				gotIDs := make([]executor.ID, len(executors))
				for i, e := range executors {
					gotIDs[i] = e.ID()
				}

				assert.Equal(t, tt.wantIDs, gotIDs, "executors should be sorted by ID")
			}
		})
	}
}

func TestProviderExecutorsAreAlsoInGlobalList(t *testing.T) {
	catalog := executor.NewCatalog()

	err := catalog.RegisterProvider(
		newTestProvider("http", "HTTP"),
		[]executor.ExecutorRegistration{
			{
				Executor: newTestExecutor("http.request", "http"),
				Runner:   newTestRunner("http.request"),
			},
		},
	)
	require.NoError(t, err)

	// The executor should also be available via the global GetExecutor/ListExecutors.
	exec, err := catalog.GetExecutor("http.request")
	require.NoError(t, err)
	assert.Equal(t, executor.ID("http.request"), exec.ID())

	// The runner should also be retrievable.
	runner, err := catalog.GetRunner("http.request")
	require.NoError(t, err)
	assert.Equal(t, executor.ID("http.request"), runner.ExecutorID())

	// ListExecutors should include the provider's executor.
	allExecutors := catalog.ListExecutors()
	assert.Len(t, allExecutors, 1)
	assert.Equal(t, executor.ID("http.request"), allExecutors[0].ID())
}
