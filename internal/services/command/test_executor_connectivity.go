// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"

	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/executors/http/auth"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/transformation"
	"github.com/google/uuid"
)

// TestExecutorConnectivityCommand handles executor connectivity testing.
type TestExecutorConnectivityCommand struct {
	repo       ExecutorConfigRepository
	httpClient *http.Client
}

// NewTestExecutorConnectivityCommand creates a new TestExecutorConnectivityCommand.
func NewTestExecutorConnectivityCommand(
	repo ExecutorConfigRepository,
	httpClient *http.Client,
) (*TestExecutorConnectivityCommand, error) {
	if repo == nil {
		return nil, ErrTestExecutorConnectivityNilRepo
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	return &TestExecutorConnectivityCommand{
		repo:       repo,
		httpClient: httpClient,
	}, nil
}

// Execute tests executor connectivity and returns detailed results.
func (c *TestExecutorConnectivityCommand) Execute(ctx context.Context, id uuid.UUID) (*model.ExecutorTestResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.executor_config.test_connectivity")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Testing executor connectivity", libLog.Any("operation", "command.executor_config.test_connectivity"), libLog.Any("executor_config.id", id))

	// Fetch executor configuration
	executorConfig, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrExecutorConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Executor configuration not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find executor configuration", err)

		return nil, fmt.Errorf("failed to find executor configuration: %w", err)
	}

	// Create test result
	testResult := model.NewExecutorTestResult(id)

	// Run test stages
	c.runConnectivityTest(ctx, executorConfig, testResult)
	c.runAuthenticationTest(ctx, executorConfig, testResult)
	c.runTransformationTest(ctx, executorConfig, testResult)
	c.runEndToEndTest(ctx, executorConfig, testResult)

	// Complete the test and calculate overall status
	testResult.Complete()

	// If all tests passed, mark executor configuration as tested
	if testResult.IsPassed() {
		previousStatus := executorConfig.Status()

		if err := executorConfig.MarkTested(); err != nil {
			// Executor configuration might not be in the right status, but test still succeeded
			logger.Log(ctx, libLog.LevelWarn, "Could not mark executor configuration as tested", libLog.Any("operation", "command.executor_config.test_connectivity"), libLog.Any("executor_config.id", executorConfig.ID()), libLog.Any("error.message", err.Error()))
		} else {
			if err := c.repo.Update(ctx, executorConfig, previousStatus); err != nil {
				if errors.Is(err, constant.ErrConflictStateChanged) {
					libOtel.HandleSpanBusinessErrorEvent(span, "State conflict detected", err)
					return nil, err
				}

				libOtel.HandleSpanError(span, "failed to update executor configuration", err)

				return nil, err
			}

			logger.Log(ctx, libLog.LevelInfo, "Executor configuration marked as tested", libLog.Any("operation", "command.executor_config.test_connectivity"), libLog.Any("executor_config.id", executorConfig.ID()))
		}
	}

	logger.Log(ctx, libLog.LevelInfo, "Executor connectivity test completed", libLog.Any("operation", "command.executor_config.test_connectivity"), libLog.Any("executor_config.id", id), libLog.Any("test.status", testResult.OverallStatus()))

	return testResult, nil
}

// runConnectivityTest tests basic HTTP connectivity to the executor.
func (c *TestExecutorConnectivityCommand) runConnectivityTest(
	ctx context.Context,
	config *model.ExecutorConfiguration,
	result *model.ExecutorTestResult,
) {
	start := time.Now()

	endpoints := config.Endpoints()
	if len(endpoints) == 0 {
		result.AddStageResult(model.NewSkippedStageResult(
			model.TestStageConnectivity,
			"No endpoints configured",
		))

		return
	}

	// Test first endpoint
	endpoint := endpoints[0]

	endpointURL, err := url.JoinPath(config.BaseURL(), endpoint.Path())
	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageConnectivity,
			durationMs,
			fmt.Sprintf("invalid base URL or endpoint path: %v", err),
			nil,
		))

		return
	}

	timeout := endpoint.Timeout()
	if timeout <= 0 {
		timeout = 30
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, endpoint.Method(), endpointURL, nil)
	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageConnectivity,
			durationMs,
			fmt.Sprintf("failed to create request: %v", err),
			nil,
		))

		return
	}

	resp, err := c.httpClient.Do(req)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageConnectivity,
			durationMs,
			fmt.Sprintf("request failed: %v", err),
			nil,
		))

		return
	}

	defer resp.Body.Close()

	details := map[string]any{
		"statusCode":     resp.StatusCode,
		"responseTimeMs": durationMs,
		"url":            endpointURL,
	}

	// Consider 2xx and 4xx as connectivity success (executor is reachable and responding)
	// 401/403 = needs auth, 400/404/422 = endpoint issue - all indicate the server is responding
	// Only 5xx indicates potential server-side problems that might affect reliability
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		result.AddStageResult(model.NewPassedStageResult(
			model.TestStageConnectivity,
			durationMs,
			"Executor is reachable",
			details,
		))

		return
	}

	result.AddStageResult(model.NewFailedStageResult(
		model.TestStageConnectivity,
		durationMs,
		fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
		details,
	))
}

