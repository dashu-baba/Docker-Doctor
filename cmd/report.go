package cmd

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/example/docker-doctor/internal/types"
	"github.com/spf13/cobra"
)

// reportCmd represents the report command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate HTML or Markdown report from JSON data",
	Long: `Generate a human-readable report in HTML or Markdown format
from the JSON output of the scan command.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		input, _ := cmd.Flags().GetString("input")
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")
		return runReport(input, format, output)
	},
}

func init() {
	rootCmd.AddCommand(reportCmd)

	reportCmd.Flags().StringP("input", "i", "report.json", "Input JSON file from scan")
	reportCmd.Flags().StringP("format", "f", "html", "Output format: html or md")
	reportCmd.Flags().StringP("output", "o", "", "Output file (default stdout)")
}

func runReport(input, format, output string) error {
	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var report types.Report
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	var result string
	switch format {
	case "html":
		result, err = generateHTML(&report)
	case "md":
		result, err = generateMarkdown(&report)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	if err != nil {
		return err
	}

	if output == "" {
		fmt.Print(result)
	} else {
		if err := os.WriteFile(output, []byte(result), 0644); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
		fmt.Printf("Report written to %s\n", output)
	}

	return nil
}

func generateHTML(report *types.Report) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Docker Doctor Report</title>
</head>
<body>
    <h1>Docker Doctor Report</h1>
    <p>Timestamp: {{.Timestamp}}</p>
    <h2>Host Info</h2>
    <p>OS: {{.Host.OS}}</p>
    <p>Arch: {{.Host.Arch}}</p>
    <h3>Disk Usage</h3>
    <ul>
    {{range $path, $usage := .Host.DiskUsage}}
        <li>{{$path}}: {{$usage}} bytes</li>
    {{end}}
    </ul>
    <h2>Docker Info</h2>
    <p>Version: {{.Docker.Version}}</p>
    <h2>Containers</h2>
    <p>Count: {{.Containers.Count}}</p>
    <h2>Images</h2>
    <p>Count: {{.Images.Count}}</p>
    <h2>Volumes</h2>
    <p>Count: {{.Volumes.Count}}</p>
</body>
</html>
`
	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := t.Execute(&buf, report); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func generateMarkdown(report *types.Report) (string, error) {
	md := fmt.Sprintf(`# Docker Doctor Report

Timestamp: %s

## Host Info
- OS: %s
- Arch: %s

### Disk Usage
`, report.Timestamp, report.Host.OS, report.Host.Arch)
	for path, usage := range report.Host.DiskUsage {
		md += fmt.Sprintf("- %s: %d bytes\n", path, usage)
	}
	md += fmt.Sprintf(`
## Docker Info
- Version: %s

## Containers
- Count: %d

## Images
- Count: %d

## Volumes
- Count: %d
`, report.Docker.Version, report.Containers.Count, report.Images.Count, report.Volumes.Count)
	return md, nil
}