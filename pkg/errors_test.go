// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package pkg

import (
	"errors"
	"github.com/LerianStudio/flowker/pkg/constant"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntityNotFoundError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      EntityNotFoundError
		expected string
	}{
		{
			name: "with message",
			err: EntityNotFoundError{
				Message: "custom not found message",
			},
			expected: "custom not found message",
		},
		{
			name: "with entity type only",
			err: EntityNotFoundError{
				EntityType: "User",
			},
			expected: "Entity User not found",
		},
		{
			name: "with wrapped error",
			err: EntityNotFoundError{
				Err: errors.New("underlying error"),
			},
			expected: "underlying error",
		},
		{
			name:     "empty error",
			err:      EntityNotFoundError{},
			expected: "entity not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntityNotFoundError_Unwrap(t *testing.T) {
	t.Run("with wrapped error", func(t *testing.T) {
		wrappedErr := errors.New("wrapped error")
		err := EntityNotFoundError{Err: wrappedErr}
		assert.Equal(t, wrappedErr, err.Unwrap())
	})

	t.Run("with nil error", func(t *testing.T) {
		err := EntityNotFoundError{}
		assert.Nil(t, err.Unwrap())
	})
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      ValidationError
		expected string
	}{
		{
			name: "with code and message",
			err: ValidationError{
				Code:    "VAL001",
				Message: "validation failed",
			},
			expected: "VAL001 - validation failed",
		},
		{
			name: "message only",
			err: ValidationError{
				Message: "simple validation error",
			},
			expected: "simple validation error",
		},
		{
			name:     "empty validation error",
			err:      ValidationError{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationError_Unwrap(t *testing.T) {
	t.Run("with wrapped error", func(t *testing.T) {
		wrappedErr := errors.New("wrapped validation error")
		err := ValidationError{Err: wrappedErr}
		assert.Equal(t, wrappedErr, err.Unwrap())
	})

	t.Run("with nil error", func(t *testing.T) {
		err := ValidationError{}
		assert.Nil(t, err.Unwrap())
	})
}

func TestEntityConflictError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      EntityConflictError
		expected string
	}{
		{
			name: "with wrapped error only",
			err: EntityConflictError{
				Err: errors.New("conflict details"),
			},
			expected: "conflict details",
		},
		{
			name: "with message",
			err: EntityConflictError{
				Message: "entity already exists",
			},
			expected: "entity already exists",
		},
		{
			name:     "empty conflict error",
			err:      EntityConflictError{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEntityConflictError_Unwrap(t *testing.T) {
	t.Run("with wrapped error", func(t *testing.T) {
		wrappedErr := errors.New("wrapped conflict error")
		err := EntityConflictError{Err: wrappedErr}
		assert.Equal(t, wrappedErr, err.Unwrap())
	})

	t.Run("with nil error", func(t *testing.T) {
		err := EntityConflictError{}
		assert.Nil(t, err.Unwrap())
	})
}

func TestSimpleErrorTypes_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"UnauthorizedError", UnauthorizedError{Message: "unauthorized access"}, "unauthorized access"},
		{"ForbiddenError", ForbiddenError{Message: "forbidden action"}, "forbidden action"},
		{"UnprocessableOperationError", UnprocessableOperationError{Message: "unprocessable operation"}, "unprocessable operation"},
		{"HTTPError", HTTPError{Message: "http error occurred"}, "http error occurred"},
		{"FailedPreconditionError", FailedPreconditionError{Message: "precondition failed"}, "precondition failed"},
		{"InternalServerError", InternalServerError{Message: "internal server error"}, "internal server error"},
		{"UnauthorizedError empty", UnauthorizedError{}, ""},
		{"ForbiddenError empty", ForbiddenError{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestResponseError_Error(t *testing.T) {
	err := ResponseError{
		Title:   "Error Title",
		Message: "error message",
		Code:    500,
	}
	assert.Equal(t, "error message", err.Error())
}

func TestValidationKnownFieldsError_Error(t *testing.T) {
	err := ValidationKnownFieldsError{
		Message: "known fields validation error",
		Fields: FieldValidations{
			"field1": "error1",
		},
	}
	assert.Equal(t, "known fields validation error", err.Error())
}

func TestValidationUnknownFieldsError_Error(t *testing.T) {
	err := ValidationUnknownFieldsError{
		Message: "unknown fields validation error",
		Fields: UnknownFields{
			"unknownField": "value",
		},
	}
	assert.Equal(t, "unknown fields validation error", err.Error())
}

func TestValidateInternalError(t *testing.T) {
	originalErr := errors.New("database connection failed")
	result := ValidateInternalError(originalErr, "User")

	internalErr, ok := result.(InternalServerError)
	require.True(t, ok, "expected InternalServerError type")
	assert.Equal(t, "User", internalErr.EntityType)
	assert.Equal(t, constant.ErrInternalServer.Error(), internalErr.Code)
	assert.Equal(t, "Internal Server Error", internalErr.Title)
	assert.Equal(t, originalErr, internalErr.Err)
}

func TestValidateBadRequestFieldsError(t *testing.T) {
	tests := []struct {
		name               string
		requiredFields     map[string]string
		knownInvalidFields map[string]string
		entityType         string
		unknownFields      map[string]any
		expectType         string
		expectCode         string
	}{
		{
			name:               "unknown fields error",
			requiredFields:     nil,
			knownInvalidFields: nil,
			entityType:         "User",
			unknownFields:      map[string]any{"extra": "value"},
			expectType:         "ValidationUnknownFieldsError",
			expectCode:         constant.ErrUnexpectedFieldsInTheRequest.Error(),
		},
		{
			name:               "required fields error",
			requiredFields:     map[string]string{"name": "required"},
			knownInvalidFields: nil,
			entityType:         "User",
			unknownFields:      nil,
			expectType:         "ValidationKnownFieldsError",
			expectCode:         constant.ErrMissingFieldsInRequest.Error(),
		},
		{
			name:               "known invalid fields error",
			requiredFields:     nil,
			knownInvalidFields: map[string]string{"email": "invalid format"},
			entityType:         "User",
			unknownFields:      nil,
			expectType:         "ValidationKnownFieldsError",
			expectCode:         constant.ErrBadRequest.Error(),
		},
		{
			name:               "all empty returns error",
			requiredFields:     nil,
			knownInvalidFields: nil,
			entityType:         "User",
			unknownFields:      nil,
			expectType:         "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateBadRequestFieldsError(tt.requiredFields, tt.knownInvalidFields, tt.entityType, tt.unknownFields)

			assert.NotNil(t, result)

			switch tt.expectType {
			case "ValidationUnknownFieldsError":
				unknownErr, ok := result.(ValidationUnknownFieldsError)
				require.True(t, ok, "expected ValidationUnknownFieldsError type")
				assert.Equal(t, tt.expectCode, unknownErr.Code)
			case "ValidationKnownFieldsError":
				knownErr, ok := result.(ValidationKnownFieldsError)
				require.True(t, ok, "expected ValidationKnownFieldsError type")
				assert.Equal(t, tt.expectCode, knownErr.Code)
			case "error":
				assert.Error(t, result)
			}
		})
	}
}

func TestValidateBusinessError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		entityType string
		args       []any
		expectType string
	}{
		{
			name:       "entity not found",
			err:        constant.ErrEntityNotFound,
			entityType: "User",
			expectType: "EntityNotFoundError",
		},
		{
			name:       "invalid query parameter",
			err:        constant.ErrInvalidQueryParameter,
			entityType: "Example",
			args:       []any{"invalid_param"},
			expectType: "ValidationError",
		},
		{
			name:       "invalid date format",
			err:        constant.ErrInvalidDateFormat,
			entityType: "Report",
			expectType: "ValidationError",
		},
		{
			name:       "invalid final date",
			err:        constant.ErrInvalidFinalDate,
			entityType: "Report",
			expectType: "ValidationError",
		},
		{
			name:       "date range exceeds limit",
			err:        constant.ErrDateRangeExceedsLimit,
			entityType: "Report",
			args:       []any{12},
			expectType: "ValidationError",
		},
		{
			name:       "invalid date range",
			err:        constant.ErrInvalidDateRange,
			entityType: "Report",
			expectType: "ValidationError",
		},
		{
			name:       "pagination limit exceeded",
			err:        constant.ErrPaginationLimitExceeded,
			entityType: "List",
			args:       []any{100},
			expectType: "ValidationError",
		},
		{
			name:       "invalid sort order",
			err:        constant.ErrInvalidSortOrder,
			entityType: "List",
			expectType: "ValidationError",
		},
		{
			name:       "action not permitted",
			err:        constant.ErrActionNotPermitted,
			entityType: "User",
			expectType: "ValidationError",
		},
		{
			name:       "parent example id not found",
			err:        constant.ErrParentExampleIDNotFound,
			entityType: "Example",
			expectType: "EntityNotFoundError",
		},
		{
			name:       "metadata key length exceeded",
			err:        constant.ErrMetadataKeyLengthExceeded,
			entityType: "Metadata",
			args:       []any{"longkey", 50},
			expectType: "ValidationError",
		},
		{
			name:       "metadata value length exceeded",
			err:        constant.ErrMetadataValueLengthExceeded,
			entityType: "Metadata",
			args:       []any{"value", 100},
			expectType: "ValidationError",
		},
		{
			name:       "invalid metadata nesting",
			err:        constant.ErrInvalidMetadataNesting,
			entityType: "Metadata",
			args:       []any{"nested_value"},
			expectType: "ValidationError",
		},
		{
			name:       "calculation field type",
			err:        constant.ErrCalculationFieldType,
			entityType: "Calculation",
			expectType: "ValidationError",
		},
		{
			name:       "unmapped error returns wrapped",
			err:        errors.New("unknown error"),
			entityType: "Unknown",
			expectType: "wrapped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateBusinessError(tt.err, tt.entityType, tt.args...)

			assert.NotNil(t, result)

			switch tt.expectType {
			case "EntityNotFoundError":
				var notFoundErr EntityNotFoundError
				assert.True(t, errors.As(result, &notFoundErr), "expected EntityNotFoundError type")
				// Verify error has required fields
				assert.NotEmpty(t, notFoundErr.Message)
				assert.Equal(t, tt.entityType, notFoundErr.EntityType)
			case "ValidationError":
				var validationErr ValidationError
				assert.True(t, errors.As(result, &validationErr), "expected ValidationError type")
				// Verify error message is not empty
				assert.NotEmpty(t, validationErr.Message)
			case "wrapped":
				// Verify the original error is wrapped with context
				assert.True(t, errors.Is(result, tt.err), "wrapped error should contain original error")
				assert.Contains(t, result.Error(), "unhandled business error")
			}
		})
	}
}
