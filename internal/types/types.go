package types

import "time"

// HostInfo holds basic host system information and disk usage.
type HostInfo struct {
	OS        string            `json:"os"`
	Arch      string            `json:"arch"`
	DiskUsage map[string]uint64 `json:"disk_usage"` // path to bytes used
}

// DockerInfo holds Docker daemon and version information.
type DockerInfo struct {
	Version     string                 `json:"version"`
	DaemonInfo  map[string]interface{} `json:"daemon_info"`
}

// Containers holds container count and basic list.
type Containers struct {
	Count int      `json:"count"`
	List  []string `json:"list"` // container IDs or names
}

// Images holds image count and basic list.
type Images struct {
	Count int      `json:"count"`
	List  []string `json:"list"` // image IDs
}

// Volumes holds volume count and basic list.
type Volumes struct {
	Count int      `json:"count"`
	List  []string `json:"list"` // volume names
}

// Report is the top-level structure for the scan report.
type Report struct {
	Host       HostInfo    `json:"host"`
	Docker     DockerInfo  `json:"docker"`
	Containers Containers  `json:"containers"`
	Images     Images      `json:"images"`
	Volumes    Volumes     `json:"volumes"`
	Timestamp  time.Time   `json:"timestamp"`
}