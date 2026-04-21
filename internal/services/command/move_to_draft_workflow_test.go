// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package command

import (
	"context"
	"errors"
	"testing"

	"github.com/LerianStudio/flowker/internal/testutil"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewMoveToDraftWorkflowCommand_NilRepository(t *testing.T) {
	cmd, err := NewMoveToDraftWorkflowCommand(nil, testutil.NewDefaultMockClock(), newNoopAuditWriter())

	require.Nil(t, cmd)
	require.ErrorIs(t, err, ErrMoveToDraftWorkflowNilRepo)
}

func TestNewMoveToDraftWorkflowCommand_NilClock(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockWorkflowRepository(ctrl)

	cmd, err := NewMoveToDraftWorkflowCommand(mockRepo, nil, newNoopAuditWriter())

	require.NotNil(t, cmd)
	require.NoError(t, err)
}

func TestMoveToDraftWorkflowCommand_Execute(t *testing.T) {
	dbError := errors.New("database error")

	tests := []struct {
		name       string
		workflowID uuid.UUID
		setup      func(ctrl *gomock.Controller, workflowID uuid.UUID) *MockWorkflowRepository
		wantErr    error
		validate   func(t *testing.T, result *model.Workflow)
	}{
		{
			name:       "success",
			workflowID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			setup: func(ctrl *gomock.Controller, workflowID uuid.UUID) *MockWorkflowRepository {
				mockRepo := NewMockWorkflowRepository(ctrl)

				node, _ := model.NewWorkflowNode("node-1", model.NodeTypeTrigger, nil, model.Position{}, nil)
				workflow, _ := model.NewWorkflow(
					"test-workflow",
					testutil.Ptr("Test description"),
					[]model.WorkflowNode{node},
					nil,
				)
				_ = workflow.Activate()
				_ = workflow.Deactivate()

				mockRepo.EXPECT().
					FindByID(gomock.Any(), workflowID).
					Return(workflow, nil)

				mockRepo.EXPECT().
					Update(gomock.Any(), gomock.Any(), gomock.Eq(model.WorkflowStatusInactive)).
					Return(nil)

				return mockRepo
			},
			validate: func(t *testing.T, result *model.Workflow) {
				t.Helper()
				assert.Equal(t, model.WorkflowStatusDraft, result.Status())
			},
		},
		{
			name:       "workflow not found",
			workflowID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			setup: func(ctrl *gomock.Controller, workflowID uuid.UUID) *MockWorkflowRepository {
				mockRepo := NewMockWorkflowRepository(ctrl)

				mockRepo.EXPECT().
					FindByID(gomock.Any(), workflowID).
					Return(nil, constant.ErrWorkflowNotFound)

				return mockRepo
			},
			wantErr: constant.ErrWorkflowNotFound,
		},
		{
			name:       "not inactive",
			workflowID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			setup: func(ctrl *gomock.Controller, workflowID uuid.UUID) *MockWorkflowRepository {
				mockRepo := NewMockWorkflowRepository(ctrl)

				node, _ := model.NewWorkflowNode("node-1", model.NodeTypeTrigger, nil, model.Position{}, nil)
				workflow, _ := model.NewWorkflow(
					"test-workflow",
					testutil.Ptr("Test description"),
					[]model.WorkflowNode{node},
					nil,
				)
				_ = workflow.Activate()

				mockRepo.EXPECT().
					FindByID(gomock.Any(), workflowID).
					Return(workflow, nil)

				return mockRepo
			},
			wantErr: constant.ErrWorkflowInvalidStatus,
		},
		{
			name:       "update error",
			workflowID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			setup: func(ctrl *gomock.Controller, workflowID uuid.UUID) *MockWorkflowRepository {
				mockRepo := NewMockWorkflowRepository(ctrl)

				node, _ := model.NewWorkflowNode("node-1", model.NodeTypeTrigger, nil, model.Position{}, nil)
				workflow, _ := model.NewWorkflow(
					"test-workflow",
					testutil.Ptr("Test description"),
					[]model.WorkflowNode{node},
					nil,
				)
				_ = workflow.Activate()
				_ = workflow.Deactivate()

				mockRepo.EXPECT().
					FindByID(gomock.Any(), workflowID).
					Return(workflow, nil)

				mockRepo.EXPECT().
					Update(gomock.Any(), gomock.Any(), gomock.Eq(model.WorkflowStatusInactive)).
					Return(dbError)

				return mockRepo
			},
			wantErr: dbError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx := context.Background()
			mockRepo := tt.setup(ctrl, tt.workflowID)

			cmd, err := NewMoveToDraftWorkflowCommand(mockRepo, testutil.NewDefaultMockClock(), newNoopAuditWriter())
			require.NoError(t, err)

			result, err := cmd.Execute(ctx, tt.workflowID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Nil(t, result)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
