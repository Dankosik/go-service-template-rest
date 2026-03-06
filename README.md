# Go Service Template REST

AI-native Go REST template for teams shipping web backends with Codex, Claude Code, Cursor, Gemini CLI, and other LLM-assisted workflows.

`go-service-template-rest` is not just a Go starter. It is a repository contract for agentic delivery: the orchestrator owns the task, read-only subagents handle focused research and review, `spec.md` records the decisions, and the service stack underneath is ready for real backend work.

- **Orchestrator-first**: frame, delegate, synthesize, plan, implement, verify.
- **Project-scoped agents**: Codex agents live in `.codex/agents/`, Claude Code agents live in `.claude/agents/`.
- **Spec-first**: non-trivial work starts in `specs/<feature-id>/spec.md`, not in prompt spaghetti.
- **Production stack underneath**: OpenAPI-first HTTP, PostgreSQL, `sqlc`, observability, tests, and CI gates are already wired.

## Why This Template Exists

Most Go templates stop at folder layout, Docker files, and a `Makefile`. That is not enough when humans and agents are both writing code in the same repository.

This template is built for teams that want:

- a backend starter that works with LLM-assisted coding instead of fighting it;
- an explicit workflow for research, planning, implementation, review, and validation;
- project-scoped agents with clear ownership instead of one giant all-purpose assistant;
- a serious Go REST baseline once the workflow moves into implementation.

If you want a Go backend template that feels natural inside Codex or Claude Code, this repository is designed for that use case.

## Workflow First

This repository treats delivery as an explicit loop, not as a single long chat:

```text
intake -> research -> synthesis -> planning -> implementation -> review -> validation
```

- `intake`: frame the change, scope it, and record assumptions.
- `research`: keep simple work local or fan out to read-only subagents.
- `synthesis`: compare specialist output and keep final decisions with the orchestrator.
- `planning`: write the implementation plan before code changes start.
- `implementation`: change the service in the main flow, not inside research agents.
- `review`: run targeted review agents only where the risk justifies them.
- `validation`: do not claim "done" without fresh command evidence.

The full contract lives in [AGENTS.md](AGENTS.md) and the supporting workflow doc lives in [docs/spec-first-workflow.md](docs/spec-first-workflow.md).

## Agent Portfolio

The repository ships with project-scoped, read-only subagents for focused reasoning and review.

- `architecture-agent`, `api-agent`, `domain-agent`, `data-agent`, `distributed-agent`: use these when the shape of the system, contract, invariants, storage, or cross-service workflow is changing.
- `design-integrator-agent`: use this when multiple specialist outputs need reconciliation into one coherent path.
- `security-agent`, `reliability-agent`, `performance-agent`, `concurrency-agent`: use these when the main risk is trust boundaries, failure behavior, hot paths, or concurrent correctness.
- `quality-agent`, `qa-agent`, `delivery-agent`: use these for maintainability, proof obligations, and release or CI policy.

All of these agents stay advisory and read-only. Final decisions always stay with the orchestrator in the main flow.

### How They Are Called

**Codex**

Codex loads the project agent registry from [.codex/config.toml](.codex/config.toml). In practice, you ask the orchestrator to fan out by agent name:

```text
Use `architecture-agent` and `api-agent` to evaluate the new async export flow.
Synthesize the result into `specs/export-flow/spec.md`.
Do not start coding until the implementation plan is explicit.
```

**Claude Code**

Claude Code project agents live in [.claude/agents](.claude/agents). You can select them directly with `--agent`:

```bash
claude -p --agent architecture-agent -- "Review boundary ownership for adding async webhook retries in this repository."
claude -p --agent qa-agent -- "List the minimum regression obligations for changing the order status flow."
```

### Common Fan-Out Patterns

- New endpoint or contract change: `api-agent` + `domain-agent` + `qa-agent`
- Storage, cache, or migration change: `data-agent` + `reliability-agent`
- Cross-service or async workflow: `architecture-agent` + `distributed-agent` + `security-agent`
- Pre-merge cleanup on a larger diff: `quality-agent` + the domain reviewer that matches the risk

