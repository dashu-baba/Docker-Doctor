package cmd

import (
	"strings"
	"testing"
	"time"

	v1 "github.com/dashu-baba/docker-doctor/internal/schema/v1"
)

func TestGenerateMarkdownv1_BasicSections(t *testing.T) {
	r := &v1.Report{
		SchemaVersion: "1.0",
		Tool:          v1.Tool{Name: "docker-host-doctor", Version: "dev"},
		Scan: v1.Scan{
			ScanID:        "test-scan",
			FinishedAt:    time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
			EffectiveMode: "basic",
		},
		Target: v1.Target{
			Host:  v1.TargetHost{OS: "linux", Arch: "amd64"},
			Docker: v1.TargetDocker{EngineVersion: "29.1.3", APIVersion: "1.41"},
		},
		Summary: v1.Summary{
			Counts: v1.SummaryCounts{ContainersRunning: 1, ContainersStopped: 0, Images: 2, Volumes: 1},
			ResourceSnapshot: v1.SummaryResourceSnapshot{
				DockerSystemDf: v1.DockerSystemDf{ImagesTotalBytes: 1024, BuildCacheTotalBytes: 2048},
			},
			FindingCounts: v1.SummaryFindingCounts{Critical: 1, Warning: 0, Info: 0},
		},
		Collectors: []v1.Collector{{Name: "docker_engine", Status: "ok", DurationMs: 10}},
		Findings: []v1.Finding{
			{
				ID:          "DOCKER_STORAGE_BLOAT",
				Fingerprint: "DOCKER_STORAGE_BLOAT:images_total",
				Severity:    "critical",
				Confidence:  "medium",
				Category:    "storage",
				Title:       "Docker storage usage is high",
				Summary:     "Example summary",
			},
		},
	}

	out, err := generateMarkdownv1(r)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"# Docker Host Doctor Report",
		"## Summary",
		"## Findings",
		"DOCKER_STORAGE_BLOAT",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("markdown missing %q\n\n%s", needle, out)
		}
	}
}

