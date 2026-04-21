// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"time"

	"github.com/google/uuid"
)

// ProviderConfigTestResultOutput represents the API response for a provider config connectivity test.
type ProviderConfigTestResultOutput struct {
	ProviderConfigID uuid.UUID               `json:"providerConfigId" swaggertype:"string" format:"uuid"`
	ProviderID       string                  `json:"providerId"`
	OverallStatus    string                  `json:"overallStatus"`
	DurationMs       int64                   `json:"durationMs"`
	Stages           []StageTestResultOutput `json:"stages"`
	Summary          string                  `json:"summary"`
	StartedAt        time.Time               `json:"startedAt"`
	CompletedAt      *time.Time              `json:"completedAt,omitempty"`
}

// ProviderConfigTestResultOutputFromDomain converts a ProviderConfigTestResult to its output representation.
func ProviderConfigTestResultOutputFromDomain(result *ProviderConfigTestResult) ProviderConfigTestResultOutput {
	if result == nil {
		return ProviderConfigTestResultOutput{}
	}

	stages := make([]StageTestResultOutput, len(result.Stages()))
	for i, stage := range result.Stages() {
		stages[i] = StageTestResultOutputFromDomain(stage)
	}

	return ProviderConfigTestResultOutput{
		ProviderConfigID: result.ProviderConfigID(),
		ProviderID:       result.ProviderID(),
		OverallStatus:    string(result.OverallStatus()),
		DurationMs:       result.DurationMs(),
		Stages:           stages,
		Summary:          result.Summary(),
		StartedAt:        result.StartedAt(),
		CompletedAt:      result.CompletedAt(),
	}
}
