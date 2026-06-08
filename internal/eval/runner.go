package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

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

	resultsDir := filepath.Join(".kiro-krew", "evals", "results", gitHash)
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
				score.Score, score.Reasoning, score.Skipped = scoreDeterministic(criterion, tc)
			} else {
				// LLM-judged criteria
				if tc.Output == "" {
					score.Score = 0
					score.Skipped = true
					score.Reasoning = "no output available for LLM judging"
				} else {
					score.Score, score.Reasoning, score.Skipped = scoreLLMJudge(criterion, tc)
				}
			}

			cr.Scores = append(cr.Scores, score)
		}

		// Estimate cost from output length as proxy when no real token data
		if tc.Output != "" {
			cr.Cost = estimateCost(tc.Input, tc.Output)
		}

		result.Cases = append(result.Cases, cr)
	}

	return result
}

func scoreDeterministic(criterion Criterion, tc TestCase) (int, string, bool) {
	if tc.Output == "" {
		return 0, "no output to evaluate", true
	}

	maxScore := parseMaxScore(criterion.Scoring)

	// Heuristic deterministic checks based on criterion name patterns
	switch {
	case strings.Contains(criterion.Name, "completeness"):
		// Check for required sections
		sections := []string{"## ", "### "}
		found := 0
		for _, s := range sections {
			if strings.Contains(tc.Output, s) {
				found++
			}
		}
		score := (found * maxScore) / len(sections)
		return score, fmt.Sprintf("found %d/%d expected structural elements", found, len(sections)), false

	case strings.Contains(criterion.Name, "file_reference"):
		// Extract candidate file paths and verify they exist
		lines := strings.Split(tc.Output, "\n")
		var candidates []string
		for _, word := range lines {
			for _, w := range strings.Fields(word) {
				w = strings.Trim(w, "`*_-•")
				if strings.Contains(w, "/") && (strings.HasSuffix(w, ".go") || strings.HasSuffix(w, ".ts") || strings.HasSuffix(w, ".yaml") || strings.HasSuffix(w, ".md") || strings.HasSuffix(w, ".json") || strings.HasSuffix(w, ".sh")) {
					candidates = append(candidates, w)
				}
			}
		}
		if len(candidates) == 0 {
			return 1, "no file references found", false
		}
		verified := 0
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				verified++
			}
		}
		score := (verified * maxScore) / len(candidates)
		if score < 1 {
			score = 1
		}
		return score, fmt.Sprintf("%d/%d referenced files verified on disk", verified, len(candidates)), false

	case strings.Contains(criterion.Name, "file_naming"):
		// Check documenter output references correct path format
		if strings.Contains(tc.Output, "app_docs/feature-") {
			return maxScore, "output references correct app_docs/feature-* path", false
		}
		return 1, "no app_docs/feature-* path found in output", false

	default:
		// Unknown deterministic criterion — skip rather than award false credit
		return 0, fmt.Sprintf("no deterministic checker implemented for %q", criterion.Name), true
	}
}

func scoreLLMJudge(criterion Criterion, tc TestCase) (int, string, bool) {
	prompt := fmt.Sprintf(`Evaluate this output against the criterion. Respond with JSON only:
{"score": <number>, "reasoning": "<explanation>", "pass": <boolean>}

CRITERION: %s
DESCRIPTION: %s  
SCORING: %s

OUTPUT TO EVALUATE:
%s`, criterion.Name, criterion.Description, criterion.Scoring, tc.Output)

	cmd := exec.Command("kiro", "chat", prompt)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Sprintf("kiro chat failed: %v", err), true
	}

	var response struct {
		Score     int    `json:"score"`
		Reasoning string `json:"reasoning"`
		Pass      bool   `json:"pass"`
	}

	if err := json.Unmarshal(output, &response); err != nil {
		return 0, fmt.Sprintf("JSON parse error: %v", err), true
	}

	return response.Score, response.Reasoning, false
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
			s.TotalCost.TokensIn += c.Cost.TokensIn
			s.TotalCost.TokensOut += c.Cost.TokensOut
			s.TotalCost.EstimatedUSD += c.Cost.EstimatedUSD
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

func getGitShortHash() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
