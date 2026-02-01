package collector

import (
	"context"
	"sort"
	"sync"
	"time"

	dtypes "github.com/docker/engine-api/types"

	"github.com/example/docker-doctor/internal/types"
)

func collectContainers(ctx context.Context, dockerHost string, apiVersion string) (*types.Containers, error) {
	cli, err := newClient(dockerHost, apiVersion)
	if err != nil {
		return nil, err
	}

	containers, err := cli.ContainerList(dtypes.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	cont := &types.Containers{
		Count: len(containers),
		List:  make([]types.ContainerInfo, 0, len(containers)),
	}

	// Bounded concurrency for better latency on hosts with many containers.
	type row struct {
		info types.ContainerInfo
	}
	rows := make([]row, len(containers))
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i, c := range containers {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, c dtypes.Container) {
			defer wg.Done()
			defer func() { <-sem }()

			inspect, err := cli.ContainerInspect(c.ID)
			oomKilled := false
			healthStatus := "none"
			var unhealthySince time.Time
			restartCount := 0
			if err == nil {
				if inspect.State != nil {
					oomKilled = inspect.State.OOMKilled
				}
				restartCount = inspect.RestartCount
			}

			name := ""
			if len(c.Names) > 0 {
				name = c.Names[0]
			}

			rows[i] = row{info: types.ContainerInfo{
				ID:             c.ID[:12],
				Name:           name,
				RestartCount:   restartCount,
				Status:         c.Status,
				OOMKilled:      oomKilled,
				HealthStatus:   healthStatus,
				UnhealthySince: unhealthySince,
			}}
		}(i, c)
	}
	wg.Wait()

	for i := range rows {
		cont.List = append(cont.List, rows[i].info)
	}

	// Deterministic ordering for diff-friendly output
	sort.Slice(cont.List, func(i, j int) bool {
		if cont.List[i].Name != cont.List[j].Name {
			return cont.List[i].Name < cont.List[j].Name
		}
		return cont.List[i].ID < cont.List[j].ID
	})

	return cont, nil
}

