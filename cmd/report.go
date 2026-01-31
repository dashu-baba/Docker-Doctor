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
    {{range $path, $disk := .Host.DiskUsage}}
        <li>{{$path}}: {{printf "%.2f" $disk.UsedPercent}}% used ({{$disk.Used}}/{{$disk.Total}} bytes)</li>
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
    <h2>Issues</h2>
    {{if .Issues}}
    <ul>
    {{range .Issues}}
        <li>
            <strong>{{.Category}} ({{.Severity}}):</strong> {{.Description}}
            <br><strong>Facts:</strong>
            <ul>
            {{range $key, $value := .Facts}}
                <li>{{$key}}: {{$value}}</li>
            {{end}}
            </ul>
            <strong>Solutions:</strong>
            <ul>
            {{range .Solutions}}
                <li>{{.}}</li>
            {{end}}
            </ul>
        </li>
    {{end}}
    </ul>
    {{else}}
    <p>No issues found.</p>
    {{end}}
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
	for path, disk := range report.Host.DiskUsage {
		md += fmt.Sprintf("- %s: %.2f%% used (%d/%d bytes)\n", path, disk.UsedPercent, disk.Used, disk.Total)
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

## Issues
`, report.Docker.Version, report.Containers.Count, report.Images.Count, report.Volumes.Count)
	if len(report.Issues) > 0 {
		for _, issue := range report.Issues {
			md += fmt.Sprintf("### %s (%s)\n%s\n\n**Facts:**\n", issue.Category, issue.Severity, issue.Description)
			for key, value := range issue.Facts {
				md += fmt.Sprintf("- %s: %v\n", key, value)
			}
			md += "\n**Solutions:**\n"
			for _, sol := range issue.Solutions {
				md += fmt.Sprintf("- %s\n", sol)
			}
			md += "\n"
		}
	} else {
		md += "No issues found.\n"
	}
	return md, nil
}
