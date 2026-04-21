// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package contextutil

import "context"

// ContextKeyClientIP is a type-safe key for storing client IP in context.
type ContextKeyClientIP struct{}

// GetClientIP extracts the client IP from context, returning empty string if not set.
func GetClientIP(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if v := ctx.Value(ContextKeyClientIP{}); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}
