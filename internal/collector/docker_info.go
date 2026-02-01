package collector

import (
	"context"

	"github.com/dashu-baba/docker-doctor/internal/types"
)

func collectDockerInfo(ctx context.Context, dockerHost string, apiVersion string) (*types.DockerInfo, error) {
	cli, err := newClient(dockerHost, apiVersion)
	if err != nil {
		return nil, err
	}

	version, err := cli.ServerVersion(ctx)
	if err != nil {
		return nil, err
	}

	info, err := cli.Info(ctx)
	if err != nil {
		return nil, err
	}

	daemonInfo := map[string]interface{}{
		"server_version":  info.ServerVersion,
		"os":              info.OSType,
		"arch":            info.Architecture,
		"storage_driver":  info.Driver,
		"experimental":    info.ExperimentalBuild,
		"logging_driver":  info.LoggingDriver,
		"registry_config": info.RegistryConfig,
	}

	// Note: CgroupVersion and DockerRootDir may not be available in older API versions
	// They will be empty in the struct, but can be populated from DaemonInfo if added later

	return &types.DockerInfo{
		Version:       version.Version,
		CgroupVersion: "", // Not available in this API version
		DataRoot:      "", // Not available in this API version
		DaemonInfo:    daemonInfo,
	}, nil
}

