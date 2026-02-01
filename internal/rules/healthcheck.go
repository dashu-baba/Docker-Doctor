package rules

import (
	"fmt"
	"time"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/types"
)

func checkHealthcheck(report *types.Report, cfg *config.Config) {
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
}