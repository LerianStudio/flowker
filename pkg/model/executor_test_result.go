// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package model

import (
	"time"

	"github.com/google/uuid"
)

// TestStageStatus represents the status of a test stage.
type TestStageStatus string

const (
	TestStageStatusPassed  TestStageStatus = "passed"
	TestStageStatusFailed  TestStageStatus = "failed"
	TestStageStatusSkipped TestStageStatus = "skipped"
)

// TestOverallStatus represents the overall status of an executor test.
type TestOverallStatus string

const (
	TestOverallStatusPassed  TestOverallStatus = "passed"
	TestOverallStatusFailed  TestOverallStatus = "failed"
	TestOverallStatusPartial TestOverallStatus = "partial"
)

// TestStageName represents the name of a test stage.
type TestStageName string

const (
	TestStageConnectivity   TestStageName = "connectivity"
	TestStageAuthentication TestStageName = "authentication"
	TestStageTransformation TestStageName = "transformation"
	TestStageEndToEnd       TestStageName = "end_to_end"
)

// StageTestResult represents the result of a single test stage.
type StageTestResult struct {
	name       TestStageName
	status     TestStageStatus
	durationMs int64
	message    string
	details    map[string]any
	err        *string
}

// NewStageTestResult creates a new StageTestResult.
func NewStageTestResult(
	name TestStageName,
	status TestStageStatus,
	durationMs int64,
	message string,
	details map[string]any,
	err *string,
) StageTestResult {
	return StageTestResult{
		name:       name,
		status:     status,
		durationMs: durationMs,
		message:    message,
		details:    cloneDetails(details),
		err:        err,
	}
}

// NewPassedStageResult creates a passed stage result.
func NewPassedStageResult(name TestStageName, durationMs int64, message string, details map[string]any) StageTestResult {
	return NewStageTestResult(name, TestStageStatusPassed, durationMs, message, details, nil)
}

// NewFailedStageResult creates a failed stage result.
func NewFailedStageResult(name TestStageName, durationMs int64, errMsg string, details map[string]any) StageTestResult {
	return NewStageTestResult(name, TestStageStatusFailed, durationMs, "", details, &errMsg)
}

// NewSkippedStageResult creates a skipped stage result.
func NewSkippedStageResult(name TestStageName, message string) StageTestResult {
	return NewStageTestResult(name, TestStageStatusSkipped, 0, message, nil, nil)
}

// Getters for StageTestResult.
func (s StageTestResult) Name() TestStageName     { return s.name }
func (s StageTestResult) Status() TestStageStatus { return s.status }
func (s StageTestResult) DurationMs() int64       { return s.durationMs }
func (s StageTestResult) Message() string         { return s.message }
func (s StageTestResult) Details() map[string]any { return cloneDetails(s.details) }
func (s StageTestResult) Error() *string          { return s.err }

// IsPassed returns true if the stage passed.
func (s StageTestResult) IsPassed() bool { return s.status == TestStageStatusPassed }

// IsFailed returns true if the stage failed.
func (s StageTestResult) IsFailed() bool { return s.status == TestStageStatusFailed }

// IsSkipped returns true if the stage was skipped.
func (s StageTestResult) IsSkipped() bool { return s.status == TestStageStatusSkipped }

// ExecutorTestResult represents the result of an executor connectivity test.
type ExecutorTestResult struct {
	executorConfigID uuid.UUID
	overallStatus    TestOverallStatus
	stages           []StageTestResult
	summary          string
	startedAt        time.Time
	completedAt      *time.Time
}

// NewExecutorTestResult creates a new ExecutorTestResult in progress.
func NewExecutorTestResult(executorConfigID uuid.UUID) *ExecutorTestResult {
	return &ExecutorTestResult{
		executorConfigID: executorConfigID,
		overallStatus:    TestOverallStatusPassed, // Will be recalculated on Complete
		stages:           make([]StageTestResult, 0, 4),
		startedAt:        time.Now().UTC(),
	}
}

// NewExecutorTestResultFromDB reconstitutes an ExecutorTestResult from database.
func NewExecutorTestResultFromDB(
	executorConfigID uuid.UUID,
	overallStatus TestOverallStatus,
	stages []StageTestResult,
	summary string,
	startedAt time.Time,
	completedAt *time.Time,
) *ExecutorTestResult {
	// Create defensive copy of stages to prevent external mutation
	stagesCopy := make([]StageTestResult, len(stages))
	copy(stagesCopy, stages)

	return &ExecutorTestResult{
		executorConfigID: executorConfigID,
		overallStatus:    overallStatus,
		stages:           stagesCopy,
		summary:          summary,
		startedAt:        startedAt,
		completedAt:      completedAt,
	}
}

// Getters for ExecutorTestResult.
func (p *ExecutorTestResult) ExecutorConfigID() uuid.UUID      { return p.executorConfigID }
func (p *ExecutorTestResult) OverallStatus() TestOverallStatus { return p.overallStatus }
func (p *ExecutorTestResult) Summary() string                  { return p.summary }
func (p *ExecutorTestResult) StartedAt() time.Time             { return p.startedAt }
func (p *ExecutorTestResult) CompletedAt() *time.Time          { return p.completedAt }

// Stages returns a copy of the stages slice.
func (p *ExecutorTestResult) Stages() []StageTestResult {
	result := make([]StageTestResult, len(p.stages))
	copy(result, p.stages)

	return result
}

// DurationMs returns the total duration in milliseconds.
func (p *ExecutorTestResult) DurationMs() int64 {
	if p.completedAt == nil {
		return time.Since(p.startedAt).Milliseconds()
	}

	return p.completedAt.Sub(p.startedAt).Milliseconds()
}

// AddStageResult adds a stage result to the test.
func (p *ExecutorTestResult) AddStageResult(stage StageTestResult) {
	p.stages = append(p.stages, stage)
}

// Complete marks the test as complete and calculates overall status.
func (p *ExecutorTestResult) Complete() {
	now := time.Now().UTC()
	p.completedAt = &now
	p.calculateOverallStatus()
	p.generateSummary()
}

// calculateOverallStatus determines the overall status based on stage results.
func (p *ExecutorTestResult) calculateOverallStatus() {
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
func (p *ExecutorTestResult) generateSummary() {
	switch p.overallStatus {
	case TestOverallStatusPassed:
		p.summary = "All tests passed - executor configuration ready for use"
	case TestOverallStatusFailed:
		p.summary = "All tests failed - executor configuration needs review"
	case TestOverallStatusPartial:
		p.summary = "Some tests failed - review failed stages before use"
	}
}

// IsPassed returns true if all tests passed.
func (p *ExecutorTestResult) IsPassed() bool {
	return p.overallStatus == TestOverallStatusPassed
}

// IsFailed returns true if all tests failed.
func (p *ExecutorTestResult) IsFailed() bool {
	return p.overallStatus == TestOverallStatusFailed
}

// IsPartial returns true if some tests passed and some failed.
func (p *ExecutorTestResult) IsPartial() bool {
	return p.overallStatus == TestOverallStatusPartial
}

// cloneDetails creates a shallow copy of the details map.
// Note: nested maps or slices are not deep-copied.
func cloneDetails(details map[string]any) map[string]any {
	if details == nil {
		return nil
	}

	result := make(map[string]any, len(details))
	for k, v := range details {
		result[k] = v
	}

	return result
}
