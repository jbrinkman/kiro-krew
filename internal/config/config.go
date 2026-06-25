package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type SessionConfig struct {
	MaxHistoryMessages int           `yaml:"max_history_messages"`
	MaxAge             time.Duration `yaml:"max_age"`
	SessionsDir        string        `yaml:"sessions_dir"`
}

type SandboxConfig struct {
	Image        string        `yaml:"image"`
	WorkspaceDir string        `yaml:"workspace_dir"`
	CPUCores     float64       `yaml:"cpu_cores"`
	MemoryMB     int           `yaml:"memory_mb"`
	Timeout      time.Duration `yaml:"timeout"`
}

type Config struct {
	Repo                string        `yaml:"repo"`
	Label               string        `yaml:"label"`
	PollInterval        time.Duration `yaml:"poll_interval"`
	MaxRetries          int           `yaml:"max_retries"`
	MaxQARetries        int           `yaml:"max_qa_retries"`
	MaxActivityLines    int           `yaml:"max_activity_lines"`
	ConsoleLogging      bool          `yaml:"console_logging"`
	Theme               string        `yaml:"theme"`
	EnableCopilotReview bool          `yaml:"enable_copilot_review"`
	Session             SessionConfig `yaml:"session"`
	Sandbox             SandboxConfig `yaml:"sandbox"`
	LoadedTheme         *Theme        `yaml:"-"`
}

func extractLeadingWhitespace(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

func Load() (*Config, error) {
	cfg := &Config{
		Label:               "kiro-krew",
		PollInterval:        5 * time.Minute,
		MaxRetries:          3,
		MaxQARetries:        3,
		MaxActivityLines:    1000,
		ConsoleLogging:      false,
		Theme:               "default",
		EnableCopilotReview: true,
		Session: SessionConfig{
			MaxHistoryMessages: 100,
			MaxAge:             24 * time.Hour,
			SessionsDir:        ".kiro-krew/sessions",
		},
		Sandbox: SandboxConfig{
			Image:        "alpine:3.19",
			WorkspaceDir: "/workspace",
			CPUCores:     1.0,
			MemoryMB:     1024,
			Timeout:      5 * time.Minute,
		},
	}

	data, err := os.ReadFile(".kiro-krew/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.Repo == "" {
		return nil, fmt.Errorf("repo field is required")
	}

	// Validate session config
	if cfg.Session.MaxHistoryMessages <= 0 {
		return nil, fmt.Errorf("session.max_history_messages must be greater than 0")
	}
	if cfg.Session.MaxAge <= 0 {
		return nil, fmt.Errorf("session.max_age must be greater than 0")
	}
	if cfg.Session.SessionsDir == "" {
		return nil, fmt.Errorf("session.sessions_dir cannot be empty")
	}

	// Validate sandbox config
	if cfg.Sandbox.CPUCores <= 0 {
		return nil, fmt.Errorf("sandbox.cpu_cores must be greater than 0")
	}
	if cfg.Sandbox.MemoryMB <= 0 {
		return nil, fmt.Errorf("sandbox.memory_mb must be greater than 0")
	}
	if cfg.Sandbox.MemoryMB < 256 {
		return nil, fmt.Errorf("sandbox.memory_mb must be at least 256")
	}
	if cfg.Sandbox.Timeout <= 0 {
		return nil, fmt.Errorf("sandbox.timeout must be greater than 0")
	}
	if cfg.Sandbox.Image == "" {
		return nil, fmt.Errorf("sandbox.image cannot be empty")
	}
	if cfg.Sandbox.WorkspaceDir == "" {
		return nil, fmt.Errorf("sandbox.workspace_dir cannot be empty")
	}

	// Load the specified theme
	theme, err := LoadTheme(cfg.Theme)
	if err != nil {
		fmt.Printf("Warning: Failed to load theme '%s': %v, using built-in default theme\n", cfg.Theme, err)
		theme = getDefaultTheme()
	}
	cfg.LoadedTheme = theme

	return cfg, nil
}

// Save writes the current configuration back to the config file,
// preserving existing YAML structure and comments
func (c *Config) Save() error {
	configPath := ".kiro-krew/config.yaml"

	// Read existing file to preserve structure and comments
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	// Find and update theme line, or add it if not present
	themeUpdated := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		leadingWhitespace := extractLeadingWhitespace(line)
		if strings.HasPrefix(trimmed, "theme:") {
			lines[i] = fmt.Sprintf("%stheme: %q", leadingWhitespace, c.Theme)
			themeUpdated = true
			break
		}
		// Check for commented theme line
		if strings.HasPrefix(trimmed, "# theme:") {
			lines[i] = fmt.Sprintf("%stheme: %q", leadingWhitespace, c.Theme)
			themeUpdated = true
			break
		}
	}

	// If theme wasn't found, add it at the end
	if !themeUpdated {
		lines = append(lines, fmt.Sprintf("theme: %q", c.Theme))
	}

	// Write back to file
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
