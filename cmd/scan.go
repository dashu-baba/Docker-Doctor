package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/dashu-baba/docker-doctor/internal/collector"
	"github.com/dashu-baba/docker-doctor/internal/config"
	v1 "github.com/dashu-baba/docker-doctor/internal/schema/v1"
	"github.com/dashu-baba/docker-doctor/internal/types"
	"github.com/spf13/cobra"
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan the Docker host and generate reports",
	Long: `Scan the Docker host to collect metadata about the host, Docker daemon,
containers, images, volumes, and disk usage. Writes scan.json + human reports.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, _ := cmd.Flags().GetString("output-dir")
		formats, _ := cmd.Flags().GetString("formats")
		apiVersion, _ := cmd.Flags().GetString("api-version")
		exitCode, _ := cmd.Flags().GetBool("exit-code")
		verbose, _ := cmd.Flags().GetBool("verbose")
		return runScan(outputDir, formats, apiVersion, exitCode, verbose)
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
	scanCmd.Flags().StringP("output-dir", "o", "./out", "Output directory. Artifacts are written to <output-dir>/<scanId>/")
	scanCmd.Flags().String("formats", "json,html,md", "Comma-separated output formats: json,html,md")
	scanCmd.Flags().String("api-version", "", "Docker API version to use (overrides config)")
	scanCmd.Flags().Bool("exit-code", false, "If set, exit non-zero when findings are WARN/CRITICAL (CI mode)")
	scanCmd.Flags().Bool("verbose", false, "Enable debug logging to stderr")
}

func runScan(outputDir string, formats string, apiVersion string, exitCode bool, verbose bool) error {
	startedAt := time.Now()

	cfg, err := config.Load(configFile)
	if err != nil {
		return ExitError{Code: 3, Err: err}
	}

	// Check full mode availability
	if cfg.Scan.Mode == "full" && runtime.GOOS != "linux" {
		fmt.Fprintf(os.Stderr, "Warning: Full scan mode is not supported on %s. Host filesystem access is required for full scans. Falling back to basic mode.\n", runtime.GOOS)
		cfg.Scan.Mode = "basic"
	}

	// Use config values, override with flags if provided
	if apiVersion == "" {
		apiVersion = cfg.Scan.Version
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Scan.Timeout)*time.Second)
	defer cancel()

	// Optional debug logging (kept off by default for clean CLI UX).
	// Note: we use a lightweight logger interface inside collector.
	if verbose {
		l := log.New(os.Stderr, "docker-doctor ", log.LstdFlags)
		ctx = collector.WithLogger(ctx, l)
	}

	report, err := collector.Collect(ctx, apiVersion, cfg)
	if err != nil {
		return ExitError{Code: 3, Err: fmt.Errorf("failed to collect data: %w", err)}
	}

	finishedAt := time.Now()

	v1Report := v1.BuildFromV0(ctx, report, cfg, apiVersion, startedAt, finishedAt, toolVersion, toolGitCommit, toolBuildTime)

	selected := parseFormats(formats)
	if len(selected) == 0 {
		return ExitError{Code: 3, Err: fmt.Errorf("no formats selected (use --formats json,html,md)")}
	}

	runDir := filepath.Join(outputDir, v1Report.Scan.ScanID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return ExitError{Code: 3, Err: fmt.Errorf("failed to create output directory: %w", err)}
	}

	written := []string{}

	// Always allow HTML/MD generation without requiring a separate command.
	if selected["json"] || selected["html"] || selected["md"] {
		data, err := json.MarshalIndent(v1Report, "", "  ")
		if err != nil {
			return ExitError{Code: 3, Err: fmt.Errorf("failed to marshal JSON: %w", err)}
		}
		scanPath := filepath.Join(runDir, "scan.json")
		if err := os.WriteFile(scanPath, data, 0o644); err != nil {
			return ExitError{Code: 3, Err: fmt.Errorf("failed to write scan.json: %w", err)}
		}
		written = append(written, scanPath)
	}

	if selected["html"] {
		html, err := generateHTMLv1(&v1Report)
		if err != nil {
			return ExitError{Code: 3, Err: fmt.Errorf("failed to generate HTML report: %w", err)}
		}
		htmlPath := filepath.Join(runDir, "report.html")
		if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
			return ExitError{Code: 3, Err: fmt.Errorf("failed to write report.html: %w", err)}
		}
		written = append(written, htmlPath)
	}

	if selected["md"] {
		md, err := generateMarkdownv1(&v1Report)
		if err != nil {
			return ExitError{Code: 3, Err: fmt.Errorf("failed to generate Markdown report: %w", err)}
		}
		mdPath := filepath.Join(runDir, "report.md")
		if err := os.WriteFile(mdPath, []byte(md), 0o644); err != nil {
			return ExitError{Code: 3, Err: fmt.Errorf("failed to write report.md: %w", err)}
		}
		written = append(written, mdPath)
	}

	if len(written) > 0 {
		fmt.Printf("Wrote %d artifact(s) to %s\n", len(written), runDir)
	}

	if exitCode {
		code := scanExitCode(report)
		if code == 0 {
			return nil
		}
		return ExitError{Code: code, Err: nil}
	}

	// Default UX: scan success is exit 0, even if findings exist.
	return nil
}

func parseFormats(s string) map[string]bool {
	out := map[string]bool{}
	for _, p := range strings.Split(s, ",") {
		k := strings.ToLower(strings.TrimSpace(p))
		if k == "" {
			continue
		}
		switch k {
		case "json", "html", "md":
			out[k] = true
		}
	}
	return out
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
