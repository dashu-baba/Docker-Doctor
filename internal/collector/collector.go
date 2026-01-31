package collector

import (
	"context"
	"fmt"
	"os"
	"runtime"
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
		List:  make([]string, 0, len(containers)),
	}
	for _, c := range containers {
		cont.List = append(cont.List, c.ID[:12]) // short ID
	}

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
}
