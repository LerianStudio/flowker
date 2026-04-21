// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package execution contains the MongoDB adapter for workflow execution persistence.
package execution

import (
	"fmt"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MongoDBModel is the MongoDB document model for workflow executions.
type MongoDBModel struct {
	ObjectID          primitive.ObjectID `bson:"_id,omitempty"`
	ExecutionID       string             `bson:"executionId"`
	WorkflowID        string             `bson:"workflowId"`
	Status            string             `bson:"status"`
	InputData         map[string]any     `bson:"inputData,omitempty"`
	OutputData        map[string]any     `bson:"outputData,omitempty"`
	ErrorMessage      *string            `bson:"errorMessage,omitempty"`
	CurrentStepNumber int                `bson:"currentStepNumber"`
	TotalSteps        int                `bson:"totalSteps"`
	Steps             []StepModel        `bson:"steps,omitempty"`
	IdempotencyKey    *string            `bson:"idempotencyKey,omitempty"`
	StartedAt         time.Time          `bson:"startedAt"`
	CompletedAt       *time.Time         `bson:"completedAt,omitempty"`
}

// StepModel is the MongoDB model for execution steps.
type StepModel struct {
	StepID              string                    `bson:"stepId"`
	StepNumber          int                       `bson:"stepNumber"`
	StepName            string                    `bson:"stepName"`
	NodeID              string                    `bson:"nodeId"`
	Status              string                    `bson:"status"`
	InputData           map[string]any            `bson:"inputData,omitempty"`
	OutputData          map[string]any            `bson:"outputData,omitempty"`
	ErrorMessage        *string                   `bson:"errorMessage,omitempty"`
	ExecutorCallDetails *ExecutorCallDetailsModel `bson:"executorCallDetails,omitempty"`
	StartedAt           time.Time                 `bson:"startedAt"`
	CompletedAt         *time.Time                `bson:"completedAt,omitempty"`
	Duration            int64                     `bson:"duration"`
	AttemptNumber       int                       `bson:"attemptNumber"`
}

// ExecutorCallDetailsModel is the MongoDB model for executor call details.
type ExecutorCallDetailsModel struct {
	ExecutorConfigID string `bson:"executorConfigId"`
	EndpointName     string `bson:"endpointName"`
	Method           string `bson:"method"`
	URL              string `bson:"url"`
	StatusCode       int    `bson:"statusCode"`
	DurationMs       int64  `bson:"durationMs"`
}

// ToEntity converts MongoDBModel to domain entity.
func (m *MongoDBModel) ToEntity() (*model.WorkflowExecution, error) {
	executionID, err := uuid.Parse(m.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution ID %q: %w", m.ExecutionID, err)
	}

	workflowID, err := uuid.Parse(m.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("invalid workflow ID %q: %w", m.WorkflowID, err)
	}

	steps := make([]model.ExecutionStep, len(m.Steps))

	for i, stepModel := range m.Steps {
		step, stepErr := stepModel.ToEntity()
		if stepErr != nil {
			return nil, fmt.Errorf("invalid step at index %d: %w", i, stepErr)
		}

		steps[i] = step
	}

	return model.NewWorkflowExecutionFromDB(
		executionID,
		workflowID,
		model.ExecutionStatus(m.Status),
		m.InputData,
		m.OutputData,
		m.ErrorMessage,
		m.CurrentStepNumber,
		m.TotalSteps,
		steps,
		m.IdempotencyKey,
		m.StartedAt,
		m.CompletedAt,
	), nil
}

// FromEntity populates MongoDBModel from domain entity.
func (m *MongoDBModel) FromEntity(e *model.WorkflowExecution) {
	m.ExecutionID = e.ExecutionID().String()
	m.WorkflowID = e.WorkflowID().String()
	m.Status = string(e.Status())
	m.InputData = e.InputData()
	m.OutputData = e.OutputData()
	m.ErrorMessage = e.ErrorMessage()
	m.CurrentStepNumber = e.CurrentStepNumber()
	m.TotalSteps = e.TotalSteps()
	m.IdempotencyKey = e.IdempotencyKey()
	m.StartedAt = e.StartedAt()
	m.CompletedAt = e.CompletedAt()

	domainSteps := e.Steps()
	m.Steps = make([]StepModel, len(domainSteps))

	for i, step := range domainSteps {
		m.Steps[i] = StepModelFromEntity(step)
	}
}

// ToEntity converts StepModel to domain entity.
func (s *StepModel) ToEntity() (model.ExecutionStep, error) {
	stepID, err := uuid.Parse(s.StepID)
	if err != nil {
		return model.ExecutionStep{}, fmt.Errorf("invalid step ID %q: %w", s.StepID, err)
	}

	var callDetails *model.ExecutorCallDetails

	if s.ExecutorCallDetails != nil {
		d := model.NewExecutorCallDetails(
			s.ExecutorCallDetails.ExecutorConfigID,
			s.ExecutorCallDetails.EndpointName,
			s.ExecutorCallDetails.Method,
			s.ExecutorCallDetails.URL,
			s.ExecutorCallDetails.StatusCode,
			s.ExecutorCallDetails.DurationMs,
		)
		callDetails = &d
	}

	return model.NewExecutionStepFromDB(
		stepID,
		s.StepNumber,
		s.StepName,
		s.NodeID,
		model.StepStatus(s.Status),
		s.InputData,
		s.OutputData,
		s.ErrorMessage,
		callDetails,
		s.StartedAt,
		s.CompletedAt,
		s.Duration,
		s.AttemptNumber,
	), nil
}

// StepModelFromEntity creates a StepModel from domain entity.
func StepModelFromEntity(s model.ExecutionStep) StepModel {
	sm := StepModel{
		StepID:        s.StepID().String(),
		StepNumber:    s.StepNumber(),
		StepName:      s.StepName(),
		NodeID:        s.NodeID(),
		Status:        string(s.Status()),
		InputData:     s.InputData(),
		OutputData:    s.OutputData(),
		ErrorMessage:  s.ErrorMessage(),
		StartedAt:     s.StartedAt(),
		CompletedAt:   s.CompletedAt(),
		Duration:      s.Duration(),
		AttemptNumber: s.AttemptNumber(),
	}

	if details := s.ExecutorCallDetails(); details != nil {
		sm.ExecutorCallDetails = &ExecutorCallDetailsModel{
			ExecutorConfigID: details.ExecutorConfigID(),
			EndpointName:     details.EndpointName(),
			Method:           details.Method(),
			URL:              details.URL(),
			StatusCode:       details.StatusCode(),
			DurationMs:       details.DurationMs(),
		}
	}

	return sm
}

// NewMongoDBModelFromEntity creates a new MongoDBModel from domain entity.
func NewMongoDBModelFromEntity(e *model.WorkflowExecution) *MongoDBModel {
	m := &MongoDBModel{}
	m.FromEntity(e)

	return m
}
