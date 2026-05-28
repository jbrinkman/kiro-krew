package version

import (
	"runtime"
)

// Build-time variables set via ldflags
var (
	Version   = "dev"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
	Arch      = runtime.GOOS + "/" + runtime.GOARCH
)

// Info returns version information
func Info() map[string]string {
	return map[string]string{
		"version":    Version,
		"build_date": BuildDate,
		"go_version": GoVersion,
		"arch":       Arch,
	}
}
