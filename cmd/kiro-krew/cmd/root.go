package cmd

import (
	"embed"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/tui"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
)

var Templates embed.FS

var rootCmd = &cobra.Command{
	Use:   "kiro-krew",
	Short: "Multi-agent development tool",
	Long:  "kiro-krew is a multi-agent development tool that watches GitHub issues and spawns agents to work on them.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		manager := agent.NewManager(cfg)
		w := watcher.New(cfg, manager)

		defer manager.StopAll()
		defer w.Stop()

		return tui.Run(w, manager, cfg)
	},
}

func SetTemplates(templates embed.FS) {
	Templates = templates
}

func Execute() error {
	return rootCmd.Execute()
}
