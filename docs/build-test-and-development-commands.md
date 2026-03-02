# Build, Test, and Development Commands

This document is the detailed command reference for local development in `go-service-template-rest`.

## Scope

Commands in this document come from:
- `Makefile` (primary interface)
- `go` toolchain commands used by make targets
- Docker and compose commands used for local environment

## Prerequisites

- Go toolchain installed (version from `go.mod`)
- Perl installed (used by `scripts/init-module.sh`)
- Docker daemon running (required for `compose` and integration scenarios)
- `golangci-lint` installed for `make lint`
- Node/npm available for OpenAPI lint (`npx @redocly/cli`)

Bootstrap shortcut for beginners:
- `make setup` to install pinned `golangci-lint`, prepare `.env`, download Go modules, and run `make doctor` checks.

## Command Groups

### Bootstrap and environment checks

- `make setup`
  - Runs: `./scripts/dev/setup.sh <golangci-lint-version>`
  - Purpose: first-run bootstrap for local environment.
  - Includes:
    - install pinned `golangci-lint`,
    - create `.env` from `env/.env.example` when missing,
    - `go mod download`,
    - `make doctor`.

- `make doctor`
  - Runs: `./scripts/dev/doctor.sh`
  - Purpose: check local machine readiness and show missing tooling.
  - Checks:
    - required: `make`, `git`, `go`, Go version vs `go.mod`, `node`, `npx`
    - optional: `golangci-lint`, Docker CLI/daemon

### Dependency and module maintenance

- `make init-module MODULE=<module_path>`
  - Runs: `./scripts/init-module.sh <module_path>`
  - Purpose: one-shot bootstrap after clone; updates `go.mod`, internal Go imports, and proto `go_package` module prefix.
  - Includes: `go mod tidy` at the end.
  - Example:
    - `make init-module MODULE=github.com/acme/my-service`

- `make tidy`
  - Runs: `go mod tidy`
  - Purpose: clean and sync `go.mod`/`go.sum` with actual imports.
  - Use when: after adding/removing dependencies.

- `make mod-check`
  - Runs:
    - `go mod tidy -diff`
    - `go mod verify`
    - `git diff --exit-code -- go.mod go.sum`
  - Purpose: enforce deterministic module integrity without local drift.
  - Use when: before push and in CI merge-gates.

- `make vendor`
  - Runs: `go mod vendor`
  - Purpose: populate `vendor/` with module dependencies.
  - Use when: your delivery process requires vendored dependencies.

### Formatting and static quality

- `make fmt`
  - Runs `gofmt` on all Go files except `vendor/`.
  - Purpose: enforce canonical Go formatting.
  - Use when: before commits/PRs and after large edits.

- `make fmt-check`
  - Runs:
    - `make fmt`
    - `git diff --exit-code`
  - Purpose: fail if formatting would change tracked Go sources.
  - Use when: CI gate or pre-push validation.

- `make lint`
  - Runs: `golangci-lint run`
  - Purpose: static checks (including `govet`, `staticcheck`, `errcheck`, `bodyclose`, `sqlclosecheck`, `errorlint`, `contextcheck` per config).
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
  - Notes:
    - local mode skips tests when Docker daemon is unavailable;
    - CI uses `REQUIRE_DOCKER=1` and fails if Docker is unavailable.

### OpenAPI and API contract workflow

- `make openapi-generate`
  - Runs: `go generate ./internal/api`
  - Purpose: regenerate Go artifacts from OpenAPI spec.
  - Source spec: `api/openapi/service.yaml`
  - Generation config: `internal/api/oapi-codegen.yaml`

- `make openapi-drift-check`
  - Runs:
    - `git diff -- internal/api` (tracked drift)
    - `git ls-files --others --exclude-standard -- internal/api` (untracked artifacts)
  - Purpose: fail if generated OpenAPI artifacts are not in the expected git state.
  - Use when: after `make openapi-generate` and in CI contract gates.

- `make openapi-runtime-contract-check`
  - Runs: `go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1`
  - Purpose: verify that HTTP runtime behavior is still aligned with OpenAPI strict handler contract.
  - Use when: after changing `api/openapi/service.yaml` or runtime handler wiring.

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
    - `openapi-drift-check`
    - `openapi-runtime-contract-check`
    - `openapi-lint`
    - `openapi-validate`
  - Purpose: run the full contract check in one command.

### CI policy helper checks

- `make guardrails-check`
  - Runs: `scripts/ci/required-guardrails-check.sh`
  - Purpose: enforce mandatory repository process files.
  - Required files:
    - `.editorconfig`
    - `.gitattributes`
    - `.github/CODEOWNERS`
    - `.github/pull_request_template.md`
    - `CONTRIBUTING.md`
    - `SECURITY.md`
    - `LICENSE`

- `make docs-drift-check BASE_REF=<base_sha> HEAD_REF=<head_sha>`
  - Runs: `scripts/ci/docs-drift-check.sh`
  - Purpose: enforce docs updates when behavior/contract/CI-sensitive paths change.
  - Trigger paths include:
    - `api/openapi/service.yaml`
    - `env/migrations/**`
    - `Makefile`
    - `.github/workflows/**`
    - `cmd/**`
    - `internal/app/**`
    - `internal/config/**`
    - `internal/infra/http/**`
    - `internal/infra/postgres/**`
    - `internal/infra/telemetry/**`
  - Required docs paths:
    - `docs/**` or `README.md`

- `make migration-validate MIGRATION_DSN=<postgres_dsn>`
  - Runs `golang-migrate` against `env/migrations`:
    - apply all up migrations
    - run `down 1`
    - run `up 1`
  - Purpose: validate migration chain on ephemeral Postgres in CI.
  - Use when: migration SQL changed (`env/migrations/**`).

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

### First run after clone

1. `make setup`
2. `make init-module MODULE=github.com/<your-org>/<your-service>`
3. `make mod-check`
4. `make test`

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
- `make mod-check` + `make guardrails-check` + `make fmt-check` + `make docs-drift-check` -> `repo-integrity`
- `make lint` -> `lint`
- `make openapi-generate` + `make openapi-drift-check` + `make openapi-runtime-contract-check` + `make openapi-validate` + `make openapi-lint` -> `openapi-contract`
- `BASE_OPENAPI=... make openapi-breaking` -> `openapi-breaking` (PR only)
- `make test` -> `test`
- `make test-race` -> `test-race`
- `make test-cover` -> `test-coverage`
- `REQUIRE_DOCKER=1 make test-integration` -> `test-integration`
- `make migration-validate` -> `migration-validate` (only when migrations changed)
- `govulncheck`, `gosec -exclude-generated`, Trivy image scan -> `go-security`, `container-security`

Nightly workflow: `.github/workflows/nightly.yml`
- Adds heavier reliability checks:
  - `go test -count=5 ./...`
  - `make test-race`
  - `make test-integration`
  - full OpenAPI/security/container checks

CD workflow: `.github/workflows/cd.yml`
- `publish-main`: after successful `ci` on `main`, builds/scans/signs/publishes image to GHCR with `main` and `sha-*` tags.
- `release-preflight`: on tag `v*`, reruns quality and security gates before publish.
- `publish-release`: on tag `v*`, runs only after `release-preflight`, then builds/scans/signs/publishes `v*`, `latest`, and `sha-*` tags, uploads CycloneDX SBOM, and pushes provenance attestation.

Repository settings checklist for hard enforcement:
- `docs/ci-cd-production-ready.md`
