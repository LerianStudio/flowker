// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package http provides an HTTP executor for making REST API calls.
package http

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
)

// ProviderID is the unique identifier for the HTTP provider.
const ProviderID executor.ProviderID = "http"

// ID is the unique identifier for the HTTP executor.
const ID executor.ID = "http"

// Version is the current version of the HTTP executor.
const Version = "v1"

// Category is the category for the HTTP executor.
const Category = "HTTP"

// HTTPExecutor is the HTTP REST API executor.
type HTTPExecutor struct {
	*base.Executor
}

// New creates a new HTTP executor.
func New() (*HTTPExecutor, error) {
	e, err := base.NewExecutor(ID, "HTTP Request", Category, Version, ProviderID, schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP executor: %w", err)
	}

	return &HTTPExecutor{Executor: e}, nil
}

// Verify HTTPExecutor implements executor.Executor interface.
var _ executor.Executor = (*HTTPExecutor)(nil)

// Register registers the HTTP provider with its HTTP Request executor into the given catalog.
func Register(catalog executor.Catalog) error {
	provider, err := base.NewProvider(
		ProviderID,
		"HTTP",
		"Generic HTTP/REST API provider for making external HTTP calls",
		"v1",
		providerConfigSchema,
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP provider: %w", err)
	}

	httpExec, err := New()
	if err != nil {
		return fmt.Errorf("failed to create HTTP executor: %w", err)
	}

	return catalog.RegisterProvider(provider, []executor.ExecutorRegistration{
		{
			Executor: httpExec,
			Runner:   NewRunner(),
		},
	})
}

