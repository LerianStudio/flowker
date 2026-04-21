// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package executor

import (
	"sort"
	"sync"
)

// Catalog manages the registry of available providers, executors, triggers, and runners.
type Catalog interface {
	// GetExecutor returns an executor by ID.
	GetExecutor(id ID) (Executor, error)

	// GetRunner returns a runner by executor ID.
	GetRunner(id ID) (Runner, error)

	// GetTrigger returns a trigger by ID.
	GetTrigger(id TriggerID) (Trigger, error)

	// ListExecutors returns all registered executors sorted by ID.
	ListExecutors() []Executor

	// ListTriggers returns all registered triggers sorted by ID.
	ListTriggers() []Trigger

	// RegisterExecutor registers an executor and its runner.
	RegisterExecutor(executor Executor, runner Runner)

	// RegisterTrigger registers a trigger.
	RegisterTrigger(trigger Trigger)

	// RegisterProvider registers a provider and all its executors in one call.
	// Each executor is linked to the provider. Returns error if the provider
	// is already registered or any executor/runner is nil.
	RegisterProvider(provider Provider, executors []ExecutorRegistration) error

	// GetProvider returns a provider by ID.
	GetProvider(id ProviderID) (Provider, error)

	// ListProviders returns all registered providers sorted by ID.
	ListProviders() []Provider

	// GetProviderExecutors returns all executors belonging to a provider.
	// Returns error if the provider is not found.
	GetProviderExecutors(id ProviderID) ([]Executor, error)

	// RegisterTemplate registers a workflow template.
	// Returns error if a template with the same ID already exists.
	RegisterTemplate(template Template) error

	// GetTemplate returns a template by ID.
	GetTemplate(id TemplateID) (Template, error)

	// ListTemplates returns all registered templates sorted by ID.
	ListTemplates() []Template
}

// InMemoryCatalog is a thread-safe in-memory implementation of Catalog.
type InMemoryCatalog struct {
	mu                sync.RWMutex
	executors         map[ID]Executor
	runners           map[ID]Runner
	triggers          map[TriggerID]Trigger
	providers         map[ProviderID]Provider
	executorProviders map[ID]ProviderID
	templates         map[TemplateID]Template
}

// NewCatalog creates a new empty in-memory catalog.
func NewCatalog() *InMemoryCatalog {
	return &InMemoryCatalog{
		executors:         make(map[ID]Executor),
		runners:           make(map[ID]Runner),
		triggers:          make(map[TriggerID]Trigger),
		providers:         make(map[ProviderID]Provider),
		executorProviders: make(map[ID]ProviderID),
		templates:         make(map[TemplateID]Template),
	}
}

// GetExecutor returns an executor by ID.
func (c *InMemoryCatalog) GetExecutor(id ID) (Executor, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, ok := c.executors[id]
	if !ok {
		return nil, NewExecutorNotFoundError(id)
	}

	return e, nil
}

// GetRunner returns a runner by executor ID.
func (c *InMemoryCatalog) GetRunner(id ID) (Runner, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	r, ok := c.runners[id]
	if !ok {
		return nil, NewRunnerNotFoundError(id)
	}

	return r, nil
}

// GetTrigger returns a trigger by ID.
func (c *InMemoryCatalog) GetTrigger(id TriggerID) (Trigger, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	t, ok := c.triggers[id]
	if !ok {
		return nil, NewTriggerNotFoundError(id)
	}

	return t, nil
}

// ListExecutors returns all registered executors sorted by ID.
func (c *InMemoryCatalog) ListExecutors() []Executor {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := make([]string, 0, len(c.executors))
	for id := range c.executors {
		ids = append(ids, string(id))
	}

	sort.Strings(ids)

	executors := make([]Executor, 0, len(ids))
	for _, id := range ids {
		executors = append(executors, c.executors[ID(id)])
	}

	return executors
}

