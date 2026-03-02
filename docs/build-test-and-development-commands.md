# Build, Test, and Development Commands

This document is the detailed command reference for local development in `go-service-template-rest`.

## Scope

Commands in this document come from:
- `Makefile` (primary interface)
- `go` toolchain commands used by make targets
- Docker and compose commands used for local environment
- Docker-based zero-setup wrapper script (`scripts/dev/docker-tooling.sh`)

## Prerequisites

This template supports two onboarding modes.

### Native mode

Required:
- Go toolchain installed (version from `go.mod`)
- Node/npm (`npx`) for OpenAPI lint
- GNU Make
- Git

Optional:
- Docker daemon (for integration tests, compose, container build/run)
- GitHub CLI (`gh`) for `make gh-protect`

### Zero-setup docker mode

Required:
- Git
- Docker CLI + running Docker daemon

Optional:
- GNU Make (convenience wrapper; commands can be called via scripts directly)
- local Go/Node toolchain (not required in this mode)
- GitHub CLI (`gh`) for `make gh-protect`

Bootstrap shortcuts:
- `make setup` (auto-select mode)
- `make setup-native`
- `make setup-docker`

## Command Groups

### Bootstrap and environment checks

- `make setup`
  - Runs: `./scripts/dev/setup.sh`
  - Purpose: first-run bootstrap with mode auto-detection.
  - Mode choice:
    - prefers native mode when local `go` exists;
    - falls back to docker mode when local `go` is absent and Docker is available.
    - if native bootstrap fails and Docker is available, switches to docker bootstrap.

- `make setup-native`
  - Runs: `./scripts/dev/setup.sh --native`
  - Includes:
    - create `.env` from `env/.env.example` when missing,
    - `go mod download`,
    - `make doctor-native`,
    - `make skills-sync`.

- `make setup-docker`
  - Runs: `./scripts/dev/setup.sh --docker`
  - Includes:
    - create `.env` from `env/.env.example` when missing,
    - pull pinned tool images,
    - `make doctor-docker`,
    - `make skills-sync`.

- `make doctor`
  - Runs: `./scripts/dev/doctor.sh --mode auto`
  - Purpose: check local readiness for the selected mode.

- `make doctor-native`
  - Runs: `./scripts/dev/doctor.sh --mode native`
  - Highlights:
    - validates local Go/Node prerequisites;
    - validates Go version against `go.mod`;
    - performs Go compile sanity check (required);
    - performs Go coverage compile sanity check (optional warning-only).

- `make doctor-docker`
  - Runs: `./scripts/dev/doctor.sh --mode docker`
  - Highlights:
    - validates `git`, `docker`, and Docker daemon reachability;
    - confirms zero-setup path is available.

- `make docker-pull-tools`
  - Runs: `scripts/dev/docker-tooling.sh pull-images`
  - Purpose: pre-pull Docker images used by zero-setup commands.

### Dependency and module maintenance

- `make init-module [MODULE=<module_path>] [CODEOWNER=@org/team]`
  - Runs: `./scripts/init-module.sh [module-path]`
  - Purpose: one-shot bootstrap after clone; updates `go.mod`, internal Go imports, proto `go_package` module prefix, and optionally replaces CODEOWNERS placeholder.
  - If `MODULE` is omitted, script auto-detects module path from `git remote origin`.
  - Includes: `go mod tidy` at the end.
  - Note: script no longer requires Perl.

- `make docker-init-module [MODULE=<module_path>] [CODEOWNER=@org/team]`
  - Runs in Docker tooling container with the same behavior as `make init-module`.

- `make gh-protect BRANCH=<branch>`
  - Runs: `./scripts/dev/configure-branch-protection.sh <branch>`
  - Purpose: apply required branch protection and CI status checks for production usage.
  - Notes:
    - `.github/CODEOWNERS` must not contain template placeholder (`@your-org/your-team`);
    - requires `gh auth login`;
    - requires admin/maintainer permissions.

- `make tidy`
  - Runs: `go mod tidy`

- `make mod-check`
  - Runs:
    - `go mod tidy -diff`
    - `go mod verify`
    - `git diff --exit-code -- go.mod go.sum`

