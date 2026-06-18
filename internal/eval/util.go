package eval

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// generateTimestampPrefix returns UTC timestamp in mmddhh-hhmmss format
func generateTimestampPrefix() string {
	now := time.Now().UTC()
	return now.Format("010215-150405")
}

// parseDirectoryName extracts commit hash from both old and new formats
func parseDirectoryName(dirName string) string {
	// New format: mmddhh-hhmmss-{hash}
	if matched, _ := regexp.MatchString(`^\d{6}-\d{6}-[a-f0-9]+$`, dirName); matched {
		parts := strings.Split(dirName, "-")
		if len(parts) == 3 {
			return parts[2]
		}
	}
	
	// Old format: just {hash}
	if matched, _ := regexp.MatchString(`^[a-f0-9]+$`, dirName); matched {
		return dirName
	}
	
	return ""
}

// getCommitTimestamp retrieves Unix timestamp for migration
func getCommitTimestamp(hash string) (int64, error) {
	out, err := exec.Command("git", "show", "-s", "--format=%ct", hash).Output()
	if err != nil {
		return 0, err
	}
	
	timestamp, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	return timestamp, err
}