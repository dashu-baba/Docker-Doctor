package rules

import (
	"fmt"
	"strings"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/types"
)

func checkVolumeSize(report *types.Report, cfg *config.Config) {
	// VOLUME_SIZE_HIGH
	if cfg.Rules.VolumeSize.Enabled {
		var largeVolumes []struct{ name string; size uint64; available bool }
		for _, vol := range report.Volumes.List {
			if vol.SizeAvailable && vol.Size > cfg.Rules.VolumeSize.SizeThreshold {
				largeVolumes = append(largeVolumes, struct{ name string; size uint64; available bool }{vol.Name, vol.Size, vol.SizeAvailable})
			} else if !vol.SizeAvailable {
				// Note unavailable
				largeVolumes = append(largeVolumes, struct{ name string; size uint64; available bool }{vol.Name, 0, false})
			}
		}

		if len(largeVolumes) > 0 {
			severity := "medium"
			if len(largeVolumes) > 3 {
				severity = "high"
			}

			var descriptions []string
			var facts map[string]interface{}

			unavailable := []string{}
			large := []string{}
			for _, lv := range largeVolumes {
				if lv.available {
					large = append(large, fmt.Sprintf("%s (%s)", lv.name, humanBytes(lv.size)))
				} else {
					unavailable = append(unavailable, lv.name)
				}
			}

			if len(large) > 0 {
				descriptions = append(descriptions, fmt.Sprintf("Found %d volumes exceeding size threshold of %s", len(large), humanBytes(cfg.Rules.VolumeSize.SizeThreshold)))
				facts = map[string]interface{}{
					"large_volumes":  large,
					"size_threshold": cfg.Rules.VolumeSize.SizeThreshold,
				}
			}

			if len(unavailable) > 0 {
				descriptions = append(descriptions, fmt.Sprintf("Volume sizes unavailable for %d volumes (host FS not accessible)", len(unavailable)))
				if facts == nil {
					facts = map[string]interface{}{}
				}
				facts["unavailable_volumes"] = unavailable
			}

			description := strings.Join(descriptions, ". ")

			solutions := []string{
				"Review volume contents and remove unnecessary data",
				"Consider archiving old data or using smaller volumes",
			}
			if len(large) > 0 {
				solutions = append(solutions, "Inspect large volumes: 'docker run --rm -v <volume>:/data alpine du -sh /data'")
			}
			if len(unavailable) > 0 {
				solutions = append(solutions, "Volume sizes are not available on this system (host FS access required)")
			}

			report.Issues = append(report.Issues, types.Issue{
				RuleID:      "VOLUME_SIZE_HIGH",
				Subject:     "volumes_large",
				Severity:    severity,
				Category:    "storage",
				Description: description,
				Facts:       facts,
				Solutions:   solutions,
			})
		}
	}
}