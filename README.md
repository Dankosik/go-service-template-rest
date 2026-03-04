# Go Service Template REST

Production-ready **Go REST API template** for **AI coding** and fast backend delivery.

`go-service-template-rest` helps Go developers (especially AI-assisted workflows) start a service with a ready architecture, OpenAPI-first contract, SQL-first data layer, and strict CI/CD quality gates.

Keywords: **Go REST API template**, **Go microservice template**, **OpenAPI Go starter**, **AI coding template**, **PostgreSQL sqlc template**.

## Why This Project

- Fast start for a Go backend service without manual boilerplate.
- One baseline for teams and AI agents (`AGENTS.md` + portable skills).
- Production-ready defaults: tests, security checks, contract checks, and container delivery.

## Who It Is For

- Go backend developers.
- AI-first workflows (Codex, ChatGPT, Claude, Copilot, etc.).
- Teams that need a reproducible microservice template.

## Quick Start

```bash
make bootstrap
make template-init   # for a new repo created from this template
make check
make run
```

## Key Directories

- `cmd/service` - composition root.
- `internal/app` - use-cases.
- `internal/domain` - domain contracts/types.
- `internal/infra` - infrastructure adapters.
- `api/openapi/service.yaml` - source of truth OpenAPI.
- `internal/api` - generated OpenAPI artifacts.
- `internal/infra/postgres/sqlcgen` - generated sqlc artifacts.

## Technologies And Libraries

Below are **all direct** project technologies/libraries (runtime + tooling).

### Runtime

- Go `1.26`
- Standard library: `net/http`, `log/slog`
- HTTP router: `github.com/go-chi/chi/v5`
- Config: `github.com/knadh/koanf/v2`, `github.com/knadh/koanf/parsers/yaml`, `github.com/knadh/koanf/providers/confmap`, `github.com/knadh/koanf/providers/rawbytes`
- OpenAPI runtime: `github.com/oapi-codegen/runtime`
- OpenAPI validation/spec work: `github.com/getkin/kin-openapi`
- PostgreSQL driver: `github.com/jackc/pgx/v5`
- Metrics: `github.com/prometheus/client_golang`
- Tracing: `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/otel/trace`, `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`, `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`
- Testing libs: `github.com/testcontainers/testcontainers-go`, `github.com/testcontainers/testcontainers-go/modules/postgres`, `go.uber.org/mock`, `go.uber.org/goleak`

### Tooling (go.mod `tool (...)`)

- `github.com/getkin/kin-openapi/cmd/validate`
- `github.com/golang-migrate/migrate/v4/cmd/migrate`
- `github.com/golangci/golangci-lint/v2/cmd/golangci-lint`
- `github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen`
- `github.com/oasdiff/oasdiff`
- `github.com/securego/gosec/v2/cmd/gosec`
- `github.com/sqlc-dev/sqlc/cmd/sqlc`
- `github.com/zricethezav/gitleaks/v8`
- `go.uber.org/mock/mockgen`
- `golang.org/x/tools/cmd/goimports`
- `golang.org/x/tools/cmd/stringer`
- `golang.org/x/vuln/cmd/govulncheck`
- `gotest.tools/gotestsum`

### Platform And Infra

- OpenAPI `3.0.3`
- PostgreSQL `17` (local compose)
- Docker multi-stage build
- Distroless runtime: `gcr.io/distroless/static-debian12:nonroot`
- Node.js `20` (OpenAPI lint via Redocly CLI)
- GitHub Actions (`ci`, `nightly`, `cd`)
- GHCR publishing
- Container scan: Trivy
- SBOM: CycloneDX
- Image signing: Cosign (keyless)
- Build provenance attestation

Full transitive dependency list: `go.mod` and `go.sum`.

## Core Commands

```bash
make help
make check
make check-full
make ci-local
make docker-ci
make openapi-check
make sqlc-check
```

## AI coding

- Agent contract: [AGENTS.md](AGENTS.md)
- Skills source: `skills/`
- Portable mirrors: `.agents/skills`, `.claude/skills`, `.cursor/skills`, `.gemini/skills`, `.github/skills`, `.opencode/skills`
