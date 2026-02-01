package collector

import (
	"context"

	"github.com/docker/engine-api/types/filters"

	"github.com/example/docker-doctor/internal/types"
)

func collectVolumes(ctx context.Context, dockerHost string, apiVersion string) (*types.Volumes, error) {
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
		vol.List = append(vol.List, types.VolumeInfo{Name: v.Name, Size: 0})
	}
	return vol, nil
}

