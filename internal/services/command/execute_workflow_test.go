// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package command

import (
	"context"
	"testing"

	"github.com/LerianStudio/flowker/pkg/circuitbreaker"
	"github.com/LerianStudio/flowker/pkg/condition"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executors"
	"github.com/LerianStudio/flowker/pkg/model"
	"github.com/LerianStudio/flowker/pkg/transformation"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestDeps(ctrl *gomock.Controller) (
	*MockExecutionRepository,
	*MockWorkflowRepository,
	*MockProviderConfigRepository,
	executor.Catalog,
	*circuitbreaker.Manager,
	*condition.Evaluator,
	*transformation.Service,
) {
	catalog := executor.NewCatalog()
	_ = executors.RegisterDefaults(catalog)

	return NewMockExecutionRepository(ctrl),
		NewMockWorkflowRepository(ctrl),
		NewMockProviderConfigRepository(ctrl),
		catalog,
		circuitbreaker.NewManager(),
		condition.NewEvaluator(),
		transformation.NewService()
}

func TestNewExecuteWorkflowCommand_NilDependencies(t *testing.T) {
	ctrl := gomock.NewController(t)
	execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc := newTestDeps(ctrl)

	tests := []struct {
		name    string
		wantErr error
		build   func() (*ExecuteWorkflowCommand, error)
	}{
		{
			name:    "nil execution repo",
			wantErr: ErrExecuteWorkflowNilRepo,
			build: func() (*ExecuteWorkflowCommand, error) {
				return NewExecuteWorkflowCommand(nil, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
			},
		},
		{
			name:    "nil workflow repo",
			wantErr: ErrExecuteWorkflowNilWorkflowRepo,
			build: func() (*ExecuteWorkflowCommand, error) {
				return NewExecuteWorkflowCommand(execRepo, nil, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
			},
		},
		{
			name:    "nil provider config repo",
			wantErr: ErrExecuteWorkflowNilProviderConfigRepo,
			build: func() (*ExecuteWorkflowCommand, error) {
				return NewExecuteWorkflowCommand(execRepo, wfRepo, nil, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
			},
		},
		{
			name:    "nil catalog",
			wantErr: ErrExecuteWorkflowNilCatalog,
			build: func() (*ExecuteWorkflowCommand, error) {
				return NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, nil, cbMgr, condEval, txSvc, newNoopAuditWriter())
			},
		},
		{
			name:    "nil circuit breaker",
			wantErr: ErrExecuteWorkflowNilCircuitBreaker,
			build: func() (*ExecuteWorkflowCommand, error) {
				return NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, nil, condEval, txSvc, newNoopAuditWriter())
			},
		},
		{
			name:    "nil condition evaluator",
			wantErr: ErrExecuteWorkflowNilCondEvaluator,
			build: func() (*ExecuteWorkflowCommand, error) {
				return NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, nil, txSvc, newNoopAuditWriter())
			},
		},
		{
			name:    "nil transform service",
			wantErr: ErrExecuteWorkflowNilTransformSvc,
			build: func() (*ExecuteWorkflowCommand, error) {
				return NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, nil, newNoopAuditWriter())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := tt.build()
			require.Nil(t, cmd)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestNewExecuteWorkflowCommand_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc := newTestDeps(ctrl)

	cmd, err := NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())

	require.NoError(t, err)
	require.NotNil(t, cmd)
}

func TestExecuteWorkflowCommand_Execute_NilInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc := newTestDeps(ctrl)

	cmd, err := NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
	require.NoError(t, err)

	key := "test-key"
	result, err := cmd.Execute(context.Background(), uuid.New(), nil, &key)

	require.ErrorIs(t, err, ErrExecuteWorkflowNilInput)
	require.Nil(t, result)
}

func TestExecuteWorkflowCommand_Execute_WorkflowNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc := newTestDeps(ctrl)

	workflowID := uuid.New()
	key := "test-wf-not-found"

	execRepo.EXPECT().
		FindByIdempotencyKey(gomock.Any(), key).
		Return(nil, constant.ErrExecutionNotFound)

	wfRepo.EXPECT().
		FindByID(gomock.Any(), workflowID).
		Return(nil, constant.ErrWorkflowNotFound)

	cmd, err := NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
	require.NoError(t, err)

	input := &model.ExecuteWorkflowInput{InputData: map[string]any{"k": "v"}}
	result, err := cmd.Execute(context.Background(), workflowID, input, &key)

	require.ErrorIs(t, err, constant.ErrWorkflowNotFound)
	require.Nil(t, result)
}

