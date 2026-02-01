package facts

// DockerSystemDfSummary is a simplified, deduplicated snapshot of Docker disk usage.
// It mirrors the high-level numbers shown in `docker system df`.
type DockerSystemDfSummary struct {
	ImagesTotalBytes             uint64
	ContainersWritableTotalBytes uint64
	VolumesTotalBytes            uint64
	BuildCacheTotalBytes         uint64
}

