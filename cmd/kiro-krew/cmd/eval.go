package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/jbrinkman/kiro-krew/internal/eval"
	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval [agent]",
	Short: "Run evaluations or show diff between runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		var agent string
		if len(args) > 0 {
			agent = args[0]
		}
		return eval.Run(agent)
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff <runA> <runB>",
	Short: "Compare two evaluation runs",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		showImprovement, _ := cmd.Flags().GetBool("improvement")
		return eval.DiffWithOptions(args[0], args[1], showImprovement)
	},
}

var improvementCmd = &cobra.Command{
	Use:   "improvement [baseline-commit]",
	Short: "Generate improvement report against baseline",
	Long: `Generate improvement report comparing current evaluation results to a baseline commit.

Stage 3 maturity requirements focus on continuous improvement tracking and
quantified performance gains to ensure agents evolve effectively.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get current commit
		currentCommit, err := getCurrentCommit()
		if err != nil {
			return fmt.Errorf("failed to get current commit: %w", err)
		}

		// Determine baseline commit
		var baselineCommit string
		if len(args) > 0 {
			baselineCommit = args[0]
		} else {
			// Load current summary to check for existing baseline
			currentRun, err := eval.FindBaselineRun(currentCommit)
			if err != nil {
				return fmt.Errorf("no baseline set and no baseline commit provided\nUse: kiro-krew eval baseline <commit> to set a baseline")
			}
			currentSummary, _, err := eval.LoadBaselineResults(currentRun)
			if err != nil {
				return fmt.Errorf("failed to load current results: %w", err)
			}
			if currentSummary.BaselineCommit == "" {
				return fmt.Errorf("no baseline set\nUse: kiro-krew eval baseline <commit> to set a baseline")
			}
			baselineCommit = currentSummary.BaselineCommit
		}

		return eval.DiffWithOptions(baselineCommit, currentCommit, true)
	},
}

var baselineCmd = &cobra.Command{
	Use:   "baseline <commit>",
	Short: "Set baseline commit for improvement tracking",
	Long: `Set the baseline commit reference for Stage 3 maturity improvement tracking.

The baseline establishes a performance reference point for measuring quantified
improvements in accuracy and error reduction over time.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		baselineCommit := args[0]
		
		// Get current commit
		currentCommit, err := getCurrentCommit()
		if err != nil {
			return fmt.Errorf("failed to get current commit: %w", err)
		}

		// Set baseline
		if err := eval.SetBaseline(currentCommit, baselineCommit); err != nil {
			return fmt.Errorf("failed to set baseline: %w", err)
		}

		fmt.Printf("Baseline set to commit %s\n", baselineCommit)
		return nil
	},
}

// getCurrentCommit returns the current git commit hash
func getCurrentCommit() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func init() {
	diffCmd.Flags().Bool("improvement", false, "Show quantified improvement metrics")
	evalCmd.AddCommand(diffCmd)
	evalCmd.AddCommand(improvementCmd)
	evalCmd.AddCommand(baselineCmd)
	rootCmd.AddCommand(evalCmd)
}