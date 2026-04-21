// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package condition provides expression evaluation for conditional workflow nodes.
package condition

import (
	"fmt"
	"strconv"
	"strings"
)

// Evaluator evaluates conditional expressions against a workflow context.
type Evaluator struct{}

// NewEvaluator creates a new condition Evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate evaluates a conditional expression against the given context.
// Supports comparison operators (>, <, >=, <=, ==, !=) and logical operators (AND, OR).
// Field references use dot notation (e.g., "workflow.field_name").
func (e *Evaluator) Evaluate(expression string, ctx map[string]any) (bool, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return false, fmt.Errorf("empty expression")
	}

	return e.evaluateOr(expression, ctx)
}

// evaluateOr splits on OR and evaluates each part.
func (e *Evaluator) evaluateOr(expr string, ctx map[string]any) (bool, error) {
	parts := splitLogical(expr, "OR")

	for _, part := range parts {
		result, err := e.evaluateAnd(part, ctx)
		if err != nil {
			return false, err
		}

		if result {
			return true, nil
		}
	}

	return false, nil
}

// evaluateAnd splits on AND and evaluates each part.
func (e *Evaluator) evaluateAnd(expr string, ctx map[string]any) (bool, error) {
	parts := splitLogical(expr, "AND")

	for _, part := range parts {
		result, err := e.evaluateComparison(part, ctx)
		if err != nil {
			return false, err
		}

		if !result {
			return false, nil
		}
	}

	return true, nil
}

// evaluateComparison evaluates a single comparison expression.
func (e *Evaluator) evaluateComparison(expr string, ctx map[string]any) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Try two-character operators first
	for _, op := range []string{">=", "<=", "==", "!="} {
		if idx := strings.Index(expr, op); idx > 0 {
			left := strings.TrimSpace(expr[:idx])
			right := strings.TrimSpace(expr[idx+len(op):])

			return e.compare(left, op, right, ctx)
		}
	}

	// Try single-character operators (but not inside >= or <=)
	for _, op := range []string{">", "<"} {
		if idx := strings.Index(expr, op); idx > 0 {
			// Ensure it's not part of >= or <=
			if idx+1 < len(expr) && expr[idx+1] == '=' {
				continue
			}

			left := strings.TrimSpace(expr[:idx])
			right := strings.TrimSpace(expr[idx+len(op):])

			return e.compare(left, op, right, ctx)
		}
	}

	// Try to evaluate as a boolean value or field reference
	val, err := e.resolveValue(expr, ctx)
	if err != nil {
		return false, fmt.Errorf("cannot evaluate expression %q: %w", expr, err)
	}

	return toBool(val), nil
}

// compare performs a comparison between two values.
func (e *Evaluator) compare(leftExpr, op, rightExpr string, ctx map[string]any) (bool, error) {
	left, err := e.resolveValue(leftExpr, ctx)
	if err != nil {
		return false, fmt.Errorf("cannot resolve left side %q: %w", leftExpr, err)
	}

	right, err := e.resolveValue(rightExpr, ctx)
	if err != nil {
		return false, fmt.Errorf("cannot resolve right side %q: %w", rightExpr, err)
	}

	// Try numeric comparison
	leftNum, leftIsNum := toFloat64(left)
	rightNum, rightIsNum := toFloat64(right)

	if leftIsNum && rightIsNum {
		return compareNumbers(leftNum, op, rightNum)
	}

	// String comparison for == and !=
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)

	switch op {
	case "==":
		return leftStr == rightStr, nil
	case "!=":
		return leftStr != rightStr, nil
	default:
		return false, fmt.Errorf("operator %q requires numeric operands", op)
	}
}

// resolveValue resolves a value from the expression - either a field reference or a literal.
func (e *Evaluator) resolveValue(expr string, ctx map[string]any) (any, error) {
	expr = strings.TrimSpace(expr)

	// Quoted string literal
	if (strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"")) ||
		(strings.HasPrefix(expr, "'") && strings.HasSuffix(expr, "'")) {
		return expr[1 : len(expr)-1], nil
	}

	// Boolean literals
	if strings.EqualFold(expr, "true") {
		return true, nil
	}

	if strings.EqualFold(expr, "false") {
		return false, nil
	}

	// Numeric literal
	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return num, nil
	}

	// Field reference (dot notation)
	if strings.Contains(expr, ".") {
		val, found := resolveField(expr, ctx)
		if found {
			return val, nil
		}

		return nil, fmt.Errorf("field %q not found in context", expr)
	}

	// Direct key lookup in context
	if val, ok := ctx[expr]; ok {
		return val, nil
	}

	return nil, fmt.Errorf("cannot resolve %q", expr)
}

// resolveField resolves a dot-notation field path from the context.
func resolveField(path string, ctx map[string]any) (any, bool) {
	parts := strings.Split(path, ".")

	var current any = ctx

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}

		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}

	return current, true
}

// splitLogical splits an expression by a logical operator (AND/OR),
// respecting word boundaries.
func splitLogical(expr, op string) []string {
	var parts []string

	remaining := expr

	for {
		idx := findLogicalOperator(remaining, op)
		if idx < 0 {
			parts = append(parts, strings.TrimSpace(remaining))
			break
		}

		parts = append(parts, strings.TrimSpace(remaining[:idx]))
		// Skip the full pattern " OP " (op + 2 spaces)
		remaining = remaining[idx+len(op)+2:]
	}

	return parts
}

// findLogicalOperator finds the index of a logical operator surrounded by spaces.
func findLogicalOperator(expr, op string) int {
	// Look for " AND " or " OR " (surrounded by spaces)
	pattern := " " + op + " "
	return strings.Index(expr, pattern)
}

// compareNumbers compares two numeric values.
func compareNumbers(left float64, op string, right float64) (bool, error) {
	switch op {
	case ">":
		return left > right, nil
	case "<":
		return left < right, nil
	case ">=":
		return left >= right, nil
	case "<=":
		return left <= right, nil
	case "==":
		return left == right, nil
	case "!=":
		return left != right, nil
	default:
		return false, fmt.Errorf("unknown operator %q", op)
	}
}

// toFloat64 attempts to convert a value to float64.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}

		return 0, false
	default:
		return 0, false
	}
}

// toBool converts a value to boolean.
func toBool(v any) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && !strings.EqualFold(val, "false")
	case float64:
		return val != 0
	case int:
		return val != 0
	case nil:
		return false
	default:
		return true
	}
}
