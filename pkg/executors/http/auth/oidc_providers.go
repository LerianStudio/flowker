// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/LerianStudio/flowker/pkg"
)

// OIDCClientCredentialsProvider provides OIDC client credentials authentication.
type OIDCClientCredentialsProvider struct {
	config       *OIDCClientCredentialsConfig
	cacheConfig  *CacheCfg
	tokenFetcher *TokenFetcher
	cacheKey     string
}

// NewOIDCClientCredentialsProvider creates a new OIDC client credentials provider.
// Returns an error if cfg is nil or required fields are missing.
func NewOIDCClientCredentialsProvider(cfg *OIDCClientCredentialsConfig, cacheCfg *CacheCfg, httpClient *http.Client) (*OIDCClientCredentialsProvider, error) {
	if cfg == nil {
		return nil, pkg.ValidationError{
			EntityType: "OIDCClientCredentialsConfig",
			Message:    "oidc_client_credentials config is required",
		}
	}

	if cfg.IssuerURL == "" {
		return nil, pkg.ValidationError{
			EntityType: "OIDCClientCredentialsConfig",
			Message:    "oidc_client_credentials config: issuer_url is required",
		}
	}

	if cfg.ClientID == "" {
		return nil, pkg.ValidationError{
			EntityType: "OIDCClientCredentialsConfig",
			Message:    "oidc_client_credentials config: client_id is required",
		}
	}

	if cfg.ClientSecret == "" {
		return nil, pkg.ValidationError{
			EntityType: "OIDCClientCredentialsConfig",
			Message:    "oidc_client_credentials config: client_secret is required",
		}
	}

	// Apply defaults
	if cacheCfg == nil {
		cacheCfg = &CacheCfg{
			Enabled:                    true,
			RefreshBeforeExpirySeconds: 60,
		}
	}

	// Generate cache key based on config
	cacheKey := generateCacheKey("cc", cfg.IssuerURL, cfg.ClientID, cfg.Scopes)

	return &OIDCClientCredentialsProvider{
		config:       cfg,
		cacheConfig:  cacheCfg,
		tokenFetcher: NewTokenFetcher(httpClient, nil),
		cacheKey:     cacheKey,
	}, nil
}

// Apply implements Provider interface.
func (p *OIDCClientCredentialsProvider) Apply(ctx context.Context, req *http.Request) error {
	token, err := p.tokenFetcher.FetchClientCredentialsToken(ctx, p.config, p.cacheKey, p.cacheConfig)
	if err != nil {
		return fmt.Errorf("fetch client credentials token: %w", err)
	}

	tokenType := token.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	req.Header.Set("Authorization", tokenType+" "+token.AccessToken)

	return nil
}

// Type implements Provider interface.
func (p *OIDCClientCredentialsProvider) Type() Type {
	return TypeOIDCClientCredentials
}

// Verify OIDCClientCredentialsProvider implements Provider interface.
var _ Provider = (*OIDCClientCredentialsProvider)(nil)

// OIDCUserProvider provides OIDC resource owner password credentials authentication.
type OIDCUserProvider struct {
	config       *OIDCUserConfig
	cacheConfig  *CacheCfg
	tokenFetcher *TokenFetcher
	cacheKey     string
}

// NewOIDCUserProvider creates a new OIDC user provider.
// Returns an error if cfg is nil or required fields are missing.
func NewOIDCUserProvider(cfg *OIDCUserConfig, cacheCfg *CacheCfg, httpClient *http.Client) (*OIDCUserProvider, error) {
	if cfg == nil {
		return nil, pkg.ValidationError{
			EntityType: "OIDCUserConfig",
			Message:    "oidc_user config is required",
		}
	}

	if cfg.IssuerURL == "" {
		return nil, pkg.ValidationError{
			EntityType: "OIDCUserConfig",
			Message:    "oidc_user config: issuer_url is required",
		}
	}

	if cfg.ClientID == "" {
		return nil, pkg.ValidationError{
			EntityType: "OIDCUserConfig",
			Message:    "oidc_user config: client_id is required",
		}
	}

	if cfg.Username == "" {
		return nil, pkg.ValidationError{
			EntityType: "OIDCUserConfig",
			Message:    "oidc_user config: username is required",
		}
	}

	if cfg.Password == "" {
		return nil, pkg.ValidationError{
			EntityType: "OIDCUserConfig",
			Message:    "oidc_user config: password is required",
		}
	}

	// Apply defaults
	if cacheCfg == nil {
		cacheCfg = &CacheCfg{
			Enabled:                    true,
			RefreshBeforeExpirySeconds: 60,
			UseRefreshToken:            true,
		}
	}

	// Generate cache key based on config
	cacheKey := generateCacheKey("user", cfg.IssuerURL, cfg.ClientID, cfg.Username, cfg.Scopes)

	return &OIDCUserProvider{
		config:       cfg,
		cacheConfig:  cacheCfg,
		tokenFetcher: NewTokenFetcher(httpClient, nil),
		cacheKey:     cacheKey,
	}, nil
}

// Apply implements Provider interface.
func (p *OIDCUserProvider) Apply(ctx context.Context, req *http.Request) error {
	token, err := p.tokenFetcher.FetchPasswordToken(ctx, p.config, p.cacheKey, p.cacheConfig)
	if err != nil {
		return fmt.Errorf("fetch password token: %w", err)
	}

	tokenType := token.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	req.Header.Set("Authorization", tokenType+" "+token.AccessToken)

	return nil
}

// Type implements Provider interface.
func (p *OIDCUserProvider) Type() Type {
	return TypeOIDCUser
}

// Verify OIDCUserProvider implements Provider interface.
var _ Provider = (*OIDCUserProvider)(nil)

// generateCacheKey generates a unique cache key from the given parts.
func generateCacheKey(parts ...any) string {
	h := sha256.New()

	for _, p := range parts {
		fmt.Fprintf(h, "%v|", p)
	}

	return hex.EncodeToString(h.Sum(nil))[:16]
}
