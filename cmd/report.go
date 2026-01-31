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
<!DOCTYPE html>
<html>
<head>
    <title>Docker Doctor Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1, h2, h3 { color: #333; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .issue { margin-bottom: 20px; border-left: 5px solid #ff6b6b; padding-left: 10px; }
        .stable { color: green; }
        .severity-high { color: red; }
        .severity-medium { color: orange; }
        .severity-low { color: yellow; }
    </style>
</head>
<body>
    <h1>Docker Doctor Report</h1>
    <p><strong>Timestamp:</strong> {{.Timestamp.Format "2006-01-02 15:04:05"}}</p>

    <h2>Host Information</h2>
    <p><strong>Operating System:</strong> {{.Host.OS}}</p>
    <p><strong>Architecture:</strong> {{.Host.Arch}}</p>

    <h3>Disk Usage</h3>
    <table>
        <tr>
            <th>Path</th>
            <th>Used (%)</th>
            <th>Used (Bytes)</th>
            <th>Total (Bytes)</th>
        </tr>
        {{range $path, $disk := .Host.DiskUsage}}
        <tr>
            <td>{{$path}}</td>
            <td>{{printf "%.2f" $disk.UsedPercent}}</td>
            <td>{{$disk.Used}}</td>
            <td>{{$disk.Total}}</td>
        </tr>
        {{end}}
    </table>

    <h2>Docker Information</h2>
    <p><strong>Version:</strong> {{.Docker.Version}}</p>

    <h2>Containers</h2>
    <p><strong>Total Count:</strong> {{.Containers.Count}}</p>
    <table>
        <tr>
            <th>ID</th>
            <th>Name</th>
            <th>Status</th>
            <th>OOM Killed</th>
            <th>Health Status</th>
        </tr>
        {{range .Containers.List}}
        <tr>
            <td>{{.ID}}</td>
            <td>{{.Name}}</td>
            <td>{{.Status}}</td>
            <td>{{if .OOMKilled}}Yes{{else}}No{{end}}</td>
            <td>{{.HealthStatus}}</td>
        </tr>
        {{end}}
    </table>

    <h2>Images</h2>
    <p><strong>Count:</strong> {{.Images.Count}}</p>
    <p><strong>Total Size:</strong> {{.Images.TotalSize}} bytes</p>

    <h2>Volumes</h2>
    <p><strong>Count:</strong> {{.Volumes.Count}}</p>

    <h2>Diagnostic Issues</h2>
    {{if .Issues}}
        {{range .Issues}}
        <div class="issue severity-{{.Severity}}">
            <h3>{{.Category}} Issue ({{.Severity}} Severity)</h3>
            <p><strong>Description:</strong> {{.Description}}</p>
            <h4>Facts</h4>
            <ul>
            {{range $key, $value := .Facts}}
                <li><strong>{{$key}}:</strong> {{$value}}</li>
            {{end}}
            </ul>
            <h4>Recommended Solutions</h4>
            <ol>
            {{range .Solutions}}
                <li>{{.}}</li>
            {{end}}
            </ol>
        </div>
        {{end}}
    {{else}}
        <p class="stable"><strong>All systems stable - No issues detected!</strong></p>
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

**Timestamp:** %s

## Host Information
- **Operating System:** %s
- **Architecture:** %s

### Disk Usage
| Path | Used (%%) | Used (Bytes) | Total (Bytes) |
|------|-----------|--------------|---------------|
`, report.Timestamp.Format("2006-01-02 15:04:05"), report.Host.OS, report.Host.Arch)
	for path, disk := range report.Host.DiskUsage {
		md += fmt.Sprintf("| %s | %.2f | %d | %d |\n", path, disk.UsedPercent, disk.Used, disk.Total)
	}
	md += fmt.Sprintf(`

## Docker Information
- **Version:** %s

## Containers
- **Total Count:** %d

| ID | Name | Status | OOM Killed | Health Status |
|----|------|--------|------------|---------------|
`, report.Docker.Version, report.Containers.Count)
	for _, container := range report.Containers.List {
		oom := "No"
		if container.OOMKilled {
			oom = "Yes"
		}
		md += fmt.Sprintf("| %s | %s | %s | %s | %s |\n", container.ID, container.Name, container.Status, oom, container.HealthStatus)
	}
	md += fmt.Sprintf(`

## Images
- **Count:** %d
- **Total Size:** %d bytes

## Volumes
- **Count:** %d

## Diagnostic Issues
`, report.Images.Count, report.Images.TotalSize, report.Volumes.Count)
	if len(report.Issues) > 0 {
		for _, issue := range report.Issues {
			md += fmt.Sprintf("### %s Issue (%s Severity)\n%s\n\n**Facts:**\n", strings.Title(issue.Category), strings.Title(issue.Severity), issue.Description)
			for key, value := range issue.Facts {
				md += fmt.Sprintf("- **%s:** %v\n", strings.Title(strings.ReplaceAll(key, "_", " ")), value)
			}
			md += "\n**Recommended Solutions:**\n"
			for i, sol := range issue.Solutions {
				md += fmt.Sprintf("%d. %s\n", i+1, sol)
			}
			md += "\n"
		}
	} else {
		md += "**All systems stable - No issues detected!**\n"
	}
	return md, nil
}