// ListTriggers returns all registered triggers sorted by ID.
func (c *InMemoryCatalog) ListTriggers() []Trigger {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := make([]string, 0, len(c.triggers))
	for id := range c.triggers {
		ids = append(ids, string(id))
	}

	sort.Strings(ids)

	triggers := make([]Trigger, 0, len(ids))
	for _, id := range ids {
		triggers = append(triggers, c.triggers[TriggerID(id)])
	}

	return triggers
}

// RegisterExecutor registers an executor and its runner.
func (c *InMemoryCatalog) RegisterExecutor(executor Executor, runner Runner) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.executors[executor.ID()] = executor
	c.runners[executor.ID()] = runner
}

// RegisterTrigger registers a trigger.
func (c *InMemoryCatalog) RegisterTrigger(trigger Trigger) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.triggers[trigger.ID()] = trigger
}

// RegisterProvider registers a provider and all its executors in one call.
// Each executor is linked to the provider via the executorProviders mapping.
func (c *InMemoryCatalog) RegisterProvider(provider Provider, executors []ExecutorRegistration) error {
	if provider == nil {
		return NewProviderNotFoundError("")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.providers[provider.ID()]; exists {
		return NewProviderDuplicateError(provider.ID())
	}

	// Validate all executors before modifying state to ensure atomicity.
	for _, reg := range executors {
		if reg.Executor == nil || reg.Runner == nil {
			return NewExecutorNotFoundError("")
		}
	}

	c.providers[provider.ID()] = provider

	for _, reg := range executors {
		c.executors[reg.Executor.ID()] = reg.Executor
		c.runners[reg.Executor.ID()] = reg.Runner
		c.executorProviders[reg.Executor.ID()] = provider.ID()
	}

	return nil
}

// GetProvider returns a provider by ID.
func (c *InMemoryCatalog) GetProvider(id ProviderID) (Provider, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	p, ok := c.providers[id]
	if !ok {
		return nil, NewProviderNotFoundError(id)
	}

	return p, nil
}

// ListProviders returns all registered providers sorted by ID.
func (c *InMemoryCatalog) ListProviders() []Provider {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := make([]string, 0, len(c.providers))
	for id := range c.providers {
		ids = append(ids, string(id))
	}

	sort.Strings(ids)

	providers := make([]Provider, 0, len(ids))
	for _, id := range ids {
		providers = append(providers, c.providers[ProviderID(id)])
	}

	return providers
}

// GetProviderExecutors returns all executors belonging to a provider, sorted by ID.
func (c *InMemoryCatalog) GetProviderExecutors(id ProviderID) ([]Executor, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, ok := c.providers[id]; !ok {
		return nil, NewProviderNotFoundError(id)
	}

	ids := make([]string, 0)

	for execID, provID := range c.executorProviders {
		if provID == id {
			ids = append(ids, string(execID))
		}
	}

	sort.Strings(ids)

	executors := make([]Executor, 0, len(ids))
	for _, execID := range ids {
		executors = append(executors, c.executors[ID(execID)])
	}

	return executors, nil
}

// RegisterTemplate registers a workflow template.
// Returns error if a template with the same ID already exists.
func (c *InMemoryCatalog) RegisterTemplate(template Template) error {
	if template == nil {
		return NewTemplateNotFoundError("")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.templates[template.ID()]; exists {
		return NewTemplateDuplicateError(template.ID())
	}

	c.templates[template.ID()] = template

	return nil
}

// GetTemplate returns a template by ID.
func (c *InMemoryCatalog) GetTemplate(id TemplateID) (Template, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	t, ok := c.templates[id]
	if !ok {
		return nil, NewTemplateNotFoundError(id)
	}

	return t, nil
}

// ListTemplates returns all registered templates sorted by ID.
func (c *InMemoryCatalog) ListTemplates() []Template {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ids := make([]string, 0, len(c.templates))
	for id := range c.templates {
		ids = append(ids, string(id))
	}

	sort.Strings(ids)

	templates := make([]Template, 0, len(ids))
	for _, id := range ids {
		templates = append(templates, c.templates[TemplateID(id)])
	}

	return templates
}

// Verify InMemoryCatalog implements Catalog interface.
var _ Catalog = (*InMemoryCatalog)(nil)
