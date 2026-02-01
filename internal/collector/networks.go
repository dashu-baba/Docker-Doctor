package collector

import (
	"context"

	dtypes "github.com/docker/docker/api/types"

	"github.com/dashu-baba/docker-doctor/internal/types"
)

func collectNetworks(ctx context.Context, dockerHost string, apiVersion string) (*types.Networks, error) {
	cli, err := newClient(dockerHost, apiVersion)
	if err != nil {
		return nil, err
	}

	networks, err := cli.NetworkList(ctx, dtypes.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	net := &types.Networks{
		Count: len(networks),
		List:  make([]types.NetworkInfo, 0, len(networks)),
	}

	// Inspect each network to get CIDR
	for _, n := range networks {
		cidr := ""
		if inspect, err := cli.NetworkInspect(ctx, n.ID, dtypes.NetworkInspectOptions{}); err == nil {
			if len(inspect.IPAM.Config) > 0 {
				cidr = inspect.IPAM.Config[0].Subnet
			}
		}
		net.List = append(net.List, types.NetworkInfo{Name: n.Name, CIDR: cidr})
	}
	return net, nil
}