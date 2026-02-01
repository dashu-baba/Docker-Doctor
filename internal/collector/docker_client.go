package collector

import (
	"github.com/docker/engine-api/client"
)

func newClient(dockerHost string, apiVersion string) (*client.Client, error) {
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}
	return client.NewClient(dockerHost, apiVersion, nil, nil)
}

