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
- OpenTelemetry tracing baseline (`otelhttp` + W3C propagators, env-driven sampler/exporter)
- API hardening middleware (`X-Content-Type-Options: nosniff`, invalid request framing guard, request body limits)
- standardized API error payloads (`application/problem+json` for request/internal failures)
- `GET /health/live`, `GET /health/ready`, `GET /api/v1/ping`, `GET /metrics`
- baseline HTTP timeouts and graceful shutdown
- optional Postgres readiness probe (via `POSTGRES_DSN`)
- portable Agent Skills in git (`skills/` as source + provider mirrors for Codex/Claude/Cursor/Gemini/Copilot)
- OpenAPI workflow: codegen (`oapi-codegen`) + lint + validate + breaking check
- Docker multi-stage + distroless runtime (image digests pinned)
- tooling image catalog (`build/docker/tooling-images.Dockerfile`) consumed by zero-setup Docker wrappers
- repository guardrails: `.editorconfig`, `.gitattributes`, `CODEOWNERS`, PR template, `CONTRIBUTING.md`, `SECURITY.md`, `LICENSE`
- CI: integrity gates (mod/fmt/docs drift), tests (unit/race/coverage/integration), OpenAPI contract gates, migration validation, security gates (govulncheck/gosec/gitleaks)
- nightly reliability workflow with repeated test runs and full security/contract checks
- CD: GHCR image publishing with Trivy scan, CycloneDX SBOM, cosign keyless signing, and provenance attestation
- Dependabot for `gomod`, GitHub Actions, Dockerfiles, and docker-compose

## Structure

```text
.
├── api/
├── build/
├── cmd/
├── internal/
├── env/
├── skills/
├── scripts/
├── test/
├── .github/workflows/
├── Makefile
├── go.mod
└── README.md
```

## Quick Start (3 Commands)

1. Bootstrap:

```bash
make bootstrap
```

2. Run quick checks:

```bash
make check
```

3. Start the service:

```bash
make run
```

`make bootstrap` is intentionally minimal:
- creates `.env` from `env/.env.example` when missing;
- downloads Go modules (or pre-pulls Docker tooling images when Go is unavailable).

`make check` is intentionally fast for daily work:
- native mode: `fmt-check`, `lint`, `test`;
- Docker fallback (when local Go is unavailable): `docker-fmt-check`, `docker-lint`, `docker-test`.

By default `POSTGRES_DSN` is empty, so the service starts without a Postgres readiness probe.
`make run` auto-loads `.env` (if present) before starting the service.

## Full Validation (CI Parity)

```bash
make check-full
```

`make check-full` runs full CI-like validation:
- with Docker daemon: `make docker-ci`;
- without Docker daemon: `make ci-local` (docker-only checks are skipped with an explicit message).

## Template/Admin Initialization (Optional)

If you cloned this template into a new repository and want template rewiring (module path, `CODEOWNERS` placeholder replacement, skills mirror sync), run:

```bash
make template-init
```

Optional explicit modes:

```bash
make template-init-native
make template-init-docker
make template-init-strict
```

Manual overrides when needed:

```bash
make init-module CODEOWNER=@your-org/your-team
make init-module MODULE=github.com/your-org/your-service CODEOWNER=@your-org/your-team
```

Branch protection bootstrap (repo admin):

```bash
make gh-protect BRANCH=main
```

### Optional Local Postgres

```bash
make compose-up
```

Set `POSTGRES_DSN` in `.env`, then restart the service.

## Endpoints

- `GET /api/v1/ping` -> `pong`
- `GET /health/live` -> `ok`
- `GET /health/ready` -> `ok` or `503 not ready`
- `GET /metrics` -> Prometheus metrics
  - includes `http_requests_total{method,route,status_code}`
  - includes `http_request_duration_seconds{method,route,status_code}`

## Main Commands

```bash
make help
make bootstrap
make check
make check-full
make run
make ci-local
make docker-ci
make template-init
make gh-protect BRANCH=main
```

