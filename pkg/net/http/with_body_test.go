// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package http

import (
	"bytes"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestFindUnknownFields(t *testing.T) {
	tests := []struct {
		name      string
		original  map[string]any
		marshaled map[string]any
		expected  map[string]any
	}{
		{
			name:      "no differences",
			original:  map[string]any{"name": "test", "age": 25},
			marshaled: map[string]any{"name": "test", "age": 25},
			expected:  map[string]any{},
		},
		{
			name:      "extra field in original",
			original:  map[string]any{"name": "test", "extra": "field"},
			marshaled: map[string]any{"name": "test"},
			expected:  map[string]any{"extra": "field"},
		},
		{
			name:      "multiple extra fields",
			original:  map[string]any{"name": "test", "extra1": "a", "extra2": "b"},
			marshaled: map[string]any{"name": "test"},
			expected:  map[string]any{"extra1": "a", "extra2": "b"},
		},
		{
			name:      "nested map with extra field",
			original:  map[string]any{"nested": map[string]any{"known": "value", "unknown": "extra"}},
			marshaled: map[string]any{"nested": map[string]any{"known": "value"}},
			expected:  map[string]any{"nested": map[string]any{"unknown": "extra"}},
		},
		{
			name:      "empty maps",
			original:  map[string]any{},
			marshaled: map[string]any{},
			expected:  map[string]any{},
		},
		{
			name:      "zero numeric value ignored",
			original:  map[string]any{"name": "test", "count": 0.0},
			marshaled: map[string]any{"name": "test"},
			expected:  map[string]any{},
		},
		{
			name:      "different values",
			original:  map[string]any{"name": "original"},
			marshaled: map[string]any{"name": "different"},
			expected:  map[string]any{"name": "original"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findUnknownFields(tt.original, tt.marshaled)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareSlices(t *testing.T) {
	tests := []struct {
		name      string
		original  []any
		marshaled []any
		expected  []any
	}{
		{
			name:      "identical slices",
			original:  []any{"a", "b", "c"},
			marshaled: []any{"a", "b", "c"},
			expected:  nil,
		},
		{
			name:      "original longer",
			original:  []any{"a", "b", "c", "d"},
			marshaled: []any{"a", "b", "c"},
			expected:  []any{"d"},
		},
		{
			name:      "marshaled longer",
			original:  []any{"a", "b"},
			marshaled: []any{"a", "b", "c", "d"},
			expected:  []any{"c", "d"},
		},
		{
			name:      "different values",
			original:  []any{"a", "x", "c"},
			marshaled: []any{"a", "b", "c"},
			expected:  []any{"x"},
		},
		{
			name:      "empty slices",
			original:  []any{},
			marshaled: []any{},
			expected:  nil,
		},
		{
			name:      "nested maps in slices with differences",
			original:  []any{map[string]any{"id": "1", "extra": "field"}},
			marshaled: []any{map[string]any{"id": "1"}},
			expected:  []any{map[string]any{"extra": "field"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareSlices(tt.original, tt.marshaled)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatErrorFieldName(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "simple field name",
			text:     "Struct.fieldName",
			expected: "fieldName",
		},
		{
			name:     "nested field name",
			text:     "Struct.nested.fieldName",
			expected: "nested.fieldName",
		},
		{
			name:     "no dot - returns original",
			text:     "fieldName",
			expected: "fieldName",
		},
		{
			name:     "empty string",
			text:     "",
			expected: "",
		},
		{
			name:     "single dot at end - returns original",
			text:     "Struct.",
			expected: "Struct.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatErrorFieldName(tt.text)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewOfType(t *testing.T) {
	type TestStruct struct {
		Name string
		Age  int
	}

	t.Run("creates new instance of type", func(t *testing.T) {
		original := &TestStruct{Name: "original", Age: 30}
		result := newOfType(original)

		assert.NotNil(t, result)
		newStruct, ok := result.(*TestStruct)
		assert.True(t, ok)
		assert.Equal(t, "", newStruct.Name)
		assert.Equal(t, 0, newStruct.Age)
	})
}

func TestValidateStruct(t *testing.T) {
	type ValidStruct struct {
		Name string `validate:"required"`
		Age  int    `validate:"required,gte=0"`
	}

	type InvalidStruct struct {
		Name string `validate:"required"`
	}

	tests := []struct {
		name        string
		input       any
		expectError bool
	}{
		{
			name:        "valid struct",
			input:       &ValidStruct{Name: "test", Age: 25},
			expectError: false,
		},
		{
			name:        "invalid struct - missing required",
			input:       &InvalidStruct{Name: ""},
			expectError: true,
		},
		{
			name:        "non-struct input",
			input:       "string value",
			expectError: false,
		},
		{
			name:        "nil pointer",
			input:       (*ValidStruct)(nil),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseMetadata(t *testing.T) {
	type WithMetadata struct {
		Name     string
		Metadata map[string]any
	}

	type WithoutMetadata struct {
		Name string
	}

	t.Run("sets empty metadata when not in original", func(t *testing.T) {
		s := &WithMetadata{Name: "test"}
		originalMap := map[string]any{"name": "test"}

		parseMetadata(s, originalMap)

		assert.NotNil(t, s.Metadata)
		assert.Empty(t, s.Metadata)
	})

	t.Run("keeps metadata when in original", func(t *testing.T) {
		s := &WithMetadata{Name: "test", Metadata: map[string]any{"key": "value"}}
		originalMap := map[string]any{"name": "test", "metadata": map[string]any{"key": "value"}}

		parseMetadata(s, originalMap)

		assert.Equal(t, map[string]any{"key": "value"}, s.Metadata)
	})

	t.Run("handles struct without metadata field", func(t *testing.T) {
		s := &WithoutMetadata{Name: "test"}
		originalMap := map[string]any{"name": "test"}

		// Should not panic
		parseMetadata(s, originalMap)
		assert.Equal(t, "test", s.Name)
	})

	t.Run("handles non-pointer input", func(t *testing.T) {
		s := WithMetadata{Name: "test"}
		originalMap := map[string]any{"name": "test"}

		// Should not panic
		parseMetadata(s, originalMap)
	})

	t.Run("handles non-struct input", func(t *testing.T) {
		s := "string"
		originalMap := map[string]any{"name": "test"}

		// Should not panic
		parseMetadata(s, originalMap)
	})
}

func TestValidateMetadataKeyMaxLength(t *testing.T) {
	// This test validates the internal validation function behavior
	// through the ValidateStruct function

	type MetadataKey struct {
		Key string `validate:"keymax=10"`
	}

	tests := []struct {
		name        string
		key         string
		expectError bool
	}{
		{
			name:        "key within limit",
			key:         "shortkey",
			expectError: false,
		},
		{
			name:        "key exactly at limit",
			key:         "1234567890",
			expectError: false,
		},
		{
			name:        "key exceeds limit",
			key:         "verylongkeythatexceedslimit",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &MetadataKey{Key: tt.key}
			err := ValidateStruct(input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFieldsRequired(t *testing.T) {
	tests := []struct {
		name     string
		input    pkg.FieldValidations
		expected pkg.FieldValidations
	}{
		{
			name: "filters required fields",
			input: pkg.FieldValidations{
				"name":  "name is a required field",
				"email": "email must be a valid email",
			},
			expected: pkg.FieldValidations{
				"name": "name is a required field",
			},
		},
		{
			name: "no required fields",
			input: pkg.FieldValidations{
				"email": "email must be a valid email",
			},
			expected: pkg.FieldValidations{},
		},
		{
			name:     "empty input",
			input:    pkg.FieldValidations{},
			expected: pkg.FieldValidations{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fieldsRequired(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWithBody(t *testing.T) {
	type TestPayload struct {
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age" validate:"required"`
	}

	t.Run("returns fiber handler", func(t *testing.T) {
		handler := WithBody(&TestPayload{}, func(p any, c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})
		assert.NotNil(t, handler)
	})
}

func TestFiberHandlerFunc(t *testing.T) {
	type TestPayload struct {
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age"`
	}

	t.Run("decodes valid JSON body", func(t *testing.T) {
		app := fiber.New()

		var received *TestPayload
		app.Post("/test", WithBody(&TestPayload{}, func(p any, c *fiber.Ctx) error {
			received = p.(*TestPayload)
			return c.SendStatus(fiber.StatusOK)
		}))

		req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"name":"test","age":25}`)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
		if assert.NotNil(t, received) {
			assert.Equal(t, "test", received.Name)
			assert.Equal(t, 25, received.Age)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		app := fiber.New()

		app.Post("/test", WithBody(&TestPayload{}, func(p any, c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		}))

		req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(`invalid json`)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.NotEqual(t, 200, resp.StatusCode)
	})

	t.Run("returns error for unknown fields", func(t *testing.T) {
		app := fiber.New()

		app.Post("/test", WithBody(&TestPayload{}, func(p any, c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		}))

		req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"name":"test","age":25,"unknown":"field"}`)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("returns error for validation failure", func(t *testing.T) {
		app := fiber.New()

		app.Post("/test", WithBody(&TestPayload{}, func(p any, c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		}))

		req := httptest.NewRequest("POST", "/test", bytes.NewReader([]byte(`{"name":"","age":25}`)))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestParseUUIDPathParameters(t *testing.T) {
	t.Run("parses valid UUID", func(t *testing.T) {
		app := fiber.New()

		var parsedID any
		app.Get("/test/:id", ParseUUIDPathParameters, func(c *fiber.Ctx) error {
			parsedID = c.Locals("id")
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test/550e8400-e29b-41d4-a716-446655440000", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
		assert.NotNil(t, parsedID)
	})

	t.Run("returns error for invalid UUID", func(t *testing.T) {
		app := fiber.New()

		app.Get("/test/:id", ParseUUIDPathParameters, func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test/invalid-uuid", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.NotEqual(t, 200, resp.StatusCode)
	})

	t.Run("allows non-UUID parameters not in UUIDPathParameters list", func(t *testing.T) {
		app := fiber.New()

		var parsedValue any
		app.Get("/test/:other", ParseUUIDPathParameters, func(c *fiber.Ctx) error {
			parsedValue = c.Locals("other")
			return c.SendStatus(fiber.StatusOK)
		})

		req := httptest.NewRequest("GET", "/test/some-string", nil)
		resp, err := app.Test(req)

		assert.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, 200, resp.StatusCode)
		assert.Equal(t, "some-string", parsedValue)
	})
}

func TestValidateMetadataValueMaxLength(t *testing.T) {
	type MetadataValue struct {
		Value string `validate:"valuemax=10"`
	}

	type MetadataIntValue struct {
		Value int `validate:"valuemax=5"`
	}

	t.Run("string value within limit", func(t *testing.T) {
		input := &MetadataValue{Value: "short"}
		err := ValidateStruct(input)
		assert.NoError(t, err)
	})

	t.Run("string value exceeds limit", func(t *testing.T) {
		input := &MetadataValue{Value: "this is a very long string that exceeds the limit"}
		err := ValidateStruct(input)
		assert.Error(t, err)
	})

	t.Run("int value within limit", func(t *testing.T) {
		input := &MetadataIntValue{Value: 123}
		err := ValidateStruct(input)
		assert.NoError(t, err)
	})
}

func TestValidateMetadataNestedValues(t *testing.T) {
	type WithNestedMap struct {
		Data map[string]any `validate:"nonested"`
	}

	type WithString struct {
		Data string `validate:"nonested"`
	}

	t.Run("map value fails nonested validation", func(t *testing.T) {
		input := &WithNestedMap{Data: map[string]any{"key": "value"}}
		err := ValidateStruct(input)
		assert.Error(t, err)
	})

	t.Run("non-map value passes nonested validation", func(t *testing.T) {
		input := &WithString{Data: "not a map"}
		err := ValidateStruct(input)
		assert.NoError(t, err)
	})
}
