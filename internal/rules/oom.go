package rules

import (
	"fmt"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/types"
)

func checkOOM(report *types.Report, cfg *config.Config) {
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
}