// providerConfigSchema is the JSON Schema for HTTP provider configuration.
const providerConfigSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "base_url": {
      "type": "string",
      "format": "uri",
      "description": "Base URL for all HTTP requests made through this provider"
    },
    "headers": {
      "type": "object",
      "additionalProperties": { "type": "string" },
      "description": "Default headers to include in all requests"
    },
    "timeout_ms": {
      "type": "integer",
      "minimum": 100,
      "maximum": 30000,
      "default": 5000,
      "description": "Default request timeout in milliseconds"
    }
  },
  "required": ["base_url"]
}`

// schema is the JSON Schema for HTTP executor configuration.
const schema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["method", "url"],
  "properties": {
    "executorConfigId": {
      "type": "string",
      "format": "uuid",
      "description": "UUID of the executor configuration to use at runtime for HTTP calls"
    },
    "endpointName": {
      "type": "string",
      "description": "Name of the endpoint in the executor configuration to call"
    },
    "method": {
      "type": "string",
      "enum": ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"],
      "description": "HTTP method to use for the request"
    },
    "url": {
      "type": "string",
      "description": "URL to send the request to. Supports variable interpolation: ${context.nodeId.field}"
    },
    "headers": {
      "type": "object",
      "additionalProperties": {
        "type": "string"
      },
      "description": "HTTP headers to include in the request"
    },
    "query": {
      "type": "object",
      "additionalProperties": {
        "type": "string"
      },
      "description": "Query parameters to append to the URL"
    },
    "body": {
      "description": "Request body. Can be a string or object (will be JSON encoded)"
    },
    "timeout_seconds": {
      "type": "integer",
      "minimum": 1,
      "maximum": 300,
      "default": 30,
      "description": "Request timeout in seconds"
    },
    "retry": {
      "type": "object",
      "properties": {
        "max_attempts": {
          "type": "integer",
          "minimum": 1,
          "maximum": 10,
          "default": 3,
          "description": "Maximum number of retry attempts"
        },
        "backoff_seconds": {
          "type": "integer",
          "minimum": 1,
          "maximum": 60,
          "default": 1,
          "description": "Initial backoff duration in seconds (doubles on each retry)"
        }
      },
      "description": "Retry configuration for failed requests"
    },
    "success_status_codes": {
      "type": "array",
      "items": {
        "type": "integer",
        "minimum": 100,
        "maximum": 599
      },
      "default": [200, 201, 202, 204],
      "description": "HTTP status codes considered successful"
    },
    "auth": {
      "type": "object",
      "description": "Authentication configuration for the request",
      "properties": {
        "type": {
          "type": "string",
          "enum": ["none", "api_key", "bearer", "basic", "oidc_client_credentials", "oidc_user"],
          "default": "none",
          "description": "Authentication type"
        },
        "config": {
          "type": "object",
          "description": "Authentication-specific configuration"
        },
        "cache": {
          "type": "object",
          "description": "Token caching configuration (for OIDC types)",
          "properties": {
            "enabled": {
              "type": "boolean",
              "default": true,
              "description": "Enable token caching"
            },
            "refresh_before_expiry_seconds": {
              "type": "integer",
              "default": 60,
              "minimum": 0,
              "description": "Refresh token this many seconds before expiry"
            },
            "use_refresh_token": {
              "type": "boolean",
              "default": true,
              "description": "Use refresh token when available (oidc_user only)"
            }
          }
        }
      },
      "allOf": [
        {
          "if": {
            "properties": { "type": { "const": "api_key" } }
          },
          "then": {
            "properties": {
              "config": {
                "type": "object",
                "required": ["key"],
                "properties": {
                  "key": { "type": "string", "minLength": 1, "description": "API key value" },
                  "header_name": { "type": "string", "default": "X-API-Key", "description": "Header name" },
                  "prefix": { "type": "string", "default": "", "description": "Optional prefix" },
                  "location": { "type": "string", "enum": ["header", "query"], "default": "header", "description": "Where to send the API key" },
                  "query_param_name": { "type": "string", "default": "api_key", "description": "Query parameter name" }
                }
              }
            }
          }
        },
        {
          "if": {
            "properties": { "type": { "const": "bearer" } }
          },
          "then": {
            "properties": {
              "config": {
                "type": "object",
                "required": ["token"],
                "properties": {
                  "token": { "type": "string", "minLength": 1, "description": "Bearer token value" }
                }
              }
            }
          }
        },
        {
          "if": {
            "properties": { "type": { "const": "basic" } }
          },
          "then": {
            "properties": {
              "config": {
                "type": "object",
                "required": ["username", "password"],
                "properties": {
                  "username": { "type": "string", "minLength": 1, "description": "Username" },
                  "password": { "type": "string", "minLength": 1, "description": "Password" }
                }
              }
            }
          }
        },
        {
          "if": {
            "properties": { "type": { "const": "oidc_client_credentials" } }
          },
          "then": {
            "properties": {
              "config": {
                "type": "object",
                "required": ["issuer_url", "client_id", "client_secret"],
                "properties": {
                  "issuer_url": { "type": "string", "format": "uri", "description": "OIDC Issuer URL (e.g., https://auth.example.com/realms/myrealm)" },
                  "client_id": { "type": "string", "minLength": 1, "description": "OAuth2 Client ID" },
                  "client_secret": { "type": "string", "minLength": 1, "description": "OAuth2 Client Secret" },
                  "scopes": { "type": "array", "items": { "type": "string" }, "description": "OAuth2 scopes to request" },
                  "audience": { "type": "string", "description": "Optional audience parameter" },
                  "token_endpoint_auth_method": { "type": "string", "enum": ["client_secret_basic", "client_secret_post"], "default": "client_secret_basic", "description": "How to send client credentials" },
                  "extra_params": { "type": "object", "additionalProperties": { "type": "string" }, "description": "Extra parameters for token request" }
                }
              }
            }
          }
        },
        {
          "if": {
            "properties": { "type": { "const": "oidc_user" } }
          },
          "then": {
            "properties": {
              "config": {
                "type": "object",
                "required": ["issuer_url", "client_id", "username", "password"],
                "properties": {
                  "issuer_url": { "type": "string", "format": "uri", "description": "OIDC Issuer URL (e.g., https://auth.example.com/realms/myrealm)" },
                  "client_id": { "type": "string", "minLength": 1, "description": "OAuth2 Client ID" },
                  "client_secret": { "type": "string", "description": "OAuth2 Client Secret (optional for public clients)" },
                  "username": { "type": "string", "minLength": 1, "description": "User's username" },
                  "password": { "type": "string", "minLength": 1, "description": "User's password" },
                  "scopes": { "type": "array", "items": { "type": "string" }, "default": ["openid"], "description": "OAuth2 scopes to request" },
                  "audience": { "type": "string", "description": "Optional audience parameter" },
                  "extra_params": { "type": "object", "additionalProperties": { "type": "string" }, "description": "Extra parameters for token request" }
                }
              }
            }
          }
        }
      ]
    },
    "inputMapping": {
      "type": "array",
      "description": "Field mappings from workflow data to executor request",
      "items": {
        "type": "object",
        "required": ["source", "target"],
        "properties": {
          "source": { "type": "string", "minLength": 1, "description": "JSONPath source in workflow context (e.g., workflow.customer.cpf)" },
          "target": { "type": "string", "minLength": 1, "description": "JSONPath target in executor request (e.g., executor.document)" },
          "required": { "type": "boolean", "default": false, "description": "If true, source field must exist" },
          "transformation": {
            "type": "object",
            "description": "Optional transformation to apply",
            "required": ["type"],
            "properties": {
              "type": { "type": "string", "minLength": 1, "description": "Transformation type (e.g., remove_characters, to_uppercase)" },
              "config": { "type": "object", "description": "Type-specific configuration" }
            }
          }
        }
      }
    },
    "outputMapping": {
      "type": "array",
      "description": "Field mappings from executor response to workflow data",
      "items": {
        "type": "object",
        "required": ["source", "target"],
        "properties": {
          "source": { "type": "string", "minLength": 1, "description": "JSONPath source in executor response (e.g., executor.accountId)" },
          "target": { "type": "string", "minLength": 1, "description": "JSONPath target in workflow context (e.g., workflow.result.id)" },
          "required": { "type": "boolean", "default": false, "description": "If true, source field must exist" },
          "transformation": {
            "type": "object",
            "description": "Optional transformation to apply",
            "required": ["type"],
            "properties": {
              "type": { "type": "string", "minLength": 1, "description": "Transformation type (e.g., remove_characters, to_uppercase)" },
              "config": { "type": "object", "description": "Type-specific configuration" }
            }
          }
        }
      }
    },
    "transforms": {
      "type": "array",
      "description": "Raw Kazaam transformation operations for advanced use cases",
      "items": {
        "type": "object",
        "required": ["operation", "spec"],
        "properties": {
          "operation": { "type": "string", "minLength": 1, "description": "Kazaam operation type (e.g., shift, concat)" },
          "spec": { "type": "object", "description": "Operation-specific specification" },
          "require": { "type": "boolean", "default": false, "description": "If true, paths must exist" }
        }
      }
    }
  },
  "additionalProperties": false
}`
