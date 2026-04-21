// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package testutil

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPointerHelpers(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "UUIDPtr returns pointer to UUID",
			test: func(t *testing.T) {
				id := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
				ptr := UUIDPtr(id)
				assert.NotNil(t, ptr)
				assert.Equal(t, id, *ptr)
			},
		},
		{
			name: "StringPtr returns pointer to string",
			test: func(t *testing.T) {
				s := "test string"
				ptr := StringPtr(s)
				assert.NotNil(t, ptr)
				assert.Equal(t, s, *ptr)
			},
		},
		{
			name: "StringPtr with empty string",
			test: func(t *testing.T) {
				s := ""
				ptr := StringPtr(s)
				assert.NotNil(t, ptr)
				assert.Equal(t, s, *ptr)
			},
		},
		{
			name: "Int64Ptr returns pointer to int64",
			test: func(t *testing.T) {
				i := int64(42)
				ptr := Int64Ptr(i)
				assert.NotNil(t, ptr)
				assert.Equal(t, i, *ptr)
			},
		},
		{
			name: "Int64Ptr with zero value",
			test: func(t *testing.T) {
				i := int64(0)
				ptr := Int64Ptr(i)
				assert.NotNil(t, ptr)
				assert.Equal(t, i, *ptr)
			},
		},
		{
			name: "Int64Ptr with negative value",
			test: func(t *testing.T) {
				i := int64(-100)
				ptr := Int64Ptr(i)
				assert.NotNil(t, ptr)
				assert.Equal(t, i, *ptr)
			},
		},
		{
			name: "IntPtr returns pointer to int",
			test: func(t *testing.T) {
				i := 42
				ptr := IntPtr(i)
				assert.NotNil(t, ptr)
				assert.Equal(t, i, *ptr)
			},
		},
		{
			name: "BoolPtr returns pointer to bool true",
			test: func(t *testing.T) {
				b := true
				ptr := BoolPtr(b)
				assert.NotNil(t, ptr)
				assert.Equal(t, b, *ptr)
			},
		},
		{
			name: "BoolPtr returns pointer to bool false",
			test: func(t *testing.T) {
				b := false
				ptr := BoolPtr(b)
				assert.NotNil(t, ptr)
				assert.Equal(t, b, *ptr)
			},
		},
		{
			name: "TimePtr returns pointer to time",
			test: func(t *testing.T) {
				now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
				ptr := TimePtr(now)
				assert.NotNil(t, ptr)
				assert.Equal(t, now, *ptr)
			},
		},
		{
			name: "Float64Ptr returns pointer to float64",
			test: func(t *testing.T) {
				f := 3.14159
				ptr := Float64Ptr(f)
				assert.NotNil(t, ptr)
				assert.Equal(t, f, *ptr)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.test(t)
		})
	}
}
