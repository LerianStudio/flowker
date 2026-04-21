// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package command

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// setSSRFAllowPrivateForTest overrides the cached SSRF policy for a single test
// and restores the previous value on cleanup. It also forces the Once to run
// (with the env-backed initializer) before saving prev, so the original value
// is preserved across nested subtests.
func setSSRFAllowPrivateForTest(t *testing.T, value bool) {
	t.Helper()
	ssrfOptions() // initialize the cached default once
	prev := ssrfAllowPrivate
	ssrfAllowPrivate = value
	t.Cleanup(func() { ssrfAllowPrivate = prev })
}

func TestNewTestProviderConfigConnectivityCommand_NilRepository(t *testing.T) {
	cmd, err := NewTestProviderConfigConnectivityCommand(nil, nil)

	require.Nil(t, cmd)
	require.ErrorIs(t, err, ErrTestProviderConfigConnectivityNilRepo)
}

func TestNewTestProviderConfigConnectivityCommand_NilHTTPClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockProviderConfigRepository(ctrl)

	cmd, err := NewTestProviderConfigConnectivityCommand(mockRepo, nil)

	require.NotNil(t, cmd)
	require.NoError(t, err)
	// Should create a default HTTP client
}

// providerTestServerConfig holds the configuration for a test HTTP server used in
// provider configuration connectivity test cases.
type providerTestServerConfig struct {
	server     *httptest.Server
	httpClient *http.Client
}

