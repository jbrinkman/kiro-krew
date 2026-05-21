package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/repl"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
)

//go:embed templates
var templates embed.FS

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := initProject(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	manager := agent.NewManager(cfg)
	w := watcher.New(cfg, manager)
	r := repl.New()

	w.Start()
	defer w.Stop()
	defer manager.StopAll()

	r.Run()
}

func initProject() error {
	return extractTemplates("templates", ".")
}

func extractTemplates(srcDir, destDir string) error {
	entries, err := templates.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read template directory %s: %w", srcDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		
		// Map template directory names to actual directory names
		destName := entry.Name()
		if entry.Name() == "kiro" {
			destName = ".kiro"
		} else if entry.Name() == "config.yaml" {
			// Special handling for config.yaml - it goes in .kiro-krew/
			destPath := filepath.Join(".kiro-krew", "config.yaml")
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				content, err := templates.ReadFile(srcPath)
				if err != nil {
					return fmt.Errorf("failed to read template file %s: %w", srcPath, err)
				}
				
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
				}
				
				if err := os.WriteFile(destPath, content, 0644); err != nil {
					return fmt.Errorf("failed to write file %s: %w", destPath, err)
				}
				fmt.Printf("Created %s\n", destPath)
			} else {
				fmt.Printf("File %s already exists\n", destPath)
			}
			continue
		}
		
		destPath := filepath.Join(destDir, destName)

		if entry.IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", destPath, err)
			}
			if err := extractTemplates(srcPath, destPath); err != nil {
				return err
			}
		} else {
			if _, err := os.Stat(destPath); os.IsNotExist(err) {
				content, err := templates.ReadFile(srcPath)
				if err != nil {
					return fmt.Errorf("failed to read template file %s: %w", srcPath, err)
				}
				
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
				}
				
				if err := os.WriteFile(destPath, content, 0644); err != nil {
					return fmt.Errorf("failed to write file %s: %w", destPath, err)
				}
				fmt.Printf("Created %s\n", destPath)
			} else {
				fmt.Printf("File %s already exists\n", destPath)
			}
		}
	}

	return nil
}
