//go:build integration
// +build integration

package collector

import (
	"context"
	"os"
	"testing"

	"github.com/example/docker-doctor/internal/config"
)

func TestCollect(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("set RUN_INTEGRATION=1 to run integration tests")
	}
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		t.Skip("set DOCKER_HOST (e.g. unix:///var/run/docker.sock or unix:///Users/<you>/.rd/docker.sock)")
	}

	ctx := context.Background()
	apiVersion := "1.40"
	cfg := &config.Config{
		Scan: config.ScanConfig{
			Mode:       "basic",
			Timeout:    30,
			DockerHost: dockerHost,
			Version:    apiVersion,
		},
		Rules: config.Rules{
			DiskUsage: config.DiskUsageRule{
				Threshold: 80,
			},
			StorageBloat: config.StorageBloatRule{
				ImageSizeThreshold:  10737418240,
				VolumeSizeThreshold: 5368709120,
			},
			Restarts: config.RestartsRule{
				Threshold: 3,
			},
			OOM: config.OOMRule{
				Enabled: true,
			},
			Healthcheck: config.HealthcheckRule{
				Enabled: true,
			},
		},
	}
	report, err := Collect(ctx, apiVersion, cfg)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if report == nil {
		t.Fatal("Report is nil")
	}

	// Basic checks
	if report.Host.OS == "" {
		t.Error("Host OS not set")
	}
	if report.Docker.Version == "" {
		t.Error("Docker version not set")
	}
	// Issues should be initialized
	if report.Issues == nil {
		t.Error("Issues not initialized")
	}
	// Add more assertions as needed
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
