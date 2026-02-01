# Docker Host Doctor

Docker Host Doctor is a lightweight CLI that scans a Docker host and produces a prioritized health report: what’s broken (or about to break), why it matters, and what to do next.

## What it does (current)

- Produces a **v1 JSON scan contract** (`scan.json`) plus **professional HTML + Markdown reports**.
- Detects and reports:
  - `DOCKER_STORAGE_BLOAT` (uses Docker `/system/df` for deduplicated disk usage when possible)
  - `DISK_USAGE_HIGH` (host disk usage thresholds)
  - `RESTART_LOOP` (restart threshold or “restarting” status)
  - `OOM_KILLED` (from container inspect)
  - `HEALTHCHECK_UNHEALTHY` (from container inspect health status)

## Install / Run

Requirements:
- Go (module uses `go.mod`)
- Docker Engine / Docker Desktop / Rancher Desktop

Run a scan and generate all artifacts (JSON + HTML + MD) in one command:

```bash
go run . scan --config doctor.yml --output-dir ./out
```

This writes to:

```
./out/<scanId>/
  scan.json
  report.html
  report.md
```

## Command reference

### `scan`

```bash
docker-doctor scan --config doctor.yml --output-dir ./out
```

Useful flags:
- `--output-dir, -o`: directory to write artifacts (default `./out`)
- `--formats`: comma-separated `json,html,md` (default `json,html,md`)
- `--exit-code`: CI mode; exit non-zero for WARN/CRITICAL findings
- `--verbose`: debug logs to stderr

### `report` (optional)

If you already have a `scan.json` and want to re-render:

```bash
docker-doctor report --input ./out/<scanId>/scan.json --format html --output report.html
docker-doctor report --input ./out/<scanId>/scan.json --format md --output report.md
```

## Configuration

Configuration is loaded from `doctor.yml` by default (override with `--config`).

Example snippet:

```yaml
scan:
  mode: basic
  timeout: 30
  dockerHost: unix:///Users/<you>/.rd/docker.sock
  version: "1.41"
rules:
  disk_usage:
    threshold: 80
  storage_bloat:
    image_size_threshold: 10737418240
  restarts:
    threshold: 3
  oom:
    enabled: true
  healthcheck:
    enabled: true
```

## Tests

Unit tests (default):

```bash
go test ./...
```

Integration test (requires Docker access explicitly):

```bash
RUN_INTEGRATION=1 DOCKER_HOST=unix:///Users/<you>/.rd/docker.sock go test -tags=integration ./internal/collector -run TestCollect -v
```

## Repro scenarios

See `docker_tests/` for small compose scenarios that trigger rules (restart loop, OOM killed, unhealthy healthcheck, etc.).

