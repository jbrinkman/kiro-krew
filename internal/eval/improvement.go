package eval

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// CalculateImprovements computes improvement metrics against baseline
func CalculateImprovements(currentSummary Summary, baselineHash string) (*ImprovementMetrics, error) {
	if baselineHash == "" {
		return nil, nil // No baseline set
	}

	// Find baseline run
	baselineSummary, err := FindBaselineRun(baselineHash)
	if err != nil {
		return nil, fmt.Errorf("baseline run not found: %w", err)
	}

	metrics := &ImprovementMetrics{
		BaselineHash:       baselineHash,
		AccuracyChange:     make(map[string]float64),
		ErrorRateChange:    make(map[string]int),
		CriterionTrends:    make(map[string][]float64),
		SignificantChanges: []string{},
	}

	// Calculate per-agent accuracy changes
	var totalChange float64
	agentCount := 0
	for agent, currentScore := range currentSummary.AgentScores {
		baselineScore, exists := baselineSummary.AgentScores[agent]
		if !exists {
			continue
		}

		change := (currentScore - baselineScore) * 100 // Convert to percentage
		metrics.AccuracyChange[agent] = change
		totalChange += change
		agentCount++

		// Track significant changes (>5% improvement or >3% regression)
		if math.Abs(change) > 5.0 {
			direction := "improved"
			if change < 0 {
				direction = "regressed"
			}
			metrics.SignificantChanges = append(metrics.SignificantChanges,
				fmt.Sprintf("%s %s by %.2f%%", agent, direction, math.Abs(change)))
		}
	}

	if agentCount > 0 {
		metrics.OverallImprovement = totalChange / float64(agentCount)
	}

	return metrics, nil
}

// FindBaselineRun locates the baseline run by git hash
func FindBaselineRun(baselineHash string) (Summary, error) {
	resultsDir := filepath.Join(".kiro-krew", "evals", "results")
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		return Summary{}, fmt.Errorf("failed to read results: %w", err)
	}

	// Search for run matching baseline hash
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Parse directory name for hash
		hash := parseDirectoryName(entry.Name())
		if hash == baselineHash {
			summaryPath := filepath.Join(resultsDir, entry.Name(), "summary.json")
			return loadSummary(summaryPath)
		}
	}

	return Summary{}, fmt.Errorf("no run found for baseline hash %s", baselineHash)
}

// AnalyzeTrends analyzes evaluation trends across multiple commits
func AnalyzeTrends(commits []string) ([]TrendPoint, error) {
	var trends []TrendPoint

	for _, commit := range commits {
		summary, err := FindBaselineRun(commit)
		if err != nil {
			continue // Skip missing runs
		}

		// Extract timestamp from directory name
		timestamp := extractTimestamp(commit)

		point := TrendPoint{
			GitHash:   summary.GitHash,
			Timestamp: timestamp,
			Scores:    summary.AgentScores,
			TotalCost: summary.TotalCost.EstimatedUSD,
		}
		trends = append(trends, point)
	}

	return trends, nil
}

// extractTimestamp extracts timestamp from directory name format: YYMMDD-HHMMSS-hash
func extractTimestamp(commit string) string {
	resultsDir := filepath.Join(".kiro-krew", "evals", "results")
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		return ""
	}

	// Search for the directory matching this commit
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		hash := parseDirectoryName(entry.Name())
		if hash == commit {
			// Extract timestamp from directory name: YYMMDD-HHMMSS-hash
			parts := strings.Split(entry.Name(), "-")
			if len(parts) >= 2 {
				// Format: YYMMDD-HHMMSS
				return parts[0] + "-" + parts[1]
			}
		}
	}

	return ""
}

// DiffWithImprovements compares two evaluation runs and highlights improvements
func DiffWithImprovements(runA, runB string) error {
	// This is integrated into the standard Diff() function
	// which now automatically shows improvement metrics when baseline is set
	return Diff(runA, runB)
}

