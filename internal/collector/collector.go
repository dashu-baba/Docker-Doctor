package collector

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/docker/engine-api/client"
	dtypes "github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/filters"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/types"
)

// Collect gathers all the required data for the report.
func Collect(ctx context.Context, apiVersion string, cfg *config.Config) (*types.Report, error) {
	report := &types.Report{
		Timestamp: time.Now(),
		Issues:    []types.Issue{},
	}

	// Collect host info
	hostInfo, err := collectHostInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to collect host info: %w", err)
	}
	report.Host = *hostInfo

	// Collect Docker info
	dockerInfo, err := collectDockerInfo(ctx, apiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to collect Docker info: %w", err)
	}
	report.Docker = *dockerInfo

	// Collect containers
	containers, err := collectContainers(ctx, apiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to collect containers: %w", err)
	}
	report.Containers = *containers

	// Collect images
	images, err := collectImages(ctx, apiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to collect images: %w", err)
	}
	report.Images = *images

	// Collect volumes
	volumes, err := collectVolumes(ctx, apiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to collect volumes: %w", err)
	}
	report.Volumes = *volumes

	// Run diagnostics
	diagnose(report, cfg)

	return report, nil
}

func collectHostInfo() (*types.HostInfo, error) {
	info := &types.HostInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		DiskUsage: make(map[string]*types.DiskInfo),
	}

	// Get disk usage for root
	if diskInfo, err := getDiskUsage("/"); err == nil {
		info.DiskUsage["/"] = diskInfo
	}

	// Get disk usage for /var/lib/docker if exists
	dockerPath := "/var/lib/docker"
	if _, err := os.Stat(dockerPath); err == nil {
		if diskInfo, err := getDiskUsage(dockerPath); err == nil {
			info.DiskUsage[dockerPath] = diskInfo
		}
	}

	return info, nil
}

func getDiskUsage(path string) (*types.DiskInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, err
	}
	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	used := total - available
	usedPercent := float64(used) / float64(total) * 100
	return &types.DiskInfo{
		Used:        used,
		Total:       total,
		UsedPercent: usedPercent,
	}, nil
}

func newClient(apiVersion string) (*client.Client, error) {
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}
	return client.NewClient(host, apiVersion, nil, nil)
}

func collectDockerInfo(ctx context.Context, apiVersion string) (*types.DockerInfo, error) {
	cli, err := newClient(apiVersion)
	if err != nil {
		return nil, err
	}

	version, err := cli.ServerVersion()
	if err != nil {
		return nil, err
	}

	info, err := cli.Info()
	if err != nil {
		return nil, err
	}

	dockerInfo := &types.DockerInfo{
		Version: version.Version,
		DaemonInfo: map[string]interface{}{
			"server_version": info.ServerVersion,
			"os":             info.OSType,
			"arch":           info.Architecture,
		},
	}

	return dockerInfo, nil
}

