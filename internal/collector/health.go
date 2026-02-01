package collector

import (
	"encoding/json"
	"strings"
	"time"
)

type inspectHealthProbe struct {
	State struct {
		Health *struct {
			Status string `json:"Status"`
			Log    []struct {
				Start    string `json:"Start"`
				End      string `json:"End"`
				ExitCode int    `json:"ExitCode"`
			} `json:"Log"`
		} `json:"Health"`
	} `json:"State"`
}

func parseHealthFromInspectRaw(raw []byte) (status string, unhealthySince time.Time) {
	var p inspectHealthProbe
	if err := json.Unmarshal(raw, &p); err != nil {
		return "none", time.Time{}
	}
	if p.State.Health == nil {
		return "none", time.Time{}
	}

	s := strings.ToLower(strings.TrimSpace(p.State.Health.Status))
	if s == "" {
		return "none", time.Time{}
	}
	if s != "unhealthy" {
		return s, time.Time{}
	}

	// Best-effort: infer when it became unhealthy using the health log.
	// We scan backward for the last "success" (ExitCode==0); unhealthySince is the next entry's Start.
	logs := p.State.Health.Log
	if len(logs) == 0 {
		return "unhealthy", time.Time{}
	}

	parse := func(ts string) time.Time {
		// Docker uses RFC3339Nano timestamps in health logs.
		t, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return time.Time{}
		}
		return t
	}

	start := parse(logs[0].Start)
	for i := len(logs) - 1; i >= 0; i-- {
		if logs[i].ExitCode == 0 {
			if i+1 < len(logs) {
				if t := parse(logs[i+1].Start); !t.IsZero() {
					start = t
				}
			}
			break
		}
	}
	return "unhealthy", start
}