// ShowTrends displays evaluation trends across multiple commits
func ShowTrends(commits []string) error {
	trends, err := AnalyzeTrends(commits)
	if err != nil {
		return fmt.Errorf("failed to analyze trends: %w", err)
	}

	if len(trends) == 0 {
		return fmt.Errorf("no evaluation runs found for specified commits")
	}

	fmt.Println("📈 Evaluation Trends")
	fmt.Println(strings.Repeat("═", 80))

	// Display summary header
	fmt.Printf("\nAnalyzing %d commits:\n", len(trends))
	for i, point := range trends {
		hashShort := point.GitHash
		if len(hashShort) > 7 {
			hashShort = hashShort[:7]
		}
		fmt.Printf("  %d. %s (%s)\n", i+1, hashShort, point.Timestamp)
	}

	fmt.Println(strings.Repeat("─", 80))

	// Display trends per agent
	agents := extractAgentNames(trends)
	if len(agents) == 0 {
		return fmt.Errorf("no agent scores found in trend data")
	}

	for _, agent := range agents {
		fmt.Printf("\n%s:\n", agent)

		for i, point := range trends {
			score, exists := point.Scores[agent]
			if !exists {
				continue
			}

			indicator := "→"
			change := ""
			if i > 0 {
				prevScore, prevExists := trends[i-1].Scores[agent]
				if prevExists {
					delta := score - prevScore
					if delta > 0.001 {
						indicator = "↑"
						change = fmt.Sprintf(" (+%.3f)", delta)
					} else if delta < -0.001 {
						indicator = "↓"
						change = fmt.Sprintf(" (%.3f)", delta)
					}
				}
			}

			hashShort := point.GitHash
			if len(hashShort) > 7 {
				hashShort = hashShort[:7]
			}
			fmt.Printf("  %s %s: %.3f %s%s\n", indicator, hashShort, score, strings.Repeat(" ", 10-len(hashShort)), change)
		}
	}

	// Display overall trend summary
	displayTrendSummary(trends, agents)

	// Display cost trend if available
	if len(trends) > 1 {
		fmt.Println(strings.Repeat("─", 80))
		fmt.Println("\n💰 Cost Trend:")
		for i, point := range trends {
			indicator := "→"
			change := ""
			if i > 0 {
				delta := point.TotalCost - trends[i-1].TotalCost
				if delta > 0.001 {
					indicator = "↑"
					change = fmt.Sprintf(" (+$%.6f)", delta)
				} else if delta < -0.001 {
					indicator = "↓"
					change = fmt.Sprintf(" ($%.6f)", delta)
				}
			}

			hashShort := point.GitHash
			if len(hashShort) > 7 {
				hashShort = hashShort[:7]
			}
			fmt.Printf("  %s %s: $%.6f%s\n", indicator, hashShort, point.TotalCost, change)
		}
	}

	return nil
}

// GenerateImprovementReport generates a detailed improvement report from baseline
func GenerateImprovementReport() error {
	baselineHash := LoadBaseline()
	if baselineHash == "" {
		return fmt.Errorf("no baseline set. Use 'kiro-krew eval --baseline <commit-hash>' to set one")
	}

	// Find most recent run
	latestRunPath, err := findLatestRun()
	if err != nil {
		return err
	}

	latestSummary, err := loadSummary(filepath.Join(latestRunPath, "summary.json"))
	if err != nil {
		return fmt.Errorf("failed to load latest run: %w", err)
	}

	baselineShort := baselineHash
	if len(baselineShort) > 7 {
		baselineShort = baselineShort[:7]
	}
	currentShort := latestSummary.GitHash
	if len(currentShort) > 7 {
		currentShort = currentShort[:7]
	}

	fmt.Println("📊 Improvement Report")
	fmt.Println(strings.Repeat("═", 80))
	fmt.Printf("Baseline: %s\n", baselineShort)
	fmt.Printf("Current:  %s\n", currentShort)
	fmt.Println(strings.Repeat("─", 80))

	if latestSummary.ImprovementData == nil {
		// Calculate improvements on the fly if not already present
		improvements, err := CalculateImprovements(latestSummary, baselineHash)
		if err != nil {
			return fmt.Errorf("failed to calculate improvements: %w", err)
		}
		if improvements == nil {
			return fmt.Errorf("no improvement data available")
		}
		latestSummary.ImprovementData = improvements
	}

	displayDetailedImprovements(latestSummary.ImprovementData)

	return nil
}

// Helper functions for report formatting

// extractAgentNames extracts unique agent names from trend points
func extractAgentNames(trends []TrendPoint) []string {
	agentSet := make(map[string]bool)
	for _, point := range trends {
		for agent := range point.Scores {
			agentSet[agent] = true
		}
	}

	var agents []string
	for agent := range agentSet {
		agents = append(agents, agent)
	}
	return agents
}

