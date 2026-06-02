package cmd

import (
	"fmt"
	"os"

	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/tui"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "kiro-krew",
	Short: "Multi-agent development tool",
	Long:  "kiro-krew is a multi-agent development tool that helps manage and coordinate development tasks.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}

		manager := agent.NewManager(cfg)
		w := watcher.New(cfg, manager)

		defer w.Stop()
		defer manager.StopAll()

		return tui.Run(w, manager, cfg)
	},
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.EnableCaseInsensitive = true
	rootCmd.Version = version
}