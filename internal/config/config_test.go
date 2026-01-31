package config

import (
	"os"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Scan: ScanConfig{
					Mode:       "basic",
					Timeout:    30,
					DockerHost: "unix:///var/run/docker.sock",
					Version:    "1.40",
				},
				Rules: Rules{
					DiskUsage: DiskUsageRule{
						Threshold: 80,
					},
					StorageBloat: StorageBloatRule{
						ImageSizeThreshold:  10737418240,
						VolumeSizeThreshold: 5368709120,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: Config{
				Scan: ScanConfig{
					Mode:       "invalid",
					Timeout:    30,
					DockerHost: "unix:///var/run/docker.sock",
					Version:    "1.40",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			config: Config{
				Scan: ScanConfig{
					Mode:       "basic",
					Timeout:    0,
					DockerHost: "unix:///var/run/docker.sock",
					Version:    "1.40",
				},
			},
			wantErr: true,
		},
		{
			name: "empty dockerHost",
			config: Config{
				Scan: ScanConfig{
					Mode:       "basic",
					Timeout:    30,
					DockerHost: "",
					Version:    "1.40",
				},
			},
			wantErr: true,
		},
		{
			name: "empty version",
			config: Config{
				Scan: ScanConfig{
					Mode:       "basic",
					Timeout:    30,
					DockerHost: "unix:///var/run/docker.sock",
					Version:    "",
				},
				Rules: Rules{
					DiskUsage: DiskUsageRule{
						Threshold: 80,
					},
					StorageBloat: StorageBloatRule{
						ImageSizeThreshold:  10737418240,
						VolumeSizeThreshold: 5368709120,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid disk threshold",
			config: Config{
				Scan: ScanConfig{
					Mode:       "basic",
					Timeout:    30,
					DockerHost: "unix:///var/run/docker.sock",
					Version:    "1.40",
				},
				Rules: Rules{
					DiskUsage: DiskUsageRule{
						Threshold: 150,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	// Create a temporary config file
	content := `scan:
  mode: basic
  timeout: 30
  dockerHost: unix:///var/run/docker.sock
  version: "1.40"
rules:
  disk_usage:
    threshold: 80
  storage_bloat:
    image_size_threshold: 10737418240
    volume_size_threshold: 5368709120
`
	tmpFile, err := os.CreateTemp("", "config.yml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Scan.Mode != "basic" {
		t.Errorf("Expected mode 'basic', got %s", cfg.Scan.Mode)
	}
	if cfg.Scan.Timeout != 30 {
		t.Errorf("Expected timeout 30, got %d", cfg.Scan.Timeout)
	}
	if cfg.Scan.DockerHost != "unix:///var/run/docker.sock" {
		t.Errorf("Expected dockerHost 'unix:///var/run/docker.sock', got %s", cfg.Scan.DockerHost)
	}
	if cfg.Scan.Version != "1.40" {
		t.Errorf("Expected version '1.40', got %s", cfg.Scan.Version)
	}
	if cfg.Rules.DiskUsage.Threshold != 80 {
		t.Errorf("Expected disk_usage threshold 80, got %d", cfg.Rules.DiskUsage.Threshold)
	}
}
