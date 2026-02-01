# Daemon Risky Settings Test

This test scenario is for validating the `DAEMON_RISKY_SETTINGS` rule.

## Prerequisites

- Docker daemon with risky settings configured

## Setup Risky Settings

To test this rule, configure your Docker daemon with risky settings. For example:

1. Edit `/etc/docker/daemon.json`:
```json
{
  "experimental": true,
  "insecure-registries": ["registry.example.com"],
  "log-driver": "none"
}
```

2. Restart Docker daemon:
```bash
sudo systemctl restart docker
```

## Expected Finding

- `DAEMON_RISKY_SETTINGS` (severity: medium or high depending on number of risky settings)

## Notes

- This test cannot be easily automated with docker-compose since daemon settings are global
- The rule checks for experimental features, insecure registries, and logging configuration
- Manual verification required for different daemon configurations