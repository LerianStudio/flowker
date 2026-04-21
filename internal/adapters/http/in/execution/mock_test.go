// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

// Code generated manually following mockgen pattern for execution handler interfaces.

package execution_test

import (
	"context"
	"reflect"

	"github.com/LerianStudio/flowker/internal/services/query"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// MockCommandService is a mock of CommandService interface.
type MockCommandService struct {
	ctrl     *gomock.Controller
	recorder *MockCommandServiceMockRecorder
}

// MockCommandServiceMockRecorder is the mock recorder for MockCommandService.
type MockCommandServiceMockRecorder struct {
	mock *MockCommandService
}

// NewMockCommandService creates a new mock instance.
func NewMockCommandService(ctrl *gomock.Controller) *MockCommandService {
	mock := &MockCommandService{ctrl: ctrl}
	mock.recorder = &MockCommandServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCommandService) EXPECT() *MockCommandServiceMockRecorder {
	return m.recorder
}

// Execute mocks base method.
func (m *MockCommandService) Execute(ctx context.Context, workflowID uuid.UUID, input *model.ExecuteWorkflowInput, idempotencyKey *string) (*model.WorkflowExecution, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Execute", ctx, workflowID, input, idempotencyKey)
	ret0, _ := ret[0].(*model.WorkflowExecution)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Execute indicates an expected call of Execute.
func (mr *MockCommandServiceMockRecorder) Execute(ctx, workflowID, input, idempotencyKey any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Execute", reflect.TypeOf((*MockCommandService)(nil).Execute), ctx, workflowID, input, idempotencyKey)
}

// MockQueryService is a mock of QueryService interface.
type MockQueryService struct {
	ctrl     *gomock.Controller
	recorder *MockQueryServiceMockRecorder
}

// MockQueryServiceMockRecorder is the mock recorder for MockQueryService.
type MockQueryServiceMockRecorder struct {
	mock *MockQueryService
}

// NewMockQueryService creates a new mock instance.
func NewMockQueryService(ctrl *gomock.Controller) *MockQueryService {
	mock := &MockQueryService{ctrl: ctrl}
	mock.recorder = &MockQueryServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQueryService) EXPECT() *MockQueryServiceMockRecorder {
	return m.recorder
}

// GetByID mocks base method.
func (m *MockQueryService) GetByID(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, id)
	ret0, _ := ret[0].(*model.WorkflowExecution)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockQueryServiceMockRecorder) GetByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockQueryService)(nil).GetByID), ctx, id)
}

// GetResults mocks base method.
func (m *MockQueryService) GetResults(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResults", ctx, id)
	ret0, _ := ret[0].(*model.WorkflowExecution)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetResults indicates an expected call of GetResults.
func (mr *MockQueryServiceMockRecorder) GetResults(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResults", reflect.TypeOf((*MockQueryService)(nil).GetResults), ctx, id)
}

// List mocks base method.
func (m *MockQueryService) List(ctx context.Context, filter query.ExecutionListFilter) (*query.ExecutionListResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, filter)
	ret0, _ := ret[0].(*query.ExecutionListResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockQueryServiceMockRecorder) List(ctx, filter any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockQueryService)(nil).List), ctx, filter)
}
