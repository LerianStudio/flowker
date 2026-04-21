// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// NewFromConfig creates an auth provider from the given configuration.
func NewFromConfig(authConfig map[string]any, httpClient *http.Client) (Provider, error) {
	if authConfig == nil {
		return NewNoneProvider(), nil
	}

	authType, ok := authConfig["type"].(string)
	if !ok || authType == "" || authType == string(TypeNone) {
		return NewNoneProvider(), nil
	}

	configData, _ := authConfig["config"].(map[string]any)
	cacheData, _ := authConfig["cache"].(map[string]any)

	switch Type(authType) {
	case TypeAPIKey:
		cfg, err := parseAPIKeyConfig(configData)
		if err != nil {
			return nil, err
		}

		return NewAPIKeyProvider(cfg)

	case TypeBearer:
		cfg, err := parseBearerConfig(configData)
		if err != nil {
			return nil, err
		}

		return NewBearerProvider(cfg)

	case TypeBasic:
		cfg, err := parseBasicConfig(configData)
		if err != nil {
			return nil, err
		}

		return NewBasicProvider(cfg)

	case TypeOIDCClientCredentials:
		cfg, err := parseOIDCClientCredentialsConfig(configData)
		if err != nil {
			return nil, err
		}

		cacheCfg := parseCacheConfig(cacheData)

		return NewOIDCClientCredentialsProvider(cfg, cacheCfg, httpClient)

	case TypeOIDCUser:
		cfg, err := parseOIDCUserConfig(configData)
		if err != nil {
			return nil, err
		}

		cacheCfg := parseCacheConfig(cacheData)

		return NewOIDCUserProvider(cfg, cacheCfg, httpClient)

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownAuthType, authType)
	}
}

func parseAPIKeyConfig(data map[string]any) (*APIKeyConfig, error) {
	cfg := &APIKeyConfig{}

	if err := mapToStruct(data, cfg); err != nil {
		return nil, fmt.Errorf("parse api_key config: %w", err)
	}

	if cfg.Key == "" {
		return nil, ErrAPIKeyKeyRequired
	}

	return cfg, nil
}

func parseBearerConfig(data map[string]any) (*BearerConfig, error) {
	cfg := &BearerConfig{}

	if err := mapToStruct(data, cfg); err != nil {
		return nil, fmt.Errorf("parse bearer config: %w", err)
	}

	if cfg.Token == "" {
		return nil, ErrBearerTokenRequired
	}

	return cfg, nil
}

func parseBasicConfig(data map[string]any) (*BasicConfig, error) {
	cfg := &BasicConfig{}

	if err := mapToStruct(data, cfg); err != nil {
		return nil, fmt.Errorf("parse basic config: %w", err)
	}

	if cfg.Username == "" {
		return nil, ErrBasicUsernameRequired
	}

	if cfg.Password == "" {
		return nil, ErrBasicPasswordRequired
	}

	return cfg, nil
}

func parseOIDCClientCredentialsConfig(data map[string]any) (*OIDCClientCredentialsConfig, error) {
	cfg := &OIDCClientCredentialsConfig{}

	if err := mapToStruct(data, cfg); err != nil {
		return nil, fmt.Errorf("parse oidc_client_credentials config: %w", err)
	}

	if cfg.IssuerURL == "" {
		return nil, ErrOIDCClientCredentialsIssuerRequired
	}

	if cfg.ClientID == "" {
		return nil, ErrOIDCClientCredentialsClientRequired
	}

	if cfg.ClientSecret == "" {
		return nil, ErrOIDCClientCredentialsSecretRequired
	}

	return cfg, nil
}

func parseOIDCUserConfig(data map[string]any) (*OIDCUserConfig, error) {
	cfg := &OIDCUserConfig{}

	if err := mapToStruct(data, cfg); err != nil {
		return nil, fmt.Errorf("parse oidc_user config: %w", err)
	}

	if cfg.IssuerURL == "" {
		return nil, ErrOIDCUserIssuerRequired
	}

	if cfg.ClientID == "" {
		return nil, ErrOIDCUserClientRequired
	}

	if cfg.Username == "" {
		return nil, ErrOIDCUserUsernameRequired
	}

	if cfg.Password == "" {
		return nil, ErrOIDCUserPasswordRequired
	}

	return cfg, nil
}

func parseCacheConfig(data map[string]any) *CacheCfg {
	if data == nil {
		return nil
	}

	cfg := &CacheCfg{}
	if err := mapToStruct(data, cfg); err != nil {
		// Invalid cache config, return nil to use defaults
		return nil
	}

	return cfg
}

func mapToStruct(data map[string]any, target any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, target)
}
