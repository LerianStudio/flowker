// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

// Code generated manually following mockgen pattern for dashboard handler interfaces.

package dashboard_test

import (
	"context"
	"reflect"
	"time"

	"github.com/LerianStudio/flowker/pkg/model"
	"go.uber.org/mock/gomock"
)

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

// WorkflowSummary mocks base method.
func (m *MockQueryService) WorkflowSummary(ctx context.Context) (*model.WorkflowSummaryOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WorkflowSummary", ctx)
	ret0, _ := ret[0].(*model.WorkflowSummaryOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// WorkflowSummary indicates an expected call of WorkflowSummary.
func (mr *MockQueryServiceMockRecorder) WorkflowSummary(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WorkflowSummary", reflect.TypeOf((*MockQueryService)(nil).WorkflowSummary), ctx)
}

// ExecutionSummary mocks base method.
func (m *MockQueryService) ExecutionSummary(ctx context.Context, startTime, endTime *time.Time, status *string) (*model.ExecutionSummaryOutput, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ExecutionSummary", ctx, startTime, endTime, status)
	ret0, _ := ret[0].(*model.ExecutionSummaryOutput)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ExecutionSummary indicates an expected call of ExecutionSummary.
func (mr *MockQueryServiceMockRecorder) ExecutionSummary(ctx, startTime, endTime, status any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ExecutionSummary", reflect.TypeOf((*MockQueryService)(nil).ExecutionSummary), ctx, startTime, endTime, status)
}
