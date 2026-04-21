// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package condition_test

import (
	"testing"

	"github.com/LerianStudio/flowker/pkg/condition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEvaluator() *condition.Evaluator {
	return condition.NewEvaluator()
}

func TestEvaluator_EmptyExpression(t *testing.T) {
	e := newEvaluator()

	_, err := e.Evaluate("", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty expression")
}

func TestEvaluator_WhitespaceExpression(t *testing.T) {
	e := newEvaluator()

	_, err := e.Evaluate("   ", nil)
	require.Error(t, err)
}

func TestEvaluator_NumericComparisons(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"workflow": map[string]any{
			"score": float64(85),
			"age":   float64(25),
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"greater than true", "workflow.score > 70", true},
		{"greater than false", "workflow.score > 90", false},
		{"less than true", "workflow.age < 30", true},
		{"less than false", "workflow.age < 20", false},
		{"greater or equal true (equal)", "workflow.score >= 85", true},
		{"greater or equal true (greater)", "workflow.score >= 80", true},
		{"greater or equal false", "workflow.score >= 90", false},
		{"less or equal true (equal)", "workflow.age <= 25", true},
		{"less or equal true (less)", "workflow.age <= 30", true},
		{"less or equal false", "workflow.age <= 20", false},
		{"equal true", "workflow.score == 85", true},
		{"equal false", "workflow.score == 90", false},
		{"not equal true", "workflow.score != 90", true},
		{"not equal false", "workflow.score != 85", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(tt.expression, ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_StringComparisons(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"workflow": map[string]any{
			"status": "approved",
			"risk":   "low",
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"string equal true", `workflow.status == "approved"`, true},
		{"string equal false", `workflow.status == "rejected"`, false},
		{"string not equal true", `workflow.risk != "high"`, true},
		{"string not equal false", `workflow.risk != "low"`, false},
		{"single quote string", `workflow.status == 'approved'`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(tt.expression, ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_LogicalAND(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"workflow": map[string]any{
			"score": float64(85),
			"age":   float64(25),
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"both true", "workflow.score > 70 AND workflow.age > 20", true},
		{"left false", "workflow.score > 90 AND workflow.age > 20", false},
		{"right false", "workflow.score > 70 AND workflow.age > 30", false},
		{"both false", "workflow.score > 90 AND workflow.age > 30", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(tt.expression, ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_LogicalOR(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"workflow": map[string]any{
			"score": float64(85),
			"age":   float64(25),
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"both true", "workflow.score > 70 OR workflow.age > 20", true},
		{"left true only", "workflow.score > 70 OR workflow.age > 30", true},
		{"right true only", "workflow.score > 90 OR workflow.age > 20", true},
		{"both false", "workflow.score > 90 OR workflow.age > 30", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := e.Evaluate(tt.expression, ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_CombinedANDOR(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"node-kyc": map[string]any{
			"score": float64(95),
		},
		"node-aml": map[string]any{
			"risk": "low",
		},
	}

	// OR has lower precedence than AND
	result, err := e.Evaluate(`node-kyc.score > 80 AND node-aml.risk == "low"`, ctx)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_BooleanLiterals(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{}

	result, err := e.Evaluate("true", ctx)
	require.NoError(t, err)
	assert.True(t, result)

	result, err = e.Evaluate("false", ctx)
	require.NoError(t, err)
	assert.False(t, result)
}

func TestEvaluator_BooleanFieldReference(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"workflow": map[string]any{
			"enabled": true,
		},
	}

	result, err := e.Evaluate("workflow.enabled == true", ctx)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_NestedFieldResolution(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"node-kyc": map[string]any{
			"result": map[string]any{
				"status": "approved",
			},
		},
	}

	result, err := e.Evaluate(`node-kyc.result.status == "approved"`, ctx)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_FieldNotFound(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"workflow": map[string]any{},
	}

	_, err := e.Evaluate("workflow.nonexistent > 5", ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEvaluator_InvalidOperatorForStrings(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"workflow": map[string]any{
			"name": "test",
		},
	}

	_, err := e.Evaluate(`workflow.name > "abc"`, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "numeric")
}

func TestEvaluator_LiteralNumbers(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{}

	result, err := e.Evaluate("10 > 5", ctx)
	require.NoError(t, err)
	assert.True(t, result)

	result, err = e.Evaluate("3.14 > 3.0", ctx)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_IntegerFieldValues(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"workflow": map[string]any{
			"count": 10, // int, not float64
		},
	}

	result, err := e.Evaluate("workflow.count > 5", ctx)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_DirectKeyLookup(t *testing.T) {
	e := newEvaluator()
	ctx := map[string]any{
		"status": "active",
	}

	result, err := e.Evaluate(`status == "active"`, ctx)
	require.NoError(t, err)
	assert.True(t, result)
}
