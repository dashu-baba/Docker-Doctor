package rules

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/facts"
	"github.com/example/docker-doctor/internal/types"
)

// Evaluate runs all rules and appends issues to report.Issues.
// It also ensures deterministic ordering of report.Issues.
func Evaluate(report *types.Report, cfg *config.Config, df *facts.DockerSystemDfSummary) {
	if report == nil || cfg == nil {
		return
	}

	// DISK_USAGE_HIGH
	for path, disk := range report.Host.DiskUsage {
		if disk.UsedPercent > float64(cfg.Rules.DiskUsage.Threshold) {
			severity := "medium"
			if disk.UsedPercent > 90 {
				severity = "high"
			} else if disk.UsedPercent < 85 {
				severity = "low"
			}

			facts := map[string]interface{}{
				"path":         path,
				"used_bytes":   disk.Used,
				"total_bytes":  disk.Total,
				"used_percent": disk.UsedPercent,
				"threshold":    cfg.Rules.DiskUsage.Threshold,
			}

			solutions := []string{
				"Identify and remove unused files or directories.",
				"Consider increasing disk space if possible.",
			}

			if path == "/var/lib/docker" || strings.Contains(path, "docker") {
				solutions = append(solutions,
					"Run 'docker system prune' to remove unused containers, images, and networks.",
					"Run 'docker volume prune' to remove unused volumes.",
					"Inspect and clean up large Docker images or logs.",
				)
			} else if path == "/" {
				solutions = append(solutions,
					"Check for large log files in /var/log and rotate them.",
					"Remove old kernel packages: 'apt autoremove' (on Ubuntu/Debian).",
				)
			}

			report.Issues = append(report.Issues, types.Issue{
				RuleID:      "DISK_USAGE_HIGH",
				Subject:     "path=" + path,
				Severity:    severity,
				Category:    "disk_usage",
				Description: fmt.Sprintf("Disk usage for %s is %.2f%%, exceeding threshold of %d%%", path, disk.UsedPercent, cfg.Rules.DiskUsage.Threshold),
				Facts:       facts,
				Solutions:   solutions,
			})
		}
	}

	// DOCKER_STORAGE_BLOAT (prefer /system/df)
	imageSizeObserved := report.Images.TotalSize
	measurement := "image_list_sum"
	buildCacheSize := uint64(0)
	if df != nil {
		if df.ImagesTotalBytes > 0 {
			imageSizeObserved = df.ImagesTotalBytes
			measurement = "system_df_layers_size"
		}
		buildCacheSize = df.BuildCacheTotalBytes
	}

	if imageSizeObserved > cfg.Rules.StorageBloat.ImageSizeThreshold {
		severity := "medium"
		if imageSizeObserved > cfg.Rules.StorageBloat.ImageSizeThreshold*2 {
			severity = "high"
		}

		facts := map[string]interface{}{
			"total_images":     report.Images.Count,
			"total_image_size": imageSizeObserved,
			"size_threshold":   cfg.Rules.StorageBloat.ImageSizeThreshold,
			"measurement":      measurement,
			"build_cache_size": buildCacheSize,
		}

		report.Issues = append(report.Issues, types.Issue{
			RuleID:      "DOCKER_STORAGE_BLOAT",
			Subject:     "images_total",
			Severity:    severity,
			Category:    "storage_bloat",
			Description: fmt.Sprintf("Docker image disk usage is %d bytes, exceeding threshold of %d bytes", imageSizeObserved, cfg.Rules.StorageBloat.ImageSizeThreshold),
			Facts:       facts,
			Solutions: []string{
				"Run 'docker system df' to see deduplicated disk usage and reclaimable space.",
				"List images: 'docker images' (or 'docker image ls') and remove unused ones.",
				"Remove unused images: 'docker image prune -a'",
				"Prune build cache: 'docker builder prune' (or 'docker builder prune -a' for more).",
				"Use multi-stage builds to reduce image sizes.",
				"Consider using smaller base images.",
			},
		})
	}

	// RESTART_LOOP
	for _, container := range report.Containers.List {
		isRestarting := strings.Contains(strings.ToLower(container.Status), "restarting")
		overThreshold := container.RestartCount > cfg.Rules.Restarts.Threshold
		if isRestarting || overThreshold {
			report.Issues = append(report.Issues, types.Issue{
				RuleID:      "RESTART_LOOP",
				Subject:     "container=" + container.ID,
				Severity:    "high",
				Category:    "restarts",
				Description: fmt.Sprintf("Container %s (%s) is restarting or exceeded restart threshold", container.Name, container.ID),
				Facts: map[string]interface{}{
					"container_id":   container.ID,
					"container_name": container.Name,
					"status":         container.Status,
					"restart_count":  container.RestartCount,
					"threshold":      cfg.Rules.Restarts.Threshold,
				},
				Solutions: []string{
					fmt.Sprintf("Check logs: 'docker logs %s'", container.ID),
					"Inspect container configuration for errors.",
					"Check resource limits (CPU/memory) that might cause crashes.",
					"Review application code for stability issues.",
					"Stop and restart the container manually if needed.",
				},
			})
		}
	}

	// OOM_KILLED
	if cfg.Rules.OOM.Enabled {
		for _, container := range report.Containers.List {
			if container.OOMKilled {
				report.Issues = append(report.Issues, types.Issue{
					RuleID:      "OOM_KILLED",
					Subject:     "container=" + container.ID,
					Severity:    "high",
					Category:    "oom",
					Description: fmt.Sprintf("Container %s (%s) was killed due to out-of-memory condition", container.Name, container.ID),
					Facts: map[string]interface{}{
						"container_id":   container.ID,
						"container_name": container.Name,
						"status":         container.Status,
					},
					Solutions: []string{
						fmt.Sprintf("Check logs: 'docker logs %s'", container.ID),
						"Increase memory limit: 'docker update --memory <limit> " + container.ID + "'",
						"Optimize application memory usage.",
						"Check for memory leaks in the application.",
						"Consider using memory profiling tools.",
						"Review container resource allocation.",
					},
				})
			}
		}
	}

	// HEALTHCHECK_UNHEALTHY (not currently populated, but keep the rule)
	if cfg.Rules.Healthcheck.Enabled {
		for _, container := range report.Containers.List {
			if container.HealthStatus == "unhealthy" {
				duration := time.Since(container.UnhealthySince)
				severity := "medium"
				if duration > time.Hour {
					severity = "high"
				}

				report.Issues = append(report.Issues, types.Issue{
					RuleID:      "HEALTHCHECK_UNHEALTHY",
					Subject:     "container=" + container.ID,
					Severity:    severity,
					Category:    "healthcheck",
					Description: fmt.Sprintf("Container %s (%s) has been unhealthy for %s", container.Name, container.ID, duration.Round(time.Second)),
					Facts: map[string]interface{}{
						"container_id":       container.ID,
						"container_name":     container.Name,
						"health_status":      container.HealthStatus,
						"unhealthy_since":    container.UnhealthySince,
						"unhealthy_duration": duration.String(),
					},
					Solutions: []string{
						fmt.Sprintf("Check healthcheck logs: 'docker inspect %s | jq .State.Health.Log'", container.ID),
						fmt.Sprintf("Check container logs: 'docker logs %s'", container.ID),
						"Review healthcheck configuration in Dockerfile or compose file.",
						"Ensure the healthcheck command is appropriate for the application.",
						"Check application responsiveness and dependencies.",
						"Consider adjusting healthcheck timeouts or intervals.",
					},
				})
			}
		}
	}

	// Deterministic ordering for diff-friendly output
	severityRank := func(s string) int {
		switch strings.ToLower(s) {
		case "high":
			return 0
		case "medium":
			return 1
		case "low":
			return 2
		default:
			return 3
		}
	}
	sort.Slice(report.Issues, func(i, j int) bool {
		if severityRank(report.Issues[i].Severity) != severityRank(report.Issues[j].Severity) {
			return severityRank(report.Issues[i].Severity) < severityRank(report.Issues[j].Severity)
		}
		if report.Issues[i].RuleID != report.Issues[j].RuleID {
			return report.Issues[i].RuleID < report.Issues[j].RuleID
		}
		return report.Issues[i].Subject < report.Issues[j].Subject
	})
}

