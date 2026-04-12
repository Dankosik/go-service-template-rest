# Repo Profile

Use this file as the always-on background layer for this skill.
It captures only durable facts that materially improve downstream prompts.

## What This Repository Is
- This repo is an AI-native Go REST service template for coding-agent workflows.
- It is not a business-specific product repo with one fixed domain model.
- The sample service is intentionally thin; do not overfit prompts to `ping` or `ping_history` unless the request actually points there.

## Core Stack
- Go `1.26.1`
- `chi` for HTTP routing
- OpenAPI-first contract in `api/openapi/service.yaml`
- generated strict server bindings in `internal/api`
- PostgreSQL with `pgx` and `sqlc`
- config loading in `internal/config`
- OpenTelemetry + Prometheus-style telemetry in `internal/infra/telemetry`
- Docker-based zero-setup flows alongside native Go workflows

## Architecture Map
- `cmd/service/internal/bootstrap/`
  - startup, shutdown, dependency checks, lifecycle wiring
- `internal/app/`
  - use-case layer
- `internal/domain/`
  - small domain contracts
- `internal/infra/http/`
  - transport, router, middleware, HTTP policy, contract tests
- `internal/infra/postgres/`
  - Postgres adapters and generated sqlc layer
- `internal/infra/telemetry/`
  - metrics and tracing
- `internal/config/`
  - config loading and validation
- `api/openapi/`
  - REST source of truth
- `internal/api/`
  - generated OpenAPI bindings
- `env/migrations/`
  - SQL migrations
- `test/`
  - integration tests under the `integration` tag
- `specs/`
  - spec-first work artifacts
- `.agents/skills/`
  - canonical local skill source

## Workflow Facts That Often Matter
- Non-trivial work is orchestrator-first and spec-first.
- Subagents are read-only research/review lanes.
- Planning should be explicit before coding.
- Validation claims should be backed by fresh commands.
- Generated artifacts are first-class and drift-checked.

## Commands Worth Mentioning When Relevant
- Quick local baseline: `make check`
- Full CI-like baseline: `make check-full`
- Unit tests: `make test`
- Race detector: `make test-race`
- Lint: `make lint`
- OpenAPI verification: `make openapi-check`
- sqlc drift check: `make sqlc-check`
- Integration tests: `make test-integration`
- Security scans: `make go-security`, `make secrets-scan`
- Migration rehearsal: `make migration-validate`
- Skill mirrors: `make skills-sync`, `make skills-check`

## Prompting Rules For This Skill
- Inject only the repo facts that help the current task.
- Prefer exact paths and source-of-truth files over broad directory descriptions.
- Mention template/bootstrap caveats only when the task is actually about repo initialization or module path setup.
