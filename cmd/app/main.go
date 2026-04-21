// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package main

import (
	"fmt"

	"github.com/LerianStudio/flowker/internal/bootstrap"
	"github.com/LerianStudio/flowker/pkg"
)

// @title					Flowker API
// @version					1.0.0
// @description				Workflow orchestration platform for financial validation
// @termsOfService			http://swagger.io/terms/
// @host					localhost:4021
// @BasePath					/
func main() {
	pkg.InitLocalEnvConfig()

	service, err := bootstrap.InitServers()
	if err != nil {
		panic(fmt.Errorf("failed to initialize servers: %w", err))
	}

	service.Run()
}
