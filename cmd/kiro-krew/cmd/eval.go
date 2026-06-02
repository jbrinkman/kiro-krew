package cmd

import (
	"github.com/spf13/cobra"
	"github.com/jbrinkman/kiro-krew/internal/eval"
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
		return eval.Diff(args[0], args[1])
	},
}

func init() {
	evalCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(evalCmd)
}