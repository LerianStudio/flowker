// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"time"

	libSecurity "github.com/LerianStudio/lib-commons/v4/commons/security"
	"github.com/google/uuid"
)

// ExecutorConfigurationOutput is the output DTO for an executor configuration.
type ExecutorConfigurationOutput struct {
	ID             uuid.UUID                    `json:"id" swaggertype:"string" format:"uuid"`
	Name           string                       `json:"name"`
	Description    *string                      `json:"description,omitempty"`
	BaseURL        string                       `json:"baseUrl"`
	Endpoints      []ExecutorEndpointOutput     `json:"endpoints"`
	Authentication ExecutorAuthenticationOutput `json:"authentication"`
	Status         string                       `json:"status"`
	Metadata       map[string]any               `json:"metadata,omitempty"`
	CreatedAt      time.Time                    `json:"createdAt"`
	UpdatedAt      time.Time                    `json:"updatedAt"`
	LastTestedAt   *time.Time                   `json:"lastTestedAt,omitempty"`
}

// ExecutorEndpointOutput is the output DTO for an executor endpoint.
type ExecutorEndpointOutput struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Method  string `json:"method"`
	Timeout int    `json:"timeout"`
}

// ExecutorAuthenticationOutput is the output DTO for executor authentication.
// Note: Sensitive config values (like secrets) are masked in output.
type ExecutorAuthenticationOutput struct {
	Type   string         `json:"type"`
	Config map[string]any `json:"config,omitempty"`
}

// ExecutorConfigurationCreateOutput is the minimal output for executor configuration creation.
type ExecutorConfigurationCreateOutput struct {
	ID        uuid.UUID `json:"id" swaggertype:"string" format:"uuid"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// ExecutorConfigurationListOutput is the output DTO for listing executor configurations.
type ExecutorConfigurationListOutput struct {
	Items      []ExecutorConfigurationOutput `json:"items"`
	NextCursor string                        `json:"nextCursor"`
	HasMore    bool                          `json:"hasMore"`
}

// ExecutorConfigurationOutputFromDomain converts an ExecutorConfiguration domain entity to output DTO.
func ExecutorConfigurationOutputFromDomain(p *ExecutorConfiguration) ExecutorConfigurationOutput {
	endpoints := make([]ExecutorEndpointOutput, len(p.Endpoints()))
	for i, ep := range p.Endpoints() {
		endpoints[i] = ExecutorEndpointOutputFromDomain(ep)
	}

	return ExecutorConfigurationOutput{
		ID:             p.ID(),
		Name:           p.Name(),
		Description:    p.Description(),
		BaseURL:        p.BaseURL(),
		Endpoints:      endpoints,
		Authentication: ExecutorAuthenticationOutputFromDomain(p.Authentication()),
		Status:         string(p.Status()),
		Metadata:       p.Metadata(),
		CreatedAt:      p.CreatedAt(),
		UpdatedAt:      p.UpdatedAt(),
		LastTestedAt:   p.LastTestedAt(),
	}
}

// ExecutorEndpointOutputFromDomain converts an ExecutorEndpoint to output DTO.
func ExecutorEndpointOutputFromDomain(ep ExecutorEndpoint) ExecutorEndpointOutput {
	return ExecutorEndpointOutput{
		Name:    ep.Name(),
		Path:    ep.Path(),
		Method:  ep.Method(),
		Timeout: ep.Timeout(),
	}
}

// ExecutorAuthenticationOutputFromDomain converts an ExecutorAuthentication to output DTO.
// Sensitive values are masked for security.
func ExecutorAuthenticationOutputFromDomain(auth ExecutorAuthentication) ExecutorAuthenticationOutput {
	// Mask sensitive config values
	maskedConfig := maskSensitiveConfig(auth.Config())

	return ExecutorAuthenticationOutput{
		Type:   auth.Type(),
		Config: maskedConfig,
	}
}

// maskSensitiveConfig masks sensitive values in a config map.
// Uses lib-commons IsSensitiveField for keyword-based detection (91+ patterns)
// and shows only the last 4 characters of masked string values.
func maskSensitiveConfig(config map[string]any) map[string]any {
	if config == nil {
		return nil
	}

	masked := make(map[string]any, len(config))
	for k, v := range config {
		if libSecurity.IsSensitiveField(k) {
			if s, ok := v.(string); ok && len(s) > 4 {
				masked[k] = "****" + s[len(s)-4:]
			} else {
				masked[k] = "********"
			}
		} else {
			masked[k] = v
		}
	}

	return masked
}

// ExecutorConfigurationCreateOutputFromDomain creates a minimal creation response.
func ExecutorConfigurationCreateOutputFromDomain(p *ExecutorConfiguration) ExecutorConfigurationCreateOutput {
	return ExecutorConfigurationCreateOutput{
		ID:        p.ID(),
		Name:      p.Name(),
		Status:    string(p.Status()),
		CreatedAt: p.CreatedAt(),
	}
}

// ExecutorConfigurationListOutputFromDomain converts a list of executor configurations to list output.
func ExecutorConfigurationListOutputFromDomain(
	configs []*ExecutorConfiguration,
	nextCursor string,
	hasMore bool,
) ExecutorConfigurationListOutput {
	items := make([]ExecutorConfigurationOutput, len(configs))
	for i, p := range configs {
		items[i] = ExecutorConfigurationOutputFromDomain(p)
	}

	return ExecutorConfigurationListOutput{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}
