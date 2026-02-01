package types

import "time"

// DiskInfo holds disk usage information.
type DiskInfo struct {
	Used        uint64  `json:"used"`
	Total       uint64  `json:"total"`
	UsedPercent float64 `json:"used_percent"`
}

// HostInfo holds basic host system information and disk usage.
type HostInfo struct {
	HostID      string                `json:"host_id"`
	Hostname    string                `json:"hostname"`
	OS          string                `json:"os"`
	Arch        string                `json:"arch"`
	Kernel      string                `json:"kernel"`
	UptimeSeconds int64               `json:"uptime_seconds"`
	DiskUsage   map[string]*DiskInfo `json:"disk_usage"` // path to disk info
}

// DockerInfo holds Docker daemon and version information.
type DockerInfo struct {
	Version       string                 `json:"version"`
	CgroupVersion string                 `json:"cgroup_version"`
	DataRoot      string                 `json:"data_root"`
	DaemonInfo    map[string]interface{} `json:"daemon_info"`
}

// ContainerInfo holds information about a container.
type ContainerInfo struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	RestartCount     int       `json:"restart_count"`
	Status           string    `json:"status"`
	OOMKilled        bool      `json:"oom_killed"`
	HealthStatus     string    `json:"health_status"`
	UnhealthySince   time.Time `json:"unhealthy_since"`
	LogSize          uint64    `json:"log_size"` // estimated log size in bytes
}

// Containers holds container count and detailed list.
type Containers struct {
	Count int             `json:"count"`
	List  []ContainerInfo `json:"list"`
}

// ImageInfo holds information about an image.
type ImageInfo struct {
	ID   string `json:"id"`
	Size uint64 `json:"size"`
}

// Images holds image count and detailed list.
type Images struct {
	Count     int          `json:"count"`
	List      []ImageInfo  `json:"list"`
	TotalSize uint64       `json:"total_size"` // total size in bytes
}

// VolumeInfo holds information about a volume.
type VolumeInfo struct {
	Name         string `json:"name"`
	Size         uint64 `json:"size"`
	SizeAvailable bool   `json:"size_available"`
	Used         bool   `json:"used"`
}

// Volumes holds volume count and detailed list.
type Volumes struct {
	Count int          `json:"count"`
	List  []VolumeInfo `json:"list"`
}

// NetworkInfo holds information about a network.
type NetworkInfo struct {
	Name string `json:"name"`
	CIDR string `json:"cidr"`
}

// Networks holds network count and detailed list.
type Networks struct {
	Count int           `json:"count"`
	List  []NetworkInfo `json:"list"`
}

// Issue represents a diagnostic finding.
type Issue struct {
	RuleID      string                 `json:"ruleId"`               // stable rule identifier (e.g., DISK_USAGE_HIGH)
	Subject     string                 `json:"subject,omitempty"`     // stable scope key (e.g., path=/, container=<id>)
	Severity    string                 `json:"severity"`    // low, medium, high
	Category    string                 `json:"category"`    // e.g., disk_usage, storage_bloat
	Description string                 `json:"description"`
	Facts       map[string]interface{} `json:"facts"`
	Solutions   []string               `json:"solutions"`
}

// Report is the top-level structure for the scan report.
type Report struct {
	Host       HostInfo   `json:"host"`
	Docker     DockerInfo  `json:"docker"`
	Containers Containers  `json:"containers"`
	Images     Images     `json:"images"`
	Volumes    Volumes    `json:"volumes"`
	Networks   Networks   `json:"networks"`
	Issues     []Issue    `json:"issues"`
	Timestamp  time.Time  `json:"timestamp"`
}