package cmd

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/tui"
	"github.com/jbrinkman/kiro-krew/internal/version"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
)

var Templates embed.FS

var rootCmd = &cobra.Command{
	Use:     "kiro-krew",
	Short:   "Multi-agent development tool",
	Long:    "kiro-krew is a multi-agent development tool that watches GitHub issues and spawns agents to work on them.",
	Version: version.String(),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		about, _ := cmd.Flags().GetBool("about")
		if about {
			info := version.Info()
			displayHash := formatCommitHash(info["commit_hash"])

			fmt.Printf("  Version:    %s\n", info["version"])
			fmt.Printf("  Build Date: %s\n", info["build_date"])
			fmt.Printf("  Commit:     %s\n", displayHash)
			fmt.Printf("  Go Version: %s\n", info["go_version"])
			fmt.Printf("  Arch:       %s\n", info["arch"])
			os.Exit(0)
		}
		return nil
	},
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

func init() {
	// Cobra automatically adds --version/-v flags when Version is set
	// Customize the version template to show just the version number
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Add --about/-a flag
	rootCmd.PersistentFlags().BoolP("about", "a", false, "display comprehensive version information")
}

// formatCommitHash returns a short display hash (7 chars) or "unknown"
func formatCommitHash(hash string) string {
	if hash == "unknown" || hash == "" {
		return "unknown"
	}
	if len(hash) >= 7 {
		return hash[:7]
	}
	return hash
}

func SetTemplates(templates embed.FS) {
	Templates = templates
}

func Execute() error {
	return rootCmd.Execute()
}
