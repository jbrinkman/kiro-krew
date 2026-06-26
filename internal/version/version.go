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

// String returns just the version number
func String() string {
	return Version
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

// ShortCommitHash returns a 7-character commit hash or "unknown"
func ShortCommitHash() string {
	return FormatCommitHash(CommitHash)
}

// FormatCommitHash formats any commit hash to 7 characters or "unknown"
func FormatCommitHash(hash string) string {
	if hash == "unknown" || hash == "" {
		return "unknown"
	}
	if len(hash) >= 7 {
		return hash[:7]
	}
	return hash
}
