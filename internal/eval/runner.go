package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ansiRegex matches all CSI (Control Sequence Introducer) escape sequences.
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// Run executes the evaluation for all agents (or a specific agent) and writes results.
func Run(agent string) error {
	gitHash, err := getGitShortHash()
	if err != nil {
		return fmt.Errorf("failed to get git hash: %w", err)
	}

	rubrics, err := loadRubrics(agent)
	if err != nil {
		return err
	}

	if len(rubrics) == 0 {
		return fmt.Errorf("no rubrics found in .kiro-krew/evals/rubrics/")
	}

	timestamp := generateTimestampPrefix()
	resultsDir := filepath.Join(".kiro-krew", "evals", "results", timestamp+"-"+gitHash)
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	var allResults []AgentResult

	for _, rubric := range rubrics {
		cases, err := loadCases(rubric.Agent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: no test cases for agent %s: %v\n", rubric.Agent, err)
			continue
		}

		result := evaluate(rubric, cases, gitHash)
		allResults = append(allResults, result)

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal results for %s: %w", rubric.Agent, err)
		}

		outPath := filepath.Join(resultsDir, rubric.Agent+".json")
		if err := os.WriteFile(outPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write results for %s: %w", rubric.Agent, err)
		}

		fmt.Printf("✓ %s: %d cases evaluated\n", rubric.Agent, len(result.Cases))
	}

	summary := buildSummary(allResults, gitHash)
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary: %w", err)
	}

	if err := os.WriteFile(filepath.Join(resultsDir, "summary.json"), data, 0644); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	fmt.Printf("\nResults written to %s\n", resultsDir)
	return nil
}

func evaluate(rubric Rubric, cases []TestCase, gitHash string) AgentResult {
	result := AgentResult{
		Agent:   rubric.Agent,
		GitHash: gitHash,
	}

	for _, tc := range cases {
		cr := CaseResult{CaseName: tc.Name}

		// Assemble prompt and invoke agent
		prompt, err := assemblePrompt(tc.Setup, tc.Input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to assemble prompt for %s: %v\n", tc.Name, err)
			cr.ActualOutput = ""
		} else {
			actualOutput, cost, err := invokeAgent(rubric.Agent, prompt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: agent invocation failed for %s: %v\n", tc.Name, err)
				cr.ActualOutput = ""
			} else {
				cr.ActualOutput = actualOutput
				cr.AgentCost = cost
			}
		}

		for _, criterion := range rubric.Criteria {
			if criterion.Type == "cost" {
				continue // cost is tracked separately
			}

			score := CriterionScore{
				Name:          criterion.Name,
				MaxScore:      parseMaxScore(criterion.Scoring),
				Deterministic: criterion.Deterministic,
			}

			if criterion.Deterministic {
				score.Score, score.Reasoning, score.Skipped = scoreDeterministic(criterion, tc, cr.ActualOutput)
			} else {
				// LLM-judged criteria
				if cr.ActualOutput == "" {
					score.Score = 0
					score.Skipped = true
					score.Reasoning = "no output available for LLM judging"
				} else {
					judgeCost, judgeScore, reasoning, skipped := scoreLLMJudge(criterion, tc, cr.ActualOutput)
					cr.JudgeCost.TokensIn += judgeCost.TokensIn
					cr.JudgeCost.TokensOut += judgeCost.TokensOut
					cr.JudgeCost.EstimatedUSD += judgeCost.EstimatedUSD
					score.Score = judgeScore
					score.Reasoning = reasoning
					score.Skipped = skipped
				}
			}

			cr.Scores = append(cr.Scores, score)
		}

		result.Cases = append(result.Cases, cr)
	}

	return result
}

// assemblePrompt combines setup entries and input into a complete agent prompt
func assemblePrompt(setup []SetupEntry, input string) (string, error) {
	var parts []string

	for _, entry := range setup {
		switch entry.Type {
		case "text":
			if entry.Label != "" {
				parts = append(parts, fmt.Sprintf("=== %s ===\n%s", entry.Label, entry.Content))
			} else {
				parts = append(parts, entry.Content)
			}
		case "file":
			if entry.Path == "" {
				return "", fmt.Errorf("setup entry '%s' has type 'file' but no path", entry.Label)
			}
			content, err := os.ReadFile(entry.Path)
			if err != nil {
				return "", fmt.Errorf("failed to read setup file %s: %w", entry.Path, err)
			}
			label := entry.Label
			if label == "" {
				label = entry.Path
			}
			parts = append(parts, fmt.Sprintf("=== %s ===\n%s", label, string(content)))
		case "url":
			return "", fmt.Errorf("url setup entries not yet supported")
		default:
			return "", fmt.Errorf("unknown setup entry type: %s", entry.Type)
		}
	}

	if len(parts) > 0 {
		parts = append(parts, "=== Task ===")
	}
	parts = append(parts, input)

	return strings.Join(parts, "\n\n"), nil
}

