// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package api

// ErrorResponse represents a standard API error response.
// Used in Swagger documentation for error responses.
type ErrorResponse struct {
	Code    string `json:"code" validate:"required" example:"FLK-0001"`
	Title   string `json:"title" validate:"required" example:"Bad Request"`
	Message string `json:"message" validate:"required" example:"Invalid input provided"`
}
