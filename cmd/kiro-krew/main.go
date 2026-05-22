package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/tui"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
)

//go:embed templates
var templates embed.FS

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
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

	if err := tui.Run(w, manager); err != nil {
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
