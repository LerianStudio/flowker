// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
	"github.com/gofiber/fiber/v2"
)

// drainingState indicates whether the service is draining (shutting down).
// When true, /readyz returns 503 to stop K8s from routing new traffic.
var drainingState atomic.Bool

func init() {
	drainingState.Store(false)
}

// IsDraining returns whether the service is in drain mode.
// Used by /readyz to short-circuit to 503 during graceful shutdown.
func IsDraining() bool {
	return drainingState.Load()
}

// SetDraining sets the draining state.
// Called when SIGTERM/SIGINT is received to signal graceful shutdown.
func SetDraining(draining bool) {
	drainingState.Store(draining)
}

// DefaultDrainGracePeriod is the default time to wait after setting draining
// before shutting down the server.
// Should be >= K8s periodSeconds * failureThreshold + buffer.
// Default: 12 seconds (covers 5s period * 2 failures + 2s buffer).
const DefaultDrainGracePeriod = 12 * time.Second

// GracefulShutdownConfig configures graceful shutdown behavior.
type GracefulShutdownConfig struct {
	App              *fiber.App                      // The Fiber app to shutdown
	Logger           libLog.Logger                   // Logger for shutdown events
	DrainGracePeriod time.Duration                   // Time to wait for K8s to observe 503
	OnShutdown       func(ctx context.Context) error // Optional cleanup callback
}

// RegisterGracefulShutdown sets up SIGTERM/SIGINT handling with drain coupling.
// This sets the drain state immediately on signal receipt, then waits for
// the grace period before actually shutting down the server.
//
// Flow:
// 1. SIGTERM/SIGINT received
// 2. Set draining state (IsDraining() returns true)
// 3. /readyz now returns 503
// 4. Wait DrainGracePeriod for K8s to stop routing traffic
// 5. Shutdown the Fiber app
// 6. Run OnShutdown callback if provided
func RegisterGracefulShutdown(cfg GracefulShutdownConfig) {
	if cfg.DrainGracePeriod == 0 {
		cfg.DrainGracePeriod = DefaultDrainGracePeriod
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sig
		signal.Stop(sig) // Stop receiving signals to prevent leak

		cfg.Logger.Log(context.Background(), libLog.LevelInfo, "received_shutdown_signal",
			libLog.String("state", "draining"))

		// Set draining state - /readyz will now return 503
		SetDraining(true)

		// Wait for K8s to observe 503 on /readyz and stop routing
		cfg.Logger.Log(context.Background(), libLog.LevelInfo, "waiting_for_drain_grace_period",
			libLog.String("grace_period", cfg.DrainGracePeriod.String()))
		time.Sleep(cfg.DrainGracePeriod)

		// Shutdown the server
		cfg.Logger.Log(context.Background(), libLog.LevelInfo, "shutting_down_server")
		if err := cfg.App.Shutdown(); err != nil {
			cfg.Logger.Log(context.Background(), libLog.LevelError, "server_shutdown_error",
				libLog.Any("error.message", err))
		}

		// Run cleanup if provided
		if cfg.OnShutdown != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if err := cfg.OnShutdown(ctx); err != nil {
				cfg.Logger.Log(context.Background(), libLog.LevelError, "cleanup_error",
					libLog.Any("error.message", err))
			}
		}
	}()
}
