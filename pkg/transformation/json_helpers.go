// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package transformation

import (
	"encoding/json"
	"fmt"
	"strings"
)

// getStringAtPath extracts a string value from JSON data at the given dot-separated path.
// Returns the value and true if found and is a string, or empty string and false otherwise.
func getStringAtPath(data []byte, path string) (string, bool) {
	if path == "" {
		return "", false
	}

	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", false
	}

	parts := strings.Split(path, ".")
	current := any(obj)

	for _, part := range parts {
		v, ok := current.(map[string]any)
		if !ok {
			return "", false
		}

		val, ok := v[part]
		if !ok {
			return "", false
		}

		current = val
	}

	str, ok := current.(string)
	if !ok {
		return "", false
	}

	return str, true
}

// setStringAtPath sets a string value in JSON data at the given dot-separated path.
func setStringAtPath(data []byte, path string, value string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	parts := strings.Split(path, ".")
	if err := setNestedValue(obj, parts, value); err != nil {
		return nil, err
	}

	return json.Marshal(obj)
}

// setNestedValue recursively sets a value in a nested map structure.
func setNestedValue(obj map[string]any, parts []string, value any) error {
	if len(parts) == 0 {
		return fmt.Errorf("empty path")
	}

	if parts[0] == "" {
		return fmt.Errorf("empty path segment")
	}

	if len(parts) == 1 {
		obj[parts[0]] = value
		return nil
	}

	next, ok := obj[parts[0]]
	if !ok {
		// Create nested map if it doesn't exist
		next = make(map[string]any)
		obj[parts[0]] = next
	}

	nextMap, ok := next.(map[string]any)
	if !ok {
		return fmt.Errorf("cannot set value: path element is not an object")
	}

	return setNestedValue(nextMap, parts[1:], value)
}
