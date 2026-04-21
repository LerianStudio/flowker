// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package triggers

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	webhooktrigger "github.com/LerianStudio/flowker/pkg/triggers/webhook"
)

// RegisterDefaults registers built-in triggers into the given catalog.
func RegisterDefaults(catalog executor.Catalog) error {
	if catalog == nil {
		return nil
	}

	webhook, err := webhooktrigger.New()
	if err != nil {
		return fmt.Errorf("failed to register webhook trigger: %w", err)
	}

	catalog.RegisterTrigger(webhook)

	return nil
}
