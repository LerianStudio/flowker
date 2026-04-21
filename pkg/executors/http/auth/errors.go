// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package auth

import (
	"errors"

	"github.com/LerianStudio/flowker/pkg"
)

// Factory errors - returned when parsing auth configuration from maps.
var (
	ErrAPIKeyKeyRequired                   = pkg.ValidationError{EntityType: "APIKeyConfig", Message: "api_key config: key is required"}
	ErrBearerTokenRequired                 = pkg.ValidationError{EntityType: "BearerConfig", Message: "bearer config: token is required"}
	ErrBasicUsernameRequired               = pkg.ValidationError{EntityType: "BasicConfig", Message: "basic config: username is required"}
	ErrBasicPasswordRequired               = pkg.ValidationError{EntityType: "BasicConfig", Message: "basic config: password is required"}
	ErrOIDCClientCredentialsIssuerRequired = pkg.ValidationError{EntityType: "OIDCClientCredentialsConfig", Message: "oidc_client_credentials config: issuer_url is required"}
	ErrOIDCClientCredentialsClientRequired = pkg.ValidationError{EntityType: "OIDCClientCredentialsConfig", Message: "oidc_client_credentials config: client_id is required"}
	ErrOIDCClientCredentialsSecretRequired = pkg.ValidationError{EntityType: "OIDCClientCredentialsConfig", Message: "oidc_client_credentials config: client_secret is required"}
	ErrOIDCUserIssuerRequired              = pkg.ValidationError{EntityType: "OIDCUserConfig", Message: "oidc_user config: issuer_url is required"}
	ErrOIDCUserClientRequired              = pkg.ValidationError{EntityType: "OIDCUserConfig", Message: "oidc_user config: client_id is required"}
	ErrOIDCUserUsernameRequired            = pkg.ValidationError{EntityType: "OIDCUserConfig", Message: "oidc_user config: username is required"}
	ErrOIDCUserPasswordRequired            = pkg.ValidationError{EntityType: "OIDCUserConfig", Message: "oidc_user config: password is required"}
)

// Provider constructor errors - returned when creating providers directly.
var (
	ErrAPIKeyConfigRequired  = pkg.ValidationError{EntityType: "APIKeyConfig", Message: "api_key config is required"}
	ErrAPIKeyInvalidLocation = pkg.ValidationError{EntityType: "APIKeyConfig", Message: "api_key config: location must be 'header' or 'query'"}
	ErrBearerConfigRequired  = pkg.ValidationError{EntityType: "BearerConfig", Message: "bearer config is required"}
	ErrBasicConfigRequired   = pkg.ValidationError{EntityType: "BasicConfig", Message: "basic config is required"}
)

// Discovery errors - returned during OIDC discovery.
var (
	ErrDiscoveryMissingTokenEndpoint = errors.New("discovery document missing token_endpoint")
)

// ErrUnknownAuthType is returned when the auth type is not recognized.
// This is a plain error (not ValidationError) because it wraps the unknown type string via fmt.Errorf.
var ErrUnknownAuthType = errors.New("unknown auth type")
