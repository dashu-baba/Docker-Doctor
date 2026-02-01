package collector

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"github.com/example/docker-doctor/internal/types"
)

func collectHostInfo() (*types.HostInfo, error) {
	hostID, _ := generateHostID()
	hostname, _ := os.Hostname()
	kernel, _ := getKernelVersion()
	uptime, _ := getUptimeSeconds()

	info := &types.HostInfo{
		HostID:        hostID,
		Hostname:      hostname,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		Kernel:        kernel,
		UptimeSeconds: uptime,
		DiskUsage:     make(map[string]*types.DiskInfo),
	}

	// Get disk usage for root
	if diskInfo, err := getDiskUsage("/"); err == nil {
		info.DiskUsage["/"] = diskInfo
	}

	// Get disk usage for /var/lib/docker if exists
	dockerPath := "/var/lib/docker"
	if _, err := os.Stat(dockerPath); err == nil {
		if diskInfo, err := getDiskUsage(dockerPath); err == nil {
			info.DiskUsage[dockerPath] = diskInfo
		}
	}

	return info, nil
}

func generateHostID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func getKernelVersion() (string, error) {
	file, err := os.Open("/proc/version")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		// Example: Linux version 5.4.0-74-generic (buildd@lgw01-amd64-060) (gcc version 9.3.0 (Ubuntu 9.3.0-17ubuntu1~20.04)) #83-Ubuntu SMP Sat May 8 02:35:39 UTC 2021
		parts := strings.Fields(line)
		if len(parts) >= 3 && parts[0] == "Linux" && parts[1] == "version" {
			return parts[2], nil
		}
	}
	return "", nil
}

func getUptimeSeconds() (int64, error) {
	file, err := os.Open("/proc/uptime")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 1 {
			uptimeFloat, err := strconv.ParseFloat(parts[0], 64)
			if err != nil {
				return 0, err
			}
			return int64(uptimeFloat), nil
		}
	}
	return 0, nil
}

func getDiskUsage(path string) (*types.DiskInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, err
	}
	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	used := total - available
	usedPercent := float64(used) / float64(total) * 100
	return &types.DiskInfo{
		Used:        used,
		Total:       total,
		UsedPercent: usedPercent,
	}, nil
}

