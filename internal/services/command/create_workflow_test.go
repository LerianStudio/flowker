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
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewCreateWorkflowCommand_NilRepository(t *testing.T) {
	cmd, err := NewCreateWorkflowCommand(nil, nil, nil, nil, testutil.NewDefaultMockClock(), newNoopAuditWriter())

	require.Nil(t, cmd)
	require.ErrorIs(t, err, ErrCreateWorkflowNilRepo)
}

func TestNewCreateWorkflowCommand_NilClock(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockWorkflowRepository(ctrl)

	cmd, err := NewCreateWorkflowCommand(mockRepo, nil, nil, nil, nil, newNoopAuditWriter())

	require.NotNil(t, cmd)
	require.NoError(t, err)
}

func TestCreateWorkflowCommand_Execute(t *testing.T) {
	dbError := errors.New("database error")

	validInput := &model.CreateWorkflowInput{
		Name:        "test-workflow",
		Description: testutil.Ptr("Test workflow description"),
		Nodes: []model.WorkflowNodeInput{
			{
				ID:   "node-1",
				Type: "trigger",
			},
		},
	}

	emptyNodesInput := &model.CreateWorkflowInput{
		Name:        "draft-workflow",
		Description: testutil.Ptr("Draft workflow with no nodes"),
	}

	tests := []struct {
		name     string
		setup    func(ctrl *gomock.Controller) *MockWorkflowRepository
		input    *model.CreateWorkflowInput
		wantErr  error
		validate func(t *testing.T, result *model.Workflow)
	}{
		{
			name: "success",
			setup: func(ctrl *gomock.Controller) *MockWorkflowRepository {
				mockRepo := NewMockWorkflowRepository(ctrl)
				mockRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)

				return mockRepo
			},
			input: validInput,
			validate: func(t *testing.T, result *model.Workflow) {
				t.Helper()
				assert.Equal(t, "test-workflow", result.Name())
			},
		},
		{
			name: "success with empty nodes (draft)",
			setup: func(ctrl *gomock.Controller) *MockWorkflowRepository {
				mockRepo := NewMockWorkflowRepository(ctrl)
				mockRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(nil)

				return mockRepo
			},
			input: emptyNodesInput,
			validate: func(t *testing.T, result *model.Workflow) {
				t.Helper()
				assert.Equal(t, "draft-workflow", result.Name())
				assert.Equal(t, model.WorkflowStatusDraft, result.Status())
				assert.Empty(t, result.Nodes())
			},
		},
		{
			name: "nil input",
			setup: func(ctrl *gomock.Controller) *MockWorkflowRepository {
				return NewMockWorkflowRepository(ctrl)
			},
			input:   nil,
			wantErr: ErrCreateWorkflowNilInput,
		},
		{
			name: "repository error",
			setup: func(ctrl *gomock.Controller) *MockWorkflowRepository {
				mockRepo := NewMockWorkflowRepository(ctrl)
				mockRepo.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(dbError)

				return mockRepo
			},
			input:   validInput,
			wantErr: dbError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			ctx := context.Background()
			mockRepo := tt.setup(ctrl)

			cmd, err := NewCreateWorkflowCommand(mockRepo, nil, nil, nil, testutil.NewDefaultMockClock(), newNoopAuditWriter())
			require.NoError(t, err)

			result, err := cmd.Execute(ctx, tt.input)

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
