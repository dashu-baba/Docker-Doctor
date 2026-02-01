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
	DiskUsage    DiskUsageRule    `yaml:"disk_usage"`
	StorageBloat StorageBloatRule `yaml:"storage_bloat"`
	Restarts     RestartsRule     `yaml:"restarts"`
	OOM          OOMRule          `yaml:"oom"`
	Healthcheck  HealthcheckRule  `yaml:"healthcheck"`
	LogBloat     LogBloatRule     `yaml:"log_bloat"`
}

// DiskUsageRule defines rules for disk usage checks.
type DiskUsageRule struct {
	Threshold int `yaml:"threshold"` // percentage (0-100)
}

// StorageBloatRule defines rules for storage bloat checks.
type StorageBloatRule struct {
	ImageSizeThreshold  uint64 `yaml:"image_size_threshold"`  // in bytes
	VolumeSizeThreshold uint64 `yaml:"volume_size_threshold"` // in bytes
}

// RestartsRule defines rules for container restart checks.
type RestartsRule struct {
	Threshold int `yaml:"threshold"` // max allowed restarts
}

// OOMRule defines rules for OOM kill checks.
type OOMRule struct {
	Enabled bool `yaml:"enabled"`
}

// HealthcheckRule defines rules for container healthcheck checks.
type HealthcheckRule struct {
	Enabled bool `yaml:"enabled"`
}

// LogBloatRule defines rules for container log bloat checks.
type LogBloatRule struct {
	Enabled       bool   `yaml:"enabled"`
	SizeThreshold uint64 `yaml:"size_threshold"` // in bytes
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
	if err := r.DiskUsage.Validate(); err != nil {
		return err
	}
	if err := r.StorageBloat.Validate(); err != nil {
		return err
	}
	if err := r.Restarts.Validate(); err != nil {
		return err
	}
	if err := r.OOM.Validate(); err != nil {
		return err
	}
	if err := r.Healthcheck.Validate(); err != nil {
		return err
	}
	return r.LogBloat.Validate()
}

// Validate checks the DiskUsageRule for correctness.
func (d *DiskUsageRule) Validate() error {
	if d.Threshold < 0 || d.Threshold > 100 {
		return fmt.Errorf("disk_usage threshold must be between 0 and 100, got %d", d.Threshold)
	}
	return nil
}

// Validate checks the StorageBloatRule for correctness.
func (s *StorageBloatRule) Validate() error {
	if s.ImageSizeThreshold < 0 {
		return fmt.Errorf("image_size_threshold must be non-negative, got %d", s.ImageSizeThreshold)
	}
	if s.VolumeSizeThreshold < 0 {
		return fmt.Errorf("volume_size_threshold must be non-negative, got %d", s.VolumeSizeThreshold)
	}
	return nil
}

// Validate checks the RestartsRule for correctness.
func (r *RestartsRule) Validate() error {
	if r.Threshold < 0 {
		return fmt.Errorf("restarts threshold must be non-negative, got %d", r.Threshold)
	}
	return nil
}

// Validate checks the OOMRule for correctness.
func (o *OOMRule) Validate() error {
	// No validation needed for boolean
	return nil
}

// Validate checks the HealthcheckRule for correctness.
func (h *HealthcheckRule) Validate() error {
	// No validation needed for boolean
	return nil
}

// Validate checks the LogBloatRule for correctness.
func (l *LogBloatRule) Validate() error {
	if l.SizeThreshold < 0 {
		return fmt.Errorf("log_bloat size_threshold must be non-negative, got %d", l.SizeThreshold)
	}
	return nil
}
