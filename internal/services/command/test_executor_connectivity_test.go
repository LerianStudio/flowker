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

	"github.com/LerianStudio/flowker/internal/testutil"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewTestExecutorConnectivityCommand_NilRepository(t *testing.T) {
	cmd, err := NewTestExecutorConnectivityCommand(nil, nil)

	require.Nil(t, cmd)
	require.ErrorIs(t, err, ErrTestExecutorConnectivityNilRepo)
}

func TestNewTestExecutorConnectivityCommand_NilHTTPClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockExecutorConfigRepository(ctrl)

	cmd, err := NewTestExecutorConnectivityCommand(mockRepo, nil)

	require.NotNil(t, cmd)
	require.NoError(t, err)
	// Should create a default HTTP client
}

// testServerConfig holds the configuration for a test HTTP server used in
// executor connectivity test cases.
type testServerConfig struct {
	server     *httptest.Server
	httpClient *http.Client
}

func TestTestExecutorConnectivityCommand_Execute(t *testing.T) {
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		executorID uuid.UUID
		setup      func(t *testing.T, ctrl *gomock.Controller, executorID uuid.UUID) (*MockExecutorConfigRepository, *testServerConfig)
		wantErr    error
		validate   func(t *testing.T, result *model.ExecutorTestResult)
	}{
		{
			name:       "success - all stages pass",
			executorID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			setup: func(t *testing.T, ctrl *gomock.Controller, executorID uuid.UUID) (*MockExecutorConfigRepository, *testServerConfig) {
				t.Helper()

				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"status": "ok"}`))
				}))

				mockRepo := NewMockExecutorConfigRepository(ctrl)

				endpoint, _ := model.NewExecutorEndpoint("health", "/health", "GET", 30)
				auth, _ := model.NewExecutorAuthentication("none", nil)
				executorConfig := model.NewExecutorConfigurationFromDB(
					executorID, "test-executor", testutil.StringPtr("Test executor"),
					mockServer.URL, []model.ExecutorEndpoint{*endpoint}, *auth,
					model.ExecutorConfigurationStatusConfigured, nil,
					fixedTime, fixedTime, nil,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), executorID).Return(executorConfig, nil)
				mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(model.ExecutorConfigurationStatusConfigured)).Return(nil)

				return mockRepo, &testServerConfig{
					server:     mockServer,
					httpClient: &http.Client{Timeout: 5 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ExecutorTestResult) {
				t.Helper()
				assert.Equal(t, uuid.MustParse("11111111-1111-1111-1111-111111111111"), result.ExecutorConfigID())
				assert.Equal(t, model.TestOverallStatusPassed, result.OverallStatus())
				assert.True(t, result.IsPassed())
				assert.Len(t, result.Stages(), 4)
			},
		},
		{
			name:       "executor config not found",
			executorID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			setup: func(t *testing.T, ctrl *gomock.Controller, executorID uuid.UUID) (*MockExecutorConfigRepository, *testServerConfig) {
				t.Helper()

				mockRepo := NewMockExecutorConfigRepository(ctrl)
				mockRepo.EXPECT().FindByID(gomock.Any(), executorID).Return(nil, constant.ErrExecutorConfigNotFound)

				return mockRepo, &testServerConfig{}
			},
			wantErr: constant.ErrExecutorConfigNotFound,
		},
		{
			name:       "connectivity failed - connection refused",
			executorID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			setup: func(t *testing.T, ctrl *gomock.Controller, executorID uuid.UUID) (*MockExecutorConfigRepository, *testServerConfig) {
				t.Helper()

				// Create and immediately close server to simulate connection refused
				closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
				closedServerURL := closedServer.URL
				closedServer.Close()

				mockRepo := NewMockExecutorConfigRepository(ctrl)

				endpoint, _ := model.NewExecutorEndpoint("health", "/health", "GET", 1)
				auth, _ := model.NewExecutorAuthentication("none", nil)
				executorConfig := model.NewExecutorConfigurationFromDB(
					executorID, "test-executor", testutil.StringPtr("Test executor"),
					closedServerURL, []model.ExecutorEndpoint{*endpoint}, *auth,
					model.ExecutorConfigurationStatusConfigured, nil,
					fixedTime, fixedTime, nil,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), executorID).Return(executorConfig, nil)

				return mockRepo, &testServerConfig{
					httpClient: &http.Client{Timeout: 1 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ExecutorTestResult) {
				t.Helper()
				assert.NotEqual(t, model.TestOverallStatusPassed, result.OverallStatus())

				var connectivityStage *model.StageTestResult

				for _, stage := range result.Stages() {
					if stage.Name() == model.TestStageConnectivity {
						connectivityStage = &stage

						break
					}
				}

				require.NotNil(t, connectivityStage)
				assert.Equal(t, model.TestStageStatusFailed, connectivityStage.Status())
			},
		},
		{
			name:       "no endpoints - stages skipped",
			executorID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			setup: func(t *testing.T, ctrl *gomock.Controller, executorID uuid.UUID) (*MockExecutorConfigRepository, *testServerConfig) {
				t.Helper()

				mockRepo := NewMockExecutorConfigRepository(ctrl)

				auth, _ := model.NewExecutorAuthentication("none", nil)
				executorConfig := model.NewExecutorConfigurationFromDB(
					executorID, "test-executor", testutil.StringPtr("Test executor"),
					"http://example.com", []model.ExecutorEndpoint{}, *auth,
					model.ExecutorConfigurationStatusConfigured, nil,
					fixedTime, fixedTime, nil,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), executorID).Return(executorConfig, nil)
				mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(model.ExecutorConfigurationStatusConfigured)).Return(nil)

				return mockRepo, &testServerConfig{}
			},
			validate: func(t *testing.T, result *model.ExecutorTestResult) {
				t.Helper()

				var connectivityStage, authStage, e2eStage *model.StageTestResult

				for _, stage := range result.Stages() {
					switch stage.Name() {
					case model.TestStageConnectivity:
						connectivityStage = &stage
					case model.TestStageAuthentication:
						authStage = &stage
					case model.TestStageEndToEnd:
						e2eStage = &stage
					}
				}

				assert.Equal(t, model.TestStageStatusSkipped, connectivityStage.Status())
				assert.Equal(t, model.TestStageStatusSkipped, authStage.Status())
				assert.Equal(t, model.TestStageStatusSkipped, e2eStage.Status())
			},
		},
		{
			name:       "authentication failed - unauthorized",
			executorID: uuid.MustParse("55555555-5555-5555-5555-555555555555"),
			setup: func(t *testing.T, ctrl *gomock.Controller, executorID uuid.UUID) (*MockExecutorConfigRepository, *testServerConfig) {
				t.Helper()

				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("X-API-Key") != "" {
						w.WriteHeader(http.StatusUnauthorized)
						_, _ = w.Write([]byte(`{"error": "unauthorized"}`))

						return
					}

					w.WriteHeader(http.StatusOK)
				}))

				mockRepo := NewMockExecutorConfigRepository(ctrl)

				endpoint, _ := model.NewExecutorEndpoint("health", "/health", "GET", 30)
				auth, _ := model.NewExecutorAuthentication("api_key", map[string]any{
					"key":    "invalid-key",
					"header": "X-API-Key",
				})
				executorConfig := model.NewExecutorConfigurationFromDB(
					executorID, "test-executor", testutil.StringPtr("Test executor"),
					mockServer.URL, []model.ExecutorEndpoint{*endpoint}, *auth,
					model.ExecutorConfigurationStatusConfigured, nil,
					fixedTime, fixedTime, nil,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), executorID).Return(executorConfig, nil)

				return mockRepo, &testServerConfig{
					server:     mockServer,
					httpClient: &http.Client{Timeout: 5 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ExecutorTestResult) {
				t.Helper()

				var authStage *model.StageTestResult

				for _, stage := range result.Stages() {
					if stage.Name() == model.TestStageAuthentication {
						authStage = &stage

						break
					}
				}

				require.NotNil(t, authStage)
				assert.Equal(t, model.TestStageStatusFailed, authStage.Status())
				assert.NotEqual(t, model.TestOverallStatusPassed, result.OverallStatus())
			},
		},
		{
			name:       "bearer auth success",
			executorID: uuid.MustParse("66666666-6666-6666-6666-666666666666"),
			setup: func(t *testing.T, ctrl *gomock.Controller, executorID uuid.UUID) (*MockExecutorConfigRepository, *testServerConfig) {
				t.Helper()

				expectedToken := "valid-token-123"
				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					authHeader := r.Header.Get("Authorization")
					if authHeader == "Bearer "+expectedToken {
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write([]byte(`{"status": "authenticated"}`))

						return
					}

					w.WriteHeader(http.StatusUnauthorized)
				}))

				mockRepo := NewMockExecutorConfigRepository(ctrl)

				endpoint, _ := model.NewExecutorEndpoint("health", "/health", "GET", 30)
				auth, _ := model.NewExecutorAuthentication("bearer", map[string]any{
					"token": expectedToken,
				})
				executorConfig := model.NewExecutorConfigurationFromDB(
					executorID, "test-executor", testutil.StringPtr("Test executor"),
					mockServer.URL, []model.ExecutorEndpoint{*endpoint}, *auth,
					model.ExecutorConfigurationStatusConfigured, nil,
					fixedTime, fixedTime, nil,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), executorID).Return(executorConfig, nil)
				mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(model.ExecutorConfigurationStatusConfigured)).Return(nil)

				return mockRepo, &testServerConfig{
					server:     mockServer,
					httpClient: &http.Client{Timeout: 5 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ExecutorTestResult) {
				t.Helper()
				assert.Equal(t, model.TestOverallStatusPassed, result.OverallStatus())
			},
		},
		{
			name:       "server error - 500",
			executorID: uuid.MustParse("77777777-7777-7777-7777-777777777777"),
			setup: func(t *testing.T, ctrl *gomock.Controller, executorID uuid.UUID) (*MockExecutorConfigRepository, *testServerConfig) {
				t.Helper()

				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error": "internal error"}`))
				}))

				mockRepo := NewMockExecutorConfigRepository(ctrl)

				endpoint, _ := model.NewExecutorEndpoint("health", "/health", "GET", 30)
				auth, _ := model.NewExecutorAuthentication("none", nil)
				executorConfig := model.NewExecutorConfigurationFromDB(
					executorID, "test-executor", testutil.StringPtr("Test executor"),
					mockServer.URL, []model.ExecutorEndpoint{*endpoint}, *auth,
					model.ExecutorConfigurationStatusConfigured, nil,
					fixedTime, fixedTime, nil,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), executorID).Return(executorConfig, nil)

				return mockRepo, &testServerConfig{
					server:     mockServer,
					httpClient: &http.Client{Timeout: 5 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ExecutorTestResult) {
				t.Helper()

				var connectivityStage *model.StageTestResult

				for _, stage := range result.Stages() {
					if stage.Name() == model.TestStageConnectivity {
						connectivityStage = &stage

						break
					}
				}

				require.NotNil(t, connectivityStage)
				assert.Equal(t, model.TestStageStatusFailed, connectivityStage.Status())
			},
		},
		{
			name:       "transformation engine test passes",
			executorID: uuid.MustParse("88888888-8888-8888-8888-888888888888"),
			setup: func(t *testing.T, ctrl *gomock.Controller, executorID uuid.UUID) (*MockExecutorConfigRepository, *testServerConfig) {
				t.Helper()

				mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"data": "test"}`))
				}))

				mockRepo := NewMockExecutorConfigRepository(ctrl)

				endpoint, _ := model.NewExecutorEndpoint("health", "/health", "GET", 30)
				auth, _ := model.NewExecutorAuthentication("none", nil)
				executorConfig := model.NewExecutorConfigurationFromDB(
					executorID, "test-executor", testutil.StringPtr("Test executor"),
					mockServer.URL, []model.ExecutorEndpoint{*endpoint}, *auth,
					model.ExecutorConfigurationStatusConfigured, nil,
					fixedTime, fixedTime, nil,
				)

				mockRepo.EXPECT().FindByID(gomock.Any(), executorID).Return(executorConfig, nil)
				mockRepo.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Eq(model.ExecutorConfigurationStatusConfigured)).Return(nil)

				return mockRepo, &testServerConfig{
					server:     mockServer,
					httpClient: &http.Client{Timeout: 5 * time.Second},
				}
			},
			validate: func(t *testing.T, result *model.ExecutorTestResult) {
				t.Helper()

				var transformStage *model.StageTestResult

				for _, stage := range result.Stages() {
					if stage.Name() == model.TestStageTransformation {
						transformStage = &stage

						break
					}
				}

				require.NotNil(t, transformStage)
				assert.Equal(t, model.TestStageStatusPassed, transformStage.Status())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx := context.Background()
			mockRepo, serverCfg := tt.setup(t, ctrl, tt.executorID)

			if serverCfg.server != nil {
				defer serverCfg.server.Close()
			}

			cmd, err := NewTestExecutorConnectivityCommand(mockRepo, serverCfg.httpClient)
			require.NoError(t, err)

			result, err := cmd.Execute(ctx, tt.executorID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
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