// displayTrendSummary shows overall trend direction across all commits
func displayTrendSummary(trends []TrendPoint, agents []string) {
	if len(trends) < 2 {
		return
	}

	fmt.Println(strings.Repeat("─", 80))
	fmt.Println("\n📊 Overall Trend Summary:")

	// Compare first and last runs for each agent
	first := trends[0]
	last := trends[len(trends)-1]

	for _, agent := range agents {
		firstScore, firstExists := first.Scores[agent]
		lastScore, lastExists := last.Scores[agent]

		if !firstExists || !lastExists {
			continue
		}

		delta := lastScore - firstScore
		percentChange := 0.0
		if firstScore > 0 {
			percentChange = (delta / firstScore) * 100
		}

		indicator := "→"
		if delta > 0.001 {
			indicator = "↑"
		} else if delta < -0.001 {
			indicator = "↓"
		}

		fmt.Printf("  %s %-20s: %.3f → %.3f  (%s%+.1f%%)\n",
			indicator, agent, firstScore, lastScore, indicator, percentChange)
	}

	// Show average improvement across all agents
	var totalDelta float64
	var validAgents int
	for _, agent := range agents {
		firstScore, firstExists := first.Scores[agent]
		lastScore, lastExists := last.Scores[agent]
		if firstExists && lastExists {
			totalDelta += (lastScore - firstScore)
			validAgents++
		}
	}

	if validAgents > 0 {
		avgDelta := totalDelta / float64(validAgents)
		fmt.Printf("\n  Average change: %+.3f\n", avgDelta)
	}
}

// displayDetailedImprovements shows comprehensive improvement metrics
func displayDetailedImprovements(metrics *ImprovementMetrics) {
	fmt.Printf("\nOverall Improvement: %+.1f%%\n", metrics.OverallImprovement)

	if len(metrics.AccuracyChange) > 0 {
		fmt.Println(strings.Repeat("─", 80))
		fmt.Println("\nPer-Agent Accuracy Changes:")
		for agent, change := range metrics.AccuracyChange {
			indicator := determineImprovementIndicator(change)
			description := getImprovementDescription(change)
			fmt.Printf("  %s %-20s: %+.1f%% %s\n", indicator, agent, change, description)
		}
	}

	if len(metrics.ErrorRateChange) > 0 {
		fmt.Println(strings.Repeat("─", 80))
		fmt.Println("\nError Rate Changes:")
		for agent, delta := range metrics.ErrorRateChange {
			indicator := "→"
			description := "(no change)"
			if delta < 0 {
				indicator = "✓"
				description = "(fewer errors)"
			} else if delta > 0 {
				indicator = "✗"
				description = "(more errors)"
			}
			fmt.Printf("  %s %-20s: %+d errors %s\n", indicator, agent, delta, description)
		}
	}

	if len(metrics.SignificantChanges) > 0 {
		fmt.Println(strings.Repeat("─", 80))
		fmt.Println("\nSignificant Changes:")
		for _, change := range metrics.SignificantChanges {
			fmt.Printf("  • %s\n", change)
		}
	}
}

// findLatestRun locates the most recent evaluation run
func findLatestRun() (string, error) {
	resultsDir := filepath.Join(".kiro-krew", "evals", "results")
	entries, err := os.ReadDir(resultsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read results directory: %w", err)
	}

	var latest string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() > latest {
			latest = entry.Name()
		}
	}

	if latest == "" {
		return "", fmt.Errorf("no evaluation runs found")
	}

	return filepath.Join(resultsDir, latest), nil
}

// getImprovementDescription returns a human-readable description of the improvement
func getImprovementDescription(changePercent float64) string {
	if changePercent > 10.0 {
		return "(major improvement)"
	} else if changePercent > 5.0 {
		return "(significant improvement)"
	} else if changePercent > 0.001 {
		return "(improvement)"
	} else if changePercent < -10.0 {
		return "(major regression)"
	} else if changePercent < -3.0 {
		return "(significant regression)"
	} else if changePercent < -0.001 {
		return "(regression)"
	}
	return "(no change)"
}

// averageScore calculates the average of all agent scores
func averageScore(scores map[string]float64) float64 {
	if len(scores) == 0 {
		return 0.0
	}
	var total float64
	for _, score := range scores {
		total += score
	}
	return total / float64(len(scores))
}
