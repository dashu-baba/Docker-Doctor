package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dashu-baba/docker-doctor/internal/facts"
)

type dockerSystemDFResponse struct {
	LayersSize int64 `json:"LayersSize"`
	Containers []struct {
		SizeRw int64 `json:"SizeRw"`
	} `json:"Containers"`
	Volumes []struct {
		UsageData struct {
			Size int64 `json:"Size"`
		} `json:"UsageData"`
	} `json:"Volumes"`
	BuildCache []struct {
		Size int64 `json:"Size"`
	} `json:"BuildCache"`
}

// CollectDockerSystemDfSummary fetches `/system/df` and returns deduplicated totals.
// Best-effort callers should treat errors as non-fatal.
func CollectDockerSystemDfSummary(ctx context.Context, dockerHost string, apiVersion string) (*facts.DockerSystemDfSummary, error) {
	if strings.TrimSpace(dockerHost) == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	u, err := url.Parse(dockerHost)
	if err != nil {
		return nil, err
	}

	base := &url.URL{}
	var transport *http.Transport

	switch u.Scheme {
	case "unix":
		socketPath := u.Path
		transport = &http.Transport{
			DisableCompression: true,
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.DialTimeout("unix", socketPath, 32*time.Second)
			},
		}
		base.Scheme = "http"
		base.Host = "docker"
	case "tcp", "http", "https":
		base.Scheme = "http"
		if u.Scheme == "https" {
			base.Scheme = "https"
		}
		base.Host = u.Host
		base.Path = strings.TrimSuffix(u.Path, "/")
		transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: 32 * time.Second,
			}).DialContext,
		}
	default:
		return nil, fmt.Errorf("unsupported DOCKER_HOST scheme: %s", u.Scheme)
	}

	client := &http.Client{Transport: transport}
	path := fmt.Sprintf("/v%s/system/df", strings.TrimPrefix(apiVersion, "v"))
	reqURL := base.ResolveReference(&url.URL{Path: path})

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("system df request failed: %s (%s)", resp.Status, strings.TrimSpace(string(b)))
	}

	var df dockerSystemDFResponse
	if err := json.NewDecoder(resp.Body).Decode(&df); err != nil {
		return nil, err
	}

	var containersRW int64
	for _, c := range df.Containers {
		if c.SizeRw > 0 {
			containersRW += c.SizeRw
		}
	}
	var volumesTotal int64
	for _, v := range df.Volumes {
		if v.UsageData.Size > 0 {
			volumesTotal += v.UsageData.Size
		}
	}
	var buildCache int64
	for _, b := range df.BuildCache {
		if b.Size > 0 {
			buildCache += b.Size
		}
	}

	out := &facts.DockerSystemDfSummary{
		ImagesTotalBytes:             uint64(max64(df.LayersSize, 0)),
		ContainersWritableTotalBytes: uint64(max64(containersRW, 0)),
		VolumesTotalBytes:            uint64(max64(volumesTotal, 0)),
		BuildCacheTotalBytes:         uint64(max64(buildCache, 0)),
	}
	return out, nil
}

func max64(v, min int64) int64 {
	if v < min {
		return min
	}
	return v
}

