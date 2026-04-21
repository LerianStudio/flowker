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
	"os"
	"sync"
	"time"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	libLog "github.com/LerianStudio/lib-commons/v4/commons/log"
	libOtel "github.com/LerianStudio/lib-commons/v4/commons/opentelemetry"
	libSSRF "github.com/LerianStudio/lib-commons/v4/commons/security/ssrf"

	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
)

// defaultProviderConfigTestTimeout is the default HTTP client timeout for connectivity tests.
const defaultProviderConfigTestTimeout = 10 * time.Second

// maxProviderConfigResponseBodySize limits the response body size to prevent memory exhaustion.
const maxProviderConfigResponseBodySize = 1 * 1024 * 1024 // 1MB

// TestProviderConfigConnectivityCommand handles provider configuration connectivity testing.
type TestProviderConfigConnectivityCommand struct {
	repo       ProviderConfigRepository
	httpClient *http.Client
}

// NewTestProviderConfigConnectivityCommand creates a new TestProviderConfigConnectivityCommand.
func NewTestProviderConfigConnectivityCommand(
	repo ProviderConfigRepository,
	httpClient *http.Client,
) (*TestProviderConfigConnectivityCommand, error) {
	if repo == nil {
		return nil, ErrTestProviderConfigConnectivityNilRepo
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultProviderConfigTestTimeout,
		}
	}

	// Always enforce SSRF redirect protection, wrapping any existing CheckRedirect.
	// NOTE: Redirect validation uses ValidateURL (static check) rather than
	// ResolveAndValidate because the net/http client has already resolved the
	// redirect target. This is acceptable: the initial request uses a DNS-pinned
	// URL, and redirect targets are validated before following.
	originalRedirect := httpClient.CheckRedirect
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if err := libSSRF.ValidateURL(req.Context(), req.URL.String(), ssrfOptions()...); err != nil {
			return fmt.Errorf("redirect blocked by SSRF policy: %w", err)
		}

		if len(via) >= 10 {
			return errors.New("too many redirects")
		}

		if originalRedirect != nil {
			return originalRedirect(req, via)
		}

		return nil
	}

	return &TestProviderConfigConnectivityCommand{
		repo:       repo,
		httpClient: httpClient,
	}, nil
}

// Execute tests provider configuration connectivity and returns detailed results.
func (c *TestProviderConfigConnectivityCommand) Execute(ctx context.Context, id uuid.UUID) (*model.ProviderConfigTestResult, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	logger, tracer, _, _ := libCommons.NewTrackingFromContext(ctx)

	ctx, span := tracer.Start(ctx, "command.provider_config.test_connectivity")
	defer span.End()

	logger.Log(ctx, libLog.LevelInfo, "Testing provider configuration connectivity", libLog.Any("operation", "command.provider_config.test_connectivity"), libLog.Any("provider_config.id", id))

	// Fetch provider configuration
	providerConfig, err := c.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, constant.ErrProviderConfigNotFound) {
			libOtel.HandleSpanBusinessErrorEvent(span, "Provider configuration not found", err)
			return nil, err
		}

		libOtel.HandleSpanError(span, "Failed to find provider configuration", err)

		return nil, fmt.Errorf("failed to find provider configuration: %w", err)
	}

	// Extract base_url from config with DNS-pinned SSRF validation
	baseURL, ssrfResult, err := extractBaseURL(ctx, providerConfig.Config())
	if err != nil {
		if errors.Is(err, libSSRF.ErrBlocked) || errors.Is(err, libSSRF.ErrDNSFailed) {
			libOtel.HandleSpanBusinessErrorEvent(span, "SSRF blocked", err)
			return nil, fmt.Errorf("%w: %s", constant.ErrProviderConfigSSRFBlocked, err.Error())
		}

		libOtel.HandleSpanBusinessErrorEvent(span, "Invalid base_url in config", err)
		return nil, fmt.Errorf("%w: %s", constant.ErrProviderConfigMissingBaseURL, err.Error())
	}

	// Use the DNS-pinned URL for actual HTTP requests to prevent TOCTOU/DNS-rebinding.
	// baseURL is retained for logging/display; pinnedURL goes on the wire.
	pinnedURL := ssrfResult.PinnedURL
	authority := ssrfResult.Authority

	// Create test result
	testResult, err := model.NewProviderConfigTestResult(id, providerConfig.ProviderID())
	if err != nil {
		libOtel.HandleSpanError(span, "failed to create test result", err)
		return nil, err
	}

	// Run test stages — skip dependent stages if a prerequisite fails
	c.runConnectivityTest(ctx, baseURL, pinnedURL, authority, testResult)

	if testResult.HasFailedStage() {
		c.skipRemainingStages(testResult, model.TestStageConnectivity)
	} else {
		c.runAuthenticationTest(ctx, baseURL, pinnedURL, authority, providerConfig.Config(), testResult)

		if testResult.HasFailedStage() {
			c.skipRemainingStages(testResult, model.TestStageAuthentication)
		} else {
			c.runEndToEndTest(ctx, baseURL, pinnedURL, authority, providerConfig.Config(), testResult)
		}
	}

	// Complete the test and calculate overall status
	testResult.Complete()

	logger.Log(ctx, libLog.LevelInfo, "Provider configuration connectivity test completed", libLog.Any("operation", "command.provider_config.test_connectivity"), libLog.Any("provider_config.id", id), libLog.Any("test.status", testResult.OverallStatus()))

	return testResult, nil
}

