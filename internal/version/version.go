package version

import (
	_ "embed"
	"encoding/json"
	"runtime"
)

//go:embed version.json
var versionJSON []byte

type versionInfo struct {
	Version    string `json:"version"`
	Prerelease string `json:"prerelease"`
}

var (
	Version    = "dev"
	BuildDate  = "unknown"
	CommitHash = "unknown"
	GoVersion  = runtime.Version()
	Arch       = runtime.GOOS + "/" + runtime.GOARCH
)

func init() {
	var info versionInfo
	if err := json.Unmarshal(versionJSON, &info); err == nil {
		Version = info.Version
		if info.Prerelease != "" {
			Version += "-" + info.Prerelease
		}
	}
}

// Info returns version information
func Info() map[string]string {
	return map[string]string{
		"version":     Version,
		"build_date":  BuildDate,
		"commit_hash": CommitHash,
		"go_version":  GoVersion,
		"arch":        Arch,
	}
}