- `make docker-mod-check`
  - Docker equivalent of `make mod-check`.

- `make vendor`
  - Runs: `go mod vendor`

### Formatting and static quality

- `make fmt`
  - Runs `gofmt` on all Go files except `vendor/`.

- `make docker-fmt`
  - Docker equivalent of `make fmt`.

- `make fmt-check`
  - Fails only when `gofmt -l` reports unformatted Go files.

- `make docker-fmt-check`
  - Docker equivalent of `make fmt-check` (same `gofmt -l` behavior).

- `make lint`
  - Runs: `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@<pinned-version> run --timeout=3m`

- `make docker-lint`
  - Docker equivalent of `make lint`.

### Unit and integration testing

- `make test`
  - Runs: `go test ./...`

- `make docker-test`
  - Docker equivalent of `make test`.

- `make test-race`
  - Runs: `go test -race ./...`

- `make docker-test-race`
  - Docker equivalent of `make test-race`.

- `make test-cover`
  - Runs:
    - `GOCOVERDIR= go test -covermode=atomic -coverprofile=coverage.out ./...`
    - `go tool cover -func=coverage.out`

- `make test-cover-local`
  - Runs same coverage flow as `test-cover`, but degrades to warning when local coverage tooling is unhealthy.
  - Intended for beginner-friendly local checks (`ci-local`) where regular tests already passed.

- `make docker-test-cover`
  - Docker equivalent of `make test-cover`.

- `make test-integration`
  - Runs: `go test -tags=integration ./test/...`
  - Local behavior:
    - skips when Docker daemon is unavailable.
  - CI behavior:
    - `REQUIRE_DOCKER=1` enforces failure when Docker is unavailable.

- `make docker-test-integration`
  - Docker tooling equivalent of integration tests.
  - Uses Docker socket passthrough when available.

### OpenAPI and API contract workflow

- `make openapi-generate`
  - Runs: `go generate ./internal/api`

- `make openapi-drift-check`
  - Checks tracked and untracked codegen drift in `internal/api`.

- `make openapi-runtime-contract-check`
  - Runs: `go test ./internal/infra/http -run '^TestOpenAPIRuntimeContract' -count=1`

- `make openapi-lint`
  - Runs: `npx @redocly/cli@2.20.0 lint --config .redocly.yaml api/openapi/service.yaml`

- `make openapi-validate`
  - Runs: `kin-openapi validate` against `api/openapi/service.yaml`

- `make openapi-breaking`
  - Runs `oasdiff breaking` against `BASE_OPENAPI` and current spec.

- `make openapi-check`
  - Composite target:
    - `openapi-generate`
    - `openapi-drift-check`
    - `go test ./internal/api`
    - `openapi-runtime-contract-check`
    - `openapi-lint`
    - `openapi-validate`

- `make docker-openapi-check`
  - Docker equivalent of `make openapi-check`.

### Security and CI-like local checks

- `make go-security`
  - Runs native `govulncheck` and `gosec -exclude-generated`.

- `make ci-local`
  - Native composite check for beginner-friendly local parity:
    - `mod-check`
    - `guardrails-check`
    - `skills-check`
    - `fmt-check`
    - `lint`
    - `test`
    - `test-race`
    - `test-cover-local`
    - `openapi-check`
    - `go-security`
  - When Docker daemon is reachable, also runs:
    - `test-integration` (`REQUIRE_DOCKER=1`)
    - `docker-migration-validate`
    - `docker-container-security`
  - When Docker is unavailable, docker-only checks are skipped with a clear message.

- `make docker-go-security`
  - Runs `govulncheck` and `gosec` through Docker tooling container.

- `make docker-guardrails-check`
  - Runs required repository guardrails check in Docker mode wrapper.

- `make docker-skills-check`
  - Runs skill mirror consistency check.

- `make docker-docs-drift-check BASE_REF=<base_sha> HEAD_REF=<head_sha>`
  - Runs docs drift policy check through Docker mode wrapper.

- `make docker-migration-validate`
  - Runs migration rehearsal (`up`, `down 1`, `up 1`) on ephemeral Docker Postgres.

- `make docker-container-security`
  - Builds `service:ci` image and runs Trivy scan (`HIGH,CRITICAL`).

