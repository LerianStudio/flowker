// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

//go:build unit

package http

import (
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/LerianStudio/flowker/pkg"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
	}{
		{
			name: "EntityNotFoundError",
			err: pkg.EntityNotFoundError{
				Code:    "NF001",
				Title:   "Not Found",
				Message: "Entity not found",
			},
			expectedStatus: 404,
		},
		{
			name: "EntityConflictError",
			err: pkg.EntityConflictError{
				Code:    "CONF001",
				Title:   "Conflict",
				Message: "Entity conflict",
			},
			expectedStatus: 409,
		},
		{
			name: "ValidationError",
			err: pkg.ValidationError{
				Code:    "VAL001",
				Title:   "Validation Error",
				Message: "Validation failed",
			},
			expectedStatus: 400,
		},
		{
			name: "UnprocessableOperationError",
			err: pkg.UnprocessableOperationError{
				Code:    "UPE001",
				Title:   "Unprocessable",
				Message: "Cannot process",
			},
			expectedStatus: 422,
		},
		{
			name: "UnauthorizedError",
			err: pkg.UnauthorizedError{
				Code:    "AUTH001",
				Title:   "Unauthorized",
				Message: "Not authorized",
			},
			expectedStatus: 401,
		},
		{
			name: "ForbiddenError",
			err: pkg.ForbiddenError{
				Code:    "FORB001",
				Title:   "Forbidden",
				Message: "Access forbidden",
			},
			expectedStatus: 403,
		},
		{
			name: "ValidationKnownFieldsError",
			err: pkg.ValidationKnownFieldsError{
				Code:    "VKF001",
				Title:   "Validation Error",
				Message: "Known fields error",
				Fields:  map[string]string{"field": "error"},
			},
			expectedStatus: 400,
		},
		{
			name: "ValidationUnknownFieldsError",
			err: pkg.ValidationUnknownFieldsError{
				Code:    "VUF001",
				Title:   "Validation Error",
				Message: "Unknown fields error",
				Fields:  map[string]any{"field": "value"},
			},
			expectedStatus: 400,
		},
		{
			name: "ResponseError",
			err: pkg.ResponseError{
				Code:    418,
				Title:   "I'm a teapot",
				Message: "Custom error",
			},
			expectedStatus: 418,
		},
		{
			name:           "Generic error - becomes internal server error",
			err:            errors.New("unknown error"),
			expectedStatus: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			app.Get("/test", func(c *fiber.Ctx) error {
				return WithError(c, tt.err)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestWithError_EntityNotFoundError_Body(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		return WithError(c, pkg.EntityNotFoundError{
			Code:    "NF001",
			Title:   "Not Found",
			Message: "User not found",
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "NF001")
	assert.Contains(t, string(body), "Not Found")
	assert.Contains(t, string(body), "User not found")
}

func TestWithError_ValidationError_Body(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		return WithError(c, pkg.ValidationError{
			Code:    "VAL001",
			Title:   "Validation Error",
			Message: "Field validation failed",
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "VAL001")
	assert.Contains(t, string(body), "Validation Error")
}

func TestWithError_InternalServerError_Body(t *testing.T) {
	app := fiber.New()

	app.Get("/test", func(c *fiber.Ctx) error {
		return WithError(c, errors.New("database connection failed"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 500, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Internal Server Error")
}
