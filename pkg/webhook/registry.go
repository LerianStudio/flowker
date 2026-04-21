// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package webhook provides the in-memory webhook route registry used to
// dynamically map HTTP method+path pairs to active workflow triggers.
package webhook

import (
	"fmt"
	"strings"
	"sync"

	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/google/uuid"
)

// Route holds the mapping between an HTTP method+path and the workflow
// that should be executed when the webhook is triggered.
type Route struct {
	WorkflowID  uuid.UUID
	Path        string
	Method      string
	VerifyToken string // optional static token for webhook validation
}

// Registry is a thread-safe in-memory registry that maps HTTP
// method+path pairs to workflow webhook routes. It is used to dynamically
// register and unregister webhook endpoints as workflows are activated
// and deactivated.
type Registry struct {
	mu     sync.RWMutex
	routes map[string]Route // key = "METHOD /path"
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		routes: make(map[string]Route),
	}
}

// RouteKey builds the map key from an HTTP method and path.
func RouteKey(method, path string) string {
	return fmt.Sprintf("%s %s", strings.ToUpper(method), NormalizePath(path))
}

// NormalizePath ensures a leading slash and removes any trailing slash.
func NormalizePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return strings.TrimRight(path, "/")
}

// Register adds a new webhook route. Returns an error if the method+path
// combination is already registered by another workflow.
func (r *Registry) Register(route Route) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := RouteKey(route.Method, route.Path)

	if existing, ok := r.routes[key]; ok {
		return fmt.Errorf("%w: path %s %s already registered by workflow %s",
			constant.ErrWebhookPathAlreadyRegistered, route.Method, route.Path, existing.WorkflowID)
	}

	r.routes[key] = route

	return nil
}

// Unregister removes all webhook routes associated with the given workflow ID.
func (r *Registry) Unregister(workflowID uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, route := range r.routes {
		if route.WorkflowID == workflowID {
			delete(r.routes, key)
		}
	}
}

// Resolve looks up a webhook route by HTTP method and path.
// Returns the route and true if found, or a zero-value route and false if not.
func (r *Registry) Resolve(method, path string) (Route, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := RouteKey(method, path)
	route, ok := r.routes[key]

	return route, ok
}

// Count returns the number of currently registered webhook routes.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.routes)
}
