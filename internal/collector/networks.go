package collector

import (
	"context"

	dockertypes "github.com/docker/engine-api/types"

	"github.com/example/docker-doctor/internal/types"
)

func collectNetworks(ctx context.Context, dockerHost string, apiVersion string) (*types.Networks, error) {
	cli, err := newClient(dockerHost, apiVersion)
	if err != nil {
		return nil, err
	}

	networks, err := cli.NetworkList(dockertypes.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	net := &types.Networks{
		Count: len(networks),
		List:  make([]string, 0, len(networks)),
	}
	for _, n := range networks {
		net.List = append(net.List, n.Name)
	}
	return net, nil
}