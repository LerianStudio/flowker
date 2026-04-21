// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package execution_test

import (
	"testing"
	"time"

	mongoExec "github.com/LerianStudio/flowker/internal/adapters/mongodb/execution"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMongoDBModel_ToEntity(t *testing.T) {
	execID := uuid.New()
	wfID := uuid.New()
	now := time.Now().UTC()
	completed := now.Add(5 * time.Second)
	errMsg := "node failed"
	key := "idem-123"

	m := &mongoExec.MongoDBModel{
		ExecutionID:       execID.String(),
		WorkflowID:        wfID.String(),
		Status:            "failed",
		InputData:         map[string]any{"cpf": "123"},
		OutputData:        map[string]any{"result": "denied"},
		ErrorMessage:      &errMsg,
		CurrentStepNumber: 2,
		TotalSteps:        3,
		Steps: []mongoExec.StepModel{
			{
				StepID:        uuid.New().String(),
				StepNumber:    1,
				StepName:      "KYC Check",
				NodeID:        "node-kyc",
				Status:        "completed",
				InputData:     map[string]any{"in": "data"},
				OutputData:    map[string]any{"out": "data"},
				StartedAt:     now,
				CompletedAt:   &completed,
				Duration:      5000,
				AttemptNumber: 1,
				ExecutorCallDetails: &mongoExec.ExecutorCallDetailsModel{
					ExecutorConfigID: "cfg-1",
					EndpointName:     "validate",
					Method:           "POST",
					URL:              "https://api.example.com/v1",
					StatusCode:       200,
					DurationMs:       150,
				},
			},
		},
		IdempotencyKey: &key,
		StartedAt:      now,
		CompletedAt:    &completed,
	}

	entity, err := m.ToEntity()
	require.NoError(t, err)
	require.NotNil(t, entity)
	assert.Equal(t, execID, entity.ExecutionID())
	assert.Equal(t, wfID, entity.WorkflowID())
	assert.Equal(t, model.ExecutionStatusFailed, entity.Status())
	assert.Equal(t, map[string]any{"cpf": "123"}, entity.InputData())
	assert.Equal(t, map[string]any{"result": "denied"}, entity.OutputData())
	require.NotNil(t, entity.ErrorMessage())
	assert.Equal(t, "node failed", *entity.ErrorMessage())
	assert.Equal(t, 2, entity.CurrentStepNumber())
	assert.Equal(t, 3, entity.TotalSteps())
	assert.Len(t, entity.Steps(), 1)
	require.NotNil(t, entity.IdempotencyKey())
	assert.Equal(t, "idem-123", *entity.IdempotencyKey())
	assert.Equal(t, now, entity.StartedAt())
	require.NotNil(t, entity.CompletedAt())

	// Verify step
	step := entity.Steps()[0]
	assert.Equal(t, 1, step.StepNumber())
	assert.Equal(t, "KYC Check", step.StepName())
	assert.Equal(t, "node-kyc", step.NodeID())
	assert.Equal(t, model.StepStatusCompleted, step.Status())
	require.NotNil(t, step.ExecutorCallDetails())
	assert.Equal(t, "cfg-1", step.ExecutorCallDetails().ExecutorConfigID())
	assert.Equal(t, 200, step.ExecutorCallDetails().StatusCode())
}

func TestMongoDBModel_FromEntity(t *testing.T) {
	wfID := uuid.New()
	key := "idem-456"
	exec := model.NewWorkflowExecution(wfID, map[string]any{"cpf": "123"}, &key, 2)
	_ = exec.MarkRunning()

	step := model.NewExecutionStep(1, "KYC", "node-kyc", map[string]any{"in": "data"})
	details := model.NewExecutorCallDetails("cfg-1", "validate", "POST", "https://api.test.com", 200, 100)
	step.SetExecutorCallDetails(details)
	_ = step.MarkCompleted(map[string]any{"out": "data"})
	exec.AddStep(step)

	_ = exec.MarkCompleted(map[string]any{"final": "ok"})

	m := &mongoExec.MongoDBModel{}
	m.FromEntity(exec)

	assert.Equal(t, exec.ExecutionID().String(), m.ExecutionID)
	assert.Equal(t, wfID.String(), m.WorkflowID)
	assert.Equal(t, "completed", m.Status)
	assert.Equal(t, map[string]any{"cpf": "123"}, m.InputData)
	assert.Equal(t, map[string]any{"final": "ok"}, m.OutputData)
	assert.Nil(t, m.ErrorMessage)
	assert.Equal(t, 1, m.CurrentStepNumber)
	assert.Equal(t, 2, m.TotalSteps)
	assert.Len(t, m.Steps, 1)
	require.NotNil(t, m.IdempotencyKey)
	assert.Equal(t, "idem-456", *m.IdempotencyKey)
	assert.NotNil(t, m.CompletedAt)

	// Verify step model
	sm := m.Steps[0]
	assert.Equal(t, step.StepID().String(), sm.StepID)
	assert.Equal(t, 1, sm.StepNumber)
	assert.Equal(t, "KYC", sm.StepName)
	assert.Equal(t, "node-kyc", sm.NodeID)
	assert.Equal(t, "completed", sm.Status)
	require.NotNil(t, sm.ExecutorCallDetails)
	assert.Equal(t, "cfg-1", sm.ExecutorCallDetails.ExecutorConfigID)
	assert.Equal(t, 200, sm.ExecutorCallDetails.StatusCode)
}

func TestMongoDBModel_RoundTrip(t *testing.T) {
	wfID := uuid.New()
	key := "round-trip-key"
	original := model.NewWorkflowExecution(wfID, map[string]any{"amount": float64(100)}, &key, 1)
	_ = original.MarkRunning()

	step := model.NewExecutionStep(1, "Fraud Check", "node-fraud", map[string]any{"tx": "data"})
	_ = step.MarkCompleted(map[string]any{"risk": "low"})
	original.AddStep(step)
	_ = original.MarkCompleted(map[string]any{"approved": true})

	// Entity -> MongoDB model -> Entity
	m := mongoExec.NewMongoDBModelFromEntity(original)
	restored, err := m.ToEntity()
	require.NoError(t, err)

	assert.Equal(t, original.ExecutionID(), restored.ExecutionID())
	assert.Equal(t, original.WorkflowID(), restored.WorkflowID())
	assert.Equal(t, original.Status(), restored.Status())
	assert.Equal(t, original.InputData(), restored.InputData())
	assert.Equal(t, original.OutputData(), restored.OutputData())
	assert.Equal(t, original.CurrentStepNumber(), restored.CurrentStepNumber())
	assert.Equal(t, original.TotalSteps(), restored.TotalSteps())
	assert.Len(t, restored.Steps(), 1)
	require.NotNil(t, restored.IdempotencyKey())
	assert.Equal(t, "round-trip-key", *restored.IdempotencyKey())
}

func TestNewMongoDBModelFromEntity(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 0)

	m := mongoExec.NewMongoDBModelFromEntity(exec)

	require.NotNil(t, m)
	assert.Equal(t, exec.ExecutionID().String(), m.ExecutionID)
}