- `make docker-ci`
  - Zero-setup composite check (closest local equivalent to CI gates):
    - `mod-check`
    - `guardrails-check`
    - `skills-check`
    - `fmt-check`
    - `lint`
    - `test`
    - `test-race`
    - `test-cover`
    - `test-integration` (`REQUIRE_DOCKER=1`)
    - `openapi-check`
    - `go-security`
    - `migration-validate`
    - `container-security`
  - If `BASE_REF` and `HEAD_REF` are provided, also runs docs drift check.

### CI policy helper checks

- `make guardrails-check`
  - Runs: `scripts/ci/required-guardrails-check.sh`

- `make docs-drift-check BASE_REF=<base_sha> HEAD_REF=<head_sha>`
  - Runs: `scripts/ci/docs-drift-check.sh`

- `make migration-validate MIGRATION_DSN=<postgres_dsn>`
  - Runs `golang-migrate` against `env/migrations`:
    - apply all up migrations
    - run `down 1`
    - run `up 1`

### Skills distribution and sync

- `make skills-sync`
  - Runs: `scripts/dev/sync-skills.sh`
  - Purpose: sync provider-specific skill directories from canonical source `skills/`.
  - Mirrors:
    - `.agents/skills/`
    - `.claude/skills/`
    - `.gemini/skills/`
    - `.github/skills/`
    - `.cursor/skills/`
    - `.opencode/skills/`
  - Note: `docs/skills/` stores documentation only.

- `make skills-check`
  - Runs: `scripts/dev/sync-skills.sh --check`
  - Purpose: validate mirror sync with source `skills/`.

### Run and build

- `make run`
  - Runs: `go run ./cmd/service` with `.env` auto-loaded when the file exists.

- `make build`
  - Builds static binary:
    - output: `bin/service`
    - flags: `CGO_ENABLED=0`, `-trimpath`, `-ldflags='-s -w'`

### Container and local environment

- `make docker-build`
  - Runs: `docker build -f build/docker/Dockerfile -t service:local .`

- `make docker-run`
  - Runs: `docker run --rm -p 8080:8080 --env-file .env service:local`

- `make compose-up`
  - Runs: `docker compose -f env/docker-compose.yml up -d`

- `make compose-down`
  - Runs: `docker compose -f env/docker-compose.yml down -v`

## Recommended Local Workflows

### First run after clone (native)

1. `make setup-native`
2. `make init-module CODEOWNER=@your-org/your-team`
3. `make gh-protect BRANCH=main`
4. `make ci-local`

### First run after clone (zero-setup)

1. `make setup-docker`
2. `make docker-init-module CODEOWNER=@your-org/your-team`
3. `make gh-protect BRANCH=main`
4. `make docker-ci`

### Feature implementation (native)

1. `make fmt`
2. `make test`
3. `make lint`
4. If API contract changed: `make openapi-check`
5. If integration behavior changed: `make test-integration`

### Feature implementation (zero-setup)

1. `make docker-fmt-check`
2. `make docker-test`
3. `make docker-lint`
4. If API contract changed: `make docker-openapi-check`
5. If integration behavior changed: `make docker-test-integration`

## CI Mapping

Main CI workflow: `.github/workflows/ci.yml`

Local commands map directly to CI jobs:
- `make mod-check` + `make guardrails-check` + `make skills-check` + `make fmt-check` + `make docs-drift-check` -> `repo-integrity`
- `make lint` -> `lint`
- `make openapi-check` -> `openapi-contract`
- `BASE_OPENAPI=... make openapi-breaking` -> `openapi-breaking` (PR only)
- `make test` -> `test`
- `make test-race` -> `test-race`
- `make test-cover` -> `test-coverage`
- `REQUIRE_DOCKER=1 make test-integration` -> `test-integration`
- `make migration-validate` -> `migration-validate` (only when migrations changed)
- `make go-security` and Trivy image scan -> `go-security`, `container-security`

Zero-setup wrappers:
- `make docker-ci` runs a near-parity local CI baseline without local Go/Node installs.
- `make docker-openapi-check`, `make docker-go-security`, `make docker-test-*`, and `make docker-container-security` mirror native/CI checks.

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
