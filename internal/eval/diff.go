package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Diff compares two eval runs and prints score/cost deltas.
func Diff(runA, runB string) error {
	resultsDir := filepath.Join(".kiro-krew", "evals", "results")

	summaryA, err := loadSummary(filepath.Join(resultsDir, runA, "summary.json"))
	if err != nil {
		return fmt.Errorf("failed to load run %s: %w", runA, err)
	}

	summaryB, err := loadSummary(filepath.Join(resultsDir, runB, "summary.json"))
	if err != nil {
		return fmt.Errorf("failed to load run %s: %w", runB, err)
	}

	fmt.Printf("Eval Diff: %s → %s\n", runA, runB)
	fmt.Println(strings.Repeat("─", 60))

	// Per-agent score deltas
	fmt.Println("\nScore Deltas (normalized 0-1):")
	allAgents := mergeKeys(summaryA.AgentScores, summaryB.AgentScores)
	for _, agent := range allAgents {
		scoreA := summaryA.AgentScores[agent]
		scoreB := summaryB.AgentScores[agent]
		delta := scoreB - scoreA
		indicator := "→"
		if delta > 0 {
			indicator = "↑"
		} else if delta < 0 {
			indicator = "↓"
		}
		fmt.Printf("  %-12s  %.3f → %.3f  %s %+.3f\n", agent, scoreA, scoreB, indicator, delta)
	}

	// Cost delta
	fmt.Println("\nCost Delta:")
	costDelta := summaryB.TotalCost.EstimatedUSD - summaryA.TotalCost.EstimatedUSD
	tokenDelta := (summaryB.TotalCost.TokensIn + summaryB.TotalCost.TokensOut) -
		(summaryA.TotalCost.TokensIn + summaryA.TotalCost.TokensOut)
	fmt.Printf("  Tokens: %+d\n", tokenDelta)
	fmt.Printf("  Cost:   %+.6f USD\n", costDelta)

	// Quality per dollar
	fmt.Println("\nQuality per Dollar:")
	avgA := avgScore(summaryA.AgentScores)
	avgB := avgScore(summaryB.AgentScores)
	if summaryA.TotalCost.EstimatedUSD > 0 {
		fmt.Printf("  %s: %.2f quality/$ \n", runA, avgA/summaryA.TotalCost.EstimatedUSD)
	}
	if summaryB.TotalCost.EstimatedUSD > 0 {
		fmt.Printf("  %s: %.2f quality/$\n", runB, avgB/summaryB.TotalCost.EstimatedUSD)
	}

	return nil
}

func loadSummary(path string) (Summary, error) {
	var s Summary
	data, err := os.ReadFile(path)
	if err != nil {
		return s, err
	}
	err = json.Unmarshal(data, &s)
	return s, err
}

func mergeKeys(a, b map[string]float64) []string {
	seen := make(map[string]bool)
	var keys []string
	for k := range a {
		if !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	for k := range b {
		if !seen[k] {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	return keys
}

func avgScore(scores map[string]float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	var sum float64
	for _, v := range scores {
		sum += v
	}
	return sum / float64(len(scores))
}
