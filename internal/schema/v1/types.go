package v1

import "time"

type Report struct {
	SchemaVersion string      `json:"schemaVersion"`
	Tool          Tool        `json:"tool"`
	Scan          Scan        `json:"scan"`
	Target        Target      `json:"target"`
	Collectors    []Collector `json:"collectors"`
	Summary       Summary     `json:"summary"`
	Findings      []Finding   `json:"findings"`
	Errors        []string    `json:"errors"`
	Raw           Raw         `json:"raw"`
}

type Tool struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	GitCommit string `json:"gitCommit"`
	BuildTime string `json:"buildTime"`
}

type Scan struct {
	ScanID          string       `json:"scanId"`
	StartedAt       time.Time    `json:"startedAt"`
	FinishedAt      time.Time    `json:"finishedAt"`
	DurationMs      int64        `json:"durationMs"`
	Mode            string       `json:"mode"`
	EffectiveMode   string       `json:"effectiveMode"`
	TimeoutSeconds  int          `json:"timeoutSeconds"`
	Capabilities    Capabilities `json:"capabilities"`
	Redaction       Redaction    `json:"redaction"`
}

type Capabilities struct {
	DockerAPI               bool `json:"dockerApi"`
	HostFSMounted           bool `json:"hostFsMounted"`
	DaemonConfigReadable    bool `json:"daemonConfigReadable"`
	ContainerLogFilesReadable bool `json:"containerLogFilesReadable"`
}

type Redaction struct {
	Enabled        bool     `json:"enabled"`
	MaskedIPs      bool     `json:"maskedIPs"`
	MaskedHostnames bool    `json:"maskedHostnames"`
	DroppedEnvVars bool     `json:"droppedEnvVars"`
	Notes          []string `json:"notes"`
}

type Target struct {
	Host  TargetHost  `json:"host"`
	Docker TargetDocker `json:"docker"`
}

type TargetHost struct {
	HostID        string `json:"hostId"`
	Hostname      string `json:"hostname"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Kernel        string `json:"kernel"`
	UptimeSeconds int64  `json:"uptimeSeconds"`
}

type TargetDocker struct {
	EngineVersion  string `json:"engineVersion"`
	APIVersion     string `json:"apiVersion"`
	StorageDriver  string `json:"storageDriver"`
	CgroupVersion  string `json:"cgroupVersion"`
	DataRoot       string `json:"dataRoot"`
}

type Collector struct {
	Name       string   `json:"name"`
	Status     string   `json:"status"` // ok | skipped | error
	DurationMs int64    `json:"durationMs"`
	Errors     []string `json:"errors"`
}

type Summary struct {
	Counts         SummaryCounts         `json:"counts"`
	ResourceSnapshot SummaryResourceSnapshot `json:"resourceSnapshot"`
	FindingCounts  SummaryFindingCounts  `json:"findingCounts"`
}

type SummaryCounts struct {
	ContainersRunning int `json:"containersRunning"`
	ContainersStopped int `json:"containersStopped"`
	Images            int `json:"images"`
	Volumes           int `json:"volumes"`
	Networks          int `json:"networks"`
}

type SummaryResourceSnapshot struct {
	DockerSystemDf DockerSystemDf `json:"dockerSystemDf"`
}

type DockerSystemDf struct {
	ImagesTotalBytes              uint64 `json:"imagesTotalBytes"`
	ContainersWritableTotalBytes  uint64 `json:"containersWritableTotalBytes"`
	VolumesTotalBytes             uint64 `json:"volumesTotalBytes"`
	BuildCacheTotalBytes          uint64 `json:"buildCacheTotalBytes"`
}

type SummaryFindingCounts struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

type Finding struct {
	ID             string           `json:"id"`
	Fingerprint    string           `json:"fingerprint"`
	Severity       string           `json:"severity"`   // critical | warning | info
	Confidence     string           `json:"confidence"` // high | medium | low
	Category       string           `json:"category"`
	Title          string           `json:"title"`
	Summary        string           `json:"summary"`
	Scope          Scope            `json:"scope"`
	Evidence       []Evidence       `json:"evidence"`
	Recommendations []Recommendation `json:"recommendations"`
	References     []Reference      `json:"references"`
}

type Scope struct {
	ContainerID   string `json:"containerId,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	Image         string `json:"image,omitempty"`
	Path          string `json:"path,omitempty"`
}

type Evidence struct {
	Type  string      `json:"type"`
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type Recommendation struct {
	Risk     string   `json:"risk"` // safe | planned | risky
	Title    string   `json:"title"`
	Steps    []string `json:"steps"`
	Commands []string `json:"commands"`
	Notes    []string `json:"notes"`
}

type Reference struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Raw struct {
	Included bool   `json:"included"`
	Reason   string `json:"reason"`
}

