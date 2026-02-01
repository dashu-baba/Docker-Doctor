package rules

import (
	"fmt"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/types"
)

func checkDaemonRisky(report *types.Report, cfg *config.Config) {
	// DAEMON_RISKY_SETTINGS
	if true { // Always check daemon settings
		var riskySettings []string
		daemonInfo := report.Docker.DaemonInfo

		if experimental, ok := daemonInfo["experimental"].(bool); ok && experimental {
			riskySettings = append(riskySettings, "experimental features enabled")
		}

		if registryConfig, ok := daemonInfo["registry_config"].(map[string]interface{}); ok {
			if insecureRegs, ok := registryConfig["InsecureRegistryCIDRs"].([]interface{}); ok && len(insecureRegs) > 0 {
				riskySettings = append(riskySettings, fmt.Sprintf("insecure registries configured: %d entries", len(insecureRegs)))
			}
		}

		if loggingDriver, ok := daemonInfo["logging_driver"].(string); ok {
			if loggingDriver == "none" {
				riskySettings = append(riskySettings, "logging driver set to 'none'")
			}
		}

		if len(riskySettings) > 0 {
			severity := "medium"
			if len(riskySettings) > 2 {
				severity = "high"
			}

			report.Issues = append(report.Issues, types.Issue{
				RuleID:      "DAEMON_RISKY_SETTINGS",
				Subject:     "daemon_config",
				Severity:    severity,
				Category:    "configuration",
				Description: fmt.Sprintf("Docker daemon has %d potentially risky settings configured", len(riskySettings)),
				Facts: map[string]interface{}{
					"risky_settings": riskySettings,
				},
				Solutions: []string{
					"Review Docker daemon configuration for security implications",
					"Disable experimental features in production",
					"Avoid insecure registries unless absolutely necessary",
					"Configure appropriate logging drivers",
					"Check /etc/docker/daemon.json for configuration details",
					"Restart Docker daemon after configuration changes",
				},
			})
		}
	}
}