func TestTestProviderConfigConnectivityCommand_Execute(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		configID  uuid.UUID
		setup     func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig)
		wantErr   error
		wantErrIs bool
		validate  func(t *testing.T, result *model.ProviderConfigTestResult)
	}{
		{
			name:     "success - all stages pass",
			configID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			setup: func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig) {
				t.Helper()
				setSSRFAllowPrivateForTest(t, true)

				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"status": "ok"}`))
				}))

				mockRepo := NewMockProviderConfigRepository(ctrl)

				providerConfig := model.ReconstructProviderConfiguration(
					configID, "test-config", nil,
					"my-provider",
					map[string]any{
						"base_url": mockServer.URL,
						"api_key":  "test-key-123",
					},
					model.ProviderConfigStatusActive,
					nil,
					fixedTime, fixedTime,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), configID).Return(providerConfig, nil)

				return mockRepo, &providerTestServerConfig{
					server:     mockServer,
					httpClient: &http.Client{Timeout: 5 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ProviderConfigTestResult) {
				t.Helper()
				assert.Equal(t, uuid.MustParse("11111111-1111-1111-1111-111111111111"), result.ProviderConfigID())
				assert.Equal(t, "my-provider", result.ProviderID())
				assert.Equal(t, model.TestOverallStatusPassed, result.OverallStatus())
				assert.True(t, result.IsPassed())
				assert.Len(t, result.Stages(), 3)
			},
		},
		{
			name:     "provider config not found",
			configID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			setup: func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig) {
				t.Helper()

				mockRepo := NewMockProviderConfigRepository(ctrl)
				mockRepo.EXPECT().FindByID(gomock.Any(), configID).Return(nil, constant.ErrProviderConfigNotFound)

				return mockRepo, &providerTestServerConfig{}
			},
			wantErr:   constant.ErrProviderConfigNotFound,
			wantErrIs: true,
		},
		{
			name:     "config missing base_url",
			configID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			setup: func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig) {
				t.Helper()

				mockRepo := NewMockProviderConfigRepository(ctrl)

				providerConfig := model.ReconstructProviderConfiguration(
					configID, "test-config", nil,
					"my-provider",
					map[string]any{
						"some_field": "value",
					},
					model.ProviderConfigStatusActive,
					nil,
					fixedTime, fixedTime,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), configID).Return(providerConfig, nil)

				return mockRepo, &providerTestServerConfig{}
			},
			wantErr:   constant.ErrProviderConfigMissingBaseURL,
			wantErrIs: true,
		},
		{
			name:     "config with malformed base_url",
			configID: uuid.MustParse("33333333-3333-3333-3333-333333333334"),
			setup: func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig) {
				t.Helper()

				mockRepo := NewMockProviderConfigRepository(ctrl)

				providerConfig := model.ReconstructProviderConfiguration(
					configID, "test-config", nil,
					"my-provider",
					map[string]any{
						"base_url": "not-a-url",
					},
					model.ProviderConfigStatusActive,
					nil,
					fixedTime, fixedTime,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), configID).Return(providerConfig, nil)

				return mockRepo, &providerTestServerConfig{}
			},
			wantErr:   constant.ErrProviderConfigMissingBaseURL,
			wantErrIs: true,
		},
		{
			name:     "SSRF blocked - private IP",
			configID: uuid.MustParse("33333333-3333-3333-3333-333333333335"),
			setup: func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig) {
				t.Helper()

				mockRepo := NewMockProviderConfigRepository(ctrl)

				providerConfig := model.ReconstructProviderConfiguration(
					configID, "test-config", nil,
					"my-provider",
					map[string]any{
						"base_url": "http://127.0.0.1:8080",
					},
					model.ProviderConfigStatusActive,
					nil,
					fixedTime, fixedTime,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), configID).Return(providerConfig, nil)

				return mockRepo, &providerTestServerConfig{}
			},
			wantErr:   constant.ErrProviderConfigSSRFBlocked,
			wantErrIs: true,
		},
		{
			name:     "connectivity failed - connection refused",
			configID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			setup: func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig) {
				t.Helper()
				setSSRFAllowPrivateForTest(t, true)

				// Create and immediately close server to simulate connection refused
				closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				closedServerURL := closedServer.URL
				closedServer.Close()

				mockRepo := NewMockProviderConfigRepository(ctrl)

				providerConfig := model.ReconstructProviderConfiguration(
					configID, "test-config", nil,
					"my-provider",
					map[string]any{
						"base_url": closedServerURL,
					},
					model.ProviderConfigStatusActive,
					nil,
					fixedTime, fixedTime,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), configID).Return(providerConfig, nil)

				return mockRepo, &providerTestServerConfig{
					httpClient: &http.Client{Timeout: 1 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ProviderConfigTestResult) {
				t.Helper()
				assert.NotEqual(t, model.TestOverallStatusPassed, result.OverallStatus())

				stages := result.Stages()
				require.Len(t, stages, 3)

				// Connectivity should fail
				assert.Equal(t, model.TestStageConnectivity, stages[0].Name())
				assert.Equal(t, model.TestStageStatusFailed, stages[0].Status())

				// Auth and E2E should be skipped due to connectivity failure
				assert.Equal(t, model.TestStageAuthentication, stages[1].Name())
				assert.Equal(t, model.TestStageStatusSkipped, stages[1].Status())

				assert.Equal(t, model.TestStageEndToEnd, stages[2].Name())
				assert.Equal(t, model.TestStageStatusSkipped, stages[2].Status())
			},
		},
		{
			name:     "no auth credentials - auth stage skipped",
			configID: uuid.MustParse("55555555-5555-5555-5555-555555555555"),
			setup: func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig) {
				t.Helper()
				setSSRFAllowPrivateForTest(t, true)

				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"status": "ok"}`))
				}))

				mockRepo := NewMockProviderConfigRepository(ctrl)

				providerConfig := model.ReconstructProviderConfiguration(
					configID, "test-config", nil,
					"my-provider",
					map[string]any{
						"base_url": mockServer.URL,
					},
					model.ProviderConfigStatusActive,
					nil,
					fixedTime, fixedTime,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), configID).Return(providerConfig, nil)

				return mockRepo, &providerTestServerConfig{
					server:     mockServer,
					httpClient: &http.Client{Timeout: 5 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ProviderConfigTestResult) {
				t.Helper()

				var authStage *model.StageTestResult

				for _, stage := range result.Stages() {
					if stage.Name() == model.TestStageAuthentication {
						authStage = &stage

						break
					}
				}

				require.NotNil(t, authStage)
				assert.Equal(t, model.TestStageStatusSkipped, authStage.Status())
			},
		},
		{
			name:     "authentication failed - unauthorized",
			configID: uuid.MustParse("66666666-6666-6666-6666-666666666666"),
			setup: func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig) {
				t.Helper()
				setSSRFAllowPrivateForTest(t, true)

				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "" {
						w.WriteHeader(http.StatusUnauthorized)
						_, _ = w.Write([]byte(`{"error": "unauthorized"}`))

						return
					}

					w.WriteHeader(http.StatusOK)
				}))

				mockRepo := NewMockProviderConfigRepository(ctrl)

				providerConfig := model.ReconstructProviderConfiguration(
					configID, "test-config", nil,
					"my-provider",
					map[string]any{
						"base_url": mockServer.URL,
						"api_key":  "invalid-key",
					},
					model.ProviderConfigStatusActive,
					nil,
					fixedTime, fixedTime,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), configID).Return(providerConfig, nil)

				return mockRepo, &providerTestServerConfig{
					server:     mockServer,
					httpClient: &http.Client{Timeout: 5 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ProviderConfigTestResult) {
				t.Helper()
				assert.NotEqual(t, model.TestOverallStatusPassed, result.OverallStatus())

				stages := result.Stages()
				require.Len(t, stages, 3)

				// Connectivity should pass
				assert.Equal(t, model.TestStageConnectivity, stages[0].Name())
				assert.Equal(t, model.TestStageStatusPassed, stages[0].Status())

				// Auth should fail
				assert.Equal(t, model.TestStageAuthentication, stages[1].Name())
				assert.Equal(t, model.TestStageStatusFailed, stages[1].Status())

				// E2E should be skipped due to auth failure
				assert.Equal(t, model.TestStageEndToEnd, stages[2].Name())
				assert.Equal(t, model.TestStageStatusSkipped, stages[2].Status())
			},
		},
		{
			name:     "server error - 500",
			configID: uuid.MustParse("77777777-7777-7777-7777-777777777777"),
			setup: func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig) {
				t.Helper()
				setSSRFAllowPrivateForTest(t, true)

				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error": "internal error"}`))
				}))

				mockRepo := NewMockProviderConfigRepository(ctrl)

				providerConfig := model.ReconstructProviderConfiguration(
					configID, "test-config", nil,
					"my-provider",
					map[string]any{
						"base_url": mockServer.URL,
					},
					model.ProviderConfigStatusActive,
					nil,
					fixedTime, fixedTime,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), configID).Return(providerConfig, nil)

				return mockRepo, &providerTestServerConfig{
					server:     mockServer,
					httpClient: &http.Client{Timeout: 5 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ProviderConfigTestResult) {
				t.Helper()

				stages := result.Stages()
				require.Len(t, stages, 3)

				// Connectivity should fail on 500
				assert.Equal(t, model.TestStageConnectivity, stages[0].Name())
				assert.Equal(t, model.TestStageStatusFailed, stages[0].Status())

				// Auth and E2E should be skipped
				assert.Equal(t, model.TestStageAuthentication, stages[1].Name())
				assert.Equal(t, model.TestStageStatusSkipped, stages[1].Status())

				assert.Equal(t, model.TestStageEndToEnd, stages[2].Name())
				assert.Equal(t, model.TestStageStatusSkipped, stages[2].Status())
			},
		},
		{
			name:     "custom headers authentication success",
			configID: uuid.MustParse("88888888-8888-8888-8888-888888888888"),
			setup: func(t *testing.T, ctrl *gomock.Controller, configID uuid.UUID) (*MockProviderConfigRepository, *providerTestServerConfig) {
				t.Helper()
				setSSRFAllowPrivateForTest(t, true)

				expectedKey := "my-secret-key"
				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-API-Key") == expectedKey {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"status": "authenticated"}`))

						return
					}

					w.WriteHeader(http.StatusUnauthorized)
				}))

				mockRepo := NewMockProviderConfigRepository(ctrl)

				providerConfig := model.ReconstructProviderConfiguration(
					configID, "test-config", nil,
					"my-provider",
					map[string]any{
						"base_url": mockServer.URL,
						"headers": map[string]any{
							"X-API-Key": expectedKey,
						},
					},
					model.ProviderConfigStatusActive,
					nil,
					fixedTime, fixedTime,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), configID).Return(providerConfig, nil)

				return mockRepo, &providerTestServerConfig{
					server:     mockServer,
					httpClient: &http.Client{Timeout: 5 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ProviderConfigTestResult) {
				t.Helper()
				assert.Equal(t, model.TestOverallStatusPassed, result.OverallStatus())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx := context.Background()
			mockRepo, serverCfg := tt.setup(t, ctrl, tt.configID)

			if serverCfg.server != nil {
				defer serverCfg.server.Close()
			}

			cmd, err := NewTestProviderConfigConnectivityCommand(mockRepo, serverCfg.httpClient)
			require.NoError(t, err)

			result, err := cmd.Execute(ctx, tt.configID)

			if tt.wantErr != nil {
				if tt.wantErrIs {
					require.ErrorIs(t, err, tt.wantErr)
				} else {
					require.Error(t, err)
				}

				require.Nil(t, result)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestTestProviderConfigConnectivityCommand_Execute_NilContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockProviderConfigRepository(ctrl)

	cmd, err := NewTestProviderConfigConnectivityCommand(mockRepo, &http.Client{Timeout: 5 * time.Second})
	require.NoError(t, err)

	//nolint:staticcheck // intentionally passing nil context to test guard
	result, err := cmd.Execute(nil, uuid.New())

	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "context cannot be nil")
}

func TestExtractBaseURL_SSRFBlocking(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		allowPriv bool
		wantErr   bool
	}{
		{
			name:    "blocks loopback 127.0.0.1 by default",
			baseURL: "http://127.0.0.1:8080",
			wantErr: true,
		},
		{
			name:    "blocks private 10.x network",
			baseURL: "http://10.0.0.1:80",
			wantErr: true,
		},
		{
			name:    "blocks cloud metadata endpoint",
			baseURL: "http://169.254.169.254/latest/meta-data",
			wantErr: true,
		},
		{
			name:    "blocks IPv6 loopback",
			baseURL: "http://[::1]:8080",
			wantErr: true,
		},
		{
			name:    "blocks localhost",
			baseURL: "http://localhost:8080",
			wantErr: true,
		},
		{
			name:      "allows 127.0.0.1 when SSRF_ALLOW_PRIVATE=true",
			baseURL:   "http://127.0.0.1:8080",
			allowPriv: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setSSRFAllowPrivateForTest(t, tt.allowPriv)

			config := map[string]any{
				"base_url": tt.baseURL,
			}

			_, _, err := extractBaseURL(context.Background(), config)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "SSRF blocked")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestTestProviderConfigConnectivityCommand_Execute_CanceledContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockProviderConfigRepository(ctrl)

	cmd, err := NewTestProviderConfigConnectivityCommand(mockRepo, &http.Client{Timeout: 5 * time.Second})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result, err := cmd.Execute(ctx, uuid.New())

	require.Error(t, err)
	require.Nil(t, result)
	assert.ErrorIs(t, err, context.Canceled)
}