For the full command reference, see `docs/build-test-and-development-commands.md`.

## Portable Agent Skills

The repository keeps skills in git for clone-and-use workflows across multiple agent tools.

- canonical source directory: `skills/`

- runnable directories: `.agents/skills/`, `.claude/skills/`, `.gemini/skills/`, `.github/skills/`, `.cursor/skills/`, `.opencode/skills/`
- documentation-only directory: `docs/skills/` (guides/specifications, no runnable `SKILL.md`)

Details and provider matrix:
- `docs/skills/portable-agent-skills.md`

Check OpenAPI breaking changes locally:

```bash
BASE_OPENAPI=/path/to/base-service.yaml make openapi-breaking
```

`make test-integration` runs tests with the `integration` tag.
- local: Docker missing -> test is skipped;
- CI: `REQUIRE_DOCKER=1` enforces Docker presence and fails the job otherwise.

## Configuration

See `env/.env.example`:

- `APP_ENV`
- `APP_VERSION`
- `HTTP_ADDR`
- `HTTP_SHUTDOWN_TIMEOUT`
- `HTTP_READ_HEADER_TIMEOUT`
- `HTTP_READ_TIMEOUT`
- `HTTP_WRITE_TIMEOUT`
- `HTTP_IDLE_TIMEOUT`
- `HTTP_MAX_HEADER_BYTES`
- `HTTP_MAX_BODY_BYTES`
- `LOG_LEVEL`
- `OTEL_SERVICE_NAME`
- `OTEL_TRACES_SAMPLER`
- `OTEL_TRACES_SAMPLER_ARG`
- `POSTGRES_DSN`
  - empty by default; when set, enables Postgres readiness check on startup
- optional OTLP exporter settings (if traces export is needed):
  - `OTEL_EXPORTER_OTLP_ENDPOINT` or `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`
  - `OTEL_EXPORTER_OTLP_HEADERS`
  - `OTEL_EXPORTER_OTLP_PROTOCOL`

Default HTTP timeout profile for this template:
- `HTTP_READ_HEADER_TIMEOUT=5s`
- `HTTP_READ_TIMEOUT=5s`
- `HTTP_WRITE_TIMEOUT=10s`
- `HTTP_IDLE_TIMEOUT=60s`
- `HTTP_SHUTDOWN_TIMEOUT=10s`
- `HTTP_MAX_HEADER_BYTES=16384`
- `HTTP_MAX_BODY_BYTES=1048576`

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
- `test-coverage`: `make test-report COVERAGE_MIN=70.0` (`gotestsum` + race + coverage threshold + JUnit/JSON artifacts + `coverage.out`)
- `test-integration`: `REQUIRE_DOCKER=1 go test -tags=integration ./test/...`
- `migration-validate` (conditional): rehearses SQL migrations on ephemeral Postgres when `env/migrations/**` changes
- `go-security`: `govulncheck` and `gosec` (generated files excluded)
- `secret-scan`: `gitleaks` scan over repository git history
- `container-security`: Trivy scan for the Docker image

Nightly reliability workflow `.github/workflows/nightly.yml` runs extended checks (repeat test runs, fuzz smoke run, race/integration, OpenAPI, security, and container scan).

## CD Pipeline

Workflow `.github/workflows/cd.yml` includes:

- `publish-main`: after successful `ci` on `main`, builds and pushes GHCR image tags `main` and `sha-*`, scans with Trivy, uploads CycloneDX SBOM, signs image (cosign keyless), and publishes provenance attestation.
- `release-preflight`: on `v*` tag push, reruns quality and security gates before artifact publish.
- `publish-release`: on `v*` tag push, runs after `release-preflight`, then builds and pushes `v*`, `latest`, and `sha-*` tags with the same security/supply-chain controls.

Repository-level enforcement checklist:
- `docs/ci-cd-production-ready.md`

To apply branch protection automatically for a cloned repository, run:

```bash
make gh-protect BRANCH=main
```
