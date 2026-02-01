package collector

import (
	"context"

	"github.com/example/docker-doctor/internal/types"
)

func collectDockerInfo(ctx context.Context, dockerHost string, apiVersion string) (*types.DockerInfo, error) {
	cli, err := newClient(dockerHost, apiVersion)
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

	return &types.DockerInfo{
		Version: version.Version,
		DaemonInfo: map[string]interface{}{
			"server_version": info.ServerVersion,
			"os":             info.OSType,
			"arch":           info.Architecture,
			"storage_driver": info.Driver,
		},
	}, nil
}

