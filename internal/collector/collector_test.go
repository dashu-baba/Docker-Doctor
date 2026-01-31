package collector

import (
	"context"
	"os"
	"testing"

	"github.com/example/docker-doctor/internal/config"
)

func TestCollect(t *testing.T) {
	// Skip if DOCKER_HOST is not set or docker not available
	if os.Getenv("DOCKER_HOST") == "" && !fileExists("/var/run/docker.sock") {
		t.Skip("Docker not available")
	}

	ctx := context.Background()
	apiVersion := "1.40"
	cfg := &config.Config{
		Rules: config.Rules{
			DiskUsage: config.DiskUsageRule{
				Threshold: 80,
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