func collectContainers(ctx context.Context, apiVersion string) (*types.Containers, error) {
	cli, err := newClient(apiVersion)
	if err != nil {
		return nil, err
	}

	containers, err := cli.ContainerList(dtypes.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	cont := &types.Containers{
		Count: len(containers),
		List:  make([]types.ContainerInfo, 0, len(containers)),
	}
	for _, c := range containers {
		// Inspect to get OOM and health status
		inspect, err := cli.ContainerInspect(c.ID)
		oomKilled := false
		healthStatus := "none"
		var unhealthySince time.Time
		if err == nil {
			oomKilled = inspect.State.OOMKilled
			// Health checks not available in this API version
			// Assume healthy if running
		}
		// Ignore inspect errors

		cont.List = append(cont.List, types.ContainerInfo{
			ID:             c.ID[:12],  // short ID
			Name:           c.Names[0], // first name
			RestartCount:   0,          // not available in list
			Status:         c.Status,
			OOMKilled:      oomKilled,
			HealthStatus:   healthStatus,
			UnhealthySince: unhealthySince,
		})
	}

	// Deterministic ordering for diff-friendly output
	sort.Slice(cont.List, func(i, j int) bool {
		if cont.List[i].Name != cont.List[j].Name {
			return cont.List[i].Name < cont.List[j].Name
		}
		return cont.List[i].ID < cont.List[j].ID
	})

	return cont, nil
}

func collectImages(ctx context.Context, apiVersion string) (*types.Images, error) {
	cli, err := newClient(apiVersion)
	if err != nil {
		return nil, err
	}

	images, err := cli.ImageList(dtypes.ImageListOptions{All: true})
	if err != nil {
		return nil, err
	}

	img := &types.Images{
		Count:     len(images),
		List:      make([]string, 0, len(images)),
		TotalSize: 0,
	}
	for _, i := range images {
		img.List = append(img.List, i.ID)
		img.TotalSize += uint64(i.Size)
	}

	return img, nil
}

func collectVolumes(ctx context.Context, apiVersion string) (*types.Volumes, error) {
	cli, err := newClient(apiVersion)
	if err != nil {
		return nil, err
	}

	volumes, err := cli.VolumeList(filters.Args{})
	if err != nil {
		return nil, err
	}

	vol := &types.Volumes{
		Count: len(volumes.Volumes),
		List:  make([]string, 0, len(volumes.Volumes)),
	}
	for _, v := range volumes.Volumes {
		vol.List = append(vol.List, v.Name)
	}

	return vol, nil
}

func diagnose(report *types.Report, cfg *config.Config) {
	// Check disk usage
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

			issue := types.Issue{
				RuleID:      "DISK_USAGE_HIGH",
				Subject:     "path=" + path,
				Severity:    severity,
				Category:    "disk_usage",
				Description: fmt.Sprintf("Disk usage for %s is %.2f%%, exceeding threshold of %d%%", path, disk.UsedPercent, cfg.Rules.DiskUsage.Threshold),
				Facts:       facts,
				Solutions:   solutions,
			}
			report.Issues = append(report.Issues, issue)
		}
	}

	// Check storage bloat
	if report.Images.TotalSize > cfg.Rules.StorageBloat.ImageSizeThreshold {
		severity := "medium"
		if report.Images.TotalSize > cfg.Rules.StorageBloat.ImageSizeThreshold*2 {
			severity = "high"
		}

		facts := map[string]interface{}{
			"total_images":     report.Images.Count,
			"total_image_size": report.Images.TotalSize,
			"size_threshold":   cfg.Rules.StorageBloat.ImageSizeThreshold,
		}

		issue := types.Issue{
			RuleID:      "DOCKER_STORAGE_BLOAT",
			Subject:     "images_total",
			Severity:    severity,
			Category:    "storage_bloat",
			Description: fmt.Sprintf("Total Docker image size is %d bytes, exceeding threshold of %d bytes", report.Images.TotalSize, cfg.Rules.StorageBloat.ImageSizeThreshold),
			Facts:       facts,
			Solutions: []string{
				"Run 'docker images' to list images and their sizes.",
				"Remove unused images: 'docker image prune -a'",
				"Use multi-stage builds to reduce image sizes.",
				"Consider using smaller base images.",
			},
		}
		report.Issues = append(report.Issues, issue)
	}

	// Check container restarts
	for _, container := range report.Containers.List {
		if strings.Contains(strings.ToLower(container.Status), "restarting") {
			facts := map[string]interface{}{
				"container_id":   container.ID,
				"container_name": container.Name,
				"status":         container.Status,
			}

			issue := types.Issue{
				RuleID:      "RESTART_LOOP",
				Subject:     "container=" + container.ID,
				Severity:    "high",
				Category:    "restarts",
				Description: fmt.Sprintf("Container %s (%s) is in restarting state", container.Name, container.ID),
				Facts:       facts,
				Solutions: []string{
					fmt.Sprintf("Check logs: 'docker logs %s'", container.ID),
					"Inspect container configuration for errors.",
					"Check resource limits (CPU/memory) that might cause crashes.",
					"Review application code for stability issues.",
					"Stop and restart the container manually if needed.",
				},
			}
			report.Issues = append(report.Issues, issue)
		}
	}

	// Check OOM kills
	if cfg.Rules.OOM.Enabled {
		for _, container := range report.Containers.List {
			if container.OOMKilled {
				facts := map[string]interface{}{
					"container_id":   container.ID,
					"container_name": container.Name,
					"status":         container.Status,
				}

				issue := types.Issue{
					RuleID:      "OOM_KILLED",
					Subject:     "container=" + container.ID,
					Severity:    "high",
					Category:    "oom",
					Description: fmt.Sprintf("Container %s (%s) was killed due to out-of-memory condition", container.Name, container.ID),
					Facts:       facts,
					Solutions: []string{
						fmt.Sprintf("Check logs: 'docker logs %s'", container.ID),
						"Increase memory limit: 'docker update --memory <limit> " + container.ID + "'",
						"Optimize application memory usage.",
						"Check for memory leaks in the application.",
						"Consider using memory profiling tools.",
						"Review container resource allocation.",
					},
				}
				report.Issues = append(report.Issues, issue)
			}
		}
	}

	// Check healthcheck failures
	if cfg.Rules.Healthcheck.Enabled {
		for _, container := range report.Containers.List {
			if container.HealthStatus == "unhealthy" {
				duration := time.Since(container.UnhealthySince)
				severity := "medium"
				if duration > time.Hour {
					severity = "high"
				}

				facts := map[string]interface{}{
					"container_id":       container.ID,
					"container_name":     container.Name,
					"health_status":      container.HealthStatus,
					"unhealthy_since":    container.UnhealthySince,
					"unhealthy_duration": duration.String(),
				}

				issue := types.Issue{
					RuleID:      "HEALTHCHECK_UNHEALTHY",
					Subject:     "container=" + container.ID,
					Severity:    severity,
					Category:    "healthcheck",
					Description: fmt.Sprintf("Container %s (%s) has been unhealthy for %s", container.Name, container.ID, duration.Round(time.Second)),
					Facts:       facts,
					Solutions: []string{
						fmt.Sprintf("Check healthcheck logs: 'docker inspect %s | jq .State.Health.Log'", container.ID),
						fmt.Sprintf("Check container logs: 'docker logs %s'", container.ID),
						"Review healthcheck configuration in Dockerfile or compose file.",
						"Ensure the healthcheck command is appropriate for the application.",
						"Check application responsiveness and dependencies.",
						"Consider adjusting healthcheck timeouts or intervals.",
					},
				}
				report.Issues = append(report.Issues, issue)
			}
		}
	}

	// Deterministic ordering for diff-friendly output
	severityRank := func(s string) int {
		switch strings.ToLower(s) {
		case "high":
			return 0
		case "medium":
			return 1
		case "low":
			return 2
		default:
			return 3
		}
	}
	sort.Slice(report.Issues, func(i, j int) bool {
		if severityRank(report.Issues[i].Severity) != severityRank(report.Issues[j].Severity) {
			return severityRank(report.Issues[i].Severity) < severityRank(report.Issues[j].Severity)
		}
		if report.Issues[i].RuleID != report.Issues[j].RuleID {
			return report.Issues[i].RuleID < report.Issues[j].RuleID
		}
		return report.Issues[i].Subject < report.Issues[j].Subject
	})
}
