# docker_tests

Small, reproducible Docker/Compose scenarios to validate Docker Host Doctor rules end-to-end.

## Prerequisites

- Docker Desktop / Rancher Desktop / Docker Engine running
- `docker compose` available
- Run Doctor from repo root using your `doctor.yml` (or `--config ...`)

## How to run any scenario

1) Start the scenario:

```bash
cd docker_tests/<SCENARIO>
docker compose up --build
```

2) Run a scan from the repo root (in another terminal):

```bash
cd /Users/nowshadurrahaman/Projects/Nowshad/Docker-Doctor
go run . scan --config doctor.yml --output-dir ./out
```

3) View findings:

```bash
jq '.findings[] | {id,severity,fingerprint,summary}' ./out/<scanId>/scan.json
```

4) Cleanup:

```bash
cd docker_tests/<SCENARIO>
docker compose down --remove-orphans
```

## Scenarios (what’s in this folder)

### `cpu-heavy/` (workload generator)

- **Purpose**: Generates sustained CPU load (useful for future “host pressure” rules).
- **Expected Doctor finding today**: **none** (we don’t scan CPU saturation yet).
- **Notes**:
  - Adjust `THREADS` / `MATRIX` in `compose.yaml` to tune load.

### `disk-heavy/` (workload generator)

- **Purpose**: Writes data into a named volume (useful for future volume bloat checks).
- **Expected Doctor finding today**: **none** (we don’t scan per-volume sizes yet).
- **Notes**:
  - Controlled by `FILE_MB` in `compose.yaml` (writes into `/data` volume).

### `restart-loop/` (rule validation)

- **Goal**: trigger `RESTART_LOOP`
- **How**: container exits with code 1 under `restart: always`
- **Expected finding**:
  - `RESTART_LOOP` (severity: `critical`)

### `oom-killed/` (rule validation)

- **Goal**: trigger `OOM_KILLED`
- **How**: memory hog container with a low memory limit (`mem_limit`)
- **Expected finding**:
  - `OOM_KILLED` (severity: `critical`)
- **Notes**:
  - The container will typically exit with code `137` after being killed.

### `healthcheck-unhealthy/` (rule validation)

- **Goal**: produce an unhealthy container health status
- **Expected finding**:
  - `HEALTHCHECK_UNHEALTHY`

### `log-bloat/` (rule validation)

- **Goal**: trigger `LOG_BLOAT`
- **How**: container generates large log output (>100MB)
- **Expected finding**:
  - `LOG_BLOAT` (severity: `medium` or `high` depending on size)

### `network-overlap/` (rule validation)

- **Goal**: trigger `NETWORK_OVERLAP`
- **How**: creates Docker networks with overlapping CIDR ranges
- **Expected finding**:
  - `NETWORK_OVERLAP` (severity: `high`)

### `volume-bloat/` (rule validation)

- **Goal**: trigger `VOLUME_BLOAT`
- **How**: creates Docker volumes where some are used by containers and some are unused
- **Expected finding**:
  - `VOLUME_BLOAT` (severity: `low` or `medium` depending on unused count)

### `volume-large/` (workload generator)

- **Purpose**: creates a Docker volume with large content for testing volume size detection
- **Expected Doctor finding today**: none (volume sizes are informational)
- **Notes**:
  - Creates a 50MB file in the volume
  - Useful for validating volume size collection

### `daemon-risky/` (configuration validation)

- **Goal**: trigger `DAEMON_RISKY_SETTINGS`
- **How**: requires Docker daemon configured with risky settings (experimental, insecure registries, etc.)
- **Expected finding**:
  - `DAEMON_RISKY_SETTINGS` (severity: medium/high)
- **Notes**:
  - Requires manual daemon configuration
  - Cannot be automated with docker-compose
  - See `daemon-risky/README.md` for setup instructions

## Notes

- The `scan` command writes artifacts to `./out/<scanId>/` by default (configurable via `--output-dir`).

