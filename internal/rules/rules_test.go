package rules

import (
	"testing"
	"time"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/facts"
	"github.com/example/docker-doctor/internal/types"
)

func TestEvaluate_ProducesExpectedRuleIDs(t *testing.T) {
	cfg := &config.Config{
		Rules: config.Rules{
			DiskUsage: config.DiskUsageRule{Threshold: 80},
			StorageBloat: config.StorageBloatRule{
				ImageSizeThreshold:  10,
				VolumeSizeThreshold: 0,
			},
			Restarts:     config.RestartsRule{Threshold: 3},
			OOM:          config.OOMRule{Enabled: true},
			Healthcheck:  config.HealthcheckRule{Enabled: true},
			LogBloat:     config.LogBloatRule{Enabled: true, SizeThreshold: 100},
			VolumeSize:   config.VolumeSizeRule{Enabled: true, SizeThreshold: 2000000000}, // 2GB
		},
	}

	report := &types.Report{
		Host: types.HostInfo{
			DiskUsage: map[string]*types.DiskInfo{
				"/": {Used: 90, Total: 100, UsedPercent: 90.0},
			},
		},
		Images: types.Images{
			Count:     1,
			TotalSize: 11, // above threshold 10
			List:      []types.ImageInfo{{ID: "img1", Size: 11}},
		},
		Containers: types.Containers{
			Count: 1,
			List: []types.ContainerInfo{
				{
					ID:             "abc123",
					Name:           "/app",
					Status:         "Up 1m",
					RestartCount:   10,
					OOMKilled:      true,
					HealthStatus:   "unhealthy",
					UnhealthySince: time.Now().Add(-2 * time.Hour),
					LogSize:        200, // above threshold 100
				},
			},
		},
		Volumes: types.Volumes{
			Count: 3,
			List: []types.VolumeInfo{
				{Name: "vol1", Size: 100, SizeAvailable: true, Used: false},
				{Name: "vol2", Size: 3000000000, SizeAvailable: true, Used: true}, // 3GB > 2GB
				{Name: "vol3", Size: 0, SizeAvailable: false, Used: false}, // unavailable
			},
		},
		Networks: types.Networks{
			Count: 3,
			List: []types.NetworkInfo{
				{Name: "bridge", CIDR: "172.17.0.0/16"},
				{Name: "net1", CIDR: "192.168.1.0/24"},
				{Name: "net2", CIDR: "192.168.1.0/25"}, // overlaps with net1
			},
		},
	}

	df := &facts.DockerSystemDfSummary{
		ImagesTotalBytes:     11, // above threshold (10)
		BuildCacheTotalBytes: 5,
	}

	Evaluate(report, cfg, df)

	seen := map[string]bool{}
	for _, is := range report.Issues {
		seen[is.RuleID] = true
	}

	for _, want := range []string{
		"DISK_USAGE_HIGH",
		"DOCKER_STORAGE_BLOAT",
		"RESTART_LOOP",
		"OOM_KILLED",
		"HEALTHCHECK_UNHEALTHY",
		"LOG_BLOAT",
		"VOLUME_BLOAT",
		"VOLUME_SIZE_HIGH",
		"NETWORK_OVERLAP",
	} {
		if !seen[want] {
			t.Fatalf("expected ruleId %s to be produced, got %+v", want, seen)
		}
	}
}

func TestEvaluate_StorageBloat_PrefersSystemDf(t *testing.T) {
	cfg := &config.Config{
		Rules: config.Rules{
			StorageBloat: config.StorageBloatRule{
				ImageSizeThreshold: 100,
			},
		},
	}
	report := &types.Report{
		Images: types.Images{
			Count:     2,
			TotalSize: 999999999, // should be ignored when df present
			List:      []types.ImageInfo{{ID: "img1", Size: 100}, {ID: "img2", Size: 1}},
		},
		Volumes: types.Volumes{
			Count: 1,
			List:  []types.VolumeInfo{{Name: "vol1", Size: 50, SizeAvailable: true, Used: false}},
		},
		Networks: types.Networks{
			Count: 2,
			List: []types.NetworkInfo{
				{Name: "bridge", CIDR: "172.17.0.0/16"},
				{Name: "net1", CIDR: "192.168.1.0/24"},
			},
		},
	}
	df := &facts.DockerSystemDfSummary{
		ImagesTotalBytes:     101,
		BuildCacheTotalBytes: 7,
	}

	Evaluate(report, cfg, df)

	if len(report.Issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(report.Issues))
	}
	found := false
	for _, issue := range report.Issues {
		if issue.RuleID == "DOCKER_STORAGE_BLOAT" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected DOCKER_STORAGE_BLOAT to be present")
	}
	if report.Issues[0].Facts["measurement"] != "system_df_layers_size" {
		t.Fatalf("expected measurement system_df_layers_size, got %#v", report.Issues[0].Facts["measurement"])
	}
	if report.Issues[0].Facts["total_image_size"] != uint64(101) {
		t.Fatalf("expected total_image_size 101, got %#v", report.Issues[0].Facts["total_image_size"])
	}
}

func TestEvaluate_DeterministicIssueOrdering(t *testing.T) {
	cfg := &config.Config{
		Rules: config.Rules{
			DiskUsage: config.DiskUsageRule{Threshold: 1},
			StorageBloat: config.StorageBloatRule{
				ImageSizeThreshold: 1,
			},
			Restarts: config.RestartsRule{Threshold: 1},
		},
	}

	report := &types.Report{
		Host: types.HostInfo{DiskUsage: map[string]*types.DiskInfo{"/": {Used: 2, Total: 2, UsedPercent: 100}}},
		Images: types.Images{
			Count:     1,
			TotalSize: 2,
		},
		Containers: types.Containers{
			List: []types.ContainerInfo{
				{ID: "b", Name: "/b", Status: "Up", RestartCount: 2},
			},
		},
	}

	Evaluate(report, cfg, nil)

	if len(report.Issues) < 3 {
		t.Fatalf("expected at least 3 issues, got %d", len(report.Issues))
	}

	// High severities should come first, then by rule ID, then by subject.
	for i := 1; i < len(report.Issues); i++ {
		prev := report.Issues[i-1]
		cur := report.Issues[i]
		if prev.Severity == "low" && (cur.Severity == "medium" || cur.Severity == "high") {
			t.Fatalf("issues not sorted by severity: prev=%+v cur=%+v", prev, cur)
		}
	}
}

