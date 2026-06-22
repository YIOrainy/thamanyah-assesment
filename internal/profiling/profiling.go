// Package profiling starts the Pyroscope continuous-profiling agent.
package profiling

import (
	"log/slog"
	"runtime"

	"github.com/grafana/pyroscope-go"

	"github.com/yazeedalorainy/thmanyah/internal/config"
)

// Start launches the Pyroscope agent when enabled, pushing to the configured
// endpoint under appName. It returns a stop function (a no-op when disabled or
// on error). Mutex/block rates are set so those profile types carry data.
func Start(cfg config.ProfilingConfig, appName string) func() {
	if !cfg.Enabled {
		return func() {}
	}
	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	p, err := pyroscope.Start(pyroscope.Config{
		ApplicationName: appName,
		ServerAddress:   cfg.Endpoint,
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects, pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects, pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount, pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount, pyroscope.ProfileBlockDuration,
		},
	})
	if err != nil {
		slog.Error("start profiling", "error", err)
		return func() {}
	}
	slog.Info("profiling started", "endpoint", cfg.Endpoint, "app", appName)
	return func() { _ = p.Stop() }
}
