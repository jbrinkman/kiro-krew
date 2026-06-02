package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/eval"
	"github.com/jbrinkman/kiro-krew/internal/tui"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
)

//go:embed templates
var templates embed.FS

// Help data structure
type CommandHelp struct {
	Brief    string
	Usage    string
	Detailed string
}

var helpData = map[string]CommandHelp{
	"init": {
		Brief:    "Extract project templates",
		Usage:    "kiro-krew init",
		Detailed: "Purpose: Extract kiro-krew project templates to current directory.\n\nBehavior:\n- Creates .kiro and .kiro-krew directories with configuration files\n- Skips files that already exist (non-destructive)\n- Sets up the project structure for kiro-krew usage",
	},
	"update": {
		Brief:    "Update project templates (force overwrite)",
		Usage:    "kiro-krew update",
		Detailed: "Purpose: Update kiro-krew project templates with force overwrite.\n\nBehavior:\n- Overwrites existing template files to update them to latest versions\n- Uses --force flag behavior by default\n- Never overwrites config.yaml to preserve user settings",
	},
	"eval": {
		Brief:    "Run evaluations or show diff between runs", 
		Usage:    "kiro-krew eval [agent]\nkiro-krew eval diff <run-a> <run-b>",
		Detailed: "Purpose: Run evaluations on your project.\n\nUsage:\n- kiro-krew eval: Run evaluation with default settings\n- kiro-krew eval [agent]: Run evaluation with specified agent (optional parameter)\n- kiro-krew eval diff <run-a> <run-b>: Compare two evaluation runs\n\nSubcommands:\n  diff    Compare two evaluation runs (see 'kiro-krew eval diff --help')",
	},
	"eval-diff": {
		Brief:    "Compare two evaluation runs",
		Usage:    "kiro-krew eval diff <run-a> <run-b>",
		Detailed: "Purpose: Compare two evaluation runs to see differences.\n\nRequired parameters:\n- run-a: First evaluation run identifier\n- run-b: Second evaluation run identifier\n\nShows detailed comparison between the specified evaluation runs.",
	},
}

func main() {
	// Check for help flags first
	if len(os.Args) > 1 {
		// Global help: kiro-krew --help or kiro-krew -h
		if os.Args[1] == "--help" || os.Args[1] == "-h" {
			showGeneralHelp()
			return
		}

		// Command-specific help: kiro-krew <command> --help or kiro-krew <command> -h
		if len(os.Args) > 2 && (os.Args[2] == "--help" || os.Args[2] == "-h") {
			showCommandHelp(strings.ToLower(os.Args[1]))
			return
		}

		// eval diff help: kiro-krew eval diff --help or kiro-krew eval diff -h
		if len(os.Args) > 3 && strings.ToLower(os.Args[1]) == "eval" && strings.ToLower(os.Args[2]) == "diff" && (os.Args[3] == "--help" || os.Args[3] == "-h") {
			showCommandHelp("eval-diff")
			return
		}
	}

	if len(os.Args) > 1 {
		switch strings.ToLower(os.Args[1]) {
		case "init":
			if err := extractTemplates("templates", ".", false); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "update":
			if err := extractTemplates("templates", ".", true); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		case "eval":
			if err := runEval(); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			return
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	manager := agent.NewManager(cfg)
	w := watcher.New(cfg, manager)

	defer w.Stop()
	defer manager.StopAll()

	if err := tui.Run(w, manager, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func extractTemplates(srcDir, destDir string, force bool) error {
	entries, err := templates.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read template directory %s: %w", srcDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())

		// Map template directory names to dot-prefixed names
		destName := entry.Name()
		switch entry.Name() {
		case "kiro":
			destName = ".kiro"
		case "kiro-krew":
			destName = ".kiro-krew"
		}

		destPath := filepath.Join(destDir, destName)

		if entry.IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			if err := extractTemplates(srcPath, destPath, force); err != nil {
				return err
			}
		} else {
			// config.yaml is NEVER overwritten
			if entry.Name() == "config.yaml" && destDir != "." {
				if _, err := os.Stat(destPath); err == nil {
					fmt.Printf("Skipped %s (config never overwritten)\n", destPath)
					continue
				}
			}

			if force {
				if err := writeTemplateFile(srcPath, destPath); err != nil {
					return err
				}
				fmt.Printf("Updated %s\n", destPath)
			} else if _, err := os.Stat(destPath); os.IsNotExist(err) {
				if err := writeTemplateFile(srcPath, destPath); err != nil {
					return err
				}
				fmt.Printf("Created %s\n", destPath)
			} else {
				fmt.Printf("Skipped %s (already exists)\n", destPath)
			}
		}
	}

	return nil
}

func writeTemplateFile(srcPath, destPath string) error {
	content, err := templates.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", srcPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
	}

	return os.WriteFile(destPath, content, 0644)
}

func runEval() error {
	// kiro-krew eval diff <run-a> <run-b>
	if len(os.Args) > 2 && strings.ToLower(os.Args[2]) == "diff" {
		if len(os.Args) < 5 {
			return fmt.Errorf("usage: kiro-krew eval diff <run-a> <run-b>")
		}
		return eval.Diff(os.Args[3], os.Args[4])
	}

	// kiro-krew eval [agent]
	var agent string
	if len(os.Args) > 2 {
		agent = os.Args[2]
	}
	return eval.Run(agent)
}

func showGeneralHelp() {
	fmt.Println("kiro-krew - Multi-agent development tool")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  kiro-krew [command]")
	fmt.Println()
	fmt.Println("AVAILABLE COMMANDS:")
	// Show commands in specific order, excluding internal eval-diff
	commands := []string{"init", "update", "eval"}
	for _, cmd := range commands {
		if help, exists := helpData[cmd]; exists {
			fmt.Printf("  %-8s %s\n", cmd, help.Brief)
		}
	}
	fmt.Println()
	fmt.Println("FLAGS:")
	fmt.Println("  -h, --help   Show help")
	fmt.Println()
	fmt.Println("Use \"kiro-krew [command] --help\" for more information about a command.")
}

func showCommandHelp(command string) {
	help, exists := helpData[command]
	if !exists {
		fmt.Printf("Unknown command: %s\n\n", command)
		showGeneralHelp()
		return
	}

	fmt.Printf("Usage: %s\n\n", help.Usage)
	fmt.Printf("%s\n", help.Detailed)
}