// skipRemainingStages adds skipped results for stages that were not run due to a prerequisite failure.
func (c *TestProviderConfigConnectivityCommand) skipRemainingStages(
	result *model.ProviderConfigTestResult,
	failedStage model.TestStageName,
) {
	reason := fmt.Sprintf("Skipped due to %s failure", failedStage)

	remaining := []model.TestStageName{
		model.TestStageConnectivity,
		model.TestStageAuthentication,
		model.TestStageEndToEnd,
	}

	// Find stages after the failed one and skip them
	skip := false

	for _, stage := range remaining {
		if stage == failedStage {
			skip = true

			continue
		}

		if skip {
			result.AddStageResult(model.NewSkippedStageResult(stage, reason))
		}
	}
}

// extractBaseURL extracts and validates a base URL from a provider configuration's config map.
// It tries multiple URL keys to support both generic providers (base_url) and
// Midaz-specific providers (onboarding_base_url, transaction_base_url).
// SSRF protection: performs DNS resolution and IP validation in a single step via
// ResolveAndValidate, eliminating the TOCTOU window between DNS lookup and connection.
func extractBaseURL(ctx context.Context, config map[string]any) (string, *libSSRF.ResolveResult, error) {
	for _, key := range []string{"base_url", "onboarding_base_url", "transaction_base_url"} {
		if u, ok := config[key].(string); ok && u != "" {
			parsed, err := url.Parse(u)
			if err != nil {
				return "", nil, fmt.Errorf("invalid %s format: %w", key, err)
			}

			if parsed.Scheme == "" || parsed.Host == "" {
				return "", nil, fmt.Errorf("%s must include scheme and host, got: %s", key, u)
			}

			// SSRF validation via DNS pinning: resolves DNS and validates all IPs
			// in a single step, eliminating the TOCTOU/DNS-rebinding window.
			ssrfOpts := ssrfOptions()

			result, err := libSSRF.ResolveAndValidate(ctx, u, ssrfOpts...)
			if err != nil {
				return "", nil, fmt.Errorf("SSRF blocked: %s targets a restricted destination: %w", key, err)
			}

			return u, result, nil
		}
	}

	return "", nil, fmt.Errorf("config missing required base URL field (expected base_url, onboarding_base_url, or transaction_base_url)")
}

// ssrfAllowPrivate caches the SSRF_ALLOW_PRIVATE env var. Tests may override
// it directly before calling ssrfOptions().
var (
	ssrfAllowPrivate     bool
	ssrfAllowPrivateOnce sync.Once
)

// ssrfOptions returns SSRF validation options based on environment configuration.
// Set SSRF_ALLOW_PRIVATE=true for local development with providers on localhost.
// The env var is read lazily on first call (zero syscall in hot path after first request).
func ssrfOptions() []libSSRF.Option {
	ssrfAllowPrivateOnce.Do(func() {
		ssrfAllowPrivate = os.Getenv("SSRF_ALLOW_PRIVATE") == "true"
	})

	if ssrfAllowPrivate {
		return []libSSRF.Option{libSSRF.WithAllowPrivateNetwork()}
	}

	return nil
}

// extractHeaders extracts custom headers from a provider configuration's config map.
func extractHeaders(config map[string]any) map[string]string {
	headersRaw, ok := config["headers"]
	if !ok {
		return nil
	}

	headersMap, ok := headersRaw.(map[string]any)
	if !ok {
		return nil
	}

	headers := make(map[string]string, len(headersMap))

	for k, v := range headersMap {
		if strVal, ok := v.(string); ok {
			headers[k] = strVal
		}
	}

	return headers
}

