package cmd

import (
	"strings"

	"github.com/jbrinkman/kiro-krew/internal/eval"
	"github.com/spf13/cobra"
)

var (
	evalList             bool
	evalResume           bool
	evalCase             string
	evalPerf             bool
	evalSandbox          bool
	evalNoSandbox        bool
	evalResourceLimit    []string
	evalDebug            bool
	evalCleanup          bool
	evalSetBaseline      string
	evalShowImprovements bool
)

var evalCmd = &cobra.Command{
	Use:   "eval [agent] [testcase]",
	Short: "Run evaluations or show diff between runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		var agent, testcase string
		if len(args) > 0 {
			agent = args[0]
		}
		if len(args) > 1 {
			testcase = args[1]
		}

		// Use --case flag if provided
		if evalCase != "" {
			testcase = evalCase
		}

		// Handle baseline setting
		if evalSetBaseline != "" {
			return eval.RunWithOptions(agent, testcase, eval.RunOptions{
				SetBaseline: evalSetBaseline,
			})
		}

		// Handle cleanup operation
		if evalCleanup {
			return eval.RunCleanup()
		}

		// Handle performance investigation
		if evalPerf {
			return eval.RunPerformanceInvestigation(agent)
		}

		// Parse resource limits
		resourceLimits := make(map[string]string)
		for _, limit := range evalResourceLimit {
			parts := strings.SplitN(limit, "=", 2)
			if len(parts) == 2 {
				resourceLimits[parts[0]] = parts[1]
			}
		}

		return eval.RunWithOptions(agent, testcase, eval.RunOptions{
			List:          evalList,
			Resume:        evalResume,
			Sandbox:       evalSandbox,
			NoSandbox:     evalNoSandbox,
			ResourceLimit: resourceLimits,
			Debug:         evalDebug,
			Cleanup:       evalCleanup,
		})
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff <runA> <runB>",
	Short: "Compare two evaluation runs",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if evalShowImprovements {
			return eval.DiffWithImprovements(args[0], args[1])
		}
		return eval.Diff(args[0], args[1])
	},
}

var trendCmd = &cobra.Command{
	Use:   "trend <commit...>",
	Short: "Show evaluation trends across multiple commits",
	Long: `Show evaluation trends across multiple commits.

Analyzes evaluation results from multiple commits and shows how metrics
have changed over time. Requires at least 2 commits to compare.

Example:
  kiro-krew eval trend HEAD~5 HEAD~3 HEAD~1 HEAD`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return eval.ShowTrends(args)
	},
}

var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate improvement report from baseline",
	Long: `Generate an improvement report comparing current results to baseline.

Analyzes evaluation results from the current state and compares them
against the configured baseline, showing improvements and regressions.
Requires a baseline to be set using --baseline flag.

Example:
  kiro-krew eval --baseline main
  kiro-krew eval report`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return eval.GenerateImprovementReport()
	},
}

func init() {
	evalCmd.Flags().BoolVar(&evalList, "list", false, "List available test cases for the agent")
	evalCmd.Flags().BoolVar(&evalResume, "resume", false, "Resume interrupted evaluation from last completed test")
	evalCmd.Flags().StringVar(&evalCase, "case", "", "Run specific test case")
	evalCmd.Flags().BoolVar(&evalPerf, "perf", false, "Run performance investigation and profiling")
	evalCmd.Flags().BoolVar(&evalSandbox, "sandbox", false, "Enable container sandboxing for agent execution")
	evalCmd.Flags().BoolVar(&evalNoSandbox, "no-sandbox", false, "Explicitly disable container sandboxing")
	evalCmd.Flags().StringSliceVar(&evalResourceLimit, "resource-limit", nil, "Override resource limits (cpu=1.0, memory=1073741824, timeout=5m)")
	evalCmd.Flags().BoolVarP(&evalDebug, "debug", "d", false, "Enable debug mode with verbose logging and container persistence")
	evalCmd.Flags().BoolVar(&evalCleanup, "cleanup", false, "Stop and remove all tracked debug containers and clean artifacts")
	evalCmd.Flags().StringVar(&evalSetBaseline, "baseline", "", "Set baseline commit for improvement tracking (hash or ref)")

	diffCmd.Flags().BoolVar(&evalShowImprovements, "show-improvements", false, "Highlight improvements and regressions in diff output")

	evalCmd.AddCommand(diffCmd)
	evalCmd.AddCommand(trendCmd)
	evalCmd.AddCommand(reportCmd)
	rootCmd.AddCommand(evalCmd)
}
