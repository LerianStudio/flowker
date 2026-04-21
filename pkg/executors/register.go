// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package executors

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executors/midaz"
	"github.com/LerianStudio/flowker/pkg/executors/tracer"
)

// RegisterDefaults registers all built-in providers and their executors into the given catalog.
// It should be called once during initialization. Duplicate registrations will return an error.
//
// Note: HTTP and S3 providers are used internally by other providers (e.g., Midaz and Tracer
// use the HTTP runner) but are not registered in the catalog as standalone providers.
func RegisterDefaults(catalog executor.Catalog) error {
	if catalog == nil {
		return nil
	}

	if err := midaz.Register(catalog); err != nil {
		return fmt.Errorf("failed to register Midaz provider: %w", err)
	}

	if err := tracer.Register(catalog); err != nil {
		return fmt.Errorf("failed to register Tracer provider: %w", err)
	}

	return nil
}
