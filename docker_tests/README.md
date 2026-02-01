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

## Notes

- The `scan` command writes artifacts to `./out/<scanId>/` by default (configurable via `--output-dir`).

