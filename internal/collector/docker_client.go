package collector

import (
	"github.com/docker/docker/client"
)

func newClient(dockerHost string, apiVersion string) (*client.Client, error) {
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}
	return client.NewClientWithOpts(client.WithHost(dockerHost), client.WithVersion(apiVersion))
}

