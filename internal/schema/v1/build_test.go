package v1

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dashu-baba/docker-doctor/internal/config"
	"github.com/dashu-baba/docker-doctor/internal/types"
)

func TestBuildFromV0_HasRequiredTopLevelFields(t *testing.T) {
	v0 := &types.Report{
		Host: types.HostInfo{OS: "linux", Arch: "amd64", DiskUsage: map[string]*types.DiskInfo{}},
		Docker: types.DockerInfo{
			Version:    "1.2.3",
			DaemonInfo: map[string]interface{}{"storage_driver": "overlay2"},
		},
		Containers: types.Containers{Count: 0, List: nil},
		Images:     types.Images{Count: 0, List: nil, TotalSize: 0},
		Volumes:    types.Volumes{Count: 0, List: nil},
		Networks:   types.Networks{Count: 0, List: nil},
		Issues:     []types.Issue{},
		Timestamp:  time.Now(),
	}
	cfg := &config.Config{
		Scan: config.ScanConfig{
			Mode:       "basic",
			Timeout:    30,
			DockerHost: "unix:///var/run/docker.sock",
			Version:    "1.41",
		},
	}

	r := BuildFromV0(context.Background(), v0, cfg, "1.41", time.Now().Add(-time.Second), time.Now(), "dev", "", "")

	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}

	for _, k := range []string{
		"schemaVersion",
		"tool",
		"scan",
		"target",
		"collectors",
		"summary",
		"findings",
		"errors",
		"raw",
	} {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing top-level key %q in v1 report JSON", k)
		}
	}
}

