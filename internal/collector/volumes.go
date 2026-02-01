package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/engine-api/types/filters"

	"github.com/example/docker-doctor/internal/types"
)

func collectVolumes(ctx context.Context, dockerHost string, apiVersion string, usedVolumes map[string]bool) (*types.Volumes, error) {
	cli, err := newClient(dockerHost, apiVersion)
	if err != nil {
		return nil, err
	}

	volumes, err := cli.VolumeList(filters.Args{})
	if err != nil {
		return nil, err
	}

	vol := &types.Volumes{
		Count: len(volumes.Volumes),
		List:  make([]types.VolumeInfo, 0, len(volumes.Volumes)),
	}
	for _, v := range volumes.Volumes {
		size := uint64(0)
		if s, err := getVolumeSize(v.Name); err == nil {
			size = s
		}
		used := usedVolumes[v.Name]
		vol.List = append(vol.List, types.VolumeInfo{Name: v.Name, Size: size, Used: used})
	}
	return vol, nil
}

func getVolumeSize(volumeName string) (uint64, error) {
	// Try to get volume size from /var/lib/docker/volumes/<name>/_data
	volumePath := filepath.Join("/var/lib/docker/volumes", volumeName, "_data")
	if stat, err := os.Stat(volumePath); err == nil && stat.IsDir() {
		// Use du-like calculation
		size, err := dirSize(volumePath)
		if err == nil {
			return size, nil
		}
	}
	return 0, fmt.Errorf("volume size not accessible")
}

func dirSize(path string) (uint64, error) {
	var size uint64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += uint64(info.Size())
		}
		return nil
	})
	return size, err
}

