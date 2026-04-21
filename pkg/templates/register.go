// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package templates registers all built-in workflow templates into the catalog.
package templates

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	tracerMidaz "github.com/LerianStudio/flowker/pkg/templates/tracer_midaz"
)

// RegisterDefaults registers all built-in workflow templates into the given catalog.
// It should be called once during initialization after providers and triggers are registered.
func RegisterDefaults(catalog executor.Catalog) error {
	if catalog == nil {
		return nil
	}

	if err := tracerMidaz.Register(catalog); err != nil {
		return fmt.Errorf("failed to register tracer-midaz template: %w", err)
	}

	return nil
}
