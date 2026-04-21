// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package auth provides authentication mechanisms for the HTTP provider.
package auth

import (
	"context"
	"net/http"
)

// Type represents the authentication type.
type Type string

const (
	// TypeNone represents no authentication.
	TypeNone Type = "none"

	// TypeAPIKey represents API key authentication.
	TypeAPIKey Type = "api_key"

	// TypeBearer represents bearer token authentication.
	TypeBearer Type = "bearer"

	// TypeBasic represents basic authentication.
	TypeBasic Type = "basic"

	// TypeOIDCClientCredentials represents OIDC client credentials flow.
	TypeOIDCClientCredentials Type = "oidc_client_credentials"

	// TypeOIDCUser represents OIDC resource owner password credentials flow.
	TypeOIDCUser Type = "oidc_user"
)

// Config represents the authentication configuration.
type Config struct {
	Type   Type      `json:"type"`
	Config any       `json:"config,omitempty"`
	Cache  *CacheCfg `json:"cache,omitempty"`
}

// CacheCfg represents token caching configuration.
type CacheCfg struct {
	Enabled                    bool `json:"enabled"`
	RefreshBeforeExpirySeconds int  `json:"refresh_before_expiry_seconds"`
	UseRefreshToken            bool `json:"use_refresh_token"`
}

// APIKeyConfig represents API key authentication configuration.
type APIKeyConfig struct {
	Key            string `json:"key" validate:"required"`
	HeaderName     string `json:"header_name"`
	Prefix         string `json:"prefix"`
	Location       string `json:"location" validate:"omitempty,oneof=header query"` // "header" or "query"
	QueryParamName string `json:"query_param_name"`
}

// BearerConfig represents bearer token authentication configuration.
type BearerConfig struct {
	Token string `json:"token" validate:"required"`
}

// BasicConfig represents basic authentication configuration.
type BasicConfig struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// OIDCClientCredentialsConfig represents OIDC client credentials configuration.
type OIDCClientCredentialsConfig struct {
	IssuerURL               string            `json:"issuer_url" validate:"required,url"`
	ClientID                string            `json:"client_id" validate:"required"`
	ClientSecret            string            `json:"client_secret" validate:"required"`
	Scopes                  []string          `json:"scopes"`
	Audience                string            `json:"audience"`
	TokenEndpointAuthMethod string            `json:"token_endpoint_auth_method" validate:"omitempty,oneof=client_secret_basic client_secret_post"`
	ExtraParams             map[string]string `json:"extra_params"`
}

// OIDCUserConfig represents OIDC resource owner password credentials configuration.
type OIDCUserConfig struct {
	IssuerURL    string            `json:"issuer_url" validate:"required,url"`
	ClientID     string            `json:"client_id" validate:"required"`
	ClientSecret string            `json:"client_secret"`
	Username     string            `json:"username" validate:"required"`
	Password     string            `json:"password" validate:"required"`
	Scopes       []string          `json:"scopes"`
	Audience     string            `json:"audience"`
	ExtraParams  map[string]string `json:"extra_params"`
}

// Provider defines the interface for authentication providers.
type Provider interface {
	// Apply applies authentication to the HTTP request.
	Apply(ctx context.Context, req *http.Request) error

	// Type returns the authentication type.
	Type() Type
}
