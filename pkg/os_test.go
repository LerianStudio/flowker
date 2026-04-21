// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package pkg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_ENV_VAR_1",
			defaultValue: "default",
			envValue:     "actual_value",
			setEnv:       true,
			expected:     "actual_value",
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_ENV_VAR_2",
			defaultValue: "default_value",
			envValue:     "",
			setEnv:       false,
			expected:     "default_value",
		},
		{
			name:         "returns default when env is empty string",
			key:          "TEST_ENV_VAR_3",
			defaultValue: "default_value",
			envValue:     "",
			setEnv:       true,
			expected:     "default_value",
		},
		{
			name:         "returns default when env is whitespace",
			key:          "TEST_ENV_VAR_4",
			defaultValue: "default_value",
			envValue:     "   ",
			setEnv:       true,
			expected:     "default_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := GetEnvOrDefault(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetenvBoolOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		setEnv       bool
		expected     bool
	}{
		{
			name:         "returns true when env is 'true'",
			key:          "TEST_BOOL_1",
			defaultValue: false,
			envValue:     "true",
			setEnv:       true,
			expected:     true,
		},
		{
			name:         "returns false when env is 'false'",
			key:          "TEST_BOOL_2",
			defaultValue: true,
			envValue:     "false",
			setEnv:       true,
			expected:     false,
		},
		{
			name:         "returns true when env is '1'",
			key:          "TEST_BOOL_3",
			defaultValue: false,
			envValue:     "1",
			setEnv:       true,
			expected:     true,
		},
		{
			name:         "returns false when env is '0'",
			key:          "TEST_BOOL_4",
			defaultValue: true,
			envValue:     "0",
			setEnv:       true,
			expected:     false,
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_BOOL_5",
			defaultValue: true,
			envValue:     "",
			setEnv:       false,
			expected:     true,
		},
		{
			name:         "returns default when env is invalid",
			key:          "TEST_BOOL_6",
			defaultValue: true,
			envValue:     "invalid",
			setEnv:       true,
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := GetenvBoolOrDefault(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetenvIntOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int64
		envValue     string
		setEnv       bool
		expected     int64
	}{
		{
			name:         "returns int value when env is valid",
			key:          "TEST_INT_1",
			defaultValue: 0,
			envValue:     "42",
			setEnv:       true,
			expected:     42,
		},
		{
			name:         "returns negative int value",
			key:          "TEST_INT_2",
			defaultValue: 0,
			envValue:     "-100",
			setEnv:       true,
			expected:     -100,
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_INT_3",
			defaultValue: 99,
			envValue:     "",
			setEnv:       false,
			expected:     99,
		},
		{
			name:         "returns default when env is invalid",
			key:          "TEST_INT_4",
			defaultValue: 50,
			envValue:     "not_a_number",
			setEnv:       true,
			expected:     50,
		},
		{
			name:         "returns default when env is float",
			key:          "TEST_INT_5",
			defaultValue: 25,
			envValue:     "3.14",
			setEnv:       true,
			expected:     25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			result := GetenvIntOrDefault(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetConfigFromEnvVars(t *testing.T) {
	type TestConfig struct {
		StringField string `env:"TEST_STRING_FIELD"`
		BoolField   bool   `env:"TEST_BOOL_FIELD"`
		IntField    int    `env:"TEST_INT_FIELD"`
		Int64Field  int64  `env:"TEST_INT64_FIELD"`
		NoTagField  string
	}

	t.Run("sets fields from env vars", func(t *testing.T) {
		os.Setenv("TEST_STRING_FIELD", "test_value")
		os.Setenv("TEST_BOOL_FIELD", "true")
		os.Setenv("TEST_INT_FIELD", "42")
		os.Setenv("TEST_INT64_FIELD", "9999")
		defer func() {
			os.Unsetenv("TEST_STRING_FIELD")
			os.Unsetenv("TEST_BOOL_FIELD")
			os.Unsetenv("TEST_INT_FIELD")
			os.Unsetenv("TEST_INT64_FIELD")
		}()

		config := &TestConfig{}
		err := SetConfigFromEnvVars(config)

		assert.NoError(t, err)
		assert.Equal(t, "test_value", config.StringField)
		assert.True(t, config.BoolField)
		assert.Equal(t, 42, config.IntField)
		assert.Equal(t, int64(9999), config.Int64Field)
	})

	t.Run("returns error for non-pointer", func(t *testing.T) {
		config := TestConfig{}
		err := SetConfigFromEnvVars(config)

		assert.ErrorIs(t, err, ErrMustBePointer)
	})

	t.Run("handles missing env vars", func(t *testing.T) {
		os.Unsetenv("TEST_STRING_FIELD")
		os.Unsetenv("TEST_BOOL_FIELD")
		os.Unsetenv("TEST_INT_FIELD")
		os.Unsetenv("TEST_INT64_FIELD")

		config := &TestConfig{
			StringField: "default",
			BoolField:   true,
			IntField:    100,
		}
		err := SetConfigFromEnvVars(config)

		assert.NoError(t, err)
		assert.Equal(t, "", config.StringField)
		assert.False(t, config.BoolField)
		assert.Equal(t, 0, config.IntField)
	})
}

func TestEnsureConfigFromEnvVars(t *testing.T) {
	type TestConfig struct {
		Field string `env:"TEST_ENSURE_FIELD"`
	}

	t.Run("returns config on success", func(t *testing.T) {
		os.Setenv("TEST_ENSURE_FIELD", "value")
		defer os.Unsetenv("TEST_ENSURE_FIELD")

		config := &TestConfig{}
		result := EnsureConfigFromEnvVars(config)

		assert.NotNil(t, result)
		assert.Equal(t, "value", config.Field)
	})

	t.Run("panics on error", func(t *testing.T) {
		config := TestConfig{}

		assert.Panics(t, func() {
			EnsureConfigFromEnvVars(config)
		})
	})
}
