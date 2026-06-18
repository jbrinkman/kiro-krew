package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindBaselineRun finds the baseline run directory by commit hash.
func FindBaselineRun(targetCommit string) (string, error) {
	resultsDir := filepath.Join(".kiro-krew", "evals", "results")
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read results directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		hash := parseDirectoryName(entry.Name())
		if hash == targetCommit {
			return entry.Name(), nil
		}
	}

	return "", fmt.Errorf("baseline run not found for commit %s", targetCommit)
}

// LoadBaselineResults loads the complete baseline dataset (summary + agent results).
func LoadBaselineResults(baselineRun string) (*Summary, map[string]AgentResult, error) {
	baselineDir := filepath.Join(".kiro-krew", "evals", "results", baselineRun)
	
	// Load summary.json
	summaryPath := filepath.Join(baselineDir, "summary.json")
	summaryData, err := os.ReadFile(summaryPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read baseline summary: %w", err)
	}
	
	var summary Summary
	if err := json.Unmarshal(summaryData, &summary); err != nil {
		return nil, nil, fmt.Errorf("failed to parse baseline summary: %w", err)
	}

	// Load agent result files
	agentResults := make(map[string]AgentResult)
	entries, err := os.ReadDir(baselineDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read baseline directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || entry.Name() == "summary.json" {
			continue
		}

		agentFile := filepath.Join(baselineDir, entry.Name())
		data, err := os.ReadFile(agentFile)
		if err != nil {
			continue // Skip files we can't read
		}

		var result AgentResult
		if err := json.Unmarshal(data, &result); err != nil {
			continue // Skip files we can't parse
		}

		agentResults[result.Agent] = result
	}

	return &summary, agentResults, nil
}

// SetBaseline sets the baseline reference by updating the current run's summary.
func SetBaseline(currentCommit, baselineCommit string) error {
	// Find current run directory
	currentRun := ""
	resultsDir := filepath.Join(".kiro-krew", "evals", "results")
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		return fmt.Errorf("failed to read results directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		hash := parseDirectoryName(entry.Name())
		if hash == currentCommit {
			currentRun = entry.Name()
			break
		}
	}

	if currentRun == "" {
		return fmt.Errorf("current run not found for commit %s", currentCommit)
	}

	// Load current summary
	summaryPath := filepath.Join(resultsDir, currentRun, "summary.json")
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		return fmt.Errorf("failed to read current summary: %w", err)
	}

	var summary Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		return fmt.Errorf("failed to parse current summary: %w", err)
	}

	// Update baseline reference
	summary.BaselineCommit = baselineCommit

	// Write updated summary
	updatedData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated summary: %w", err)
	}

	if err := os.WriteFile(summaryPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write updated summary: %w", err)
	}

	return nil
}