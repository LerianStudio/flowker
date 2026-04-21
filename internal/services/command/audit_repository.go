// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package command

import (
	"context"

	"github.com/LerianStudio/flowker/pkg/model"
)

// AuditWriteRepository provides write access to the audit trail store.
type AuditWriteRepository interface {
	// Insert persists a new audit entry.
	Insert(ctx context.Context, entry *model.AuditEntry) error
}
