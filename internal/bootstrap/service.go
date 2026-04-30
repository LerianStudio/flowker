// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	libCommons "github.com/LerianStudio/lib-commons/v5/commons"
	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
)

// Service is the application glue where we put all top level components to be used.
type Service struct {
	*HTTPServer
	libLog.Logger
	// TenantInfra holds multi-tenant infrastructure components.
	// Nil when MULTI_TENANT_ENABLED=false (single-tenant mode).
	TenantInfra *TenantInfrastructure
}

// Run starts the application.
// This is the only necessary code to run an app in main.go
func (app *Service) Run() {
	libCommons.NewLauncher(
		libCommons.WithLogger(app.Logger),
		libCommons.RunApp("HTTP Service", app.HTTPServer),
	).Run()
}
