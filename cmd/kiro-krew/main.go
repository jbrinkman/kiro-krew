package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbrinkman/kiro-krew/internal/agent"
	"github.com/jbrinkman/kiro-krew/internal/config"
	"github.com/jbrinkman/kiro-krew/internal/repl"
	"github.com/jbrinkman/kiro-krew/internal/watcher"
)

//go:embed templates/config.yaml
var configTemplate string

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
	if err := os.MkdirAll(".kiro-krew", 0755); err != nil {
		return fmt.Errorf("failed to create .kiro-krew directory: %w", err)
	}

	configPath := filepath.Join(".kiro-krew", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte(configTemplate), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
		fmt.Printf("Created %s\n", configPath)
	} else {
		fmt.Printf("Config file %s already exists\n", configPath)
	}

	return nil
}