func TestExecuteWorkflowCommand_Execute_WorkflowNotActive(t *testing.T) {
	ctrl := gomock.NewController(t)
	execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc := newTestDeps(ctrl)

	// Create a draft workflow (not active)
	triggerNode, err := model.NewWorkflowNode("trigger-1", model.NodeTypeTrigger, nil, model.Position{}, map[string]any{"triggerType": "http"})
	require.NoError(t, err)

	workflow, err := model.NewWorkflow("Test", nil, []model.WorkflowNode{triggerNode}, nil)
	require.NoError(t, err)
	// workflow is draft, not active

	key := "test-wf-not-active"

	execRepo.EXPECT().
		FindByIdempotencyKey(gomock.Any(), key).
		Return(nil, constant.ErrExecutionNotFound)

	wfRepo.EXPECT().
		FindByID(gomock.Any(), workflow.ID()).
		Return(workflow, nil)

	cmd, err := NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
	require.NoError(t, err)

	input := &model.ExecuteWorkflowInput{InputData: map[string]any{"k": "v"}}
	result, err := cmd.Execute(context.Background(), workflow.ID(), input, &key)

	require.ErrorIs(t, err, constant.ErrExecutionNotActive)
	require.Nil(t, result)
}

func TestExecuteWorkflowCommand_Execute_IdempotencyKeyReturnsExisting(t *testing.T) {
	ctrl := gomock.NewController(t)
	execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc := newTestDeps(ctrl)

	key := "idem-key-1"
	existingExec := model.NewWorkflowExecution(uuid.New(), map[string]any{"k": "v"}, &key, 1)

	execRepo.EXPECT().
		FindByIdempotencyKey(gomock.Any(), key).
		Return(existingExec, nil)

	// Should NOT call FindByID since we found existing execution
	wfRepo.EXPECT().FindByID(gomock.Any(), gomock.Any()).Times(0)

	cmd, err := NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
	require.NoError(t, err)

	input := &model.ExecuteWorkflowInput{InputData: map[string]any{"k": "v"}}
	result, err := cmd.Execute(context.Background(), uuid.New(), input, &key)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, existingExec.ExecutionID(), result.ExecutionID())
}

func TestExecuteWorkflowCommand_Execute_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc := newTestDeps(ctrl)

	// Create active workflow
	triggerNode, err := model.NewWorkflowNode("trigger-1", model.NodeTypeTrigger, nil, model.Position{}, map[string]any{"triggerType": "http"})
	require.NoError(t, err)

	workflow, err := model.NewWorkflow("Test", nil, []model.WorkflowNode{triggerNode}, nil)
	require.NoError(t, err)
	require.NoError(t, workflow.Activate())

	key := "test-success-key"

	execRepo.EXPECT().
		FindByIdempotencyKey(gomock.Any(), key).
		Return(nil, constant.ErrExecutionNotFound)

	wfRepo.EXPECT().
		FindByID(gomock.Any(), workflow.ID()).
		Return(workflow, nil)

	execRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(nil)

	// MarkRunning + Update is now called synchronously before the goroutine,
	// plus the state machine runs in background so Update may be called again.
	execRepo.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).
		AnyTimes()

	cmd, err := NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
	require.NoError(t, err)

	input := &model.ExecuteWorkflowInput{InputData: map[string]any{"cpf": "123"}}
	result, err := cmd.Execute(context.Background(), workflow.ID(), input, &key)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, model.ExecutionStatusRunning, result.Status())
	assert.Equal(t, workflow.ID(), result.WorkflowID())
}

func TestExecuteWorkflowCommand_Execute_DuplicateIdempotencyKey_Resolved(t *testing.T) {
	ctrl := gomock.NewController(t)
	execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc := newTestDeps(ctrl)

	// Create active workflow
	triggerNode, err := model.NewWorkflowNode("trigger-1", model.NodeTypeTrigger, nil, model.Position{}, map[string]any{"triggerType": "http"})
	require.NoError(t, err)

	workflow, err := model.NewWorkflow("Test", nil, []model.WorkflowNode{triggerNode}, nil)
	require.NoError(t, err)
	require.NoError(t, workflow.Activate())

	key := "dup-key"
	existingExec := model.NewWorkflowExecution(workflow.ID(), map[string]any{"k": "v"}, &key, 1)

	// Fast-path: first lookup misses
	firstLookup := execRepo.EXPECT().
		FindByIdempotencyKey(gomock.Any(), key).
		Return(nil, constant.ErrExecutionNotFound)

	wfRepo.EXPECT().
		FindByID(gomock.Any(), workflow.ID()).
		Return(workflow, nil)

	// Create fails with duplicate (concurrent insert)
	execRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(constant.ErrExecutionDuplicate)

	// Insert-first fallback: second lookup finds the existing execution
	execRepo.EXPECT().
		FindByIdempotencyKey(gomock.Any(), key).
		Return(existingExec, nil).
		After(firstLookup)

	cmd, err := NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
	require.NoError(t, err)

	input := &model.ExecuteWorkflowInput{InputData: map[string]any{"k": "v"}}
	result, err := cmd.Execute(context.Background(), workflow.ID(), input, &key)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, existingExec.ExecutionID(), result.ExecutionID())
}

