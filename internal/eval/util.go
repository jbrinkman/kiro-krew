package eval

import (
	"regexp"
	"strings"
	"time"
)

var (
	timestampedDirRe = regexp.MustCompile(`^\d{6}-\d{6}-[a-f0-9]+$`)
	hashOnlyRe       = regexp.MustCompile(`^[a-f0-9]+$`)
)

// generateTimestampPrefix returns UTC timestamp in YYMMDD-HHMMSS format
func generateTimestampPrefix() string {
	return time.Now().UTC().Format("060102-150405")
}

// parseDirectoryName extracts commit hash from both old and new formats
func parseDirectoryName(dirName string) string {
	// New format: YYMMDD-HHMMSS-{hash}
	if timestampedDirRe.MatchString(dirName) {
		parts := strings.Split(dirName, "-")
		if len(parts) == 3 {
			return parts[2]
		}
	}

	// Old format: just {hash}
	if hashOnlyRe.MatchString(dirName) {
		return dirName
	}

	return ""
}
