// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package transformation

import (
	"fmt"
	"strings"

	"github.com/qntfy/kazaam/v4"
	"github.com/qntfy/kazaam/v4/transform"
)

// customTransforms holds the custom transform functions.
var customTransforms = map[string]kazaam.TransformFunc{
	"remove_characters": removeCharactersTransform,
	"add_prefix":        addPrefixTransform,
	"add_suffix":        addSuffixTransform,
	"to_uppercase":      toUppercaseTransform,
	"to_lowercase":      toLowercaseTransform,
}

// NewConfigWithCustomTransforms creates a Kazaam config with custom transforms registered.
// Panics if registration fails, which indicates a programming error.
func NewConfigWithCustomTransforms() kazaam.Config {
	config := kazaam.NewDefaultConfig()

	for name, fn := range customTransforms {
		if err := config.RegisterTransform(name, fn); err != nil {
			panic(fmt.Sprintf("failed to register transform %q: %v", name, err))
		}
	}

	return config
}

// removeCharactersTransform removes specified characters from a string value at a path.
// Spec format:
//
//	{
//	  "operation": "remove_characters",
//	  "spec": {
//	    "path": "provider.document",
//	    "characters": ".-"
//	  }
//	}
func removeCharactersTransform(spec *transform.Config, data []byte) ([]byte, error) {
	path, ok := (*spec.Spec)["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("remove_characters: 'path' is required")
	}

	characters, ok := (*spec.Spec)["characters"].(string)
	if !ok || strings.TrimSpace(characters) == "" {
		return nil, fmt.Errorf("remove_characters: 'characters' is required and must not be empty or whitespace-only")
	}

	// Get the value at path
	value, ok := getStringAtPath(data, path)
	if !ok {
		// Path doesn't exist or value is not a string - return data unchanged
		return data, nil
	}

	// Remove each character using strings.Map for efficiency
	result := strings.Map(func(r rune) rune {
		if strings.ContainsRune(characters, r) {
			return -1 // Remove this rune
		}

		return r
	}, value)

	// Set the new value
	return setStringAtPath(data, path, result)
}

// addPrefixTransform adds a prefix to a string value at a path.
// Spec format:
//
//	{
//	  "operation": "add_prefix",
//	  "spec": {
//	    "path": "provider.id",
//	    "prefix": "BR-"
//	  }
//	}
func addPrefixTransform(spec *transform.Config, data []byte) ([]byte, error) {
	path, ok := (*spec.Spec)["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("add_prefix: 'path' is required")
	}

	prefix, ok := (*spec.Spec)["prefix"].(string)
	if !ok || prefix == "" {
		return nil, fmt.Errorf("add_prefix: 'prefix' is required and must not be empty")
	}

	value, ok := getStringAtPath(data, path)
	if !ok {
		// Path doesn't exist or value is not a string - return data unchanged
		return data, nil
	}

	return setStringAtPath(data, path, prefix+value)
}

// addSuffixTransform adds a suffix to a string value at a path.
// Spec format:
//
//	{
//	  "operation": "add_suffix",
//	  "spec": {
//	    "path": "provider.id",
//	    "suffix": "-2025"
//	  }
//	}
func addSuffixTransform(spec *transform.Config, data []byte) ([]byte, error) {
	path, ok := (*spec.Spec)["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("add_suffix: 'path' is required")
	}

	suffix, ok := (*spec.Spec)["suffix"].(string)
	if !ok || suffix == "" {
		return nil, fmt.Errorf("add_suffix: 'suffix' is required and must not be empty")
	}

	value, ok := getStringAtPath(data, path)
	if !ok {
		// Path doesn't exist or value is not a string - return data unchanged
		return data, nil
	}

	return setStringAtPath(data, path, value+suffix)
}

// toUppercaseTransform converts a string value at a path to uppercase.
// Spec format:
//
//	{
//	  "operation": "to_uppercase",
//	  "spec": {
//	    "path": "provider.name"
//	  }
//	}
func toUppercaseTransform(spec *transform.Config, data []byte) ([]byte, error) {
	path, ok := (*spec.Spec)["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("to_uppercase: 'path' is required")
	}

	value, ok := getStringAtPath(data, path)
	if !ok {
		// Path doesn't exist or value is not a string - return data unchanged
		return data, nil
	}

	return setStringAtPath(data, path, strings.ToUpper(value))
}

// toLowercaseTransform converts a string value at a path to lowercase.
// Spec format:
//
//	{
//	  "operation": "to_lowercase",
//	  "spec": {
//	    "path": "provider.email"
//	  }
//	}
func toLowercaseTransform(spec *transform.Config, data []byte) ([]byte, error) {
	path, ok := (*spec.Spec)["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("to_lowercase: 'path' is required")
	}

	value, ok := getStringAtPath(data, path)
	if !ok {
		// Path doesn't exist or value is not a string - return data unchanged
		return data, nil
	}

	return setStringAtPath(data, path, strings.ToLower(value))
}
