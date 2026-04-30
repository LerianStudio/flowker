// Copyright (c) 2026 Lerian Studio. All rights reserved.
// Use of this source code is governed by the Elastic License 2.0
// that can be found in the LICENSE file.

package bootstrap

import (
	"context"
	"sync/atomic"

	libLog "github.com/LerianStudio/lib-commons/v5/commons/log"
)

// selfProbeOK indicates whether the startup self-probe passed.
// Initialized to false; set to true only after all dependencies are verified healthy.
var selfProbeOK atomic.Bool

func init() {
	selfProbeOK.Store(false) // unhealthy until proven otherwise
}

// SelfProbeOK returns whether the startup self-probe passed.
// Used by /health to gate liveness: if false, /health returns 503.
func SelfProbeOK() bool {
	return selfProbeOK.Load()
}

// SetSelfProbeOK sets the self-probe result.
// Should only be called once at startup after RunSelfProbe completes.
func SetSelfProbeOK(ok bool) {
	selfProbeOK.Store(ok)
}

// SelfProbeResult represents the result of a dependency's self-probe.
type SelfProbeResult struct {
	Name   string // Dependency name (e.g., "mongodb", "postgresql")
	Status string // up, down, skipped
	Error  error  // Non-nil when status is "down"
}

// LogSelfProbeStart logs the start of the self-probe.
func LogSelfProbeStart(logger libLog.Logger) {
	logger.Log(context.Background(), libLog.LevelInfo, "startup_self_probe_started",
		libLog.String("probe", "self"))
}

// LogSelfProbeResult logs a single dependency's probe result.
func LogSelfProbeResult(logger libLog.Logger, result SelfProbeResult) {
	if result.Status == "up" || result.Status == "skipped" {
		logger.Log(context.Background(), libLog.LevelInfo, "self_probe_check",
			libLog.String("probe", "self"),
			libLog.String("name", result.Name),
			libLog.String("status", result.Status),
		)
	} else {
		logger.Log(context.Background(), libLog.LevelError, "self_probe_check",
			libLog.String("probe", "self"),
			libLog.String("name", result.Name),
			libLog.String("status", result.Status),
			libLog.Any("error.message", result.Error),
		)
	}
}

// LogSelfProbeComplete logs the completion of the self-probe.
func LogSelfProbeComplete(logger libLog.Logger, passed bool) {
	if passed {
		logger.Log(context.Background(), libLog.LevelInfo, "startup_self_probe_passed",
			libLog.String("probe", "self"))
	} else {
		logger.Log(context.Background(), libLog.LevelError, "startup_self_probe_failed",
			libLog.String("probe", "self"))
	}
}
