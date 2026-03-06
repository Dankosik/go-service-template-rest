# Go Service Template REST

Production-ready **Go REST API template** for building **OpenAPI-first**, **PostgreSQL-backed** services with a **spec-first** and **agent-centric** workflow.

`go-service-template-rest` is a Go microservice template for developers who want more than boilerplate. It gives you a working service layout, an OpenAPI source of truth, SQL-first data access with `sqlc`, local and CI quality gates, and explicit repository rules for AI coding workflows.

If you are looking for a **Go REST API template**, **Go microservice template**, **OpenAPI Go starter**, or **PostgreSQL + sqlc Go backend template**, this repository is designed for that use case.

## Why Use This Template

- Start a Go backend service with a clear module layout instead of inventing one from scratch.
- Keep the REST contract in `api/openapi/service.yaml` and generate runtime artifacts from it.
- Use SQL-first PostgreSQL access with `sqlc` instead of ad hoc query code.
- Keep local development, CI, and container delivery aligned through shared commands and guardrails.
- Give humans and AI agents the same repository contract through `AGENTS.md`, `spec.md`, and mirrored skills.

## What You Get

- **OpenAPI-first HTTP service** with generated bindings in `internal/api`.
- **Chi-based routing** and HTTP runtime adapters in `internal/infra/http`.
- **PostgreSQL + pgx + sqlc** for typed SQL access.
- **Config loading** with `koanf`.
- **Metrics and tracing** with Prometheus and OpenTelemetry.
- **Deterministic tests** with unit, race, coverage, and integration paths.
- **Security and delivery guardrails** for linting, vulnerability checks, secrets scanning, image scanning, SBOM generation, and signed container release flows.

## Why This Repository Is Spec-First And Agent-Centric

This template treats implementation as a controlled engineering workflow, not as a free-form coding session.

- **Spec-first** means non-trivial work starts with `specs/<feature-id>/spec.md`, where decisions, scope, assumptions, implementation steps, validation, and outcome are recorded.
- **Agent-centric** means the repository is built for orchestrated AI workflows, not one-off prompts. The main flow stays with the orchestrator, while subagents handle narrow read-only research or review tasks.
- **Planning before implementation** is a repository rule, not a suggestion. `AGENTS.md` and `docs/spec-first-workflow.md` require an explicit implementation plan before code changes start.
- **Skills are on-demand tools**, mirrored to multiple runtimes, instead of being the main workflow themselves.

Why this matters in practice:

- fewer undocumented decisions,
- better reviewability,
- easier reuse of validated research,
- safer AI-assisted changes,
- less drift between human and agent workflows.

Read the workflow contract in [AGENTS.md](AGENTS.md) and the supporting process document in [docs/spec-first-workflow.md](docs/spec-first-workflow.md).

## Quick Start

```bash
make bootstrap
make template-init   # run this when you create a new repo from the template
make check
make run
```

Typical next steps:

1. Copy `env/.env.example` to `.env` if `make bootstrap` did not already do it.
2. Run `make template-init` after cloning into a new service repository to rewire module path, CODEOWNERS, and skill mirrors.
3. Use `make check-full` before larger changes or before opening a PR.

## Repository Layout

- `cmd/service` - service entrypoint and bootstrap lifecycle orchestration.
- `internal/app` - use-case layer.
- `internal/domain` - domain contracts and types.
- `internal/infra` - HTTP, Postgres, telemetry, and other infrastructure adapters.
- `api/openapi/service.yaml` - REST API source of truth.
- `internal/api` - generated OpenAPI artifacts.
- `env/migrations` - SQL migrations for the local PostgreSQL environment.
- `internal/infra/postgres/sqlcgen` - generated `sqlc` artifacts.
- `specs/` - spec-first decision records and implementation history for non-trivial work.
- `skills/` - canonical skill definitions mirrored into agent runtime directories.

More detail: [docs/project-structure-and-module-organization.md](docs/project-structure-and-module-organization.md)

## Quality Gates And CI/CD

This template ships with a production-oriented quality baseline instead of a bare `go test`.

Local entry points:

- `make check` - quick local checks.
- `make check-full` - CI-like verification.
- `make ci-local` - native CI-style flow.
- `make docker-ci` - Docker-based CI-style flow.

Repository and CI guardrails include:

- formatting and module integrity checks,
- `golangci-lint`,
- unit tests, race tests, and coverage thresholds,
- OpenAPI generation drift, validation, lint, and breaking-change checks,
- `sqlc` generation drift checks,
- docs and skills mirror drift checks,
- `govulncheck`, `gosec`, and `gitleaks`,
- container image scanning with Trivy,
- GHCR publishing, CycloneDX SBOM generation, and Cosign signing in release flows.

See `.github/workflows/` and `Makefile` for the exact pipeline steps.

## Technology Stack

Core stack:

- Go `1.26`
- `chi` for HTTP routing
- `kin-openapi` and `oapi-codegen` for contract-first API work
- PostgreSQL `17`, `pgx/v5`, and `sqlc` for SQL-first data access
- `koanf` for configuration
- Prometheus and OpenTelemetry for observability
- `testcontainers-go`, `go.uber.org/mock`, and `goleak` for testing
- Docker multi-stage builds and distroless runtime images
- GitHub Actions for CI, nightly checks, and CD

For the full dependency graph, see [`go.mod`](go.mod) and [`go.sum`](go.sum).

## Core Commands

```bash
make help
make check
make check-full
make ci-local
make docker-ci
make openapi-check
make sqlc-check
make test-integration
make gh-protect BRANCH=main
```

## AI Tooling And Skills

The repository is prepared for multi-tool AI workflows:

- [`AGENTS.md`](AGENTS.md) defines the repository contract for orchestrator/subagent-first execution.
- `skills/` is the canonical source of skill content.
- Skills are mirrored to `.agents/skills`, `.claude/skills`, `.cursor/skills`, `.gemini/skills`, `.github/skills`, and `.opencode/skills`.
- `CLAUDE.md` keeps Claude-facing instructions aligned with `AGENTS.md`.

This makes the template usable for AI-assisted development across Codex, Claude Code, Cursor, Gemini, GitHub, and similar agent runtimes without maintaining separate workflow definitions per tool.