// runAuthenticationTest tests authentication with the executor.
func (c *TestExecutorConnectivityCommand) runAuthenticationTest(
	ctx context.Context,
	config *model.ExecutorConfiguration,
	result *model.ExecutorTestResult,
) {
	start := time.Now()

	authConfig := config.Authentication()
	if authConfig.Type() == "" || authConfig.Type() == "none" {
		result.AddStageResult(model.NewSkippedStageResult(
			model.TestStageAuthentication,
			"No authentication configured",
		))

		return
	}

	endpoints := config.Endpoints()
	if len(endpoints) == 0 {
		result.AddStageResult(model.NewSkippedStageResult(
			model.TestStageAuthentication,
			"No endpoints to test authentication",
		))

		return
	}

	// Create auth provider
	authConfigMap := map[string]any{
		"type":   authConfig.Type(),
		"config": authConfig.Config(),
	}

	authProvider, err := auth.NewFromConfig(authConfigMap, c.httpClient)
	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageAuthentication,
			durationMs,
			fmt.Sprintf("failed to create auth provider: %v", err),
			nil,
		))

		return
	}

	// Test first endpoint with auth
	endpoint := endpoints[0]

	endpointURL, err := url.JoinPath(config.BaseURL(), endpoint.Path())
	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageAuthentication,
			durationMs,
			fmt.Sprintf("invalid base URL or endpoint path: %v", err),
			nil,
		))

		return
	}

	timeout := endpoint.Timeout()
	if timeout <= 0 {
		timeout = 30
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, endpoint.Method(), endpointURL, nil)
	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageAuthentication,
			durationMs,
			fmt.Sprintf("failed to create request: %v", err),
			nil,
		))

		return
	}

	// Apply authentication
	if err := authProvider.Apply(reqCtx, req); err != nil {
		durationMs := time.Since(start).Milliseconds()
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageAuthentication,
			durationMs,
			fmt.Sprintf("failed to apply authentication: %v", err),
			nil,
		))

		return
	}

	resp, err := c.httpClient.Do(req)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageAuthentication,
			durationMs,
			fmt.Sprintf("authenticated request failed: %v", err),
			nil,
		))

		return
	}

	defer resp.Body.Close()

	details := map[string]any{
		"statusCode":     resp.StatusCode,
		"responseTimeMs": durationMs,
		"authType":       authConfig.Type(),
	}

	// 2xx means auth succeeded
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.AddStageResult(model.NewPassedStageResult(
			model.TestStageAuthentication,
			durationMs,
			"Authentication successful",
			details,
		))

		return
	}

	// 401/403 means auth failed
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageAuthentication,
			durationMs,
			fmt.Sprintf("authentication rejected: status %d", resp.StatusCode),
			details,
		))

		return
	}

	// Other status codes - auth might have worked but endpoint returned error
	result.AddStageResult(model.NewPassedStageResult(
		model.TestStageAuthentication,
		durationMs,
		"Authentication accepted (endpoint returned non-auth error)",
		details,
	))
}

