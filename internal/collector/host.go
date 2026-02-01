package collector

import (
	"os"
	"runtime"
	"syscall"

	"github.com/example/docker-doctor/internal/types"
)

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

