// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrDatabaseItemNotFound(t *testing.T) {
	tests := []struct {
		name          string
		expectedError string
	}{
		{
			name:          "error message is correct",
			expectedError: "errDatabaseItemNotFound",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, ErrDatabaseItemNotFound)
			assert.Equal(t, tt.expectedError, ErrDatabaseItemNotFound.Error())
		})
	}
}
