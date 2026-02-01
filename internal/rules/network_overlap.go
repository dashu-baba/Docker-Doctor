package rules

import (
	"fmt"

	"github.com/dashu-baba/docker-doctor/internal/config"
	"github.com/dashu-baba/docker-doctor/internal/types"
)

func checkNetworkOverlap(report *types.Report, cfg *config.Config) {
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
}