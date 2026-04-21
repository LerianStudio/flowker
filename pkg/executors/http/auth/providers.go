// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package auth

import (
	"context"
	"encoding/base64"
	"net/http"
)

// NoneProvider provides no authentication.
type NoneProvider struct{}

// NewNoneProvider creates a new none provider.
func NewNoneProvider() *NoneProvider {
	return &NoneProvider{}
}

// Apply implements Provider interface.
func (p *NoneProvider) Apply(_ context.Context, _ *http.Request) error {
	return nil
}

// Type implements Provider interface.
func (p *NoneProvider) Type() Type {
	return TypeNone
}

// Verify NoneProvider implements Provider interface.
var _ Provider = (*NoneProvider)(nil)

// APIKeyProvider provides API key authentication.
type APIKeyProvider struct {
	config *APIKeyConfig
}

// NewAPIKeyProvider creates a new API key provider.
// Returns an error if cfg is nil or required fields are missing.
func NewAPIKeyProvider(cfg *APIKeyConfig) (*APIKeyProvider, error) {
	if cfg == nil {
		return nil, ErrAPIKeyConfigRequired
	}

	if cfg.Key == "" {
		return nil, ErrAPIKeyKeyRequired
	}

	// Apply defaults
	if cfg.HeaderName == "" {
		cfg.HeaderName = "X-API-Key"
	}

	if cfg.Location == "" {
		cfg.Location = "header"
	}

	if cfg.QueryParamName == "" {
		cfg.QueryParamName = "api_key"
	}

	// Validate location
	if cfg.Location != "header" && cfg.Location != "query" {
		return nil, ErrAPIKeyInvalidLocation
	}

	return &APIKeyProvider{config: cfg}, nil
}

// Apply implements Provider interface.
func (p *APIKeyProvider) Apply(_ context.Context, req *http.Request) error {
	value := p.config.Prefix + p.config.Key

	switch p.config.Location {
	case "header":
		req.Header.Set(p.config.HeaderName, value)
	case "query":
		q := req.URL.Query()
		q.Set(p.config.QueryParamName, value)
		req.URL.RawQuery = q.Encode()
	}

	return nil
}

// Type implements Provider interface.
func (p *APIKeyProvider) Type() Type {
	return TypeAPIKey
}

// Verify APIKeyProvider implements Provider interface.
var _ Provider = (*APIKeyProvider)(nil)

// BearerProvider provides bearer token authentication.
type BearerProvider struct {
	config *BearerConfig
}

// NewBearerProvider creates a new bearer provider.
// Returns an error if cfg is nil or token is missing.
func NewBearerProvider(cfg *BearerConfig) (*BearerProvider, error) {
	if cfg == nil {
		return nil, ErrBearerConfigRequired
	}

	if cfg.Token == "" {
		return nil, ErrBearerTokenRequired
	}

	return &BearerProvider{config: cfg}, nil
}

// Apply implements Provider interface.
func (p *BearerProvider) Apply(_ context.Context, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+p.config.Token)
	return nil
}

// Type implements Provider interface.
func (p *BearerProvider) Type() Type {
	return TypeBearer
}

// Verify BearerProvider implements Provider interface.
var _ Provider = (*BearerProvider)(nil)

// BasicProvider provides basic authentication.
type BasicProvider struct {
	config *BasicConfig
}

// NewBasicProvider creates a new basic provider.
// Returns an error if cfg is nil or required fields are missing.
func NewBasicProvider(cfg *BasicConfig) (*BasicProvider, error) {
	if cfg == nil {
		return nil, ErrBasicConfigRequired
	}

	if cfg.Username == "" {
		return nil, ErrBasicUsernameRequired
	}

	if cfg.Password == "" {
		return nil, ErrBasicPasswordRequired
	}

	return &BasicProvider{config: cfg}, nil
}

// Apply implements Provider interface.
func (p *BasicProvider) Apply(_ context.Context, req *http.Request) error {
	auth := base64.StdEncoding.EncodeToString([]byte(p.config.Username + ":" + p.config.Password))
	req.Header.Set("Authorization", "Basic "+auth)

	return nil
}

// Type implements Provider interface.
func (p *BasicProvider) Type() Type {
	return TypeBasic
}

// Verify BasicProvider implements Provider interface.
var _ Provider = (*BasicProvider)(nil)
