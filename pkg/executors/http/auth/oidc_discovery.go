// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OIDCDiscoveryDocument represents the OpenID Connect Discovery document.
type OIDCDiscoveryDocument struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	JwksURI                           string   `json:"jwks_uri"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// DiscoveryClient fetches and caches OIDC discovery documents.
type DiscoveryClient struct {
	httpClient *http.Client
	cache      map[string]*cachedDiscovery
	cacheMu    sync.RWMutex
	cacheTTL   time.Duration
}

type cachedDiscovery struct {
	doc       *OIDCDiscoveryDocument
	fetchedAt time.Time
}

// NewDiscoveryClient creates a new OIDC discovery client.
func NewDiscoveryClient(httpClient *http.Client) *DiscoveryClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &DiscoveryClient{
		httpClient: httpClient,
		cache:      make(map[string]*cachedDiscovery),
		cacheTTL:   1 * time.Hour,
	}
}

// Discover fetches the OIDC discovery document for the given issuer.
func (c *DiscoveryClient) Discover(ctx context.Context, issuerURL string) (*OIDCDiscoveryDocument, error) {
	// Normalize and validate issuer URL
	issuerURL = strings.TrimSpace(issuerURL)
	issuerURL = strings.TrimSuffix(issuerURL, "/")

	if issuerURL == "" {
		return nil, fmt.Errorf("issuer URL is required")
	}

	// Check cache
	c.cacheMu.RLock()
	cached, ok := c.cache[issuerURL]
	c.cacheMu.RUnlock()

	if ok && time.Since(cached.fetchedAt) < c.cacheTTL {
		return cached.doc, nil
	}

	// Fetch discovery document
	wellKnownURL := issuerURL + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnownURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch discovery document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discovery endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var doc OIDCDiscoveryDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode discovery document: %w", err)
	}

	// Validate required fields
	if doc.TokenEndpoint == "" {
		return nil, ErrDiscoveryMissingTokenEndpoint
	}

	// Cache the result
	c.cacheMu.Lock()
	c.cache[issuerURL] = &cachedDiscovery{
		doc:       &doc,
		fetchedAt: time.Now(),
	}
	c.cacheMu.Unlock()

	return &doc, nil
}

// InvalidateCache removes a cached discovery document.
func (c *DiscoveryClient) InvalidateCache(issuerURL string) {
	issuerURL = strings.TrimSuffix(issuerURL, "/")

	c.cacheMu.Lock()
	delete(c.cache, issuerURL)
	c.cacheMu.Unlock()
}
