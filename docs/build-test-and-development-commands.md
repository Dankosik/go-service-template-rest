# Build, Test, and Development Commands

This document is the detailed command reference for local development in `go-service-template-rest`.

## Scope

Commands in this document come from:
- `Makefile` (primary interface)
- `go` toolchain commands used by make targets
- Docker and compose commands used for local environment

## Prerequisites

- Go toolchain installed (version from `go.mod`)
- Docker daemon running (required for `compose` and integration scenarios)
- `golangci-lint` installed for `make lint`
- Node/npm available for OpenAPI lint (`npx @redocly/cli`)

## Command Groups

### Dependency and module maintenance

- `make tidy`
  - Runs: `go mod tidy`
  - Purpose: clean and sync `go.mod`/`go.sum` with actual imports.
  - Use when: after adding/removing dependencies.

- `make vendor`
  - Runs: `go mod vendor`
  - Purpose: populate `vendor/` with module dependencies.
  - Use when: your delivery process requires vendored dependencies.

### Formatting and static quality

- `make fmt`
  - Runs `gofmt` on all Go files except `vendor/`.
  - Purpose: enforce canonical Go formatting.
  - Use when: before commits/PRs and after large edits.

- `make lint`
  - Runs: `golangci-lint run`
  - Purpose: static checks (including vet/staticcheck/errcheck/revive per config).
  - Use when: before pushing and to reproduce CI lint failures.

### Unit and integration testing

- `make test`
  - Runs: `go test ./...`
  - Purpose: execute default package test suite.
  - Use when: baseline local verification.

- `make test-race`
  - Runs: `go test -race ./...`
  - Purpose: detect data races and concurrency issues.
  - Use when: touching goroutines/channels/shared mutable state.

- `make test-cover`
  - Runs:
    - `go test -covermode=atomic -coverprofile=coverage.out ./...`
    - `go tool cover -func=coverage.out`
  - Purpose: produce coverage report and summary.
  - Use when: validating coverage impact of changes.

- `make test-integration`
  - Runs: `go test -tags=integration ./test/...`
  - Purpose: execute integration tests under `test/`.
  - Use when: validating DB/container-dependent behavior.
  - Notes: requires Docker (used by integration setup).

### OpenAPI and API contract workflow

- `make openapi-generate`
  - Runs: `go generate ./internal/api`
  - Purpose: regenerate Go artifacts from OpenAPI spec.
  - Source spec: `api/openapi/service.yaml`
  - Generation config: `internal/api/oapi-codegen.yaml`

- `make openapi-lint`
  - Runs: `npx @redocly/cli@2.20.0 lint api/openapi/service.yaml`
  - Purpose: OpenAPI style/rule validation.

- `make openapi-validate`
  - Runs: `kin-openapi validate` against `api/openapi/service.yaml`
  - Purpose: schema-level OpenAPI correctness check.

- `make openapi-breaking`
  - Runs `oasdiff breaking` against:
    - `BASE_OPENAPI` (required environment variable)
    - current `api/openapi/service.yaml`
  - Purpose: detect breaking API changes.
  - Example:
    - `BASE_OPENAPI=/path/to/base-service.yaml make openapi-breaking`

- `make openapi-check`
  - Composite target:
    - `openapi-generate`
    - `openapi-lint`
    - `openapi-validate`
  - Purpose: run the full contract check in one command.

### Run and build

- `make run`
  - Runs: `go run ./cmd/service`
  - Purpose: start the service locally without building a binary.

- `make build`
  - Builds static binary:
    - output: `bin/service`
    - flags: `CGO_ENABLED=0`, `-trimpath`, `-ldflags='-s -w'`
  - Purpose: produce a lightweight local artifact.

### Container and local environment

- `make docker-build`
  - Runs: `docker build -f build/docker/Dockerfile -t service:local .`
  - Purpose: build local container image.

- `make compose-up`
  - Runs: `docker compose -f env/docker-compose.yml up -d`
  - Purpose: start local infrastructure (for example Postgres).

- `make compose-down`
  - Runs: `docker compose -f env/docker-compose.yml down -v`
  - Purpose: stop and remove local infrastructure volumes.

## Recommended Local Workflows

### Feature implementation (typical)

1. `make fmt`
2. `make test`
3. `make lint`
4. If API contract changed: `make openapi-check`
5. If integration behavior changed: `make test-integration`

### Before opening a PR

1. `make fmt`
2. `make lint`
3. `make test`
4. `make test-race`
5. `make openapi-check`
6. `make test-integration` (if relevant to your change)

## CI Mapping

Main CI workflow: `.github/workflows/ci.yml`

Local commands map directly to CI jobs:
- `make test` -> unit tests
- `make test-race` -> race detector
- `make test-cover` -> coverage job
- `make test-integration` -> integration job
- `make lint` -> lint job
- `make openapi-check` -> openapi contract job

Security jobs (`govulncheck`, `gosec`, Trivy) run in CI and are not wrapped by dedicated make targets in this repository.
