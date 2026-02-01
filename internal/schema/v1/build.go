package v1

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"time"

	"github.com/example/docker-doctor/internal/collector"
	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/types"
)

func BuildFromV0(ctx context.Context, v0 *types.Report, cfg *config.Config, apiVersion string, startedAt time.Time, finishedAt time.Time, version, gitCommit, buildTime string) Report {
	df, dfErr := collector.CollectDockerSystemDfSummary(ctx, cfg.Scan.DockerHost, apiVersion)

	containersRunning := 0
	containersStopped := 0
	for _, c := range v0.Containers.List {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(c.Status)), "up") {
			containersRunning++
		} else {
			containersStopped++
		}
	}

	findings := make([]Finding, 0, len(v0.Issues))
	counts := SummaryFindingCounts{}
	for _, is := range v0.Issues {
		f := mapIssueToFinding(is)
		findings = append(findings, f)
		switch f.Severity {
		case "critical":
			counts.Critical++
		case "warning":
			counts.Warning++
		default:
			counts.Info++
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		sr := func(s string) int {
			switch s {
			case "critical":
				return 0
			case "warning":
				return 1
			default:
				return 2
			}
		}
		if sr(findings[i].Severity) != sr(findings[j].Severity) {
			return sr(findings[i].Severity) < sr(findings[j].Severity)
		}
		if findings[i].ID != findings[j].ID {
			return findings[i].ID < findings[j].ID
		}
		return findings[i].Fingerprint < findings[j].Fingerprint
	})

	collectors := []Collector{
		{
			Name:       "docker_engine",
			Status:     "ok",
			DurationMs: finishedAt.Sub(startedAt).Milliseconds(),
			Errors:     []string{},
		},
		{
			Name:       "docker_system_df",
			Status:     "ok",
			DurationMs: 0,
			Errors:     []string{},
		},
		{
			Name:       "host_fs",
			Status:     "skipped",
			DurationMs: 0,
			Errors:     []string{"host filesystem collector not enabled in this build"},
		},
	}
	if dfErr != nil {
		for i := range collectors {
			if collectors[i].Name == "docker_system_df" {
				collectors[i].Status = "error"
				collectors[i].Errors = []string{dfErr.Error()}
			}
		}
	}
	sort.Slice(collectors, func(i, j int) bool { return collectors[i].Name < collectors[j].Name })

	systemDf := DockerSystemDf{}
	if df != nil {
		systemDf = DockerSystemDf{
			ImagesTotalBytes:             df.ImagesTotalBytes,
			ContainersWritableTotalBytes: df.ContainersWritableTotalBytes,
			VolumesTotalBytes:            df.VolumesTotalBytes,
			BuildCacheTotalBytes:         df.BuildCacheTotalBytes,
		}
	}

	return Report{
		SchemaVersion: "1.0",
		Tool: Tool{
			Name:      "docker-host-doctor",
			Version:   version,
			GitCommit: gitCommit,
			BuildTime: buildTime,
		},
		Scan: Scan{
			ScanID:         newScanID(finishedAt),
			StartedAt:      startedAt.UTC(),
			FinishedAt:     finishedAt.UTC(),
			DurationMs:     finishedAt.Sub(startedAt).Milliseconds(),
			Mode:           cfg.Scan.Mode,
			EffectiveMode:  cfg.Scan.Mode,
			TimeoutSeconds: cfg.Scan.Timeout,
			Capabilities: Capabilities{
				DockerAPI:                 true,
				HostFSMounted:             false,
				DaemonConfigReadable:      false,
				ContainerLogFilesReadable: false,
			},
			Redaction: Redaction{
				Enabled:         false,
				MaskedIPs:       false,
				MaskedHostnames: false,
				DroppedEnvVars:  false,
				Notes:           []string{},
			},
		},
		Target: Target{
			Host: TargetHost{
				HostID:        v0.Host.HostID,
				Hostname:      v0.Host.Hostname,
				OS:            v0.Host.OS,
				Arch:          v0.Host.Arch,
				Kernel:        v0.Host.Kernel,
				UptimeSeconds: v0.Host.UptimeSeconds,
			},
			Docker: TargetDocker{
				EngineVersion:  v0.Docker.Version,
				APIVersion:     apiVersion,
				StorageDriver:  stringFromDaemonInfo(v0.Docker.DaemonInfo, "storage_driver"),
				CgroupVersion:  v0.Docker.CgroupVersion,
				DataRoot:       v0.Docker.DataRoot,
			},
		},
		Collectors: collectors,
		Summary: Summary{
			Counts: SummaryCounts{
				ContainersRunning: containersRunning,
				ContainersStopped: containersStopped,
				Images:            v0.Images.Count,
				Volumes:           v0.Volumes.Count,
				Networks:          v0.Networks.Count,
			},
			ResourceSnapshot: SummaryResourceSnapshot{
				DockerSystemDf: systemDf,
			},
			FindingCounts: counts,
		},
		Findings: findings,
		Errors:   []string{},
		Raw: Raw{
			Included: false,
			Reason:   "privacy_and_size",
		},
	}
}

