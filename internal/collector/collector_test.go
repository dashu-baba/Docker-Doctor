package collector

import (
	"context"
	"os"
	"testing"
)

func TestCollect(t *testing.T) {
	// Skip if DOCKER_HOST is not set or docker not available
	if os.Getenv("DOCKER_HOST") == "" && !fileExists("/var/run/docker.sock") {
		t.Skip("Docker not available")
	}

	ctx := context.Background()
	apiVersion := "1.40"
	report, err := Collect(ctx, apiVersion)
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
	// Add more assertions as needed
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}