package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level configuration structure.
type Config struct {
	Scan ScanConfig `yaml:"scan"`
}

// ScanConfig holds configuration for the scan operation.
type ScanConfig struct {
	Mode       string `yaml:"mode"`
	Timeout    int    `yaml:"timeout"`
	DockerHost string `yaml:"dockerHost"`
	Version    string `yaml:"version"`
}

// Load reads and parses the config file.
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate checks the configuration for correctness.
func (c *Config) Validate() error {
	return c.Scan.Validate()
}

// Validate checks the ScanConfig for correctness.
func (s *ScanConfig) Validate() error {
	validModes := map[string]bool{
		"auto":  true,
		"basic": true,
		"full":  true,
	}
	if !validModes[strings.ToLower(s.Mode)] {
		return fmt.Errorf("invalid mode '%s', must be one of: auto, basic, full", s.Mode)
	}

	if s.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0, got %d", s.Timeout)
	}

	if strings.TrimSpace(s.DockerHost) == "" {
		return fmt.Errorf("dockerHost cannot be empty")
	}

	if strings.TrimSpace(s.Version) == "" {
		return fmt.Errorf("version cannot be empty")
	}

	return nil
}