func newScanID(t time.Time) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return t.UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(b)
}

func stringFromDaemonInfo(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func categorizeSolutions(solutions []string) (steps, commands, notes []string) {
	for _, sol := range solutions {
		sol = strings.TrimSpace(sol)
		if strings.Contains(sol, "docker image rm") || strings.Contains(sol, "docker builder prune") || strings.Contains(sol, "docker system df") {
			commands = append(commands, sol)
		} else if strings.Contains(sol, "Total") || strings.Contains(sol, "Top") || strings.Contains(sol, "Build cache") || strings.Contains(sol, "Consider") {
			notes = append(notes, sol)
		} else {
			steps = append(steps, sol)
		}
	}
	return
}

func mapIssueToFinding(is types.Issue) Finding {
	severity := "info"
	switch strings.ToLower(is.Severity) {
	case "high":
		severity = "critical"
	case "medium":
		severity = "warning"
	case "low":
		severity = "info"
	}

	category := "general"
	switch is.RuleID {
	case "DISK_USAGE_HIGH":
		category = "host"
	case "DOCKER_STORAGE_BLOAT", "LOG_BLOAT", "VOLUME_BLOAT", "VOLUME_SIZE_HIGH":
		category = "storage"
	case "RESTART_LOOP", "OOM_KILLED", "HEALTHCHECK_UNHEALTHY":
		category = "stability"
	case "NETWORK_OVERLAP":
		category = "networking"
	case "DAEMON_RISKY_SETTINGS":
		category = "configuration"
	}

	title := is.RuleID
	switch is.RuleID {
	case "DISK_USAGE_HIGH":
		title = "Disk usage is above threshold"
	case "DOCKER_STORAGE_BLOAT":
		title = "Docker storage usage is high"
	case "RESTART_LOOP":
		title = "Container is restarting frequently"
	case "OOM_KILLED":
		title = "Container was killed by OOM"
	case "HEALTHCHECK_UNHEALTHY":
		title = "Container healthcheck is unhealthy"
	case "LOG_BLOAT":
		title = "Container logs are bloated"
	case "VOLUME_BLOAT":
		title = "Unused Docker volumes detected"
	case "VOLUME_SIZE_HIGH":
		title = "Large Docker volumes detected"
	case "NETWORK_OVERLAP":
		title = "Docker network CIDRs overlap"
	case "DAEMON_RISKY_SETTINGS":
		title = "Docker daemon has risky settings"
	}

	scope := Scope{}
	if strings.HasPrefix(is.Subject, "container=") {
		scope.ContainerID = strings.TrimPrefix(is.Subject, "container=")
	}
	if strings.HasPrefix(is.Subject, "path=") {
		scope.Path = strings.TrimPrefix(is.Subject, "path=")
	}
	if v, ok := is.Facts["container_name"]; ok {
		if s, ok := v.(string); ok {
			scope.ContainerName = s
		}
	}

	evidence := make([]Evidence, 0, len(is.Facts))
	for k, v := range is.Facts {
		evidence = append(evidence, Evidence{
			Type:  "fact",
			Key:   k,
			Value: v,
		})
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].Key < evidence[j].Key })

	steps, commands, notes := categorizeSolutions(is.Solutions)

	reco := Recommendation{
		Risk:     "planned",
		Title:    "Recommended actions",
		Steps:    steps,
		Commands: commands,
		Notes:    notes,
	}

	fp := is.RuleID
	if strings.TrimSpace(is.Subject) != "" {
		fp += ":" + is.Subject
	} else {
		fp += ":global"
	}

	confidence := "medium"
	switch is.RuleID {
	case "DISK_USAGE_HIGH", "VOLUME_SIZE_HIGH", "LOG_BLOAT":
		confidence = "high" // Relies on host FS access
	case "DOCKER_STORAGE_BLOAT", "RESTART_LOOP", "OOM_KILLED", "HEALTHCHECK_UNHEALTHY", "VOLUME_BLOAT", "NETWORK_OVERLAP", "DAEMON_RISKY_SETTINGS":
		confidence = "medium" // API-based
	default:
		confidence = "low"
	}

	return Finding{
		ID:              is.RuleID,
		Fingerprint:     fp,
		Severity:        severity,
		Confidence:      confidence,
		Category:        category,
		Title:           title,
		Summary:         is.Description,
		Scope:           scope,
		Evidence:        evidence,
		Recommendations: []Recommendation{reco},
		References:      []Reference{},
	}
}

