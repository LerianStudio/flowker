// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package constant

const (
	// ApplicationName is the name of the Flowker service.
	ApplicationName = "flowker"

	// ModuleManager is the single module name for dispatch layer registration.
	// Flowker is a single-module service with MongoDB (primary) + PostgreSQL (audit).
	ModuleManager = "manager"
)
