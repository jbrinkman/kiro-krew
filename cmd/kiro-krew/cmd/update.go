package cmd

import (
	"github.com/spf13/cobra"
	"github.com/jbrinkman/kiro-krew/internal/templates"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update project templates (force overwrite)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return templates.Extract("templates", ".", true)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}