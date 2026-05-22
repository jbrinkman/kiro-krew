package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Repo             string        `yaml:"repo"`
	Label            string        `yaml:"label"`
	PollInterval     time.Duration `yaml:"poll_interval"`
	MaxRetries       int           `yaml:"max_retries"`
	MaxActivityLines int           `yaml:"max_activity_lines"`
}

func Load() (*Config, error) {
	cfg := &Config{
		Label:            "kiro-krew",
		PollInterval:     5 * time.Minute,
		MaxRetries:       3,
		MaxActivityLines: 1000,
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

	return cfg, nil
}