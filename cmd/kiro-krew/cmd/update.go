package cmd

import (
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update kiro-krew project files",
	Long:  "Update the project with the latest kiro-krew configuration files and templates, preserving config.yaml.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return extractTemplates("templates", ".", true)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}