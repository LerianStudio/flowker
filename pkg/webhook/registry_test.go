// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package webhook

import (
	"fmt"
	"sync"
	"testing"

	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	require.NotNil(t, registry)
	assert.Equal(t, 0, registry.Count())
}

func TestRegistry_Register_Success(t *testing.T) {
	registry := NewRegistry()
	route := Route{
		WorkflowID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Path:       "/my-webhook",
		Method:     "POST",
	}

	err := registry.Register(route)

	require.NoError(t, err)
	assert.Equal(t, 1, registry.Count())
}

func TestRegistry_Register_DuplicatePathRejected(t *testing.T) {
	registry := NewRegistry()
	route1 := Route{
		WorkflowID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Path:       "/my-webhook",
		Method:     "POST",
	}
	route2 := Route{
		WorkflowID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Path:       "/my-webhook",
		Method:     "POST",
	}

	err := registry.Register(route1)
	require.NoError(t, err)

	err = registry.Register(route2)
	require.Error(t, err)
	assert.ErrorIs(t, err, constant.ErrWebhookPathAlreadyRegistered)
	assert.Contains(t, err.Error(), "already registered")
	assert.Equal(t, 1, registry.Count())
}

func TestRegistry_Register_DifferentMethodsAllowed(t *testing.T) {
	registry := NewRegistry()
	routePOST := Route{
		WorkflowID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Path:       "/my-webhook",
		Method:     "POST",
	}
	routeGET := Route{
		WorkflowID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Path:       "/my-webhook",
		Method:     "GET",
	}

	require.NoError(t, registry.Register(routePOST))
	require.NoError(t, registry.Register(routeGET))
	assert.Equal(t, 2, registry.Count())
}

func TestRegistry_Resolve_Found(t *testing.T) {
	registry := NewRegistry()
	wfID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	err := registry.Register(Route{
		WorkflowID:  wfID,
		Path:        "/payment/callback",
		Method:      "POST",
		VerifyToken: "secret-token",
	})
	require.NoError(t, err)

	route, ok := registry.Resolve("POST", "/payment/callback")
	require.True(t, ok)
	assert.Equal(t, wfID, route.WorkflowID)
	assert.Equal(t, "secret-token", route.VerifyToken)
}

func TestRegistry_Resolve_NotFound(t *testing.T) {
	registry := NewRegistry()

	route, ok := registry.Resolve("POST", "/nonexistent")
	assert.False(t, ok)
	assert.Equal(t, Route{}, route)
}

func TestRegistry_Resolve_CaseInsensitiveMethod(t *testing.T) {
	registry := NewRegistry()

	err := registry.Register(Route{
		WorkflowID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Path:       "/callback",
		Method:     "post",
	})
	require.NoError(t, err)

	// Resolve with uppercase method should match
	route, ok := registry.Resolve("POST", "/callback")
	require.True(t, ok)
	assert.Equal(t, uuid.MustParse("11111111-1111-1111-1111-111111111111"), route.WorkflowID)
}

func TestRegistry_PathNormalization(t *testing.T) {
	tests := []struct {
		name         string
		registerPath string
		resolvePath  string
		shouldMatch  bool
	}{
		{
			name:         "trailing slash removed",
			registerPath: "/webhook/",
			resolvePath:  "/webhook",
			shouldMatch:  true,
		},
		{
			name:         "leading slash added",
			registerPath: "webhook",
			resolvePath:  "/webhook",
			shouldMatch:  true,
		},
		{
			name:         "both normalizations",
			registerPath: "webhook/",
			resolvePath:  "/webhook",
			shouldMatch:  true,
		},
		{
			name:         "different paths do not match",
			registerPath: "/webhook-a",
			resolvePath:  "/webhook-b",
			shouldMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()

			err := registry.Register(Route{
				WorkflowID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
				Path:       tt.registerPath,
				Method:     "POST",
			})
			require.NoError(t, err)

			_, ok := registry.Resolve("POST", tt.resolvePath)
			assert.Equal(t, tt.shouldMatch, ok)
		})
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()
	wfID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	err := registry.Register(Route{
		WorkflowID: wfID,
		Path:       "/hook-a",
		Method:     "POST",
	})
	require.NoError(t, err)

	err = registry.Register(Route{
		WorkflowID: wfID,
		Path:       "/hook-b",
		Method:     "GET",
	})
	require.NoError(t, err)

	assert.Equal(t, 2, registry.Count())

	registry.Unregister(wfID)
	assert.Equal(t, 0, registry.Count())

	// Resolve should now return false
	_, ok := registry.Resolve("POST", "/hook-a")
	assert.False(t, ok)

	_, ok = registry.Resolve("GET", "/hook-b")
	assert.False(t, ok)
}

func TestRegistry_Unregister_OnlyAffectsTargetWorkflow(t *testing.T) {
	registry := NewRegistry()
	wfID1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	wfID2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	require.NoError(t, registry.Register(Route{WorkflowID: wfID1, Path: "/hook-1", Method: "POST"}))
	require.NoError(t, registry.Register(Route{WorkflowID: wfID2, Path: "/hook-2", Method: "POST"}))

	registry.Unregister(wfID1)

	assert.Equal(t, 1, registry.Count())
	_, ok := registry.Resolve("POST", "/hook-2")
	assert.True(t, ok)
}

func TestRegistry_Unregister_NonExistentWorkflow(t *testing.T) {
	registry := NewRegistry()

	// Should not panic
	registry.Unregister(uuid.MustParse("99999999-9999-9999-9999-999999999999"))
	assert.Equal(t, 0, registry.Count())
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()
	const goroutines = 50

	var wg sync.WaitGroup

	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			wfID := uuid.New()
			path := fmt.Sprintf("/hook-%d", idx)

			_ = registry.Register(Route{
				WorkflowID: wfID,
				Path:       path,
				Method:     "POST",
			})

			registry.Resolve("POST", path)
			registry.Count()
			registry.Unregister(wfID)
		}(i)
	}

	wg.Wait()

	// Registry should be empty after all goroutines unregister
	assert.Equal(t, 0, registry.Count())
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/webhook", "/webhook"},
		{"webhook", "/webhook"},
		{"/webhook/", "/webhook"},
		{"webhook/", "/webhook"},
		{"/a/b/c", "/a/b/c"},
		{"/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizePath(tt.input))
		})
	}
}

func TestRouteKey(t *testing.T) {
	assert.Equal(t, "POST /webhook", RouteKey("post", "/webhook"))
	assert.Equal(t, "GET /webhook", RouteKey("GET", "/webhook"))
	assert.Equal(t, "PUT /webhook", RouteKey("put", "webhook"))
}
