.PHONY: help test test-integration scan scan-ci clean

help:
	@echo "Targets:"
	@echo "  make test              - Run unit tests"
	@echo "  make test-integration  - Run integration tests (requires env vars)"
	@echo "  make scan              - Run scan and write to ./out"
	@echo "  make scan-ci           - Run scan with non-zero exit codes"
	@echo "  make clean             - Remove ./out"
	@echo ""
	@echo "Integration env vars:"
	@echo "  RUN_INTEGRATION=1"
	@echo "  DOCKER_HOST=unix:///Users/<you>/.rd/docker.sock (or your engine socket)"

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

