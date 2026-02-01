package rules

import (
	"fmt"
	"strings"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/facts"
	"github.com/example/docker-doctor/internal/types"
)

func checkStorageBloat(report *types.Report, cfg *config.Config, df *facts.DockerSystemDfSummary) {
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

		factsMap := map[string]interface{}{
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
			solutions = append(solutions, "Remove specific images: 'docker image rm <image_id>'")
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
			Facts:       factsMap,
			Solutions:   solutions,
		})
	}
}