// invokeAgent executes kiro-cli with the given agent and prompt
func invokeAgent(agent, prompt string) (string, CostInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kiro-cli", "chat", "--agent", agent, "--no-interactive", "--trust-all-tools")
	cmd.Stdin = strings.NewReader(prompt)

	output, err := cmd.Output()
	if err != nil {
		return "", CostInfo{}, fmt.Errorf("kiro-cli invocation failed: %w", err)
	}

	result := stripANSISequences(string(output))
	cost := estimateCost(prompt, result)

	return result, cost, nil
}

func scoreDeterministic(criterion Criterion, tc TestCase, actualOutput string) (int, string, bool) {
	output := actualOutput
	if output == "" {
		return 0, "no output to evaluate", true
	}

	maxScore := parseMaxScore(criterion.Scoring)

	// Check against context facts if available
	contextViolations := 0
	if len(tc.Context) > 0 {
		lowerOutput := strings.ToLower(output)
		for _, fact := range tc.Context {
			lowerFact := strings.ToLower(fact)
			// Simple heuristic: if output contradicts a context fact
			if strings.Contains(lowerFact, "not") || strings.Contains(lowerFact, "no ") {
				// Look for positive assertions that contradict negative facts
				factWords := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(lowerFact, "not", ""), "no ", ""))
				for _, word := range factWords {
					if len(word) > 3 && strings.Contains(lowerOutput, word) {
						contextViolations++
						break
					}
				}
			}
		}
	}

	// Heuristic deterministic checks based on criterion name patterns
	switch {
	case strings.Contains(criterion.Name, "completeness"):
		// Check for required sections
		sections := []string{"## ", "### "}
		found := 0
		for _, s := range sections {
			if strings.Contains(output, s) {
				found++
			}
		}
		score := (found * maxScore) / len(sections)
		if contextViolations > 0 {
			score = max(1, score-contextViolations)
		}
		return score, fmt.Sprintf("found %d/%d expected structural elements", found, len(sections)), false

	case strings.Contains(criterion.Name, "file_reference"):
		// Extract candidate file paths and verify they exist
		lines := strings.Split(output, "\n")
		var candidates []string
		for _, word := range lines {
			for _, w := range strings.Fields(word) {
				w = strings.Trim(w, "`*_-•")
				if strings.Contains(w, "/") && (strings.HasSuffix(w, ".go") || strings.HasSuffix(w, ".ts") || strings.HasSuffix(w, ".yaml") || strings.HasSuffix(w, ".md") || strings.HasSuffix(w, ".json") || strings.HasSuffix(w, ".sh")) {
					candidates = append(candidates, w)
				}
			}
		}

		// Check against context facts for file existence
		if len(tc.Context) > 0 {
			for i, path := range candidates {
				for _, fact := range tc.Context {
					if strings.Contains(fact, path) && (strings.Contains(fact, "exists") || strings.Contains(fact, "present")) {
						candidates[i] = path + " (verified by context)"
					}
				}
			}
		}

		if len(candidates) == 0 {
			return 1, "no file references found", false
		}
		verified := 0
		for _, path := range candidates {
			cleanPath := strings.Split(path, " ")[0]
			if _, err := os.Stat(cleanPath); err == nil || strings.Contains(path, "verified by context") {
				verified++
			}
		}
		score := (verified * maxScore) / len(candidates)
		if score < 1 {
			score = 1
		}
		if contextViolations > 0 {
			score = max(1, score-contextViolations)
		}
		return score, fmt.Sprintf("%d/%d referenced files verified", verified, len(candidates)), false

	case strings.Contains(criterion.Name, "file_naming"):
		// Check documenter output references correct path format
		if strings.Contains(output, "app_docs/feature-") {
			return maxScore, "output references correct app_docs/feature-* path", false
		}
		return 1, "no app_docs/feature-* path found in output", false

	case strings.Contains(criterion.Name, "acceptance_criteria_quality"):
		// Check for testable acceptance criteria patterns
		testablePatterns := []string{"- [ ]", "- [x]", "```", "go test", "go build", "curl ", "exit code", "status code", "returns ", "outputs "}
		found := 0
		for _, pattern := range testablePatterns {
			if strings.Contains(strings.ToLower(output), pattern) {
				found++
			}
		}
		score := max(1, (found*maxScore)/3)
		if found >= 3 {
			score = maxScore
		}
		if contextViolations > 0 {
			score = max(1, score-contextViolations)
		}
		return score, fmt.Sprintf("found %d testable criteria indicators", found), false

	case strings.Contains(criterion.Name, "test_execution"):
		// Check for evidence of actual command execution and results
		executionIndicators := []string{"exit code", "$ ", "PASS", "FAIL", "ok  \t", "--- FAIL", "--- PASS", "go test", "npm test", "pytest"}
		found := 0
		for _, indicator := range executionIndicators {
			if strings.Contains(output, indicator) {
				found++
			}
		}
		score := max(1, (found*maxScore)/2)
		if found >= 2 {
			score = maxScore
		}
		if contextViolations > 0 {
			score = max(1, score-contextViolations)
		}
		return score, fmt.Sprintf("found %d execution evidence indicators", found), false

	case strings.Contains(criterion.Name, "code_correctness"):
		// Check for code compilation/execution success indicators
		lowerOutput := strings.ToLower(output)
		successIndicators := []string{"build passes", "compiled successfully", "no errors", "exit code 0", "ok  \t"}
		errorIndicators := []string{"compilation error", "syntax error", "build failed", "does not compile"}
		successCount := 0
		errorCount := 0
		for _, indicator := range successIndicators {
			if strings.Contains(lowerOutput, indicator) {
				successCount++
			}
		}
		for _, indicator := range errorIndicators {
			if strings.Contains(lowerOutput, indicator) {
				errorCount++
			}
		}

		score := maxScore / 2
		if successCount > 0 && errorCount == 0 {
			score = maxScore
		}
		if errorCount > 0 {
			score = 1
		}
		if contextViolations > 0 {
			score = max(1, score-contextViolations)
		}

		reasoning := "no clear success or error indicators"
		if successCount > 0 && errorCount == 0 {
			reasoning = "code appears to compile/run successfully"
		} else if errorCount > 0 {
			reasoning = "compilation or runtime errors detected"
		}

		return score, reasoning, false

	case strings.Contains(criterion.Name, "test_coverage"):
		// Check for test file references and test execution
		testPatterns := []string{"_test.go", ".test.js", ".spec.ts", "func Test", "go test", "npm test", "pytest", "describe(", "it("}
		found := 0
		for _, pattern := range testPatterns {
			if strings.Contains(output, pattern) {
				found++
			}
		}
		score := max(1, (found*maxScore)/2)
		if found >= 2 {
			score = maxScore
		}
		if contextViolations > 0 {
			score = max(1, score-contextViolations)
		}
		return score, fmt.Sprintf("found %d test coverage indicators", found), false

	default:
		// Unknown deterministic criterion — skip rather than award false credit
		return 0, fmt.Sprintf("no deterministic checker implemented for %q", criterion.Name), true
	}
}

