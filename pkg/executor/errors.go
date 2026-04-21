// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package executor

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
)

// ErrExecutorNotFound is returned when an executor is not found in the catalog.
var ErrExecutorNotFound = pkg.EntityNotFoundError{
	EntityType: "Executor",
	Code:       constant.ErrExecutorNotFound.Error(),
	Message:    "executor not found",
}

// ErrTriggerNotFound is returned when a trigger is not found in the catalog.
var ErrTriggerNotFound = pkg.EntityNotFoundError{
	EntityType: "Trigger",
	Code:       constant.ErrTriggerNotFound.Error(),
	Message:    "trigger not found",
}

// ErrRunnerNotFound is returned when a runner is not found in the catalog.
var ErrRunnerNotFound = pkg.EntityNotFoundError{
	EntityType: "Runner",
	Code:       constant.ErrRunnerNotFound.Error(),
	Message:    "runner not found",
}

// ErrProviderNotFound is returned when a provider is not found in the catalog.
var ErrProviderNotFound = pkg.EntityNotFoundError{
	EntityType: "Provider",
	Code:       constant.ErrProviderNotFound.Error(),
	Message:    "provider not found",
}

// ErrProviderDuplicate is returned when attempting to register a provider with a duplicate ID.
var ErrProviderDuplicate = pkg.ValidationError{
	EntityType: "Provider",
	Code:       constant.ErrProviderDuplicate.Error(),
	Message:    "provider already registered",
}

// NewExecutorConfigError creates a new ValidationError for an executor config validation failure.
func NewExecutorConfigError(executorID ID, err error) pkg.ValidationError {
	return pkg.ValidationError{
		EntityType: "Executor",
		Code:       constant.ErrExecutorInvalidConfig.Error(),
		Message:    fmt.Sprintf("invalid config for executor %s: %v", executorID, err),
		Err:        err,
	}
}

// NewTriggerConfigError creates a new ValidationError for a trigger config validation failure.
func NewTriggerConfigError(triggerID TriggerID, err error) pkg.ValidationError {
	return pkg.ValidationError{
		EntityType: "Trigger",
		Code:       constant.ErrTriggerInvalidConfig.Error(),
		Message:    fmt.Sprintf("invalid config for trigger %s: %v", triggerID, err),
		Err:        err,
	}
}

// NewExecutorNotFoundError creates a new EntityNotFoundError for an executor.
func NewExecutorNotFoundError(executorID ID) pkg.EntityNotFoundError {
	return pkg.EntityNotFoundError{
		EntityType: "Executor",
		Code:       constant.ErrExecutorNotFound.Error(),
		Message:    fmt.Sprintf("executor not found: %s", executorID),
	}
}

// NewTriggerNotFoundError creates a new EntityNotFoundError for a trigger.
func NewTriggerNotFoundError(triggerID TriggerID) pkg.EntityNotFoundError {
	return pkg.EntityNotFoundError{
		EntityType: "Trigger",
		Code:       constant.ErrTriggerNotFound.Error(),
		Message:    fmt.Sprintf("trigger not found: %s", triggerID),
	}
}

// NewRunnerNotFoundError creates a new EntityNotFoundError for a runner.
func NewRunnerNotFoundError(executorID ID) pkg.EntityNotFoundError {
	return pkg.EntityNotFoundError{
		EntityType: "Runner",
		Code:       constant.ErrRunnerNotFound.Error(),
		Message:    fmt.Sprintf("runner not found for executor: %s", executorID),
	}
}

// NewProviderNotFoundError creates a new EntityNotFoundError for a provider.
func NewProviderNotFoundError(providerID ProviderID) pkg.EntityNotFoundError {
	return pkg.EntityNotFoundError{
		EntityType: "Provider",
		Code:       constant.ErrProviderNotFound.Error(),
		Message:    fmt.Sprintf("provider not found: %s", providerID),
	}
}

// NewProviderDuplicateError creates a new ValidationError for a duplicate provider registration.
func NewProviderDuplicateError(providerID ProviderID) pkg.ValidationError {
	return pkg.ValidationError{
		EntityType: "Provider",
		Code:       constant.ErrProviderDuplicate.Error(),
		Message:    fmt.Sprintf("provider already registered: %s", providerID),
	}
}

// ErrTemplateNotFound is returned when a template is not found in the catalog.
var ErrTemplateNotFound = pkg.EntityNotFoundError{
	EntityType: "Template",
	Code:       constant.ErrTemplateNotFound.Error(),
	Message:    "template not found",
}

// ErrTemplateDuplicate is returned when attempting to register a template with a duplicate ID.
var ErrTemplateDuplicate = pkg.ValidationError{
	EntityType: "Template",
	Code:       constant.ErrTemplateDuplicate.Error(),
	Message:    "template already registered",
}

// NewTemplateNotFoundError creates a new EntityNotFoundError for a template.
func NewTemplateNotFoundError(templateID TemplateID) pkg.EntityNotFoundError {
	return pkg.EntityNotFoundError{
		EntityType: "Template",
		Code:       constant.ErrTemplateNotFound.Error(),
		Message:    fmt.Sprintf("template not found: %s", templateID),
	}
}

// NewTemplateDuplicateError creates a new ValidationError for a duplicate template registration.
func NewTemplateDuplicateError(templateID TemplateID) pkg.ValidationError {
	return pkg.ValidationError{
		EntityType: "Template",
		Code:       constant.ErrTemplateDuplicate.Error(),
		Message:    fmt.Sprintf("template already registered: %s", templateID),
	}
}

// NewTemplateParamError creates a new ValidationError for a template parameter validation failure.
func NewTemplateParamError(templateID TemplateID, err error) pkg.ValidationError {
	return pkg.ValidationError{
		EntityType: "Template",
		Code:       constant.ErrTemplateInvalidParams.Error(),
		Message:    fmt.Sprintf("invalid params for template %s: %v", templateID, err),
		Err:        err,
	}
}
