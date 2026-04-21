// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executors/http/auth"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// DefaultTimeout is the default request timeout in seconds.
const DefaultTimeout = 30

// DefaultSuccessStatusCodes are the default HTTP status codes considered successful.
var DefaultSuccessStatusCodes = []int{200, 201, 202, 204}

// Runner executes HTTP requests.
type Runner struct{}

// NewRunner creates a new HTTP runner.
func NewRunner() *Runner {
	return &Runner{}
}

// ExecutorID returns the ID of the executor this runner handles.
func (r *Runner) ExecutorID() executor.ID {
	return ID
}

// Execute performs an HTTP request based on the input configuration.
func (r *Runner) Execute(ctx context.Context, input executor.ExecutionInput) (executor.ExecutionResult, error) {
	// Extract configuration
	method := getStringValue(input.Config, "method", http.MethodGet)
	urlValue := getStringValue(input.Config, "url", "")

	if urlValue == "" {
		return executor.NewErrorResult("url is required"), nil
	}

	// Build URL with query parameters
	finalURL, err := buildURL(urlValue, input.Config["query"])
	if err != nil {
		return executor.NewErrorResult(fmt.Sprintf("invalid url: %v", err)), nil
	}

	// Prepare request body
	var bodyReader io.Reader

	if body, ok := input.Config["body"]; ok && body != nil {
		bodyReader, err = prepareBody(body)
		if err != nil {
			return executor.NewErrorResult(fmt.Sprintf("failed to prepare body: %v", err)), nil
		}
	}

	// Get timeout
	timeout := getIntValue(input.Config, "timeout_seconds", DefaultTimeout)

	// Create context with timeout
	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	// Create request
	req, err := http.NewRequestWithContext(reqCtx, method, finalURL, bodyReader)
	if err != nil {
		return executor.NewErrorResult(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// Set headers
	setHeaders(req, input.Config["headers"])

	// Set default Content-Type for body requests
	if bodyReader != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Setup HTTP client
	client := setupHTTPClient(input.HTTPClient, timeout)

	// Apply authentication
	if errResult, hasErr := applyAuth(reqCtx, req, input, client); hasErr {
		return errResult, nil
	}

	// Log request
	if input.Emit != nil {
		input.Emit("info", "http request", map[string]any{
			"method": method,
			"url":    finalURL,
		})
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return executor.NewErrorResult(fmt.Sprintf("request failed: %v", err)), nil
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return executor.NewErrorResult(fmt.Sprintf("failed to read response: %v", err)), nil
	}

	// Build result data
	result := buildResultData(resp, finalURL, respBody)

	// Log response
	if input.Emit != nil {
		input.Emit("info", "http response", map[string]any{
			"status": resp.StatusCode,
		})
	}

	// Check if status code is successful
	successCodes := getSuccessStatusCodes(input.Config)
	if !isSuccessStatus(resp.StatusCode, successCodes) {
		errMsg := fmt.Sprintf("http request failed with status %d", resp.StatusCode)
		result["error"] = errMsg

		return executor.ExecutionResult{
			Data:   result,
			Status: executor.ExecutionStatusError,
			Error:  errMsg,
		}, nil
	}

	return executor.NewSuccessResult(result), nil
}

// setupHTTPClient creates or configures an HTTP client with the given timeout.
func setupHTTPClient(client *http.Client, timeout int) *http.Client {
	if client == nil {
		return &http.Client{
			Timeout:   time.Duration(timeout) * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}
	}

	if client.Timeout == 0 {
		client.Timeout = time.Duration(timeout) * time.Second
	}

	if client.Transport == nil {
		client.Transport = otelhttp.NewTransport(http.DefaultTransport)
	}

	return client
}

// applyAuth applies authentication to the request if configured.
// Returns the error result and true if authentication failed.
func applyAuth(ctx context.Context, req *http.Request, input executor.ExecutionInput, client *http.Client) (executor.ExecutionResult, bool) {
	authConfig, ok := input.Config["auth"].(map[string]any)
	if !ok {
		return executor.ExecutionResult{}, false
	}

	authProvider, err := auth.NewFromConfig(authConfig, client)
	if err != nil {
		return executor.NewErrorResult(fmt.Sprintf("failed to create auth provider: %v", err)), true
	}

	if err := authProvider.Apply(ctx, req); err != nil {
		return executor.NewErrorResult(fmt.Sprintf("failed to apply authentication: %v", err)), true
	}

	if input.Emit != nil {
		input.Emit("debug", "authentication applied", map[string]any{
			"type": string(authProvider.Type()),
		})
	}

	return executor.ExecutionResult{}, false
}

// buildResultData builds the result map from an HTTP response.
func buildResultData(resp *http.Response, finalURL string, respBody []byte) map[string]any {
	result := map[string]any{
		"status":      resp.StatusCode,
		"status_text": resp.Status,
		"url":         finalURL,
		"headers":     headersToMap(resp.Header),
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var jsonBody any
		if err := json.Unmarshal(respBody, &jsonBody); err == nil {
			result["body"] = jsonBody
		} else {
			result["body"] = string(respBody)
		}
	} else if len(respBody) > 0 {
		result["body"] = string(respBody)
	}

	return result
}

// Verify Runner implements executor.Runner interface.
var _ executor.Runner = (*Runner)(nil)

// Helper functions

func getStringValue(config map[string]any, key, defaultValue string) string {
	if v, ok := config[key].(string); ok {
		return v
	}

	return defaultValue
}

func getIntValue(config map[string]any, key string, defaultValue int) int {
	switch v := config[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
	}

	return defaultValue
}

func buildURL(baseURL string, query any) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	if queryMap, ok := query.(map[string]any); ok {
		q := u.Query()

		for k, v := range queryMap {
			if s, ok := v.(string); ok {
				q.Set(k, s)
			}
		}

		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

func prepareBody(body any) (io.Reader, error) {
	switch v := body.(type) {
	case string:
		return strings.NewReader(v), nil
	case []byte:
		return bytes.NewReader(v), nil
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}

		return bytes.NewReader(encoded), nil
	}
}

func setHeaders(req *http.Request, headers any) {
	if headerMap, ok := headers.(map[string]any); ok {
		for k, v := range headerMap {
			if s, ok := v.(string); ok {
				req.Header.Set(k, s)
			}
		}
	}
}

func headersToMap(headers http.Header) map[string]string {
	result := make(map[string]string)

	for k, v := range headers {
		result[k] = strings.Join(v, ", ")
	}

	return result
}

func getSuccessStatusCodes(config map[string]any) []int {
	if codes, ok := config["success_status_codes"].([]any); ok {
		result := make([]int, 0, len(codes))

		for _, c := range codes {
			switch v := c.(type) {
			case int:
				result = append(result, v)
			case int64:
				result = append(result, int(v))
			case float64:
				result = append(result, int(v))
			}
		}

		if len(result) > 0 {
			return result
		}
	}

	return DefaultSuccessStatusCodes
}

func isSuccessStatus(status int, successCodes []int) bool {
	return slices.Contains(successCodes, status)
}
