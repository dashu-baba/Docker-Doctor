package rules

import (
	"fmt"
	"strings"

	"github.com/dashu-baba/docker-doctor/internal/config"
	"github.com/dashu-baba/docker-doctor/internal/types"
)

func checkDiskUsage(report *types.Report, cfg *config.Config) {
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
}