# Go Service Template REST

AI-native Go REST template for solo developers who want coding agents that can work inside real Go constraints.

Generic AI-native repos are good at teaching agents how to spec, plan, and delegate. They are usually much weaker at teaching them how to operate inside idiomatic Go boundaries, preserve invariants, work with `context`, respect generated artifacts, reason about `chi` and `sqlc`, and ship code that survives review. `go-service-template-rest` is built around that exact gap.

This repository is for people who code with Codex, Claude Code, Cursor, Gemini CLI, and other LLM-assisted workflows, but do not want a generic process layer floating above the language. The workflow is agent-native. The instructions, skills, review surfaces, and validation loop are Go-native.

- **Orchestrator-first**: frame, delegate, synthesize, plan, implement, verify.
- **Go-native guidance**: the repository does not stop at language-agnostic workflow advice.
- **Project-scoped agents**: Codex agents live in `.codex/agents/`, Claude Code agents live in `.claude/agents/`.
- **Portable skills**: reusable workflow expertise lives in `skills/` and is mirrored to multiple agent runtimes.
- **Spec-first**: non-trivial work starts in `specs/<feature-id>/spec.md`, not in prompt spaghetti.
- **Production stack underneath**: OpenAPI-first HTTP, PostgreSQL, `sqlc`, observability, tests, and CI gates are already wired.

## Why This Template Exists

Most AI-native coding today is solo. Most generic AI-native repos intentionally stay technology-independent. Most Go templates still stop at folder layout, Docker files, and a `Makefile`. That combination leaves a real hole:

- the workflow knows how to spec and delegate, but not how to reason in Go;
- the stack knows how to compile, but not how to guide an agent through non-trivial changes;
- the repo has commands, but no explicit ownership model for research, planning, implementation, review, and validation.

This template is built from the opposite assumption: if you want agents to be useful in a Go backend, the workflow and the language need to be wired together on purpose.

That is why this repository is opinionated in four places:

1. **The workflow is explicit.**
   Non-trivial work starts with framing, research, synthesis, planning, implementation, review, and validation. The loop is visible, not implied.
2. **The specialists are real.**
   Subagents have narrow ownership areas like API, domain, data, reliability, performance, and security. They are not generic “helper” personas.
3. **The skills are Go-native.**
   The skill library does not stop at abstract design advice. It covers Go architecture, routing, DB/cache contracts, invariants, reliability, security, review, debugging, testing, and verification.
4. **The backend substrate is real.**
   OpenAPI, `chi`, PostgreSQL, `sqlc`, observability, tests, and CI gates are already in the template, so the workflow lands on an actual service baseline.

If you want a Go backend template that feels natural inside Codex or Claude Code and still respects how Go services are actually built, this repository is designed for that use case.

## How This Template Solves The Problem

The fix is not a single block of text. It shapes the whole repository:

- **Specs before edits**: `specs/<feature-id>/spec.md` is where decisions, assumptions, implementation steps, and validation evidence live.
- **Go-aware subagents**: the agent portfolio is organized around real backend concerns instead of generic brainstorming personas.
- **Go-native skills**: the skill library gives the orchestrator and subagents concrete playbooks for Go design, implementation, review, and verification.
- **Verification as a first-class rule**: “done” is tied to fresh command evidence, not to confident prose from an LLM.
- **A serious service template underneath**: once the workflow moves into implementation, the repo already has OpenAPI-first HTTP, PostgreSQL, `sqlc`, telemetry, and CI guardrails.

## Workflow First

This repository treats delivery as an explicit loop, not as a single long chat and not as process theater:

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

This repository distinguishes between two different things:

- **Subagents** are read-only specialists you fan out to for focused research or review.
- **Skills** are portable workflow playbooks loaded on demand by the orchestrator or a subagent.

The repository ships with project-scoped, read-only subagents for focused reasoning and review.

| Agent | Owns | Use when | Returns |
|---|---|---|---|
| `architecture-agent` | boundaries, ownership, interaction style, failure-domain shape | a feature or refactor may change module or service shape | boundary call, interaction recommendation, handoffs |
| `api-agent` | client-visible contract behavior and transport semantics | endpoints, statuses, errors, idempotency, or async acknowledgment change | contract recommendation, compatibility notes |
| `concurrency-agent` | goroutine, channel, cancellation, and shutdown correctness | a diff touches worker pools, goroutines, shared state, or race-prone code | concurrency findings, validation gaps |
| `data-agent` | source of truth, schema evolution, transaction and cache rules | schema, query, migration, or cache behavior changes | data contract, rollout implications |
| `delivery-agent` | CI/CD gates, rollout policy, runtime hardening, release trust | release controls, deployment policy, or platform constraints change | delivery policy, gating recommendations |
| `design-integrator-agent` | cross-domain reconciliation and simplification | multiple specialist outputs conflict or the design feels over-layered | integrated path, contradictions, reopen conditions |
| `distributed-agent` | cross-service consistency, outbox/inbox, replay, reconciliation | the workflow crosses service boundaries or depends on eventual consistency | flow model, recovery stance |
| `domain-agent` | business invariants, state transitions, acceptance semantics | behavior changes touch lifecycle, rules, duplicates, or forbidden paths | invariant set, corner cases, handoffs |
| `performance-agent` | performance budgets, bottleneck hypotheses, proof strategy | the change is hot-path sensitive or justified mainly by speed | performance stance, proof obligations |
| `qa-agent` | test obligations, proving levels, validation readiness | a non-trivial behavior change needs a real regression plan | scenario matrix, validation strategy |
| `quality-agent` | idiomatic Go review and simplification | the diff feels noisy, over-abstracted, or hard to maintain | maintainability findings, cleanup guidance |
| `reliability-agent` | timeouts, retries, overload, startup, shutdown, degradation | failure behavior, degraded mode, or lifecycle semantics change | reliability contract, residual risks |
| `security-agent` | trust boundaries, auth, tenant isolation, abuse resistance | changed paths handle untrusted input or cross security boundaries | threat/control map, verification expectations |

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

