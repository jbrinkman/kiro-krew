package cmd

import (
	"github.com/jbrinkman/kiro-krew/internal/eval"
	"github.com/spf13/cobra"
)

var evalCmd = &cobra.Command{
	Use:   "eval [agent]",
	Short: "Run evaluations for agents",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var agent string
		if len(args) > 0 {
			agent = args[0]
		}
		return eval.Run(agent)
	},
}

var evalDiffCmd = &cobra.Command{
	Use:   "diff <run-a> <run-b>",
	Short: "Compare two evaluation runs",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return eval.Diff(args[0], args[1])
	},
}

func init() {
	evalCmd.AddCommand(evalDiffCmd)
	rootCmd.AddCommand(evalCmd)
}