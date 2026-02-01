.PHONY: help build test test-integration scan scan-ci clean

GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || echo "unknown")

help:
	@echo "Targets:"
	@echo "  make build             - Build docker-doctor binary with version info"
	@echo "  make test              - Run unit tests"
	@echo "  make test-integration  - Run integration tests (requires env vars)"
	@echo "  make scan              - Run scan and write to ./out"
	@echo "  make scan-ci           - Run scan with non-zero exit codes"
	@echo "  make clean             - Remove ./out"
	@echo ""
	@echo "Integration env vars:"
	@echo "  RUN_INTEGRATION=1"
	@echo "  DOCKER_HOST=unix:///Users/<you>/.rd/docker.sock (or your engine socket)"

build:
	go build -ldflags "-X main.version=$(shell git describe --tags --abbrev=0 2>/dev/null || echo dev) -X main.gitCommit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME)" .

test:
	go test ./...

test-integration:
	RUN_INTEGRATION=1 go test -tags=integration ./internal/collector -run TestCollect -v

scan:
	go run . scan --config doctor.yml --output-dir ./out

scan-ci:
	go run . scan --config doctor.yml --output-dir ./out --exit-code

clean:
	rm -rf ./out

