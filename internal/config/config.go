package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Repo             string        `yaml:"repo"`
	Label            string        `yaml:"label"`
	PollInterval     time.Duration `yaml:"poll_interval"`
	MaxRetries       int           `yaml:"max_retries"`
	MaxActivityLines int           `yaml:"max_activity_lines"`
	ConsoleLogging   bool          `yaml:"console_logging"`
	Theme            string        `yaml:"theme"`
	LoadedTheme      *Theme        `yaml:"-"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Label:            "kiro-krew",
		PollInterval:     5 * time.Minute,
		MaxRetries:       3,
		MaxActivityLines: 1000,
		ConsoleLogging:   false,
		Theme:            "default",
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
		leadingWhitespace := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
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
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
