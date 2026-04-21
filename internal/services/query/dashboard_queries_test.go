// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package query

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// MockDashboardRepository is a mock of DashboardRepository interface.
type MockDashboardRepository struct {
	ctrl     *gomock.Controller
	recorder *MockDashboardRepositoryMockRecorder
}

// MockDashboardRepositoryMockRecorder is the mock recorder for MockDashboardRepository.
type MockDashboardRepositoryMockRecorder struct {
	mock *MockDashboardRepository
}

// NewMockDashboardRepository creates a new mock instance.
func NewMockDashboardRepository(ctrl *gomock.Controller) *MockDashboardRepository {
	mock := &MockDashboardRepository{ctrl: ctrl}
	mock.recorder = &MockDashboardRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDashboardRepository) EXPECT() *MockDashboardRepositoryMockRecorder {
	return m.recorder
}

// WorkflowSummary mocks base method.
func (m *MockDashboardRepository) WorkflowSummary(ctx context.Context) (*model.WorkflowSummaryOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WorkflowSummary", ctx)
	ret0, _ := ret[0].(*model.WorkflowSummaryOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WorkflowSummary indicates an expected call of WorkflowSummary.
func (mr *MockDashboardRepositoryMockRecorder) WorkflowSummary(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WorkflowSummary", reflect.TypeOf((*MockDashboardRepository)(nil).WorkflowSummary), ctx)
}

// ExecutionSummary mocks base method.
func (m *MockDashboardRepository) ExecutionSummary(ctx context.Context, filter ExecutionSummaryFilter) (*model.ExecutionSummaryOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecutionSummary", ctx, filter)
	ret0, _ := ret[0].(*model.ExecutionSummaryOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecutionSummary indicates an expected call of ExecutionSummary.
func (mr *MockDashboardRepositoryMockRecorder) ExecutionSummary(ctx, filter any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecutionSummary", reflect.TypeOf((*MockDashboardRepository)(nil).ExecutionSummary), ctx, filter)
}

// --- GetWorkflowSummaryQuery Tests ---

func TestNewGetWorkflowSummaryQuery_NilRepo(t *testing.T) {
	q, err := NewGetWorkflowSummaryQuery(nil)

	require.Nil(t, q)
	require.ErrorIs(t, err, ErrDashboardNilRepo)
}

func TestGetWorkflowSummaryQuery_NilContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockDashboardRepository(ctrl)

	q, err := NewGetWorkflowSummaryQuery(mockRepo)
	require.NoError(t, err)

	//nolint:staticcheck // testing nil context behavior
	result, err := q.Execute(nil)
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "context cannot be nil")
}

func TestGetWorkflowSummaryQuery_CanceledContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockDashboardRepository(ctrl)

	q, err := NewGetWorkflowSummaryQuery(mockRepo)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := q.Execute(ctx)

	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, result)
}

func TestGetWorkflowSummaryQuery_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockDashboardRepository(ctrl)

	expected := &model.WorkflowSummaryOutput{
		Total:  10,
		Active: 7,
		ByStatus: []model.StatusCountOutput{
			{Status: "active", Count: 7},
			{Status: "draft", Count: 3},
		},
	}

	mockRepo.EXPECT().
		WorkflowSummary(gomock.Any()).
		Return(expected, nil)

	q, err := NewGetWorkflowSummaryQuery(mockRepo)
	require.NoError(t, err)

	result, err := q.Execute(context.Background())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(10), result.Total)
	assert.Equal(t, int64(7), result.Active)
	assert.Len(t, result.ByStatus, 2)
}

func TestGetWorkflowSummaryQuery_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockDashboardRepository(ctrl)

	repoErr := errors.New("aggregation pipeline failed")

	mockRepo.EXPECT().
		WorkflowSummary(gomock.Any()).
		Return(nil, repoErr)

	q, err := NewGetWorkflowSummaryQuery(mockRepo)
	require.NoError(t, err)

	result, err := q.Execute(context.Background())

	require.ErrorIs(t, err, repoErr)
	require.Nil(t, result)
}

// --- GetExecutionSummaryQuery Tests ---

func TestNewGetExecutionSummaryQuery_NilRepo(t *testing.T) {
	q, err := NewGetExecutionSummaryQuery(nil)

	require.Nil(t, q)
	require.ErrorIs(t, err, ErrDashboardNilRepo)
}

func TestGetExecutionSummaryQuery_NilContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockDashboardRepository(ctrl)

	q, err := NewGetExecutionSummaryQuery(mockRepo)
	require.NoError(t, err)

	//nolint:staticcheck // testing nil context behavior
	result, err := q.Execute(nil, ExecutionSummaryFilter{})
	require.Error(t, err)
	require.Nil(t, result)
	assert.Contains(t, err.Error(), "context cannot be nil")
}

func TestGetExecutionSummaryQuery_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockDashboardRepository(ctrl)

	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	status := "completed"

	filter := ExecutionSummaryFilter{
		StartTime: &startTime,
		Status:    &status,
	}

	expected := &model.ExecutionSummaryOutput{
		Total:     5,
		Completed: 5,
	}

	mockRepo.EXPECT().
		ExecutionSummary(gomock.Any(), filter).
		Return(expected, nil)

	q, err := NewGetExecutionSummaryQuery(mockRepo)
	require.NoError(t, err)

	result, err := q.Execute(context.Background(), filter)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, int64(5), result.Total)
	assert.Equal(t, int64(5), result.Completed)
}

func TestGetExecutionSummaryQuery_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRepo := NewMockDashboardRepository(ctrl)

	repoErr := errors.New("database connection lost")

	mockRepo.EXPECT().
		ExecutionSummary(gomock.Any(), ExecutionSummaryFilter{}).
		Return(nil, repoErr)

	q, err := NewGetExecutionSummaryQuery(mockRepo)
	require.NoError(t, err)

	result, err := q.Execute(context.Background(), ExecutionSummaryFilter{})

	require.ErrorIs(t, err, repoErr)
	require.Nil(t, result)
}
