// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

// Code generated manually following mockgen pattern for execution repository interface.

package command

import (
	"context"
	"reflect"

	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

// MockExecutionRepository is a mock of ExecutionRepository interface.
type MockExecutionRepository struct {
	ctrl     *gomock.Controller
	recorder *MockExecutionRepositoryMockRecorder
}

// MockExecutionRepositoryMockRecorder is the mock recorder for MockExecutionRepository.
type MockExecutionRepositoryMockRecorder struct {
	mock *MockExecutionRepository
}

// NewMockExecutionRepository creates a new mock instance.
func NewMockExecutionRepository(ctrl *gomock.Controller) *MockExecutionRepository {
	mock := &MockExecutionRepository{ctrl: ctrl}
	mock.recorder = &MockExecutionRepositoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockExecutionRepository) EXPECT() *MockExecutionRepositoryMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockExecutionRepository) Create(ctx context.Context, execution *model.WorkflowExecution) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, execution)
	ret0, _ := ret[0].(error)
	return ret0
}

// Create indicates an expected call of Create.
func (mr *MockExecutionRepositoryMockRecorder) Create(ctx, execution any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockExecutionRepository)(nil).Create), ctx, execution)
}

// FindByID mocks base method.
func (m *MockExecutionRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.WorkflowExecution, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByID", ctx, id)
	ret0, _ := ret[0].(*model.WorkflowExecution)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByID indicates an expected call of FindByID.
func (mr *MockExecutionRepositoryMockRecorder) FindByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByID", reflect.TypeOf((*MockExecutionRepository)(nil).FindByID), ctx, id)
}

// FindByIdempotencyKey mocks base method.
func (m *MockExecutionRepository) FindByIdempotencyKey(ctx context.Context, key string) (*model.WorkflowExecution, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindByIdempotencyKey", ctx, key)
	ret0, _ := ret[0].(*model.WorkflowExecution)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByIdempotencyKey indicates an expected call of FindByIdempotencyKey.
func (mr *MockExecutionRepositoryMockRecorder) FindByIdempotencyKey(ctx, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByIdempotencyKey", reflect.TypeOf((*MockExecutionRepository)(nil).FindByIdempotencyKey), ctx, key)
}

// Update mocks base method.
func (m *MockExecutionRepository) Update(ctx context.Context, execution *model.WorkflowExecution, expectedStatus model.ExecutionStatus) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, execution, expectedStatus)
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update.
func (mr *MockExecutionRepositoryMockRecorder) Update(ctx, execution, expectedStatus any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockExecutionRepository)(nil).Update), ctx, execution, expectedStatus)
}

// FindIncomplete mocks base method.
func (m *MockExecutionRepository) FindIncomplete(ctx context.Context) ([]*model.WorkflowExecution, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindIncomplete", ctx)
	ret0, _ := ret[0].([]*model.WorkflowExecution)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindIncomplete indicates an expected call of FindIncomplete.
func (mr *MockExecutionRepositoryMockRecorder) FindIncomplete(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindIncomplete", reflect.TypeOf((*MockExecutionRepository)(nil).FindIncomplete), ctx)
}

// List mocks base method.
func (m *MockExecutionRepository) List(ctx context.Context, filter ExecutionListFilter) (*ExecutionListResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, filter)
	ret0, _ := ret[0].(*ExecutionListResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockExecutionRepositoryMockRecorder) List(ctx, filter any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockExecutionRepository)(nil).List), ctx, filter)
}
