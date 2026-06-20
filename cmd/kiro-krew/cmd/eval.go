package cmd

import (
	"github.com/jbrinkman/kiro-krew/internal/eval"
	"github.com/spf13/cobra"
)

var (
	evalList   bool
	evalResume bool
	evalCase   string
	evalPerf   bool
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

		// Handle performance investigation
		if evalPerf {
			return eval.RunPerformanceInvestigation(agent)
		}

		return eval.RunWithOptions(agent, testcase, eval.RunOptions{
			List:   evalList,
			Resume: evalResume,
		})
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff <runA> <runB>",
	Short: "Compare two evaluation runs",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return eval.Diff(args[0], args[1])
	},
}

func init() {
	evalCmd.Flags().BoolVar(&evalList, "list", false, "List available test cases for the agent")
	evalCmd.Flags().BoolVar(&evalResume, "resume", false, "Resume interrupted evaluation from last completed test")
	evalCmd.Flags().StringVar(&evalCase, "case", "", "Run specific test case")
	evalCmd.Flags().BoolVar(&evalPerf, "perf", false, "Run performance investigation and profiling")

	evalCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(evalCmd)
}
