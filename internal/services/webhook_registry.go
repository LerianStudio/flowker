// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package services

import (
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/webhook"
)

// PopulateRegistryFromWorkflows scans a slice of workflows and registers
// webhook triggers for all active workflows. Returns the number of routes
// registered. Errors from individual registrations (e.g., duplicate paths)
// are skipped silently since this is used at startup for best-effort population.
func PopulateRegistryFromWorkflows(registry *webhook.Registry, workflows []*model.Workflow) int {
	registered := 0

	for _, wf := range workflows {
		if wf.Status() != model.WorkflowStatusActive {
			continue
		}

		for _, node := range wf.Nodes() {
			if node.Type() != model.NodeTypeTrigger {
				continue
			}

			if node.TriggerType() != "webhook" {
				continue
			}

			data := node.Data()

			path, _ := data["path"].(string)
			method, _ := data["method"].(string)
			verifyToken, _ := data["verify_token"].(string)

			if path == "" || method == "" {
				continue
			}

			route := webhook.Route{
				WorkflowID:  wf.ID(),
				Path:        path,
				Method:      method,
				VerifyToken: verifyToken,
			}

			if err := registry.Register(route); err == nil {
				registered++
			}
		}
	}

	return registered
}
