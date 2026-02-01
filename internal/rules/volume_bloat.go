package rules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/example/docker-doctor/internal/types"
)

func checkVolumeBloat(report *types.Report) {
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

			// Sort unused by size
			sort.Slice(unusedVolumes, func(i, j int) bool {
				return unusedVolumes[i].size > unusedVolumes[j].size
			})

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
}