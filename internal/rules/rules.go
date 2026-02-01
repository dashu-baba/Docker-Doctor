package rules

import (
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/example/docker-doctor/internal/config"
	"github.com/example/docker-doctor/internal/facts"
	"github.com/example/docker-doctor/internal/types"
)

// topOffenders returns the top N items by size, formatted as strings
func topOffenders(items []struct{ id string; size uint64 }, n int) []string {
	sort.Slice(items, func(i, j int) bool {
		return items[i].size > items[j].size // descending
	})
	var result []string
	for i, item := range items {
		if i >= n {
			break
		}
		result = append(result, fmt.Sprintf("%s (%s)", item.id, humanBytes(item.size)))
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// cidrsOverlap checks if two CIDR strings overlap
func cidrsOverlap(cidr1, cidr2 string) bool {
	if cidr1 == "" || cidr2 == "" {
		return false
	}
	_, net1, err1 := net.ParseCIDR(cidr1)
	_, net2, err2 := net.ParseCIDR(cidr2)
	if err1 != nil || err2 != nil {
		return false
	}
	return net1.Contains(net2.IP) || net2.Contains(net1.IP) || net1.IP.Equal(net2.IP)
}

func humanBytes(v uint64) string {
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	f := float64(v)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	// show 2dp for GB+, 1dp for MB, none for KB/B
	decimals := 0
	if units[i] == "MB" {
		decimals = 1
	} else if units[i] == "GB" || units[i] == "TB" || units[i] == "PB" {
		decimals = 2
	}
	pow := 1.0
	for d := 0; d < decimals; d++ {
		pow *= 10
	}
	f = float64(int(f*pow)) / pow
	return fmt.Sprintf("%.*f %s", decimals, f, units[i])
}

// Evaluate runs all rules and appends issues to report.Issues.
// It also ensures deterministic ordering of report.Issues.
func Evaluate(report *types.Report, cfg *config.Config, df *facts.DockerSystemDfSummary) {
	if report == nil || cfg == nil {
		return
	}


	// Run all rule checks
	checkDiskUsage(report, cfg)
	checkStorageBloat(report, cfg, df)
	checkRestarts(report, cfg)
	checkOOM(report, cfg)
	checkHealthcheck(report, cfg)
	checkLogBloat(report, cfg)

	// Deterministic ordering for diff-friendly output
	severityRank := func(s string) int {
		switch strings.ToLower(s) {
		case "high":
			return 0
		case "medium":
			return 1
		case "low":
			return 2
		default:
			return 3
		}
	}
	sort.Slice(report.Issues, func(i, j int) bool {
		if severityRank(report.Issues[i].Severity) != severityRank(report.Issues[j].Severity) {
			return severityRank(report.Issues[i].Severity) < severityRank(report.Issues[j].Severity)
		}
		if report.Issues[i].RuleID != report.Issues[j].RuleID {
			return report.Issues[i].RuleID < report.Issues[j].RuleID
		}
		return report.Issues[i].Subject < report.Issues[j].Subject
	})
	checkVolumeBloat(report)
	checkVolumeSize(report, cfg)
	checkNetworkOverlap(report, cfg)
	checkDaemonRisky(report, cfg)
	sort.Slice(report.Issues, func(i, j int) bool {
		if severityRank(report.Issues[i].Severity) != severityRank(report.Issues[j].Severity) {
			return severityRank(report.Issues[i].Severity) < severityRank(report.Issues[j].Severity)
		}
		if report.Issues[i].RuleID != report.Issues[j].RuleID {
			return report.Issues[i].RuleID < report.Issues[j].RuleID
		}
		return report.Issues[i].Subject < report.Issues[j].Subject
	})
}

