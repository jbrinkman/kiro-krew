package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveRunDirectory handles both old and new format detection
func resolveRunDirectory(runName string) (string, error) {
	resultsDir := filepath.Join(".kiro-krew", "evals", "results")

	// If directory exists as-is, use it
	fullPath := filepath.Join(resultsDir, runName)
	if _, err := os.Stat(fullPath); err == nil {
		return runName, nil
	}

	// If runName looks like a hash only, search for timestamped version
	hash := parseDirectoryName(runName)
	if hash == runName {
		// This is just a hash, look for a timestamped directory with this hash
		entries, err := os.ReadDir(resultsDir)
		if err != nil {
			return "", fmt.Errorf("failed to read results directory: %w", err)
		}

		var latest string
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			dirHash := parseDirectoryName(entry.Name())
			if dirHash == hash {
				latest = entry.Name()
			}
		}
		if latest != "" {
			return latest, nil
		}
	}

	return "", fmt.Errorf("run directory %s not found", runName)
}

// Diff compares two eval runs and prints score/cost deltas.
func Diff(runA, runB string) error {
	return DiffWithOptions(runA, runB, false)
}

// DiffWithImprovement compares two eval runs with improvement metrics.
func DiffWithImprovement(runA, runB string) error {
	return DiffWithOptions(runA, runB, true)
}

