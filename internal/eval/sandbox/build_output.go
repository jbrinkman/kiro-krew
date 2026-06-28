package sandbox

import (
	"bufio"
	"encoding/json"
	"fmt"
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

// stepRegex matches Docker build step lines
var stepRegex = regexp.MustCompile(`^Step (\d+/\d+) : (.+)$`)

// dockerStreamMessage represents a Docker JSON stream message
type dockerStreamMessage struct {
	Stream      string             `json:"stream"`
	Error       string             `json:"error"`
	ErrorDetail *dockerErrorDetail `json:"errorDetail"`
}

// dockerErrorDetail contains error code and message from Docker
type dockerErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ParseBuildStream parses Docker build output from JSON stream format
func (p *DockerBuildOutputParser) ParseBuildStream(reader io.Reader) (string, error) {
	var result strings.Builder
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var parsed int

	for scanner.Scan() {
		line := scanner.Text()

		var msg dockerStreamMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		parsed++

		// Check for build errors
		if msg.Error != "" {
			errMsg := msg.Error
			if msg.ErrorDetail != nil && msg.ErrorDetail.Message != "" {
				errMsg = msg.ErrorDetail.Message
			}
			return "", fmt.Errorf("docker build error: %s", errMsg)
		}

		if msg.Stream != "" {
			clean := ansiRegex.ReplaceAllString(msg.Stream, "")

			if p.Debug {
				// Format debug output for better readability
				formatted := p.formatDebugOutput(clean)
				result.WriteString(formatted)
			} else {
				result.WriteString(clean)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanning build output: %w", err)
	}

	if parsed == 0 {
		return "", fmt.Errorf("no valid build output received")
	}

	return result.String(), nil
}

// formatDebugOutput formats Docker build steps for better debug readability
func (p *DockerBuildOutputParser) formatDebugOutput(clean string) string {
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return ""
	}

	// Format "Step X/Y : INSTRUCTION" lines
	if match := stepRegex.FindStringSubmatch(clean); match != nil {
		return fmt.Sprintf("Step %s: %s\n", match[1], match[2])
	}

	// Format "--->" hash lines to show progression
	if strings.HasPrefix(clean, " ---> ") {
		hash := strings.TrimPrefix(clean, " ---> ")
		return fmt.Sprintf(" → %s\n", hash)
	}

	// Format "Successfully built" and "Successfully tagged" lines
	if strings.HasPrefix(clean, "Successfully built ") {
		hash := strings.TrimPrefix(clean, "Successfully built ")
		return fmt.Sprintf("✅ Built: %s\n", hash)
	}

	if strings.HasPrefix(clean, "Successfully tagged ") {
		tag := strings.TrimPrefix(clean, "Successfully tagged ")
		return fmt.Sprintf("✅ Tagged: %s\n", tag)
	}

	// Return other lines as-is with newline
	return clean + "\n"
}