func TestStepModel_ToEntity_WithoutCallDetails(t *testing.T) {
	stepID := uuid.New()
	now := time.Now().UTC()

	sm := &mongoExec.StepModel{
		StepID:        stepID.String(),
		StepNumber:    1,
		StepName:      "Evaluate",
		NodeID:        "node-cond",
		Status:        "completed",
		StartedAt:     now,
		AttemptNumber: 1,
	}

	entity, err := sm.ToEntity()
	require.NoError(t, err)

	assert.Equal(t, stepID, entity.StepID())
	assert.Equal(t, "Evaluate", entity.StepName())
	assert.Nil(t, entity.ExecutorCallDetails())
}

func TestStepModelFromEntity_WithFailedStep(t *testing.T) {
	step := model.NewExecutionStep(1, "Failed Step", "node-fail", nil)
	_ = step.MarkFailed("timeout exceeded")

	sm := mongoExec.StepModelFromEntity(step)

	assert.Equal(t, "failed", sm.Status)
	require.NotNil(t, sm.ErrorMessage)
	assert.Equal(t, "timeout exceeded", *sm.ErrorMessage)
	assert.Nil(t, sm.ExecutorCallDetails)
}

func TestMongoDBModel_ToEntity_InvalidExecutionID(t *testing.T) {
	m := &mongoExec.MongoDBModel{
		ExecutionID: "not-a-uuid",
		WorkflowID:  uuid.New().String(),
		Status:      "pending",
		StartedAt:   time.Now().UTC(),
	}

	_, err := m.ToEntity()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid execution ID")
}

func TestMongoDBModel_ToEntity_InvalidWorkflowID(t *testing.T) {
	m := &mongoExec.MongoDBModel{
		ExecutionID: uuid.New().String(),
		WorkflowID:  "not-a-uuid",
		Status:      "pending",
		StartedAt:   time.Now().UTC(),
	}

	_, err := m.ToEntity()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid workflow ID")
}

func TestStepModel_ToEntity_InvalidStepID(t *testing.T) {
	sm := &mongoExec.StepModel{
		StepID:     "not-a-uuid",
		StepNumber: 1,
		StepName:   "Bad Step",
		NodeID:     "node-1",
		Status:     "completed",
		StartedAt:  time.Now().UTC(),
	}

	_, err := sm.ToEntity()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid step ID")
}

func TestMongoDBModel_FromEntity_NoSteps(t *testing.T) {
	exec := model.NewWorkflowExecution(uuid.New(), nil, nil, 0)

	m := &mongoExec.MongoDBModel{}
	m.FromEntity(exec)

	assert.Empty(t, m.Steps)
}
