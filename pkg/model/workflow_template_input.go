// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

// CreateWorkflowFromTemplateInput is the input DTO for creating a workflow from a template.
type CreateWorkflowFromTemplateInput struct {
	TemplateID string         `json:"templateId" validate:"required"`
	Params     map[string]any `json:"params" validate:"required"`
}
