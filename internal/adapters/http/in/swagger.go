// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package in

import (
	"github.com/LerianStudio/flowker/api"

	libCommons "github.com/LerianStudio/lib-commons/v4/commons"
	"github.com/gofiber/fiber/v2"
)

// SwaggerConfig holds swagger documentation configuration.
type SwaggerConfig struct {
	Title       string
	Description string
	Version     string
	Host        string
	BasePath    string
	LeftDelim   string
	RightDelim  string
	Schemes     string
}

// WithSwaggerConfig sets the Swagger configuration for the API documentation from the provided config.
func WithSwaggerConfig(cfg SwaggerConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		configMap := map[string]struct {
			value string
			field *string
		}{
			"title":       {cfg.Title, &api.SwaggerInfo.Title},
			"description": {cfg.Description, &api.SwaggerInfo.Description},
			"version":     {cfg.Version, &api.SwaggerInfo.Version},
			"host":        {cfg.Host, &api.SwaggerInfo.Host},
			"basePath":    {cfg.BasePath, &api.SwaggerInfo.BasePath},
			"leftDelim":   {cfg.LeftDelim, &api.SwaggerInfo.LeftDelim},
			"rightDelim":  {cfg.RightDelim, &api.SwaggerInfo.RightDelim},
		}

		for key, item := range configMap {
			if !libCommons.IsNilOrEmpty(&item.value) {
				if key == "host" && libCommons.ValidateServerAddress(item.value) == "" {
					continue
				}

				*item.field = item.value
			}
		}

		if cfg.Schemes != "" {
			api.SwaggerInfo.Schemes = []string{cfg.Schemes}
		}

		return c.Next()
	}
}
