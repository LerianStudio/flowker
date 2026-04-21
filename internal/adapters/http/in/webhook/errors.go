// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package webhook

import "errors"

// Sentinel errors for webhook handler construction.
var (
	ErrWebhookHandlerNilRegistry       = errors.New("webhook handler: registry cannot be nil")
	ErrWebhookHandlerNilExecuteService = errors.New("webhook handler: execute service cannot be nil")
)
