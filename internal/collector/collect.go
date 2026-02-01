package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/dashu-baba/docker-doctor/internal/config"
	"github.com/dashu-baba/docker-doctor/internal/rules"
	"github.com/dashu-baba/docker-doctor/internal/types"
)

// Collect gathers all the required data for the report.
func Collect(ctx context.Context, apiVersion string, cfg *config.Config) (*types.Report, error) {
	log := loggerFromContext(ctx)
	report := &types.Report{
		Timestamp: time.Now(),
		Issues:    []types.Issue{},
	}

	// Host
	hostStart := time.Now()
	hostInfo, err := collectHostInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to collect host info: %w", err)
	}
	report.Host = *hostInfo
	if log != nil {
		log.Printf("collector host: ok (%dms)", time.Since(hostStart).Milliseconds())
	}

	// Docker
	dockerStart := time.Now()
	dockerInfo, err := collectDockerInfo(ctx, cfg.Scan.DockerHost, apiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to collect Docker info: %w", err)
	}
	report.Docker = *dockerInfo

	containers, usedVolumes, err := collectContainers(ctx, cfg.Scan.DockerHost, apiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to collect containers: %w", err)
	}
	report.Containers = *containers

	images, err := collectImages(ctx, cfg.Scan.DockerHost, apiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to collect images: %w", err)
	}
	report.Images = *images

	volumes, err := collectVolumes(ctx, cfg.Scan.DockerHost, apiVersion, usedVolumes)
	if err != nil {
		return nil, fmt.Errorf("failed to collect volumes: %w", err)
	}
	report.Volumes = *volumes

	networks, err := collectNetworks(ctx, cfg.Scan.DockerHost, apiVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to collect networks: %w", err)
	}
	report.Networks = *networks

	if log != nil {
		log.Printf("collector docker: ok (%dms)", time.Since(dockerStart).Milliseconds())
	}

	// Best-effort: Docker system df (deduplicated disk usage)
	dfStart := time.Now()
	df, _ := CollectDockerSystemDfSummary(ctx, cfg.Scan.DockerHost, apiVersion)
	if log != nil {
		if df == nil {
			log.Printf("collector docker_system_df: skipped/error (%dms)", time.Since(dfStart).Milliseconds())
		} else {
			log.Printf("collector docker_system_df: ok (%dms)", time.Since(dfStart).Milliseconds())
		}
	}

	// Rules/diagnostics
	rulesStart := time.Now()
	rules.Evaluate(report, cfg, df)
	if log != nil {
		log.Printf("rules: %d issue(s) (%dms)", len(report.Issues), time.Since(rulesStart).Milliseconds())
	}

	return report, nil
}