func TestExecuteWorkflowCommand_Execute_DuplicateIdempotencyKey_LookupFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc := newTestDeps(ctrl)

	// Create active workflow
	triggerNode, err := model.NewWorkflowNode("trigger-1", model.NodeTypeTrigger, nil, model.Position{}, map[string]any{"triggerType": "http"})
	require.NoError(t, err)

	workflow, err := model.NewWorkflow("Test", nil, []model.WorkflowNode{triggerNode}, nil)
	require.NoError(t, err)
	require.NoError(t, workflow.Activate())

	key := "dup-key"

	// Fast-path: first lookup misses
	firstLookup := execRepo.EXPECT().
		FindByIdempotencyKey(gomock.Any(), key).
		Return(nil, constant.ErrExecutionNotFound)

	wfRepo.EXPECT().
		FindByID(gomock.Any(), workflow.ID()).
		Return(workflow, nil)

	// Create fails with duplicate
	execRepo.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		Return(constant.ErrExecutionDuplicate)

	// Insert-first fallback: second lookup also fails
	execRepo.EXPECT().
		FindByIdempotencyKey(gomock.Any(), key).
		Return(nil, constant.ErrExecutionNotFound).
		After(firstLookup)

	cmd, err := NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
	require.NoError(t, err)

	input := &model.ExecuteWorkflowInput{InputData: map[string]any{"k": "v"}}
	result, err := cmd.Execute(context.Background(), workflow.ID(), input, &key)

	require.ErrorIs(t, err, constant.ErrExecutionDuplicate)
	require.Nil(t, result)
}

func TestExecuteWorkflowCommand_Execute_CanceledContext(t *testing.T) {
	ctrl := gomock.NewController(t)
	execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc := newTestDeps(ctrl)

	cmd, err := NewExecuteWorkflowCommand(execRepo, wfRepo, pcRepo, catalog, cbMgr, condEval, txSvc, newNoopAuditWriter())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	input := &model.ExecuteWorkflowInput{InputData: map[string]any{"k": "v"}}
	result, err := cmd.Execute(ctx, uuid.New(), input, nil)

	require.Error(t, err)
	require.Nil(t, result)
}

// --- Helper function tests ---

func TestCountExecutableNodes(t *testing.T) {
	trigger, _ := model.NewWorkflowNode("t1", model.NodeTypeTrigger, nil, model.Position{}, nil)
	executor, _ := model.NewWorkflowNode("e1", model.NodeTypeExecutor, nil, model.Position{}, nil)
	cond, _ := model.NewWorkflowNode("c1", model.NodeTypeConditional, nil, model.Position{}, nil)
	action, _ := model.NewWorkflowNode("a1", model.NodeTypeAction, nil, model.Position{}, nil)

	nodes := []model.WorkflowNode{trigger, executor, cond, action}
	count := countExecutableNodes(nodes)

	assert.Equal(t, 3, count) // executor + conditional + action, excludes trigger
}

func TestBuildGraph(t *testing.T) {
	e1, _ := model.NewWorkflowEdge("edge-1", "node-a", "node-b")
	e2, _ := model.NewWorkflowEdge("edge-2", "node-a", "node-c")
	e3, _ := model.NewWorkflowEdge("edge-3", "node-b", "node-d")

	graph := buildGraph([]model.WorkflowEdge{e1, e2, e3})

	assert.Len(t, graph["node-a"], 2)
	assert.Len(t, graph["node-b"], 1)
	assert.Len(t, graph["node-d"], 0)
}

func TestBuildNodeMap(t *testing.T) {
	n1, _ := model.NewWorkflowNode("node-1", model.NodeTypeTrigger, nil, model.Position{}, nil)
	n2, _ := model.NewWorkflowNode("node-2", model.NodeTypeExecutor, nil, model.Position{}, nil)

	nodeMap := buildNodeMap([]model.WorkflowNode{n1, n2})

	assert.Len(t, nodeMap, 2)
	assert.Equal(t, model.NodeTypeTrigger, nodeMap["node-1"].Type())
	assert.Equal(t, model.NodeTypeExecutor, nodeMap["node-2"].Type())
}

func TestFindTriggerNode(t *testing.T) {
	trigger, _ := model.NewWorkflowNode("trigger-1", model.NodeTypeTrigger, nil, model.Position{}, nil)
	executor, _ := model.NewWorkflowNode("exec-1", model.NodeTypeExecutor, nil, model.Position{}, nil)

	t.Run("found", func(t *testing.T) {
		id := findTriggerNode([]model.WorkflowNode{executor, trigger})
		assert.Equal(t, "trigger-1", id)
	})

	t.Run("not found", func(t *testing.T) {
		id := findTriggerNode([]model.WorkflowNode{executor})
		assert.Equal(t, "", id)
	})
}

func TestNodeDisplayName(t *testing.T) {
	t.Run("with name", func(t *testing.T) {
		name := "KYC Check"
		node, _ := model.NewWorkflowNode("node-1", model.NodeTypeExecutor, &name, model.Position{}, nil)
		assert.Equal(t, "KYC Check", nodeDisplayName(node))
	})

	t.Run("without name", func(t *testing.T) {
		node, _ := model.NewWorkflowNode("node-1", model.NodeTypeExecutor, nil, model.Position{}, nil)
		assert.Equal(t, "executor_node-1", nodeDisplayName(node))
	})

	t.Run("with empty name", func(t *testing.T) {
		empty := ""
		node, _ := model.NewWorkflowNode("node-1", model.NodeTypeExecutor, &empty, model.Position{}, nil)
		assert.Equal(t, "executor_node-1", nodeDisplayName(node))
	})
}