## This Is An Orchestrator Project

The repository is designed so the main agent acts like an orchestrator, not like a single monolithic coder.

- The orchestrator owns framing, scope, synthesis, planning, implementation, reconciliation, and validation.
- Subagents own narrow research or review tracks only.
- Skills are tools, not the workflow itself.
- `spec.md` is the canonical decisions artifact.
- `research/*.md` is optional supporting evidence, not a competing source of truth.

For non-trivial work, the artifact shape is intentionally simple:

```text
specs/<feature-id>/
  spec.md
  research/
```

If you want the short version: plan first, delegate only where it reduces uncertainty, keep decisions in `spec.md`, and always close with fresh validation evidence.

## Quickstart

### Human Quickstart

```bash
make bootstrap
make template-init   # run this when you create a new repo from the template
make check
make run
```

Typical next steps:

1. Copy `env/.env.example` to `.env` if `make bootstrap` did not already do it.
2. Run `make template-init` after cloning into a new service repository to rewire module path, `CODEOWNERS`, and skill mirrors.
3. Use `make check-full` before larger changes or before opening a PR.

### Agent Quickstart

1. Open the repository in Codex or Claude Code.
2. Read [AGENTS.md](AGENTS.md). Claude-facing compatibility is mirrored in [CLAUDE.md](CLAUDE.md).
3. Start with a spec-driven prompt, not with direct code generation.

Example kickoff prompt:

```text
Frame a change to add tenant-aware export jobs.
Fan out to `architecture-agent`, `data-agent`, and `qa-agent` only if needed.
Write decisions and the implementation plan to `specs/tenant-export-jobs/spec.md` before coding.
```

### Cross-Runtime Skill Mirrors

The repository also mirrors skills for other agent runtimes:

- `.agents/skills`
- `.claude/skills`
- `.cursor/skills`
- `.gemini/skills`
- `.github/skills`
- `.opencode/skills`

The skill source of truth stays in `skills/`, so you do not have to hand-maintain separate workflow instructions per tool.

## Repository Layout

- `cmd/service` - service entrypoint and bootstrap lifecycle orchestration
- `internal/app` - use-case layer
- `internal/domain` - domain contracts and types
- `internal/infra` - HTTP, Postgres, telemetry, and other infrastructure adapters
- `api/openapi/service.yaml` - REST API source of truth
- `internal/api` - generated OpenAPI artifacts
- `env/migrations` - SQL migrations for the local PostgreSQL environment
- `internal/infra/postgres/sqlcgen` - generated `sqlc` artifacts
- `specs/` - spec-first decision records and implementation history
- `skills/` - canonical skill definitions mirrored into runtime-specific directories

More detail: [docs/project-structure-and-module-organization.md](docs/project-structure-and-module-organization.md)

## Technology Stack

Workflow comes first, but this is still a serious Go backend template.

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

## Quality Gates And Verification

Local entry points:

- `make check` - quick local checks
- `make check-full` - CI-like verification
- `make ci-local` - native CI-style flow
- `make docker-ci` - Docker-based CI-style flow
- `make openapi-check` - OpenAPI generation, drift, lint, and compatibility checks
- `make sqlc-check` - generated SQL artifact drift checks
- `make test-integration` - integration tests
- `make gh-protect BRANCH=main` - branch protection setup helper

Repository and CI guardrails include:

- formatting and module integrity checks
- `golangci-lint`
- unit tests, race tests, and coverage thresholds
- OpenAPI generation drift, validation, lint, and breaking-change checks
- `sqlc` generation drift checks
- docs and skills mirror drift checks
- `govulncheck`, `gosec`, and `gitleaks`
- container image scanning with Trivy
- GHCR publishing, CycloneDX SBOM generation, and Cosign signing in release flows

See `.github/workflows/` and `Makefile` for the exact pipeline steps.
