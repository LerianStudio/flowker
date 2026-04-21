// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

// Package s3 provides the AWS S3 object storage provider registration.
package s3

import (
	"fmt"

	"github.com/LerianStudio/flowker/pkg/executor"
	"github.com/LerianStudio/flowker/pkg/executor/base"
	httpExecutor "github.com/LerianStudio/flowker/pkg/executors/http"
)

// ProviderID is the unique identifier for the S3 provider.
const ProviderID executor.ProviderID = "s3"

// Register registers the S3 provider with its executors into the given catalog.
func Register(catalog executor.Catalog) error {
	provider, err := base.NewProvider(
		ProviderID,
		"S3",
		"AWS S3 object storage provider for uploading and retrieving files",
		"v1",
		providerConfigSchema,
	)
	if err != nil {
		return fmt.Errorf("failed to create S3 provider: %w", err)
	}

	putObjectExec, err := base.NewExecutor(
		"s3.put-object",
		"Put Object",
		"S3",
		"v1",
		ProviderID,
		putObjectSchema,
	)
	if err != nil {
		return fmt.Errorf("failed to create S3 PutObject executor: %w", err)
	}

	return catalog.RegisterProvider(provider, []executor.ExecutorRegistration{
		{
			Executor: putObjectExec,
			Runner:   httpExecutor.NewRunner(),
		},
	})
}

// providerConfigSchema is the JSON Schema for S3 provider configuration.
const providerConfigSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "region": {
      "type": "string",
      "description": "AWS region (e.g., us-east-1)"
    },
    "bucket": {
      "type": "string",
      "description": "S3 bucket name"
    },
    "access_key_id": {
      "type": "string",
      "description": "AWS access key ID"
    },
    "secret_access_key": {
      "type": "string",
      "description": "AWS secret access key"
    }
  },
  "required": ["region", "bucket", "access_key_id", "secret_access_key"]
}`

// putObjectSchema is the JSON Schema for the S3 PutObject executor.
const putObjectSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "key": {
      "type": "string",
      "minLength": 1,
      "description": "Object key (path) in the S3 bucket"
    },
    "content_type": {
      "type": "string",
      "default": "application/octet-stream",
      "description": "MIME type of the object"
    },
    "body": {
      "description": "Object content to upload"
    }
  },
  "required": ["key"]
}`
