package rules

import (
	"fmt"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/types"
)

func checkLogBloat(report *types.Report, cfg *config.Config) {
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
}