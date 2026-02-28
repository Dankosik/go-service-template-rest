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
- `GET /health/live`, `GET /health/ready`, `GET /api/v1/ping`, `GET /metrics`
- baseline HTTP timeouts and graceful shutdown
- optional Postgres readiness probe (via `POSTGRES_DSN`)
- OpenAPI workflow: codegen (`oapi-codegen`) + lint + validate + breaking check
- Docker multi-stage + distroless runtime
- CI: unit tests + race detector + coverage artifact + integration tests + lint + OpenAPI gates + security gates
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

1. Start local Postgres (optional):

```bash
make compose-up
```

2. Copy the env template and adjust values if needed:

```bash
cp env/.env.example .env
```

3. Run the service:

```bash
set -a
source .env
set +a
go run ./cmd/service
```

If Go is not installed locally, you can build the container:

```bash
make docker-build
```

## Endpoints

- `GET /api/v1/ping` -> `pong`
- `GET /health/live` -> `ok`
- `GET /health/ready` -> `ok` or `503 not ready`
- `GET /metrics` -> Prometheus metrics

## Main Commands

```bash
make fmt
make test
make test-race
make test-cover
make test-integration
make lint
make openapi-generate
make openapi-lint
make openapi-validate
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

## API Contracts

- OpenAPI: `api/openapi/service.yaml`
- Protobuf (optional): `api/proto/service/v1/service.proto`

### OpenAPI codegen

Go bindings are generated with `oapi-codegen`:

```bash
make openapi-generate
```

The generation entrypoint is in `internal/api/doc.go` (`go:generate`), and the config file is `internal/api/oapi-codegen.yaml`.

## Migrations

SQL migrations are stored in `env/migrations`.
The migration tool (`migrate`/`goose`/`atlas`/`tern`) is chosen at the team level.

## CI Quality Gates

Workflow `.github/workflows/ci.yml` includes:

- `test`: `go test ./...`
- `test-race`: `go test -race ./...`
- `test-coverage`: `go test -covermode=atomic -coverprofile=coverage.out ./...` + publish `coverage.out` as an artifact
- `test-integration`: `go test -tags=integration ./test/...`
- `lint`: `golangci-lint`
- `openapi-contract`: generate + validate + lint OpenAPI
- `openapi-breaking` (PR): check breaking changes between base and current OpenAPI spec
- `go-security`: `govulncheck` and `gosec`
- `container-security`: Trivy scan for the Docker image
