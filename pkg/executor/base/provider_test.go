// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package base_test

import (
	"testing"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name         string
		id           executor.ProviderID
		provName     string
		description  string
		version      string
		configSchema string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid provider with full fields",
			id:           "http",
			provName:     "HTTP",
			description:  "HTTP provider",
			version:      "v1",
			configSchema: `{"type":"object","properties":{"base_url":{"type":"string"}},"required":["base_url"]}`,
			wantErr:      false,
		},
		{
			name:         "valid provider without description",
			id:           "s3",
			provName:     "S3",
			description:  "",
			version:      "v1",
			configSchema: `{"type":"object"}`,
			wantErr:      false,
		},
		{
			name:         "empty id returns error",
			id:           "",
			provName:     "HTTP",
			version:      "v1",
			configSchema: `{"type":"object"}`,
			wantErr:      true,
			errContains:  "provider id is required",
		},
		{
			name:         "empty name returns error",
			id:           "http",
			provName:     "",
			version:      "v1",
			configSchema: `{"type":"object"}`,
			wantErr:      true,
			errContains:  "provider name is required",
		},
		{
			name:         "empty version returns error",
			id:           "http",
			provName:     "HTTP",
			version:      "",
			configSchema: `{"type":"object"}`,
			wantErr:      true,
			errContains:  "provider version is required",
		},
		{
			name:         "empty config schema returns error",
			id:           "http",
			provName:     "HTTP",
			version:      "v1",
			configSchema: "",
			wantErr:      true,
			errContains:  "provider config schema is required",
		},
		{
			name:         "invalid JSON config schema returns error",
			id:           "http",
			provName:     "HTTP",
			version:      "v1",
			configSchema: "not-json",
			wantErr:      true,
			errContains:  "invalid config schema JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := base.NewProvider(
				tt.id,
				tt.provName,
				tt.description,
				tt.version,
				tt.configSchema,
			)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, provider)

				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.id, provider.ID())
			assert.Equal(t, tt.provName, provider.Name())
			assert.Equal(t, tt.description, provider.Description())
			assert.Equal(t, tt.version, provider.Version())
			assert.Equal(t, tt.configSchema, provider.ConfigSchema())
		})
	}
}
