// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/LerianStudio/flowker/pkg"
)

// TokenResponse represents an OAuth2 token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// CachedToken represents a cached token with expiry info.
type CachedToken struct {
	Token     *TokenResponse
	ExpiresAt time.Time
}

// IsExpired checks if the token is expired or will expire within the buffer.
func (ct *CachedToken) IsExpired(bufferSeconds int) bool {
	buffer := time.Duration(bufferSeconds) * time.Second
	return time.Now().Add(buffer).After(ct.ExpiresAt)
}

// TokenFetcher fetches and caches OAuth2 tokens.
type TokenFetcher struct {
	httpClient      *http.Client
	discoveryClient *DiscoveryClient
	cache           map[string]*CachedToken
	cacheMu         sync.RWMutex
}

// NewTokenFetcher creates a new token fetcher.
func NewTokenFetcher(httpClient *http.Client, discoveryClient *DiscoveryClient) *TokenFetcher {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	if discoveryClient == nil {
		discoveryClient = NewDiscoveryClient(httpClient)
	}

	return &TokenFetcher{
		httpClient:      httpClient,
		discoveryClient: discoveryClient,
		cache:           make(map[string]*CachedToken),
	}
}

// FetchClientCredentialsToken fetches a token using client credentials flow.
func (f *TokenFetcher) FetchClientCredentialsToken(
	ctx context.Context,
	cfg *OIDCClientCredentialsConfig,
	cacheKey string,
	cacheConfig *CacheCfg,
) (*TokenResponse, error) {
	// Check cache first
	if token := f.getCachedToken(cacheKey, cacheConfig); token != nil {
		return token, nil
	}

	// Fetch new token
	token, err := f.fetchNewClientCredentialsToken(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Cache token
	f.cacheToken(cacheKey, token, cacheConfig)

	return token, nil
}

func (f *TokenFetcher) getCachedToken(cacheKey string, cacheConfig *CacheCfg) *TokenResponse {
	if cacheConfig == nil || !cacheConfig.Enabled {
		return nil
	}

	f.cacheMu.RLock()
	cached, ok := f.cache[cacheKey]
	f.cacheMu.RUnlock()

	if ok && !cached.IsExpired(cacheConfig.RefreshBeforeExpirySeconds) {
		return cached.Token
	}

	return nil
}

func (f *TokenFetcher) fetchNewClientCredentialsToken(ctx context.Context, cfg *OIDCClientCredentialsConfig) (*TokenResponse, error) {
	discovery, err := f.discoveryClient.Discover(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}

	authMethod := cfg.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = "client_secret_basic"
	}

	if authMethod != "client_secret_basic" && authMethod != "client_secret_post" {
		return nil, pkg.ValidationError{
			EntityType: "OIDCClientCredentialsConfig",
			Message:    fmt.Sprintf("unsupported token endpoint auth method: %s", authMethod),
		}
	}

	data := f.buildClientCredentialsData(cfg, authMethod)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if authMethod == "client_secret_basic" {
		auth := base64.StdEncoding.EncodeToString([]byte(cfg.ClientID + ":" + cfg.ClientSecret))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	return f.executeTokenRequest(req)
}

func (f *TokenFetcher) buildClientCredentialsData(cfg *OIDCClientCredentialsConfig, authMethod string) url.Values {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	if len(cfg.Scopes) > 0 {
		data.Set("scope", strings.Join(cfg.Scopes, " "))
	}

	if cfg.Audience != "" {
		data.Set("audience", cfg.Audience)
	}

	for k, v := range cfg.ExtraParams {
		data.Set(k, v)
	}

	if authMethod == "client_secret_post" {
		data.Set("client_id", cfg.ClientID)
		data.Set("client_secret", cfg.ClientSecret)
	}

	return data
}

// FetchPasswordToken fetches a token using resource owner password credentials flow.
func (f *TokenFetcher) FetchPasswordToken(
	ctx context.Context,
	cfg *OIDCUserConfig,
	cacheKey string,
	cacheConfig *CacheCfg,
) (*TokenResponse, error) {
	// Try to get from cache or refresh
	if token := f.tryGetCachedOrRefreshedToken(ctx, cfg, cacheKey, cacheConfig); token != nil {
		return token, nil
	}

	// Fetch new token
	token, err := f.fetchNewPasswordToken(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// Cache the token
	f.cacheToken(cacheKey, token, cacheConfig)

	return token, nil
}

func (f *TokenFetcher) tryGetCachedOrRefreshedToken(
	ctx context.Context,
	cfg *OIDCUserConfig,
	cacheKey string,
	cacheConfig *CacheCfg,
) *TokenResponse {
	if cacheConfig == nil || !cacheConfig.Enabled {
		return nil
	}

	f.cacheMu.RLock()
	cached, ok := f.cache[cacheKey]
	f.cacheMu.RUnlock()

	if !ok {
		return nil
	}

	// Return cached if not expired
	if !cached.IsExpired(cacheConfig.RefreshBeforeExpirySeconds) {
		return cached.Token
	}

	// Try refresh token if available
	if cached.Token.RefreshToken != "" && cacheConfig.UseRefreshToken {
		token, err := f.refreshToken(ctx, cfg.IssuerURL, cfg.ClientID, cfg.ClientSecret, cached.Token.RefreshToken)
		if err == nil {
			f.cacheToken(cacheKey, token, cacheConfig)

			return token
		}
	}

	return nil
}

func (f *TokenFetcher) fetchNewPasswordToken(ctx context.Context, cfg *OIDCUserConfig) (*TokenResponse, error) {
	discovery, err := f.discoveryClient.Discover(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}

	data := f.buildPasswordTokenData(cfg)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return f.executeTokenRequest(req)
}

func (f *TokenFetcher) buildPasswordTokenData(cfg *OIDCUserConfig) url.Values {
	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", cfg.Username)
	data.Set("password", cfg.Password)
	data.Set("client_id", cfg.ClientID)

	if cfg.ClientSecret != "" {
		data.Set("client_secret", cfg.ClientSecret)
	}

	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid"}
	}

	data.Set("scope", strings.Join(scopes, " "))

	if cfg.Audience != "" {
		data.Set("audience", cfg.Audience)
	}

	for k, v := range cfg.ExtraParams {
		data.Set(k, v)
	}

	return data
}

func (f *TokenFetcher) cacheToken(cacheKey string, token *TokenResponse, cacheConfig *CacheCfg) {
	if cacheConfig == nil || !cacheConfig.Enabled || token.ExpiresIn <= 0 {
		return
	}

	f.cacheMu.Lock()
	f.cache[cacheKey] = &CachedToken{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
	}
	f.cacheMu.Unlock()
}

func (f *TokenFetcher) refreshToken(ctx context.Context, issuerURL, clientID, clientSecret, refreshToken string) (*TokenResponse, error) {
	discovery, err := f.discoveryClient.Discover(ctx, issuerURL)
	if err != nil {
		return nil, err
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", clientID)

	if clientSecret != "" {
		data.Set("client_secret", clientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discovery.TokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return f.executeTokenRequest(req)
}

func (f *TokenFetcher) executeTokenRequest(req *http.Request) (*TokenResponse, error) {
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	return &token, nil
}

// InvalidateCache removes a cached token.
func (f *TokenFetcher) InvalidateCache(cacheKey string) {
	f.cacheMu.Lock()
	delete(f.cache, cacheKey)
	f.cacheMu.Unlock()
}
