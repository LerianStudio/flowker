// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package transformation provides JSON-to-JSON transformation capabilities
// using Kazaam for field mapping between workflow and provider data.
package transformation

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/LerianStudio/flowker/pkg/constant"
	"github.com/qntfy/kazaam/v4"
)

// Service provides JSON transformation operations using Kazaam.
type Service struct {
	config  kazaam.Config
	cache   map[string]*kazaam.Kazaam
	cacheMu sync.RWMutex
}

// NewService creates a new transformation service with custom transforms registered.
func NewService() *Service {
	return &Service{
		config: NewConfigWithCustomTransforms(),
		cache:  make(map[string]*kazaam.Kazaam),
	}
}

// Transform applies a Kazaam transformation spec to input data.
// The spec should be a valid Kazaam specification JSON string.
func (s *Service) Transform(ctx context.Context, input []byte, spec string) ([]byte, error) {
	k, err := s.getOrCreateKazaam(spec)
	if err != nil {
		return nil, pkg.ValidationError{
			Code:    constant.ErrWorkflowInvalidTransforms.Error(),
			Message: "invalid transformation spec",
			Err:     err,
		}
	}

	output, err := k.Transform(input)
	if err != nil {
		return nil, pkg.ValidationError{
			Code:    constant.ErrWorkflowInvalidTransforms.Error(),
			Message: "transformation execution failed",
			Err:     err,
		}
	}

	return output, nil
}

// TransformMap applies a Kazaam transformation spec to input data as maps.
// This is a convenience method that handles JSON marshaling/unmarshaling.
func (s *Service) TransformMap(ctx context.Context, input map[string]any, spec string) (map[string]any, error) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, pkg.ValidationError{
			Code:    constant.ErrWorkflowInvalidInputMapping.Error(),
			Message: "failed to marshal input for transformation",
			Err:     err,
		}
	}

	outputBytes, err := s.Transform(ctx, inputBytes, spec)
	if err != nil {
		return nil, err
	}

	var output map[string]any
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		return nil, pkg.ValidationError{
			Code:    constant.ErrWorkflowInvalidOutputMapping.Error(),
			Message: "failed to unmarshal transformation output",
			Err:     err,
		}
	}

	return output, nil
}

// ValidateSpec validates a Kazaam transformation spec without executing it.
func (s *Service) ValidateSpec(spec string) error {
	_, err := kazaam.New(spec, s.config)
	if err != nil {
		return pkg.ValidationError{
			Code:    constant.ErrWorkflowInvalidTransforms.Error(),
			Message: "invalid transformation spec",
			Err:     err,
		}
	}

	return nil
}

// getOrCreateKazaam returns a cached Kazaam instance or creates a new one.
func (s *Service) getOrCreateKazaam(spec string) (*kazaam.Kazaam, error) {
	// Try read lock first for cache hit
	s.cacheMu.RLock()

	if k, ok := s.cache[spec]; ok {
		s.cacheMu.RUnlock()

		return k, nil
	}

	s.cacheMu.RUnlock()

	// Upgrade to write lock for cache miss
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// Double-check after acquiring write lock
	if k, ok := s.cache[spec]; ok {
		return k, nil
	}

	k, err := kazaam.New(spec, s.config)
	if err != nil {
		return nil, err
	}

	s.cache[spec] = k

	return k, nil
}

// ClearCache clears the Kazaam instance cache.
func (s *Service) ClearCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.cache = make(map[string]*kazaam.Kazaam)
}

// Validator returns a TransformationValidator that can be used by the model package.
// This breaks the import cycle by returning an interface implementation.
func (s *Service) Validator() *Validator {
	return &Validator{service: s}
}

// Validator implements model.TransformationValidator interface.
type Validator struct {
	service *Service
}
