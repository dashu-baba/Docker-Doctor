package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/example/docker-doctor/internal/collector"
	"github.com/example/docker-doctor/internal/config"
	v1 "github.com/example/docker-doctor/internal/schema/v1"
	"github.com/example/docker-doctor/internal/types"
	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan the Docker host and generate a JSON report",
	Long: `Scan the Docker host to collect metadata about the host, Docker daemon,
containers, images, volumes, and disk usage. Outputs the report in JSON format.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		output, _ := cmd.Flags().GetString("output")
		apiVersion, _ := cmd.Flags().GetString("api-version")
		return runScan(output, apiVersion)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// scanCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	scanCmd.Flags().StringP("output", "o", "", "Output file for the JSON report (default stdout)")
	scanCmd.Flags().String("api-version", "", "Docker API version to use (overrides config)")
}

func runScan(output string, apiVersion string) error {
	startedAt := time.Now()

	cfg, err := config.Load(configFile)
	if err != nil {
		return ExitError{Code: 3, Err: err}
	}

	// Use config values, override with flags if provided
	if apiVersion == "" {
		apiVersion = cfg.Scan.Version
	}

	// Set DOCKER_HOST
	os.Setenv("DOCKER_HOST", cfg.Scan.DockerHost)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Scan.Timeout)*time.Second)
	defer cancel()

	report, err := collector.Collect(ctx, apiVersion, cfg)
	if err != nil {
		return ExitError{Code: 3, Err: fmt.Errorf("failed to collect data: %w", err)}
	}

	finishedAt := time.Now()

	// v0 output is deprecated; emit v1 schema by default.
	v1Report := v1.BuildFromV0(ctx, report, cfg, apiVersion, startedAt, finishedAt)
	data, err := json.MarshalIndent(v1Report, "", "  ")
	if err != nil {
		return ExitError{Code: 3, Err: fmt.Errorf("failed to marshal JSON: %w", err)}
	}

	if output == "" {
		fmt.Println(string(data))
	} else {
		if err := os.WriteFile(output, data, 0644); err != nil {
			return ExitError{Code: 3, Err: fmt.Errorf("failed to write to file: %w", err)}
		}
		fmt.Printf("Report written to %s\n", output)
	}

	code := scanExitCode(report)
	if code == 0 {
		return nil
	}
	return ExitError{Code: code, Err: nil}
}

func scanExitCode(report *types.Report) int {
	issues := report.Issues
	hasMedium := false
	for _, is := range issues {
		switch strings.ToLower(is.Severity) {
		case "high":
			return 2
		case "medium":
			hasMedium = true
		}
	}
	if hasMedium {
		return 1
	}
	// Treat only-low issues as OK for now (info-level).
	return 0
}
