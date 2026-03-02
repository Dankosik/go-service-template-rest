# go-service-template-rest

A production-ready Go REST microservice template with a deeply curated [AGENTS.md](AGENTS.md).

This repository is built first for:
- vibe coders who develop with AI coding agents;
- developers who have never written Go by hand but want a safe production-oriented starting point.

In short: this is a Go microservice starter template optimized for AI-assisted development, beginner-friendly onboarding, and production-ready engineering defaults.

## Why This Template Exists

- `AGENTS.md` is the main feature of this repository, not an extra file.
- It gives coding agents explicit rules for idiomatic Go, architecture boundaries, testing, security, and review quality.
- It reduces guesswork for beginners and keeps generated code consistent with real Go service practices.

## Best For

- Go beginners building their first backend service
- AI-first or vibe coding workflows (Codex, ChatGPT, Claude, Copilot)
- Teams that want one repeatable Go microservice baseline with strict engineering guardrails

## What's Included

- `cmd/service` as a thin entry point
- `internal` for private logic (`app/domain/infra/config`)
- configuration via environment variables
- structured JSON logs via `log/slog`
- request correlation via `X-Request-ID` (`request_id`/`trace_id`/`span_id` in request logs)
- `GET /health/live`, `GET /health/ready`, `GET /api/v1/ping`, `GET /metrics`
- baseline HTTP timeouts and graceful shutdown
- optional Postgres readiness probe (via `POSTGRES_DSN`)
- OpenAPI workflow: codegen (`oapi-codegen`) + lint + validate + breaking check
- Docker multi-stage + distroless runtime
- repository guardrails: `.editorconfig`, `.gitattributes`, `CODEOWNERS`, PR template, `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE`
- CI: integrity gates (mod/fmt/docs drift), tests (unit/race/coverage/integration), OpenAPI contract gates, migration validation, security gates
- nightly reliability workflow with repeated test runs and full security/contract checks
- CD: GHCR image publishing with Trivy scan, CycloneDX SBOM, cosign keyless signing, and provenance attestation
- Dependabot for `gomod` and GitHub Actions

## Structure

```text
.
├── api/
├── build/
├── cmd/
├── internal/
├── env/
├── scripts/
├── test/
├── .github/workflows/
├── Makefile
├── go.mod
└── README.md
```

## Quick Start

1. Bootstrap local tools and run environment checks:

```bash
make setup
```

2. Initialize module path once after clone:

```bash
make init-module MODULE=github.com/your-org/your-service
```

3. Copy the env template and adjust values if needed:

```bash
cp env/.env.example .env
```

4. Run the service:

```bash
set -a
source .env
set +a
go run ./cmd/service
```

By default `POSTGRES_DSN` is empty, so the service starts without a Postgres readiness probe.

5. Optional: enable local Postgres readiness probe:

```bash
make compose-up
```

Set `POSTGRES_DSN` in `.env`, then restart the service.

If Go is not installed locally, you can build the container:

```bash
make docker-build
```

## Endpoints

- `GET /api/v1/ping` -> `pong`
- `GET /health/live` -> `ok`
- `GET /health/ready` -> `ok` or `503 not ready`
- `GET /metrics` -> Prometheus metrics
  - includes `http_requests_total{method,route,status_code}`
  - includes `http_request_duration_seconds{method,route,status_code}`

## Main Commands

```bash
make fmt
make setup
make doctor
make init-module MODULE=github.com/your-org/your-service
make mod-check
make guardrails-check
make fmt-check
make test
make test-race
make test-cover
make test-integration
make lint
make openapi-generate
make openapi-lint
make openapi-validate
make docs-drift-check BASE_REF=<base_sha> HEAD_REF=<head_sha>
make migration-validate MIGRATION_DSN=<postgres_dsn>
make build
make run
make docker-build
```

Check OpenAPI breaking changes locally:

```bash
BASE_OPENAPI=/path/to/base-service.yaml make openapi-breaking
```

`make test-integration` runs tests with the `integration` tag and requires Docker.

## Configuration

See `env/.env.example`:

- `APP_ENV`
- `HTTP_ADDR`
- `HTTP_SHUTDOWN_TIMEOUT`
- `HTTP_READ_HEADER_TIMEOUT`
- `HTTP_READ_TIMEOUT`
- `HTTP_WRITE_TIMEOUT`
- `HTTP_IDLE_TIMEOUT`
- `HTTP_MAX_HEADER_BYTES`
- `LOG_LEVEL`
- `POSTGRES_DSN`
  - empty by default; when set, enables Postgres readiness check on startup

Default HTTP timeout profile for this template:
- `HTTP_READ_HEADER_TIMEOUT=5s`
- `HTTP_READ_TIMEOUT=5s`
- `HTTP_WRITE_TIMEOUT=10s`
- `HTTP_IDLE_TIMEOUT=60s`
- `HTTP_SHUTDOWN_TIMEOUT=10s`

These values are safe defaults for typical JSON APIs. If you add streaming/long-running responses, tune timeouts explicitly for that path or use a dedicated server profile.

## API Contracts

- OpenAPI: `api/openapi/service.yaml`
- Protobuf (optional): `api/proto/service/v1/service.proto`

### OpenAPI codegen

Go bindings are generated with `oapi-codegen`:

```bash
make openapi-generate
make openapi-drift-check
make openapi-runtime-contract-check
```

The generation entrypoint is in `internal/api/doc.go` (`go:generate`), and the config file is `internal/api/oapi-codegen.yaml`.

## Migrations

SQL migrations are stored in `env/migrations`.
CI migration rehearsal command:

```bash
make migration-validate MIGRATION_DSN='postgres://app:app@localhost:5432/app?sslmode=disable'
```

## CI Quality Gates

Workflow `.github/workflows/ci.yml` includes:

- `repo-integrity`: `go mod tidy -diff`, `go mod verify`, required guardrails check, format drift check, docs drift check
- `lint`: `golangci-lint`
- `openapi-contract`: generate + codegen drift check + runtime contract check + validate + lint OpenAPI
- `openapi-breaking` (PR): check breaking changes between base and current OpenAPI spec
- `test`: `go test ./...`
- `test-race`: `go test -race ./...`
- `test-coverage`: `go test -covermode=atomic -coverprofile=coverage.out ./...` + publish `coverage.out` as an artifact
- `test-integration`: `go test -tags=integration ./test/...`
- `migration-validate` (conditional): rehearses SQL migrations on ephemeral Postgres when `env/migrations/**` changes
- `go-security`: `govulncheck` and `gosec`
- `container-security`: Trivy scan for the Docker image

Nightly reliability workflow `.github/workflows/nightly.yml` runs extended checks (repeat test runs, race/integration, OpenAPI, security, and container scan).

## CD Pipeline

Workflow `.github/workflows/cd.yml` includes:

- `publish-main`: after successful `ci` on `main`, builds and pushes GHCR image tags `main` and `sha-*`, scans with Trivy, uploads CycloneDX SBOM, signs image (cosign keyless), and publishes provenance attestation.
- `release-preflight`: on `v*` tag push, reruns quality and security gates before artifact publish.
- `publish-release`: on `v*` tag push, runs after `release-preflight`, then builds and pushes `v*`, `latest`, and `sha-*` tags with the same security/supply-chain controls.

Repository-level enforcement checklist:
- `docs/ci-cd-production-ready.md`
