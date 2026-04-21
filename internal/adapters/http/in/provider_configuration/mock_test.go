// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

// Code generated manually following mockgen pattern for provider configuration handler interfaces.

package providerconfiguration_test

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

// Create mocks base method.
func (m *MockCommandService) Create(ctx context.Context, input *model.CreateProviderConfigurationInput) (*model.ProviderConfiguration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, input)
	ret0, _ := ret[0].(*model.ProviderConfiguration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockCommandServiceMockRecorder) Create(ctx, input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockCommandService)(nil).Create), ctx, input)
}

// Update mocks base method.
func (m *MockCommandService) Update(ctx context.Context, id uuid.UUID, input *model.UpdateProviderConfigurationInput) (*model.ProviderConfiguration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", ctx, id, input)
	ret0, _ := ret[0].(*model.ProviderConfiguration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockCommandServiceMockRecorder) Update(ctx, id, input any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockCommandService)(nil).Update), ctx, id, input)
}

// Delete mocks base method.
func (m *MockCommandService) Delete(ctx context.Context, id uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockCommandServiceMockRecorder) Delete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockCommandService)(nil).Delete), ctx, id)
}

// Disable mocks base method.
func (m *MockCommandService) Disable(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Disable", ctx, id)
	ret0, _ := ret[0].(*model.ProviderConfiguration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Disable indicates an expected call of Disable.
func (mr *MockCommandServiceMockRecorder) Disable(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Disable", reflect.TypeOf((*MockCommandService)(nil).Disable), ctx, id)
}

// Enable mocks base method.
func (m *MockCommandService) Enable(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Enable", ctx, id)
	ret0, _ := ret[0].(*model.ProviderConfiguration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Enable indicates an expected call of Enable.
func (mr *MockCommandServiceMockRecorder) Enable(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enable", reflect.TypeOf((*MockCommandService)(nil).Enable), ctx, id)
}

// TestConnectivity mocks base method.
func (m *MockCommandService) TestConnectivity(ctx context.Context, id uuid.UUID) (*model.ProviderConfigTestResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TestConnectivity", ctx, id)
	ret0, _ := ret[0].(*model.ProviderConfigTestResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TestConnectivity indicates an expected call of TestConnectivity.
func (mr *MockCommandServiceMockRecorder) TestConnectivity(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TestConnectivity", reflect.TypeOf((*MockCommandService)(nil).TestConnectivity), ctx, id)
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
func (m *MockQueryService) GetByID(ctx context.Context, id uuid.UUID) (*model.ProviderConfiguration, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByID", ctx, id)
	ret0, _ := ret[0].(*model.ProviderConfiguration)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByID indicates an expected call of GetByID.
func (mr *MockQueryServiceMockRecorder) GetByID(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByID", reflect.TypeOf((*MockQueryService)(nil).GetByID), ctx, id)
}

// List mocks base method.
func (m *MockQueryService) List(ctx context.Context, filter query.ProviderConfigListFilter) (*query.ProviderConfigListResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, filter)
	ret0, _ := ret[0].(*query.ProviderConfigListResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockQueryServiceMockRecorder) List(ctx, filter any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockQueryService)(nil).List), ctx, filter)
}
