// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"time"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/google/uuid"
)

// ProviderConfigTestResult represents the result of a provider configuration connectivity test.
type ProviderConfigTestResult struct {
	providerConfigID uuid.UUID
	providerID       string
	overallStatus    TestOverallStatus
	stages           []StageTestResult
	summary          string
	startedAt        time.Time
	completedAt      *time.Time
}

// stageCapacityProviderConfig is the expected number of test stages for provider config tests.
const stageCapacityProviderConfig = 3

// NewProviderConfigTestResult creates a new ProviderConfigTestResult in progress.
func NewProviderConfigTestResult(providerConfigID uuid.UUID, providerID string) (*ProviderConfigTestResult, error) {
	if providerConfigID == uuid.Nil {
		return nil, pkg.ValidationError{
			EntityType: "ProviderConfigTestResult",
			Code:       constant.ErrProviderConfigIDRequired.Error(),
			Message:    "provider config ID is required",
		}
	}

	if providerID == "" {
		return nil, pkg.ValidationError{
			EntityType: "ProviderConfigTestResult",
			Code:       constant.ErrProviderConfigProviderIDRequired.Error(),
			Message:    "provider ID is required",
		}
	}

	return &ProviderConfigTestResult{
		providerConfigID: providerConfigID,
		providerID:       providerID,
		overallStatus:    TestOverallStatusPassed, // Will be recalculated on Complete
		stages:           make([]StageTestResult, 0, stageCapacityProviderConfig),
		startedAt:        time.Now().UTC(),
	}, nil
}

// Getters for ProviderConfigTestResult.
func (p *ProviderConfigTestResult) ProviderConfigID() uuid.UUID      { return p.providerConfigID }
func (p *ProviderConfigTestResult) ProviderID() string               { return p.providerID }
func (p *ProviderConfigTestResult) OverallStatus() TestOverallStatus { return p.overallStatus }
func (p *ProviderConfigTestResult) Summary() string                  { return p.summary }
func (p *ProviderConfigTestResult) StartedAt() time.Time             { return p.startedAt }
func (p *ProviderConfigTestResult) CompletedAt() *time.Time {
	if p.completedAt == nil {
		return nil
	}

	t := *p.completedAt

	return &t
}

// Stages returns a copy of the stages slice.
func (p *ProviderConfigTestResult) Stages() []StageTestResult {
	result := make([]StageTestResult, len(p.stages))
	copy(result, p.stages)

	return result
}

// DurationMs returns the total duration in milliseconds.
func (p *ProviderConfigTestResult) DurationMs() int64 {
	if p.completedAt == nil {
		return time.Since(p.startedAt).Milliseconds()
	}

	return p.completedAt.Sub(p.startedAt).Milliseconds()
}

// AddStageResult adds a stage result to the test.
func (p *ProviderConfigTestResult) AddStageResult(stage StageTestResult) {
	p.stages = append(p.stages, stage)
}

// Complete marks the test as complete and calculates overall status.
func (p *ProviderConfigTestResult) Complete() {
	now := time.Now().UTC()
	p.completedAt = &now
	p.calculateOverallStatus()
	p.generateSummary()
}

// calculateOverallStatus determines the overall status based on stage results.
func (p *ProviderConfigTestResult) calculateOverallStatus() {
	passedCount := 0
	failedCount := 0
	totalCount := 0

	for _, stage := range p.stages {
		if stage.IsSkipped() {
			continue
		}

		totalCount++

		if stage.IsPassed() {
			passedCount++
		} else if stage.IsFailed() {
			failedCount++
		}
	}

	if totalCount == 0 {
		p.overallStatus = TestOverallStatusPassed

		return
	}

	if failedCount == 0 {
		p.overallStatus = TestOverallStatusPassed
	} else if passedCount == 0 {
		p.overallStatus = TestOverallStatusFailed
	} else {
		p.overallStatus = TestOverallStatusPartial
	}
}

// generateSummary generates a human-readable summary.
func (p *ProviderConfigTestResult) generateSummary() {
	switch p.overallStatus {
	case TestOverallStatusPassed:
		p.summary = "All tests passed - provider configuration connectivity verified"
	case TestOverallStatusFailed:
		p.summary = "All tests failed - provider configuration needs review"
	case TestOverallStatusPartial:
		p.summary = "Some tests failed - review failed stages before use"
	}
}

// IsPassed returns true if all tests passed.
func (p *ProviderConfigTestResult) IsPassed() bool {
	return p.overallStatus == TestOverallStatusPassed
}

// IsFailed returns true if all tests failed.
func (p *ProviderConfigTestResult) IsFailed() bool {
	return p.overallStatus == TestOverallStatusFailed
}

// IsPartial returns true if some tests passed and some failed.
func (p *ProviderConfigTestResult) IsPartial() bool {
	return p.overallStatus == TestOverallStatusPartial
}

// HasFailedStage returns true if any non-skipped stage has failed.
func (p *ProviderConfigTestResult) HasFailedStage() bool {
	for _, stage := range p.stages {
		if stage.IsFailed() {
			return true
		}
	}

	return false
}
