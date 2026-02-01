# Daemon Risky Settings Test

This test scenario is for validating the `DAEMON_RISKY_SETTINGS` rule.

## What this does

Starts a **Docker-in-Docker (DinD)** daemon with intentionally risky settings (via `daemon.json`) so you can run Docker-Doctor against it without changing your real host daemon.

## How to run

1) Start the scenario:

```bash
cd docker_tests/daemon-risky
docker compose up --build
```

2) In another terminal, run a scan from repo root against the DinD daemon:

```bash
cd /Users/nowshadurrahaman/Projects/Nowshad/Docker-Doctor
go run . scan --config docker_tests/daemon-risky/doctor.yml --output-dir ./out
```

## Expected Finding

- `DAEMON_RISKY_SETTINGS` (severity: medium or high depending on number of risky settings)

## Notes

- The scenario uses `docker_tests/daemon-risky/daemon.json` to enable:
  - `experimental: true`
  - `insecure-registries: [...]`
  - `log-driver: none`
- The DinD daemon is exposed on `tcp://localhost:23750` for the scan.