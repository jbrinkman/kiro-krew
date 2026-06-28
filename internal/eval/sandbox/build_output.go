package sandbox

import (
	"bufio"
	"encoding/json"
	"io"
	"regexp"
	"strings"
)

// DockerBuildOutputParser parses Docker build JSON streams into clean text
type DockerBuildOutputParser struct {
	Debug bool
}

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// dockerStreamMessage represents a Docker JSON stream message
type dockerStreamMessage struct {
	Stream string `json:"stream"`
}

// ParseBuildStream parses Docker build output from JSON stream format
func (p *DockerBuildOutputParser) ParseBuildStream(reader io.Reader) (string, error) {
	var result strings.Builder
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		if p.Debug {
			result.WriteString(line + "\n")
			continue
		}

		var msg dockerStreamMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// Handle malformed JSON gracefully - skip invalid lines
			continue
		}

		if msg.Stream != "" {
			// Strip ANSI escape sequences
			clean := ansiRegex.ReplaceAllString(msg.Stream, "")
			result.WriteString(clean)
		}
	}

	return result.String(), scanner.Err()
}
