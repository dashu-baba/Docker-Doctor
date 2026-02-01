package rules

import (
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/facts"
	"github.com/example/docker-doctor/internal/types"
)

// topOffenders returns the top N items by size, formatted as strings
func topOffenders(items []struct{ id string; size uint64 }, n int) []string {
	sort.Slice(items, func(i, j int) bool {
		return items[i].size > items[j].size // descending
	})
	var result []string
	for i, item := range items {
		if i >= n {
			break
		}
		result = append(result, fmt.Sprintf("%s (%s)", item.id, humanBytes(item.size)))
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// cidrsOverlap checks if two CIDR strings overlap
func cidrsOverlap(cidr1, cidr2 string) bool {
	if cidr1 == "" || cidr2 == "" {
		return false
	}
	_, net1, err1 := net.ParseCIDR(cidr1)
	_, net2, err2 := net.ParseCIDR(cidr2)
	if err1 != nil || err2 != nil {
		return false
	}
	return net1.Contains(net2.IP) || net2.Contains(net1.IP) || net1.IP.Equal(net2.IP)
}

func humanBytes(v uint64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	f := float64(v)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	// show 2dp for GB+, 1dp for MB, none for KB/B
	decimals := 0
	if units[i] == "MB" {
		decimals = 1
	} else if units[i] == "GB" || units[i] == "TB" || units[i] == "PB" {
		decimals = 2
	}
	pow := 1.0
	for d := 0; d < decimals; d++ {
		pow *= 10
	}
	f = float64(int(f*pow)) / pow
	return fmt.Sprintf("%.*f %s", decimals, f, units[i])
}

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

		// Prepare top images
		var imageItems []struct{ id string; size uint64 }
		for _, img := range report.Images.List {
			imageItems = append(imageItems, struct{ id string; size uint64 }{img.ID, img.Size})
		}
		topImages := topOffenders(imageItems, 5) // top 5

		facts := map[string]interface{}{
			"total_images":     report.Images.Count,
			"total_image_size": imageSizeObserved,
			"size_threshold":   cfg.Rules.StorageBloat.ImageSizeThreshold,
			"measurement":      measurement,
			"build_cache_size": buildCacheSize,
			"top_images":       topImages,
		}

		solutions := []string{
			"Run 'docker system df' to see deduplicated disk usage and reclaimable space.",
			fmt.Sprintf("Total images: %d, total size: %s", report.Images.Count, humanBytes(imageSizeObserved)),
		}
		if len(topImages) > 0 {
			solutions = append(solutions, fmt.Sprintf("Top images by size: %s", strings.Join(topImages, ", ")))
			var removeCmds []string
			for _, img := range imageItems[:min(3, len(imageItems))] { // top 3
				if img.size > 0 {
					removeCmds = append(removeCmds, fmt.Sprintf("'docker image rm %s'", img.id))
				}
			}
			if len(removeCmds) > 0 {
				solutions = append(solutions, fmt.Sprintf("Remove largest images: %s", strings.Join(removeCmds, " ")))
			}
		}
		solutions = append(solutions, []string{
			"List images: 'docker images' (or 'docker image ls') and remove unused ones.",
			"Remove unused images: 'docker image prune -a'",
			"Prune build cache: 'docker builder prune' (or 'docker builder prune -a' for more).",
			"Use multi-stage builds to reduce image sizes.",
			"Consider using smaller base images.",
		}...)
		if buildCacheSize > 0 {
			solutions = append(solutions, fmt.Sprintf("Build cache size: %s - consider pruning if large.", humanBytes(buildCacheSize)))
		}

		report.Issues = append(report.Issues, types.Issue{
			RuleID:      "DOCKER_STORAGE_BLOAT",
			Subject:     "images_total",
			Severity:    severity,
			Category:    "storage_bloat",
			Description: fmt.Sprintf("Docker image disk usage is %d bytes, exceeding threshold of %d bytes", imageSizeObserved, cfg.Rules.StorageBloat.ImageSizeThreshold),
			Facts:       facts,
			Solutions:   solutions,
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

	// LOG_BLOAT
	if cfg.Rules.LogBloat.Enabled {
		for _, container := range report.Containers.List {
			if container.LogSize > cfg.Rules.LogBloat.SizeThreshold {
				severity := "medium"
				if container.LogSize > cfg.Rules.LogBloat.SizeThreshold*2 {
					severity = "high"
				}

				report.Issues = append(report.Issues, types.Issue{
					RuleID:      "LOG_BLOAT",
					Subject:     "container=" + container.ID,
					Severity:    severity,
					Category:    "log_bloat",
					Description: fmt.Sprintf("Container %s (%s) has large log files (%d bytes), exceeding threshold of %d bytes", container.Name, container.ID, container.LogSize, cfg.Rules.LogBloat.SizeThreshold),
					Facts: map[string]interface{}{
						"container_id":   container.ID,
						"container_name": container.Name,
						"log_size":       container.LogSize,
						"threshold":      cfg.Rules.LogBloat.SizeThreshold,
					},
					Solutions: []string{
						fmt.Sprintf("Check log size: 'docker logs %s | wc -c'", container.ID),
						fmt.Sprintf("Rotate logs: 'docker logs %s > /tmp/logs && docker logs %s --tail 0'", container.ID, container.ID),
						"Use log drivers like 'json-file' with 'max-size' and 'max-file' options.",
						"Configure logging in docker-compose.yml or Dockerfile.",
						"Consider using external logging solutions (e.g., ELK stack, Fluentd).",
						"Monitor application logging levels to reduce verbosity.",
					},
				})
			}
		}
	}

	// VOLUME_BLOAT
	if true { // Always check volumes
		var unusedVolumes []struct{ id string; size uint64 }
		totalVolumeSize := uint64(0)
		usedVolumeCount := 0

		for _, vol := range report.Volumes.List {
			totalVolumeSize += vol.Size
			if vol.Used {
				usedVolumeCount++
			} else {
				unusedVolumes = append(unusedVolumes, struct{ id string; size uint64 }{vol.Name, vol.Size})
			}
		}

		// Report unused volumes
		if len(unusedVolumes) > 0 {
			severity := "low"
			if len(unusedVolumes) > 5 {
				severity = "medium"
			}

			topUnused := topOffenders(unusedVolumes, 5)

			facts := map[string]interface{}{
				"total_volumes":     report.Volumes.Count,
				"used_volumes":      usedVolumeCount,
				"unused_volumes":    len(unusedVolumes),
				"total_volume_size": totalVolumeSize,
				"top_unused":        topUnused,
			}

			solutions := []string{
				fmt.Sprintf("Found %d unused volumes out of %d total", len(unusedVolumes), report.Volumes.Count),
			}
			if len(topUnused) > 0 {
				solutions = append(solutions, fmt.Sprintf("Largest unused volumes: %s", strings.Join(topUnused, ", ")))
			}
			solutions = append(solutions, []string{
				"Remove unused volumes: 'docker volume rm <volume_name>'",
				"List all volumes: 'docker volume ls'",
				"Prune unused volumes: 'docker volume prune'",
				"Review container configurations to ensure volumes are properly attached.",
			}...)

			report.Issues = append(report.Issues, types.Issue{
				RuleID:      "VOLUME_BLOAT",
				Subject:     "volumes_unused",
				Severity:    severity,
				Category:    "storage_bloat",
				Description: fmt.Sprintf("Found %d unused Docker volumes that can be cleaned up", len(unusedVolumes)),
				Facts:       facts,
				Solutions:   solutions,
			})
		}
	}

	// NETWORK_OVERLAP
	if true { // Always check networks
		var overlapping []string
		checked := make(map[int]bool)

		for i, net1 := range report.Networks.List {
			for j, net2 := range report.Networks.List {
				if i >= j || checked[i*len(report.Networks.List)+j] {
					continue
				}
				checked[i*len(report.Networks.List)+j] = true
				if cidrsOverlap(net1.CIDR, net2.CIDR) {
					overlapping = append(overlapping, fmt.Sprintf("%s (%s) and %s (%s)", net1.Name, net1.CIDR, net2.Name, net2.CIDR))
				}
			}
		}

		if len(overlapping) > 0 {
			report.Issues = append(report.Issues, types.Issue{
				RuleID:      "NETWORK_OVERLAP",
				Subject:     "networks_overlap",
				Severity:    "high",
				Category:    "networking",
				Description: fmt.Sprintf("Found %d overlapping Docker network CIDRs that may cause connectivity issues", len(overlapping)),
				Facts: map[string]interface{}{
					"overlapping_networks": overlapping,
					"total_networks":       report.Networks.Count,
				},
				Solutions: []string{
					"Review and reconfigure overlapping network subnets",
					"Use non-overlapping CIDR ranges for Docker networks",
					"Remove unnecessary networks: 'docker network rm <network_name>'",
					"Recreate networks with proper subnets: 'docker network create --subnet <cidr> <name>'",
					"Check network configurations in docker-compose files",
				},
			})
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

