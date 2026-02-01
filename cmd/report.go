package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"sort"
	"strings"

	v1 "github.com/dashu-baba/docker-doctor/internal/schema/v1"
	"github.com/dashu-baba/docker-doctor/internal/types"
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

	reportCmd.Flags().StringP("input", "i", "scan.json", "Input JSON file from scan")
	reportCmd.Flags().StringP("format", "f", "html", "Output format: html or md")
	reportCmd.Flags().StringP("output", "o", "", "Output file (default stdout)")
}

func runReport(input, format, output string) error {
	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	var result string
	switch detectSchemaVersion(data) {
	case "1.0":
		var reportV1 v1.Report
		if err := json.Unmarshal(data, &reportV1); err != nil {
			return fmt.Errorf("failed to unmarshal v1 JSON: %w", err)
		}

		switch format {
		case "html":
			result, err = generateHTMLv1(&reportV1)
		case "md":
			result, err = generateMarkdownv1(&reportV1)
		default:
			return fmt.Errorf("unsupported format: %s", format)
		}
	default:
		// Backward-compat: support legacy v0 scans.
		var report types.Report
		if err := json.Unmarshal(data, &report); err != nil {
			return fmt.Errorf("failed to unmarshal JSON: %w", err)
		}
		switch format {
		case "html":
			result, err = generateHTML(&report)
		case "md":
			result, err = generateMarkdown(&report)
		default:
			return fmt.Errorf("unsupported format: %s", format)
		}
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

func detectSchemaVersion(data []byte) string {
	var probe struct {
		SchemaVersion string `json:"schemaVersion"`
	}
	_ = json.Unmarshal(data, &probe)
	return probe.SchemaVersion
}

func generateHTMLv1(report *v1.Report) (string, error) {
	funcs := template.FuncMap{
		"bytes":     humanBytes,
		"title":     strings.Title,
		"sevClass":  severityClass,
		"riskClass": riskClass,
		"json": func(v interface{}) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"b64": func(s string) string {
			return base64.RawURLEncoding.EncodeToString([]byte(s))
		},
	}

	// Sort evidence keys inside each finding for stable rendering.
	for i := range report.Findings {
		sort.Slice(report.Findings[i].Evidence, func(a, b int) bool {
			if report.Findings[i].Evidence[a].Type != report.Findings[i].Evidence[b].Type {
				return report.Findings[i].Evidence[a].Type < report.Findings[i].Evidence[b].Type
			}
			return report.Findings[i].Evidence[a].Key < report.Findings[i].Evidence[b].Key
		})
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Docker Host Doctor Report</title>
    <style>
        :root {
          --bg: #0b1020;
          --panel: rgba(255,255,255,0.06);
          --panel2: rgba(255,255,255,0.09);
          --text: #e8ecf3;
          --muted: rgba(232,236,243,0.72);
          --border: rgba(255,255,255,0.12);
          --critical: #ff3b5c;
          --warning: #ffb020;
          --info: #36d399;
          --planned: #60a5fa;
          --safe: #34d399;
          --risky: #fb7185;
        }
        * { box-sizing: border-box; }
        body {
          margin: 0;
          font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, "Apple Color Emoji","Segoe UI Emoji";
          background: radial-gradient(1200px 600px at 10% 0%, #1b2a6b 0%, var(--bg) 50%) fixed;
          color: var(--text);
          line-height: 1.4;
        }
        a { color: #93c5fd; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .container { max-width: 1100px; margin: 0 auto; padding: 28px 18px 60px; }
        .header { display: flex; gap: 18px; align-items: flex-start; justify-content: space-between; }
        .title { margin: 0; font-size: 28px; letter-spacing: 0.2px; }
        .meta { text-align: right; color: var(--muted); font-size: 13px; }
        .pill { display: inline-flex; align-items: center; gap: 8px; padding: 6px 10px; border-radius: 999px; background: var(--panel); border: 1px solid var(--border); font-size: 12px; color: var(--muted); }
        .cards { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; margin-top: 16px; }
        .card { background: var(--panel); border: 1px solid var(--border); border-radius: 14px; padding: 14px; }
        .card h3 { margin: 0 0 8px 0; font-size: 13px; color: var(--muted); font-weight: 600; }
        .card .big { font-size: 20px; font-weight: 700; }
        .row { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
        .kv { color: var(--muted); font-size: 13px; }
        .section { margin-top: 18px; }
        .section h2 { margin: 0 0 10px 0; font-size: 16px; }
        table { width: 100%; border-collapse: collapse; }
        th, td { border-bottom: 1px solid var(--border); padding: 10px 8px; vertical-align: top; font-size: 13px; }
        th { text-align: left; color: var(--muted); font-weight: 600; }
        code, pre { font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; }
        code { background: rgba(255,255,255,0.08); padding: 2px 6px; border-radius: 8px; border: 1px solid rgba(255,255,255,0.08); }
        pre { background: rgba(255,255,255,0.06); border: 1px solid var(--border); border-radius: 12px; padding: 12px; overflow: auto; }
        .badge { display: inline-flex; align-items: center; justify-content: center; padding: 3px 8px; border-radius: 999px; font-size: 12px; font-weight: 700; border: 1px solid var(--border); background: var(--panel2); }
        .badge.critical { color: var(--critical); }
        .badge.warning { color: var(--warning); }
        .badge.info { color: var(--info); }
        .badge.safe { color: var(--safe); }
        .badge.planned { color: var(--planned); }
        .badge.risky { color: var(--risky); }
        .finding { background: rgba(0,0,0,0.10); border: 1px solid var(--border); border-radius: 16px; padding: 14px; margin-top: 12px; }
        .finding h3 { margin: 0; font-size: 14px; }
        .finding .subtitle { margin-top: 6px; color: var(--muted); font-size: 13px; }
        details { margin-top: 10px; }
        summary { cursor: pointer; color: var(--muted); }
        .grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
        .muted { color: var(--muted); }
        @media (max-width: 900px) { .cards { grid-template-columns: 1fr; } .header { flex-direction: column; } .meta { text-align: left; } .grid2 { grid-template-columns: 1fr; } }
    </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div>
        <h1 class="title">Docker Host Doctor Report</h1>
        <div class="row" style="margin-top:10px;">
          <span class="pill">Schema <code>{{.SchemaVersion}}</code></span>
          <span class="pill">Scan <code>{{.Scan.ScanID}}</code></span>
          <span class="pill">Mode <code>{{.Scan.EffectiveMode}}</code></span>
          <span class="pill">Duration <code>{{.Scan.DurationMs}}ms</code></span>
        </div>
      </div>
      <div class="meta">
        <div><strong>Finished:</strong> {{.Scan.FinishedAt.UTC.Format "2006-01-02 15:04:05"}} UTC</div>
        <div><strong>Tool:</strong> {{.Tool.Name}} <span class="muted">{{if .Tool.Version}}{{.Tool.Version}}{{else}}dev{{end}}</span></div>
      </div>
    </div>

    <div class="cards">
      <div class="card">
        <h3>Target</h3>
        <div class="big">{{.Target.Host.OS}} / {{.Target.Host.Arch}}</div>
        <div class="kv">Docker {{.Target.Docker.EngineVersion}} (API {{.Target.Docker.APIVersion}})</div>
        {{if .Target.Host.Hostname}}<div class="kv">Hostname: {{.Target.Host.Hostname}}</div>{{end}}
        {{if .Target.Host.Kernel}}<div class="kv">Kernel: {{.Target.Host.Kernel}}</div>{{end}}
        {{if gt .Target.Host.UptimeSeconds 0}}<div class="kv">Uptime: {{.Target.Host.UptimeSeconds}}s</div>{{end}}
        {{if .Target.Docker.CgroupVersion}}<div class="kv">Cgroup Version: {{.Target.Docker.CgroupVersion}}</div>{{end}}
        {{if .Target.Docker.DataRoot}}<div class="kv">Data Root: {{.Target.Docker.DataRoot}}</div>{{end}}
      </div>
      <div class="card">
        <h3>Counts</h3>
        <div class="big">{{.Summary.Counts.ContainersRunning}} running</div>
        <div class="kv">{{.Summary.Counts.ContainersStopped}} stopped · {{.Summary.Counts.Images}} images · {{.Summary.Counts.Volumes}} volumes</div>
      </div>
      <div class="card">
        <h3>Findings</h3>
        <div class="row">
          <span class="badge critical">{{.Summary.FindingCounts.Critical}} critical</span>
          <span class="badge warning">{{.Summary.FindingCounts.Warning}} warning</span>
          <span class="badge info">{{.Summary.FindingCounts.Info}} info</span>
        </div>
        <div class="kv" style="margin-top:8px;">This scan is read-only and offline-friendly.</div>
      </div>
    </div>

    <div class="section">
      <h2>Resource snapshot (deduplicated)</h2>
      <div class="grid2">
        <div class="card">
          <h3>Docker disk usage</h3>
          <div class="kv">Images: <strong>{{bytes .Summary.ResourceSnapshot.DockerSystemDf.ImagesTotalBytes}}</strong></div>
          <div class="kv">Build cache: <strong>{{bytes .Summary.ResourceSnapshot.DockerSystemDf.BuildCacheTotalBytes}}</strong></div>
          <div class="kv">Volumes: <strong>{{bytes .Summary.ResourceSnapshot.DockerSystemDf.VolumesTotalBytes}}</strong></div>
          <div class="kv">Containers writable: <strong>{{bytes .Summary.ResourceSnapshot.DockerSystemDf.ContainersWritableTotalBytes}}</strong></div>
        </div>
        <div class="card">
          <h3>Capabilities</h3>
          <div class="kv">Docker API: <strong>{{.Scan.Capabilities.DockerAPI}}</strong></div>
          <div class="kv">Host FS mounted: <strong>{{.Scan.Capabilities.HostFSMounted}}</strong></div>
          <div class="kv">Daemon config readable: <strong>{{.Scan.Capabilities.DaemonConfigReadable}}</strong></div>
          <div class="kv">Container log files readable: <strong>{{.Scan.Capabilities.ContainerLogFilesReadable}}</strong></div>
        </div>
      </div>
    </div>

    <div class="section">
      <h2>Collectors</h2>
      <table>
        <tr><th>Name</th><th>Status</th><th>Duration</th><th>Errors</th></tr>
        {{range .Collectors}}
        <tr>
          <td><code>{{.Name}}</code></td>
          <td class="muted">{{.Status}}</td>
          <td class="muted">{{.DurationMs}}ms</td>
          <td class="muted">{{if .Errors}}{{range .Errors}}<div>{{.}}</div>{{end}}{{else}}none{{end}}</td>
        </tr>
        {{end}}
      </table>
    </div>

    <div class="section">
      <h2>Findings</h2>
      {{if .Findings}}
        {{range .Findings}}
          <div class="finding" id="f-{{b64 .Fingerprint}}">
            <div class="row" style="justify-content: space-between;">
              <div class="row">
                <span class="badge {{.Severity}}">{{title .Severity}}</span>
                <h3 style="margin:0;">{{.Title}}</h3>
              </div>
              <div class="row">
                <span class="pill"><code>{{.ID}}</code></span>
                <span class="pill">Confidence <code>{{.Confidence}}</code></span>
              </div>
            </div>
            <div class="subtitle">{{.Summary}}</div>
            <div class="row" style="margin-top:10px;">
              {{if .Scope.ContainerName}}<span class="pill">Container <code>{{.Scope.ContainerName}}</code> <span class="muted">{{.Scope.ContainerID}}</span></span>{{end}}
              {{if .Scope.Path}}<span class="pill">Path <code>{{.Scope.Path}}</code></span>{{end}}
              <span class="pill">Fingerprint <code>{{.Fingerprint}}</code></span>
            </div>

            <details>
              <summary>Evidence</summary>
              {{if .Evidence}}
              <table style="margin-top:10px;">
                <tr><th>Type</th><th>Key</th><th>Value</th></tr>
                {{range .Evidence}}
                <tr>
                  <td class="muted">{{.Type}}</td>
                  <td><code>{{.Key}}</code></td>
                  <td class="muted"><code>{{json .Value}}</code></td>
                </tr>
                {{end}}
              </table>
              {{else}}
                <div class="muted" style="margin-top:10px;">No evidence provided.</div>
              {{end}}
            </details>

            <details open>
              <summary>Recommended actions</summary>
              {{if .Recommendations}}
                {{range .Recommendations}}
                  <div class="card" style="margin-top:10px;">
                    <div class="row">
                      <span class="badge {{riskClass .Risk}}">{{.Risk}}</span>
                      <strong>{{.Title}}</strong>
                    </div>
                    {{if .Steps}}
                      <ol class="muted" style="margin-top:10px; padding-left: 20px;">
                        {{range .Steps}}<li>{{.}}</li>{{end}}
                      </ol>
                    {{end}}
                    {{if .Commands}}
                      <div class="muted" style="margin-top:10px;">Commands</div>
                      <ul class="muted" style="padding-left:20px;">
                        {{range .Commands}}<li><code>{{.}}</code></li>{{end}}
                      </ul>
                    {{end}}
                    {{if .Notes}}
                      <div class="muted">Notes</div>
                      <ul class="muted">
                        {{range .Notes}}<li>{{.}}</li>{{end}}
                      </ul>
                    {{end}}
                  </div>
                {{end}}
              {{else}}
                <div class="muted" style="margin-top:10px;">No recommendations provided.</div>
              {{end}}
            </details>
          </div>
        {{end}}
      {{else}}
        <div class="card"><strong>No findings.</strong> <span class="muted">Host looks stable based on available signals.</span></div>
      {{end}}
    </div>
  </div>
</body>
</html>
`
	t, err := template.New("report-v1").Funcs(funcs).Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	if err := t.Execute(&buf, report); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func generateMarkdownv1(report *v1.Report) (string, error) {
	md := fmt.Sprintf(`# Docker Host Doctor Report

**Scan ID:** %s  
**Finished:** %s UTC  
**Mode:** %s  
**Tool:** %s %s

## Target
- **Host:** %s / %s
- **Docker Engine:** %s (API %s)
- **Hostname:** %s
- **Kernel:** %s
- **Uptime:** %d s
- **Cgroup Version:** %s
- **Data Root:** %s

## Summary

| Metric | Value |
|---|---:|
| Containers (running) | %d |
| Containers (stopped) | %d |
| Images | %d |
| Volumes | %d |
| Findings (critical) | %d |
| Findings (warning) | %d |
| Findings (info) | %d |

## Resource snapshot (deduplicated)

| Resource | Size |
|---|---:|
| Images | %s |
| Build cache | %s |
| Volumes | %s |
| Containers writable | %s |

## Collectors

| Name | Status | Duration | Errors |
|---|---|---:|---|
`, report.Scan.ScanID, report.Scan.FinishedAt.UTC().Format("2006-01-02 15:04:05"),
		report.Scan.EffectiveMode,
		report.Tool.Name, fallback(report.Tool.Version, "dev"),
		report.Target.Host.OS, report.Target.Host.Arch,
		report.Target.Docker.EngineVersion, report.Target.Docker.APIVersion,
		report.Target.Host.Hostname,
		report.Target.Host.Kernel,
		report.Target.Host.UptimeSeconds,
		report.Target.Docker.CgroupVersion,
		report.Target.Docker.DataRoot,
		report.Summary.Counts.ContainersRunning,
		report.Summary.Counts.ContainersStopped,
		report.Summary.Counts.Images,
		report.Summary.Counts.Volumes,
		report.Summary.FindingCounts.Critical,
		report.Summary.FindingCounts.Warning,
		report.Summary.FindingCounts.Info,
		humanBytes(report.Summary.ResourceSnapshot.DockerSystemDf.ImagesTotalBytes),
		humanBytes(report.Summary.ResourceSnapshot.DockerSystemDf.BuildCacheTotalBytes),
		humanBytes(report.Summary.ResourceSnapshot.DockerSystemDf.VolumesTotalBytes),
		humanBytes(report.Summary.ResourceSnapshot.DockerSystemDf.ContainersWritableTotalBytes),
	)

	for _, c := range report.Collectors {
		errs := "none"
		if len(c.Errors) > 0 {
			errs = strings.Join(c.Errors, "; ")
		}
		md += fmt.Sprintf("| `%s` | %s | %dms | %s |\n", c.Name, c.Status, c.DurationMs, escapePipes(errs))
	}

	md += "\n## Findings\n\n"

	md += "This report is **read-only**. It suggests actions but does not execute them.\n\n"

	if len(report.Findings) == 0 {
		md += "**No findings.**\n"
		return md, nil
	}

	for _, f := range report.Findings {
		md += fmt.Sprintf("### %s — `%s`\n\n", strings.ToUpper(f.Severity), f.ID)

		if f.Title != "" {
			md += fmt.Sprintf("**Title:** %s\n\n", f.Title)
		}
		md += fmt.Sprintf("**Fingerprint:** `%s`\n\n", f.Fingerprint)
		md += fmt.Sprintf("**Confidence:** %s\n\n", f.Confidence)
		if f.Category != "" {
			md += fmt.Sprintf("**Category:** %s\n\n", f.Category)
		}

		scope := []string{}
		if f.Scope.ContainerName != "" || f.Scope.ContainerID != "" {
			scope = append(scope, fmt.Sprintf("container=%s(%s)", f.Scope.ContainerName, f.Scope.ContainerID))
		}
		if f.Scope.Path != "" {
			scope = append(scope, fmt.Sprintf("path=%s", f.Scope.Path))
		}
		if len(scope) > 0 {
			md += fmt.Sprintf("**Scope:** `%s`\n\n", strings.Join(scope, " "))
		}

		if f.Summary != "" {
			md += fmt.Sprintf("**Summary:** %s\n\n", f.Summary)
		}

		if len(f.Evidence) > 0 {
			md += "**Evidence**\n\n| Type | Key | Value |\n|---|---|---|\n"
			for _, e := range f.Evidence {
				b, _ := json.Marshal(e.Value)
				md += fmt.Sprintf("| %s | `%s` | `%s` |\n", e.Type, escapePipes(e.Key), escapePipes(string(b)))
			}
			md += "\n"
		}

		if len(f.Recommendations) > 0 {
			md += "**Recommended actions**\n\n"
			for _, r := range f.Recommendations {
				md += fmt.Sprintf("- **%s (%s)**\n", r.Title, r.Risk)
				for _, s := range r.Steps {
					md += fmt.Sprintf("  - %s\n", s)
				}
				if len(r.Commands) > 0 {
					md += "\n```bash\n"
					for _, c := range r.Commands {
						md += c + "\n"
					}
					md += "```\n\n"
				}
				for _, n := range r.Notes {
					md += fmt.Sprintf("  - _Note_: %s\n", n)
				}
			}
			md += "\n"
		}
	}

	return md, nil
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
	pow := math.Pow10(decimals)
	f = math.Round(f*pow) / pow
	return fmt.Sprintf("%.*f %s", decimals, f, units[i])
}

func severityClass(sev string) string {
	switch strings.ToLower(strings.TrimSpace(sev)) {
	case "critical":
		return "critical"
	case "warning":
		return "warning"
	default:
		return "info"
	}
}

func riskClass(risk string) string {
	switch strings.ToLower(strings.TrimSpace(risk)) {
	case "safe":
		return "safe"
	case "risky":
		return "risky"
	default:
		return "planned"
	}
}

func fallback(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
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
