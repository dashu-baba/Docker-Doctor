package rules

import (
	"fmt"
	"strings"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/types"
)

func checkRestarts(report *types.Report, cfg *config.Config) {
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
}