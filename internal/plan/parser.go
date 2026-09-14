package plan

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParsePlanFromMarkdown extracts and parses a plan artifact from markdown content.
// It searches for a fenced code block with the language identifier 'kiro-plan',
// extracts the YAML content, and unmarshals it into a Plan struct.
//
// Returns:
//   - (*Plan, nil) if a valid plan is found and parsed successfully
//   - (nil, nil) if no plan block is found (backward compatibility)
//   - (nil, error) if a plan block is found but is invalid or multiple blocks exist
func ParsePlanFromMarkdown(content []byte) (*Plan, error) {
	// Regex pattern to match fenced code blocks with 'kiro-plan' language identifier
	// Matches both ```kiro-plan and ~~~kiro-plan with optional whitespace
	pattern := regexp.MustCompile("(?m)^```+\\s*kiro-plan\\s*$")
	matches := pattern.FindAllIndex(content, -1)

	if len(matches) == 0 {
		// No plan block found - this is okay for backward compatibility
		return nil, nil
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple plan blocks found (expected at most 1, found %d)", len(matches))
	}

	// Extract the YAML content between the opening and closing fence
	startIdx := matches[0][1] // End of opening fence marker

	// Find the closing fence (any line starting with ``` or ~~~)
	closingPattern := regexp.MustCompile("(?m)^```+\\s*$")
	remaining := content[startIdx:]
	closingMatch := closingPattern.FindIndex(remaining)

	if closingMatch == nil {
		return nil, fmt.Errorf("unclosed plan code block (no closing fence found)")
	}

	endIdx := startIdx + closingMatch[0]
	yamlContent := content[startIdx:endIdx]

	// Trim leading/trailing whitespace from YAML content
	yamlStr := strings.TrimSpace(string(yamlContent))

	if yamlStr == "" {
		return nil, fmt.Errorf("empty plan block")
	}

	// Parse YAML into Plan struct
	var plan Plan
	if err := yaml.Unmarshal([]byte(yamlStr), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan YAML: %w", err)
	}

	return &plan, nil
}

// ParsePlanFromFile reads a file and parses its plan artifact
func ParsePlanFromFile(filePath string) (*Plan, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}
	return ParsePlanFromMarkdown(content)
}