// runTransformationTest validates that transformation service is functional.
// Note: Actual transformation specs are defined in Workflows, not ExecutorConfiguration.
// This test verifies the transformation engine is ready for use.
func (c *TestExecutorConnectivityCommand) runTransformationTest(
	ctx context.Context,
	_ *model.ExecutorConfiguration,
	result *model.ExecutorTestResult,
) {
	start := time.Now()

	// Verify transformation service is functional with a simple test.
	// Service is stateless, so creating a new instance per test is acceptable
	// and avoids the need for dependency injection of a rarely-used component.
	svc := transformation.NewService()

	// Test with a simple valid Kazaam spec (JSON string)
	testSpec := `[{"operation": "shift", "spec": {"output": "input"}}]`

	if err := svc.ValidateSpec(testSpec); err != nil {
		durationMs := time.Since(start).Milliseconds()
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageTransformation,
			durationMs,
			fmt.Sprintf("transformation service validation failed: %v", err),
			nil,
		))

		return
	}

	// Test actual transformation
	testInput := []byte(`{"input": "test-value"}`)

	_, err := svc.Transform(ctx, testInput, testSpec)
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageTransformation,
			durationMs,
			fmt.Sprintf("transformation execution failed: %v", err),
			nil,
		))

		return
	}

	result.AddStageResult(model.NewPassedStageResult(
		model.TestStageTransformation,
		durationMs,
		"Transformation engine is functional",
		nil,
	))
}

// runEndToEndTest performs a full request/response cycle.
func (c *TestExecutorConnectivityCommand) runEndToEndTest(
	ctx context.Context,
	config *model.ExecutorConfiguration,
	result *model.ExecutorTestResult,
) {
	start := time.Now()

	endpoints := config.Endpoints()
	if len(endpoints) == 0 {
		result.AddStageResult(model.NewSkippedStageResult(
			model.TestStageEndToEnd,
			"No endpoints configured",
		))

		return
	}

	// Use first endpoint for e2e test
	endpoint := endpoints[0]

	endpointURL, err := url.JoinPath(config.BaseURL(), endpoint.Path())
	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageEndToEnd,
			durationMs,
			fmt.Sprintf("invalid base URL or endpoint path: %v", err),
			nil,
		))

		return
	}

	timeout := endpoint.Timeout()
	if timeout <= 0 {
		timeout = 30
	}

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, endpoint.Method(), endpointURL, nil)
	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageEndToEnd,
			durationMs,
			fmt.Sprintf("failed to create request: %v", err),
			nil,
		))

		return
	}

	// Apply authentication if configured
	authConfig := config.Authentication()
	if authConfig.Type() != "" && authConfig.Type() != "none" {
		authConfigMap := map[string]any{
			"type":   authConfig.Type(),
			"config": authConfig.Config(),
		}

		authProvider, err := auth.NewFromConfig(authConfigMap, c.httpClient)
		if err != nil {
			durationMs := time.Since(start).Milliseconds()
			result.AddStageResult(model.NewFailedStageResult(
				model.TestStageEndToEnd,
				durationMs,
				fmt.Sprintf("failed to create auth provider: %v", err),
				nil,
			))

			return
		}

		if err := authProvider.Apply(reqCtx, req); err != nil {
			durationMs := time.Since(start).Milliseconds()
			result.AddStageResult(model.NewFailedStageResult(
				model.TestStageEndToEnd,
				durationMs,
				fmt.Sprintf("failed to apply authentication: %v", err),
				nil,
			))

			return
		}
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageEndToEnd,
			durationMs,
			fmt.Sprintf("request failed: %v", err),
			nil,
		))

		return
	}

	defer resp.Body.Close()

	// Read response body with size limit to prevent memory exhaustion
	const maxResponseBodySize = 1 * 1024 * 1024 // 1MB limit for test responses

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	durationMs := time.Since(start).Milliseconds()

	if err != nil {
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageEndToEnd,
			durationMs,
			fmt.Sprintf("failed to read response: %v", err),
			nil,
		))

		return
	}

	details := map[string]any{
		"statusCode":      resp.StatusCode,
		"responseTimeMs":  durationMs,
		"responseBodyLen": len(body),
		"contentType":     resp.Header.Get("Content-Type"),
	}

	// Consider 2xx as success
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.AddStageResult(model.NewPassedStageResult(
			model.TestStageEndToEnd,
			durationMs,
			"Full request/response cycle completed successfully",
			details,
		))

		return
	}

	// Non-2xx is a failure for e2e
	result.AddStageResult(model.NewFailedStageResult(
		model.TestStageEndToEnd,
		durationMs,
		fmt.Sprintf("request returned status %d", resp.StatusCode),
		details,
	))
}