// extractAPIKey extracts the api_key from a provider configuration's config map.
func extractAPIKey(config map[string]any) string {
	apiKey, _ := config["api_key"].(string)
	return apiKey
}

// runConnectivityTest tests basic HTTP connectivity to the provider's base URL.
// pinnedURL is the DNS-pinned URL (IP-based) used for the actual request;
// baseURL is the original hostname-based URL used for logging/display.
// authority is the original Host header value to send so the target routes correctly.
func (c *TestProviderConfigConnectivityCommand) runConnectivityTest(
	ctx context.Context,
	baseURL string,
	pinnedURL string,
	authority string,
	result *model.ProviderConfigTestResult,
) {
	start := time.Now()

	reqCtx, cancel := context.WithTimeout(ctx, defaultProviderConfigTestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, pinnedURL, nil)
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

	// Set the Host header to the original authority so the target server routes correctly.
	req.Host = authority

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
		"url":            baseURL,
	}

	// Consider 2xx and 4xx as connectivity success (provider is reachable and responding)
	// 401/403 = needs auth, 400/404/422 = endpoint issue - all indicate the server is responding
	// Only 5xx indicates potential server-side problems that might affect reliability
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusInternalServerError {
		result.AddStageResult(model.NewPassedStageResult(
			model.TestStageConnectivity,
			durationMs,
			"Provider is reachable",
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

// runAuthenticationTest tests authentication credentials against the provider.
// pinnedURL is the DNS-pinned URL used for the actual request; baseURL is for display only.
// authority is the original Host header value to send so the target routes correctly.
func (c *TestProviderConfigConnectivityCommand) runAuthenticationTest(
	ctx context.Context,
	baseURL string,
	pinnedURL string,
	authority string,
	config map[string]any,
	result *model.ProviderConfigTestResult,
) {
	_ = baseURL // retained for future logging/display use

	start := time.Now()

	apiKey := extractAPIKey(config)
	headers := extractHeaders(config)

	// If there are no authentication credentials, skip
	if apiKey == "" && len(headers) == 0 {
		result.AddStageResult(model.NewSkippedStageResult(
			model.TestStageAuthentication,
			"No authentication credentials configured",
		))

		return
	}

	reqCtx, cancel := context.WithTimeout(ctx, defaultProviderConfigTestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, pinnedURL, nil)
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

	// Set the Host header to the original authority so the target server routes correctly.
	req.Host = authority

	// Apply authentication: api_key sets Bearer token first, then custom headers
	// are applied. If headers contains "Authorization", it intentionally overrides
	// the api_key Bearer token, allowing full control via custom headers.
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
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
		"hasApiKey":      apiKey != "",
		"headerCount":    len(headers),
	}

	// 2xx means auth succeeded
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
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

	// 5xx means server error — cannot determine auth status
	if resp.StatusCode >= http.StatusInternalServerError {
		result.AddStageResult(model.NewFailedStageResult(
			model.TestStageAuthentication,
			durationMs,
			fmt.Sprintf("server error during authentication: status %d", resp.StatusCode),
			details,
		))

		return
	}

	// Other 4xx status codes — auth was accepted but endpoint returned a client error
	result.AddStageResult(model.NewPassedStageResult(
		model.TestStageAuthentication,
		durationMs,
		"Authentication accepted (endpoint returned non-auth error)",
		details,
	))
}

// runEndToEndTest performs a full request/response cycle with all credentials applied.
// pinnedURL is the DNS-pinned URL used for the actual request; baseURL is for display only.
// authority is the original Host header value to send so the target routes correctly.
func (c *TestProviderConfigConnectivityCommand) runEndToEndTest(
	ctx context.Context,
	baseURL string,
	pinnedURL string,
	authority string,
	config map[string]any,
	result *model.ProviderConfigTestResult,
) {
	_ = baseURL // retained for future logging/display use

	start := time.Now()

	reqCtx, cancel := context.WithTimeout(ctx, defaultProviderConfigTestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, pinnedURL, nil)
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

	// Set the Host header to the original authority so the target server routes correctly.
	req.Host = authority

	// Apply all credentials: api_key sets Bearer token, custom headers may override it.
	apiKey := extractAPIKey(config)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	headers := extractHeaders(config)
	for k, v := range headers {
		req.Header.Set(k, v)
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
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxProviderConfigResponseBodySize))
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
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
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
