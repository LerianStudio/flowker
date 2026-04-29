// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package http

import (
	libHTTP "github.com/LerianStudio/lib-commons/v5/commons/net/http"
	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"

	"github.com/LerianStudio/flowker/pkg"
)

// WithError returns an error with the given status code and message.
func WithError(c *fiber.Ctx, err error) error {
	switch e := err.(type) {
	case pkg.EntityNotFoundError:
		return libHTTP.Respond(c, fiber.StatusNotFound, e)
	case pkg.EntityConflictError:
		return libHTTP.Respond(c, fiber.StatusConflict, e)
	case pkg.ValidationError:
		return libHTTP.Respond(c, fiber.StatusBadRequest, pkg.ValidationKnownFieldsError{
			Code:    e.Code,
			Title:   e.Title,
			Message: e.Message,
			Fields:  nil,
		})
	case pkg.UnprocessableOperationError:
		return libHTTP.Respond(c, fiber.StatusUnprocessableEntity, e)
	case pkg.UnauthorizedError:
		return libHTTP.Respond(c, fiber.StatusUnauthorized, e)
	case pkg.ForbiddenError:
		return libHTTP.Respond(c, fiber.StatusForbidden, e)
	case pkg.ValidationKnownFieldsError, pkg.ValidationUnknownFieldsError:
		return libHTTP.Respond(c, fiber.StatusBadRequest, e)
	case pkg.ResponseError:
		var rErr pkg.ResponseError

		_ = errors.As(err, &rErr)

		return libHTTP.Respond(c, rErr.Code, rErr)
	default:
		var iErr pkg.InternalServerError

		_ = errors.As(pkg.ValidateInternalError(err, ""), &iErr)

		return libHTTP.Respond(c, fiber.StatusInternalServerError, iErr)
	}
}
