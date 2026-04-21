// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package base

import (
	"encoding/json"
	"fmt"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/executor"
)

// Provider is a base implementation of the executor.Provider interface.
// It validates the config schema JSON at construction time (Rich Domain Model).
type Provider struct {
	id           executor.ProviderID
	name         string
	description  string
	version      string
	configSchema string
}

// NewProvider creates a new base Provider with validated fields.
// Returns error if required fields are missing or the config schema is invalid JSON.
func NewProvider(id executor.ProviderID, name, description, version, configSchema string) (*Provider, error) {
	if id == "" {
		return nil, pkg.ValidationError{
			EntityType: "Provider",
			Code:       constant.ErrProviderInvalidConfig.Error(),
			Message:    "provider id is required",
		}
	}

	if name == "" {
		return nil, pkg.ValidationError{
			EntityType: "Provider",
			Code:       constant.ErrProviderInvalidConfig.Error(),
			Message:    "provider name is required",
		}
	}

	if version == "" {
		return nil, pkg.ValidationError{
			EntityType: "Provider",
			Code:       constant.ErrProviderInvalidConfig.Error(),
			Message:    "provider version is required",
		}
	}

	if configSchema == "" {
		return nil, pkg.ValidationError{
			EntityType: "Provider",
			Code:       constant.ErrProviderInvalidConfig.Error(),
			Message:    "provider config schema is required",
		}
	}

	var schemaDoc any
	if err := json.Unmarshal([]byte(configSchema), &schemaDoc); err != nil {
		return nil, pkg.ValidationError{
			EntityType: "Provider",
			Code:       constant.ErrProviderInvalidConfig.Error(),
			Message:    fmt.Sprintf("invalid config schema JSON for provider %s: %v", id, err),
			Err:        err,
		}
	}

	return &Provider{
		id:           id,
		name:         name,
		description:  description,
		version:      version,
		configSchema: configSchema,
	}, nil
}

// ID returns the provider's unique identifier.
func (p *Provider) ID() executor.ProviderID {
	return p.id
}

// Name returns the provider's human-readable name.
func (p *Provider) Name() string {
	return p.name
}

// Description returns the provider's description.
func (p *Provider) Description() string {
	return p.description
}

// Version returns the provider's version.
func (p *Provider) Version() string {
	return p.version
}

// ConfigSchema returns the provider's JSON Schema for configuration validation.
func (p *Provider) ConfigSchema() string {
	return p.configSchema
}

// Verify Provider implements executor.Provider interface.
var _ executor.Provider = (*Provider)(nil)