// DiffWithOptions compares two eval runs with optional improvement metrics.
func DiffWithOptions(runA, runB string, showImprovement bool) error {
	resultsDir := filepath.Join(".kiro-krew", "evals", "results")

	resolvedA, err := resolveRunDirectory(runA)
	if err != nil {
		return fmt.Errorf("failed to resolve run %s: %w", runA, err)
	}

	resolvedB, err := resolveRunDirectory(runB)
	if err != nil {
		return fmt.Errorf("failed to resolve run %s: %w", runB, err)
	}

	summaryA, err := loadSummary(filepath.Join(resultsDir, resolvedA, "summary.json"))
	if err != nil {
		return fmt.Errorf("failed to load run %s: %w", runA, err)
	}

	summaryB, err := loadSummary(filepath.Join(resultsDir, resolvedB, "summary.json"))
	if err != nil {
		return fmt.Errorf("failed to load run %s: %w", runB, err)
	}

	fmt.Printf("Eval Diff: %s → %s\n", runA, runB)
	fmt.Println(strings.Repeat("─", 60))

	// Per-agent, per-criterion deltas
	allAgents := mergeKeys(summaryA.AgentScores, summaryB.AgentScores)
	for _, agent := range allAgents {
		resultA, errA := loadAgentResult(filepath.Join(resultsDir, resolvedA, agent+".json"))
		resultB, errB := loadAgentResult(filepath.Join(resultsDir, resolvedB, agent+".json"))

		scoreA := summaryA.AgentScores[agent]
		scoreB := summaryB.AgentScores[agent]
		delta := scoreB - scoreA
		indicator := "→"
		if delta > 0.001 {
			indicator = "↑"
		} else if delta < -0.001 {
			indicator = "↓"
		}
		fmt.Printf("\n%s: %.3f → %.3f  %s %+.3f\n", agent, scoreA, scoreB, indicator, delta)

		if errA != nil || errB != nil {
			continue
		}

		// Build criterion averages for each run
		avgA := criterionAverages(resultA)
		avgB := criterionAverages(resultB)
		allCriteria := mergeKeys(avgA, avgB)

		for _, crit := range allCriteria {
			cA, hasA := avgA[crit]
			cB, hasB := avgB[crit]
			if !hasA && !hasB {
				continue
			}
			cDelta := cB - cA
			cInd := "→"
			if cDelta > 0.001 {
				cInd = "↑"
			} else if cDelta < -0.001 {
				cInd = "↓"
			}
			if !hasA {
				fmt.Printf("  %-30s  [new]  %.3f\n", crit, cB)
			} else if !hasB {
				fmt.Printf("  %-30s  %.3f  [removed]\n", crit, cA)
			} else {
				fmt.Printf("  %-30s  %.3f → %.3f  %s %+.3f\n", crit, cA, cB, cInd, cDelta)
			}
		}
	}

	// Cost delta
	fmt.Printf("\nCost Delta:\n")
	costDelta := summaryB.TotalCost.EstimatedUSD - summaryA.TotalCost.EstimatedUSD
	tokenDelta := (summaryB.TotalCost.TokensIn + summaryB.TotalCost.TokensOut) -
		(summaryA.TotalCost.TokensIn + summaryA.TotalCost.TokensOut)
	fmt.Printf("  Tokens: %+d\n", tokenDelta)
	fmt.Printf("  Cost:   %+.6f USD\n", costDelta)

	// Quality per dollar
	fmt.Printf("\nQuality per Dollar:\n")
	avgA := avgScore(summaryA.AgentScores)
	avgB := avgScore(summaryB.AgentScores)
	if summaryA.TotalCost.EstimatedUSD > 0 {
		fmt.Printf("  %s: %.2f quality/$ \n", runA, avgA/summaryA.TotalCost.EstimatedUSD)
	}
	if summaryB.TotalCost.EstimatedUSD > 0 {
		fmt.Printf("  %s: %.2f quality/$\n", runB, avgB/summaryB.TotalCost.EstimatedUSD)
	}

	// Show improvement metrics if requested or available
	if showImprovement || summaryB.ImprovementData != nil {
		displayImprovementMetrics(runA, runB, &summaryA, &summaryB, resolvedA, resolvedB)
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

func loadAgentResult(path string) (AgentResult, error) {
	var r AgentResult
	data, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	err = json.Unmarshal(data, &r)
	return r, err
}

func criterionAverages(result AgentResult) map[string]float64 {
	totals := make(map[string]float64)
	counts := make(map[string]int)
	for _, c := range result.Cases {
		for _, sc := range c.Scores {
			if sc.Skipped || sc.MaxScore == 0 {
				continue
			}
			totals[sc.Name] += float64(sc.Score) / float64(sc.MaxScore)
			counts[sc.Name]++
		}
	}
	avgs := make(map[string]float64)
	for name, total := range totals {
		if counts[name] > 0 {
			avgs[name] = total / float64(counts[name])
		}
	}
	return avgs
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

// displayImprovementMetrics shows improvement metrics when available.
func displayImprovementMetrics(runA, runB string, summaryA, summaryB *Summary, resolvedA, resolvedB string) {
	var improvements *ImprovementData
	
	// Use existing improvement data if available, otherwise calculate it
	if summaryB.ImprovementData != nil {
		improvements = summaryB.ImprovementData
	} else {
		// Load full results to calculate improvements
		resultsDir := filepath.Join(".kiro-krew", "evals", "results")
		
		// Load current results
		currentResults := make(map[string]AgentResult)
		entries, err := os.ReadDir(filepath.Join(resultsDir, resolvedB))
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && entry.Name() != "summary.json" {
					result, err := loadAgentResult(filepath.Join(resultsDir, resolvedB, entry.Name()))
					if err == nil {
						currentResults[result.Agent] = result
					}
				}
			}
		}
		
		// Load baseline results
		baselineResults := make(map[string]AgentResult)
		entries, err = os.ReadDir(filepath.Join(resultsDir, resolvedA))
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") && entry.Name() != "summary.json" {
					result, err := loadAgentResult(filepath.Join(resultsDir, resolvedA, entry.Name()))
					if err == nil {
						baselineResults[result.Agent] = result
					}
				}
			}
		}
		
		improvements = CalculateImprovements(summaryB, currentResults, summaryA, baselineResults)
	}
	
	if improvements == nil {
		return
	}
	
	fmt.Printf("\nImprovement Metrics:\n")
	fmt.Println(strings.Repeat("─", 60))
	
	// Overall improvements
	if improvements.AccuracyChange != 0 {
		indicator := "→"
		if improvements.AccuracyChange > 0 {
			indicator = "↑"
		} else if improvements.AccuracyChange < 0 {
			indicator = "↓"
		}
		fmt.Printf("Overall Accuracy: %s %+.1f%%\n", indicator, improvements.AccuracyChange)
	}
	
	if improvements.ErrorRateChange != 0 {
		indicator := "→"
		if improvements.ErrorRateChange > 0 {
			indicator = "↑"
		} else if improvements.ErrorRateChange < 0 {
			indicator = "↓"
		}
		fmt.Printf("Errors Reduced: %s %+.0f\n", indicator, improvements.ErrorRateChange)
	}
	
	// Per-agent improvements
	if len(improvements.AgentImprovements) > 0 {
		fmt.Printf("\nPer-Agent Improvements:\n")
		for agent, agentImprovement := range improvements.AgentImprovements {
			if agentImprovement.AccuracyGained != 0 || agentImprovement.ErrorsReduced != 0 {
				fmt.Printf("\n%s:\n", agent)
				if agentImprovement.AccuracyGained != 0 {
					indicator := "→"
					if agentImprovement.AccuracyGained > 0 {
						indicator = "↑"
					} else if agentImprovement.AccuracyGained < 0 {
						indicator = "↓"
					}
					fmt.Printf("  Accuracy: %s %+.1f%%\n", indicator, agentImprovement.AccuracyGained)
				}
				if agentImprovement.ErrorsReduced != 0 {
					indicator := "→"
					if agentImprovement.ErrorsReduced > 0 {
						indicator = "↑"
					} else if agentImprovement.ErrorsReduced < 0 {
						indicator = "↓"
					}
					fmt.Printf("  Errors Reduced: %s %+d\n", indicator, agentImprovement.ErrorsReduced)
				}
			}
		}
	}
}
