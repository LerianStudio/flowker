// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutionService_NilDependencies(t *testing.T) {
	t.Run("all dependencies nil", func(t *testing.T) {
		svc, err := NewExecutionService(nil, nil, nil, nil)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrExecutionServiceNilDependency)
		assert.Nil(t, svc)
	})
}