func runKiroCLI(prompt string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "kiro-cli", "chat", "--no-interactive")
	cmd.Stdin = strings.NewReader(prompt)
	return cmd.Output()
}

func scoreLLMJudge(criterion Criterion, tc TestCase, actualOutput string) (CostInfo, int, string, bool) {
	contextSection := ""
	if len(tc.Context) > 0 {
		contextSection = fmt.Sprintf("\nCONTEXT FACTS (use for hallucination detection — deduct for contradictions):\n%s\n", strings.Join(tc.Context, "\n"))
	}

	expectedSection := ""
	if tc.ExpectedOutput != "" {
		expectedSection = fmt.Sprintf("\nEXPECTED OUTPUT (compare similarity and completeness):\n%s\n", tc.ExpectedOutput)
	}

	prompt := fmt.Sprintf(`Evaluate this output against the criterion.
Wrap your JSON response between ===JSON_START=== and ===JSON_END=== delimiters.

{"score": <number 1-5>, "reasoning": "<explanation>", "pass": <boolean>}

CRITERION: %s
DESCRIPTION: %s

SCORING SCALE:
1 = Does not meet the criterion at all
2 = Minimally addresses the criterion with major gaps
3 = Partially meets the criterion with notable room for improvement
4 = Mostly meets the criterion with minor gaps
5 = Fully satisfies the criterion
%s%s
INPUT:
%s

ACTUAL OUTPUT TO EVALUATE:
%s`, criterion.Name, criterion.Description, contextSection, expectedSection, tc.Input, actualOutput)

	output, err := runKiroCLI(prompt)
	if err != nil {
		return CostInfo{}, 0, fmt.Sprintf("kiro-cli chat failed: %v", err), true
	}

	raw := string(output)
	start := strings.Index(raw, "===JSON_START===")
	end := strings.Index(raw, "===JSON_END===")
	if start == -1 || end == -1 || end <= start {
		return CostInfo{}, 0, fmt.Sprintf("JSON delimiters not found in output"), true
	}
	jsonStr := raw[start+len("===JSON_START===") : end]
	jsonStr = stripANSISequences(strings.TrimSpace(jsonStr))

	var response struct {
		Score     int    `json:"score"`
		Reasoning string `json:"reasoning"`
		Pass      bool   `json:"pass"`
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonStr)), &response); err != nil {
		return CostInfo{}, 0, fmt.Sprintf("JSON parse error: %v", err), true
	}

	score := max(1, min(response.Score, 5))
	cost := estimateCost(prompt, raw)
	return cost, score, response.Reasoning, false
}

