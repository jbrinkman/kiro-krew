package config

import (
"fmt"
"os"
"time"

"gopkg.in/yaml.v3"
)

type Config struct {
Repo                string        `yaml:"repo"`
Label               string        `yaml:"label"`
PollInterval        time.Duration `yaml:"poll_interval"`
MaxRetries          int           `yaml:"max_retries"`
MaxActivityLines    int           `yaml:"max_activity_lines"`
ConsoleLogging      bool          `yaml:"console_logging"`
Theme               string        `yaml:"theme"`
EnableCopilotReview bool          `yaml:"enable_copilot_review"`
LoadedTheme         *Theme        `yaml:"-"`
}

func Load() (*Config, error) {
cfg := &Config{
Label:               "kiro-krew",
PollInterval:        5 * time.Minute,
MaxRetries:          3,
MaxActivityLines:    1000,
ConsoleLogging:      false,
Theme:               "default",
EnableCopilotReview: true,
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
