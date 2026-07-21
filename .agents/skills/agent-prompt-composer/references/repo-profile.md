# Repo Profile

Load this file only when a durable repository fact could change the handoff's owner, source-of-truth guidance, starting surface, or proof command. Skip it when the source task is already repository-grounded.

## What This Repository Is
- This repo is a harness-portable Go REST service template; it runs natively in both the Codex App and Claude Code (`docs/agent-harness.md` owns the control mapping).
- It is not a business-specific product repo with one fixed domain model.
- The sample service is intentionally thin; do not overfit prompts to `ping` or `ping_history` unless the request actually points there.

## Core Stack
- Go version and optional toolchain directive from the current `go.mod`; read them live before making version-sensitive claims.
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
  - canonical local skill source (Claude Code reads it via per-skill symlinks in `.claude/skills/`; `make claude-skills-sync` resyncs)
- `.codex/agents/` and `.claude/agents/`
  - the same specialist subagent roles for the Codex App and Claude Code

## Workflow Facts That Often Matter
- Non-trivial work is orchestrator-first and spec-first.
- Subagents are read-only research/review lanes.
- Planning should be explicit before coding.
- Validation claims need current claim-scoped evidence; exact immutable-tree proof remains reusable when its relevant preconditions are unchanged.
- Generated artifacts are first-class and drift-checked.

## Commands Worth Mentioning When Relevant
- Everyday loop: focused package proof plus `make lint-fast` when useful; `make check` is a broad local baseline
- Full CI-like baseline: `make check-full`
- Unit tests: `make test`
- Race detector: `make test-race`
- Lint: `make lint`
- OpenAPI verification: `make openapi-check`
- sqlc drift check: `make sqlc-check`
- Integration tests: `make test-integration`
- Security scans: `make go-security`, `make secret-scan`
- Migration rehearsal: `make migration-validate`
- Workflow and skill instructions: `git diff --check`

## Context Rules For This Skill
- Do not inject this profile as a standard project summary.
- Inject only the repo facts that help the current task context.
- Prefer exact paths and source-of-truth files over broad directory descriptions.
- Mention template/bootstrap caveats only when the task is actually about repo initialization or module path setup.