func estimateCost(input, output string) CostInfo {
	// Rough estimate: ~4 chars per token
	tokensIn := len(input) / 4
	tokensOut := len(output) / 4
	// Claude Sonnet pricing estimate: $3/M input, $15/M output
	cost := (float64(tokensIn) * 3.0 / 1_000_000) + (float64(tokensOut) * 15.0 / 1_000_000)
	return CostInfo{
		TokensIn:     tokensIn,
		TokensOut:    tokensOut,
		EstimatedUSD: cost,
	}
}

func buildSummary(results []AgentResult, gitHash string) Summary {
	s := Summary{
		GitHash:     gitHash,
		AgentScores: make(map[string]float64),
	}

	for _, r := range results {
		var totalScore, totalMax float64
		for _, c := range r.Cases {
			for _, sc := range c.Scores {
				if sc.Skipped {
					continue
				}
				totalScore += float64(sc.Score)
				totalMax += float64(sc.MaxScore)
			}
			s.TotalCost.TokensIn += c.AgentCost.TokensIn
			s.TotalCost.TokensOut += c.AgentCost.TokensOut
			s.TotalCost.EstimatedUSD += c.AgentCost.EstimatedUSD
		}
		if totalMax > 0 {
			s.AgentScores[r.Agent] = totalScore / totalMax
		}
	}

	return s
}

func parseMaxScore(scoring string) int {
	// Parse "1-5" -> 5
	parts := strings.Split(scoring, "-")
	if len(parts) == 2 {
		var max int
		fmt.Sscanf(parts[1], "%d", &max)
		if max > 0 {
			return max
		}
	}
	return 5
}

func loadRubrics(agentFilter string) ([]Rubric, error) {
	dir := filepath.Join(".kiro-krew", "evals", "rubrics")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read rubrics directory: %w", err)
	}

	var rubrics []Rubric
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read rubric %s: %w", e.Name(), err)
		}

		var r Rubric
		if err := yaml.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("failed to parse rubric %s: %w", e.Name(), err)
		}

		if agentFilter == "" || r.Agent == agentFilter {
			rubrics = append(rubrics, r)
		}
	}

	return rubrics, nil
}

func loadCases(agent string) ([]TestCase, error) {
	dir := filepath.Join(".kiro-krew", "evals", "cases", agent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cases directory for %s: %w", agent, err)
	}

	var cases []TestCase
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read case %s: %w", e.Name(), err)
		}

		var tc TestCase
		if err := yaml.Unmarshal(data, &tc); err != nil {
			return nil, fmt.Errorf("failed to parse case %s: %w", e.Name(), err)
		}

		tc.Agent = agent
		cases = append(cases, tc)
	}

	return cases, nil
}

func stripANSISequences(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func getGitShortHash() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