## Skill Library

`skills/` is the canonical repository skill set. These skills are procedural building blocks, not autonomous owners of the workflow.

### Framing, Planning, Implementation, And Verification

| Skill | What it does | Load when |
|---|---|---|
| `spec-first-brainstorming` | turns a raw request into scope, constraints, assumptions, and design-readiness | the task is still fuzzy and needs framing before design |
| `go-coder-plan-spec` | turns approved decisions into atomic coding steps, checkpoints, and evidence expectations | planning is complete enough to prepare implementation, but coding has not started |
| `go-coder` | implements approved Go changes without semantic drift | the implementation plan is explicit and code work is next |
| `go-qa-tester` | writes deterministic Go tests from approved test obligations | test code itself needs to be added or upgraded |
| `go-systematic-debugging` | drives root-cause-first debugging with reproducible evidence | a bug, flaky test, build failure, or incident needs diagnosis |
| `go-verification-before-completion` | maps completion claims to fresh command evidence | you are about to say “fixed”, “ready”, or “done” |

### System Design And Control Surfaces

| Skill | Focus | Load when |
|---|---|---|
| `go-architect-spec` | service boundaries, ownership, sync vs async interaction style | system shape or module ownership may change |
| `go-design-spec` | integrated pre-coding design pass across domains | the draft design feels contradictory, layered, or overly complex |
| `go-devops-spec` | CI/CD policy, rollout controls, runtime hardening, release trust | delivery or release behavior is part of the change |
| `go-observability-engineer-spec` | logs, metrics, traces, correlation, telemetry cost | observability behavior needs an explicit contract |
| `go-performance-spec` | latency, throughput, contention, benchmark strategy | performance budgets or hot paths drive the design |
| `go-reliability-spec` | timeouts, retries, degradation, lifecycle behavior | failure handling or operational resilience changes |
| `go-security-spec` | trust boundaries, auth, tenant isolation, abuse resistance | the change touches security-critical surfaces |
| `go-qa-tester-spec` | test levels, scenario coverage, proof strategy | you need an explicit verification plan before coding |

### API, Routing, Domain, Data, And Distributed Semantics

| Skill | Focus | Load when |
|---|---|---|
| `api-contract-designer-spec` | resources, methods, statuses, errors, idempotency, async contracts | client-visible API behavior is changing |
| `go-chi-spec` | chi router topology, middleware ordering, fallback and CORS semantics | routing shape or HTTP middleware policy changes |
| `go-data-architect-spec` | source of truth, schema ownership, migration and rollback shape | schema or persistence model changes |
| `go-db-cache-spec` | query discipline, transaction rules, cache strategy and staleness | runtime DB or cache behavior needs an explicit contract |
| `go-domain-invariant-spec` | business invariants, state transitions, acceptance rules | lifecycle or core domain behavior changes |
| `go-distributed-architect-spec` | saga shape, outbox/inbox, replay safety, reconciliation | a flow crosses service boundaries or depends on eventual consistency |

### Review Skills

| Skill | Focus | Load when |
|---|---|---|
| `go-design-review` | architecture alignment, boundary integrity, accidental complexity | a diff may hide broader design drift |
| `go-chi-review` | router ownership, middleware order, HTTP fallback semantics | chi routing or transport behavior changed |
| `go-db-cache-review` | SQL safety, transaction scope, cache correctness, fallback risk | DB or cache code changed |
| `go-domain-invariant-review` | business-invariant preservation and side-effect safety | behavior changes carry semantic risk |
| `go-idiomatic-review` | idiomatic Go, error handling, context flow, naming | you want merge-risk review on Go code quality |
| `go-language-simplifier-review` | lower cognitive complexity and cleaner control flow | the code works but feels noisy or over-abstracted |
| `go-concurrency-review` | goroutines, channels, cancellation, shutdown safety | concurrent behavior changed or races are suspected |
| `go-performance-review` | hot-path regression, allocation and contention risk | performance is a review concern |
| `go-qa-review` | coverage quality, assertion strength, determinism | review depends on test quality and proof strength |
| `go-reliability-review` | retries, backpressure, startup, shutdown, degraded mode | failure-path behavior changed |
| `go-security-review` | authz, isolation, injection/SSRF, secret handling | changed paths accept untrusted input or cross trust boundaries |

### Skill Mirrors Across Runtimes

These repository-native skills are mirrored to multiple runtimes so the workflow stays portable:

- `.agents/skills`
- `.claude/skills`
- `.cursor/skills`
- `.gemini/skills`
- `.github/skills`
- `.opencode/skills`

The source of truth stays in `skills/`, so you do not have to hand-maintain separate skill instructions per tool.

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
