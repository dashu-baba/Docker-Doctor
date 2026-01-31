package collector

import (
	"context"
	"fmt"
	"os"
	"runtime"
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
		Issues:    []string{},
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
		Count: len(images),
		List:  make([]string, 0, len(images)),
	}
	for _, i := range images {
		img.List = append(img.List, i.ID)
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
			report.Issues = append(report.Issues, fmt.Sprintf("Disk usage for %s is %.2f%%, exceeds threshold %d%%", path, disk.UsedPercent, cfg.Rules.DiskUsage.Threshold))
		}
	}
}