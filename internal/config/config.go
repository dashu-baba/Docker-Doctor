package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level configuration structure.
type Config struct {
	Scan  ScanConfig `yaml:"scan"`
	Rules Rules      `yaml:"rules"`
}

// ScanConfig holds configuration for the scan operation.
type ScanConfig struct {
	Mode       string `yaml:"mode"`
	Timeout    int    `yaml:"timeout"`
	DockerHost string `yaml:"dockerHost"`
	Version    string `yaml:"version"`
}

// Rules holds the diagnostic rules.
type Rules struct {
	DiskUsage DiskUsageRule `yaml:"disk_usage"`
}

// DiskUsageRule defines rules for disk usage checks.
type DiskUsageRule struct {
	Threshold int `yaml:"threshold"` // percentage (0-100)
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
	if err := c.Scan.Validate(); err != nil {
		return err
	}
	return c.Rules.Validate()
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

// Validate checks the Rules for correctness.
func (r *Rules) Validate() error {
	return r.DiskUsage.Validate()
}

// Validate checks the DiskUsageRule for correctness.
func (d *DiskUsageRule) Validate() error {
	if d.Threshold < 0 || d.Threshold > 100 {
		return fmt.Errorf("disk_usage threshold must be between 0 and 100, got %d", d.Threshold)
	}
	return nil
}