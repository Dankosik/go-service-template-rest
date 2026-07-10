# Go Service Template REST

AI-native Go REST template for solo developers who want coding agents to work inside real Go, API, data, and operational constraints without carrying a heavyweight process prompt.

The repository combines:

- outcome-first workflow guidance with risk-proportional artifacts;
- project-scoped read-only specialist agents;
- portable Go design, implementation, review, and verification skills;
- OpenAPI-first HTTP, PostgreSQL, `sqlc`, telemetry, tests, and CI gates;
- explicit repository ownership and generated-source discipline.

Before adding production feature code, use the [first production feature checklist](docs/project-structure-and-module-organization.md#first-production-feature-checklist).

## Why This Template Exists

Generic agent workflows know how to produce plans but often miss Go-specific ownership, `context`, error identity, generated sources, `chi`, `sqlc`, migrations, and operational proof. Traditional Go templates provide code and commands but little guidance for agents making non-trivial changes.

This template connects the two while keeping the prompt surface lean:

- global invariants live once in [AGENTS.md](AGENTS.md);
- the stable router is [docs/spec-first-workflow.md](docs/spec-first-workflow.md);
- phase files contain only their unique decisions;
- task artifacts exist only when another phase, actor, or session needs them;
- fresh evidence, not process completion, supports “done.”

## Workflow

```text
intake -> research -> specification -> system/integration design -> Go ownership design -> test design -> planning -> implementation and verification
```

The workflow chooses among three paths:

- `direct`: clear, small, reversible work with obvious ownership and proof;
- `structured`: the normal non-trivial case, with reviewed `spec.md` and `tasks.md` plus only the design/test artifacts that carry live decisions;
- `orchestrated`: broad, hard-to-reverse, multi-owner, evidence-heavy, explicitly multi-agent, or multi-session work.

Protected concerns such as public contracts, persisted data, security, money, concurrency/lifecycle, deployment, and cross-service ownership require explicit relevant decisions and proof. They do not automatically require full-depth work or a durable artifact in every phase.

Structured and orchestrated work evaluates every phase boundary in order. Research, design, or test design may be scoped down when its question is already closed, with a concrete reason; specification, planning, and their independent review gates remain required. One authorized request may cross several phases without collapsing their ownership or gates. An explicit boundary such as `research only`, `planning only`, `read-only`, or `docs-only` stops the work there. Review, in-scope repair, fresh re-review, validation, and closeout stay inside the owning macro phase; a next-session prompt is reserved for an intentional next macro phase or an honest blocker the current root cannot resolve.

### Artifacts

| Artifact | Use when | Owns |
| --- | --- | --- |
| `spec.md` | Required for structured/orchestrated work; optional for direct work. | Outcome, behavior, invariants, constraints, risks, proof expectations. |
| `design/` | Implementation would otherwise choose mechanism or ownership. | Contracts, source of truth, sequence/failures, data, rollout, Go package/file ownership. |
| `test-plan.md` | Proof spans meaningful scenarios or levels. | Scenario obligations and observables. |
| `tasks.md` | Required for structured/orchestrated work; direct work may plan inline. | Executable order, owners, proof, progress, completion condition. |
| `research/` | Evidence must be reused, refreshed, or audited. | Findings, limits, conflicts, decision impact. |
| `rollout.md` | Deployment/migration/backfill has a real sequence. | Operational gates, rollback/failback, observables. |
| `workflow-plan.md` | Cross-session or multi-lane resume needs a control point. | Goal, current phase, active artifacts, blocker, next action. |

Use `status: draft | ready | blocked | done` when durable status is useful. Do not create per-phase control files or parallel state fields.

## Agents And Skills

`.codex/agents/*.toml` contains project-scoped read-only specialist roles. `.agents/skills` is the canonical portable skill set. Claude, Cursor, Gemini, GitHub, and OpenCode mirrors are generated; do not hand-maintain them.

Useful commands:

```bash
make agents-sync
make agents-check
make skills-sync
make skills-check
make workflow-behavior-evals-check
```

The behavior-eval check validates the E01–E21 manifest only. Actual baseline/candidate model comparison uses `make workflow-behavior-evals` with the external adapters documented in [Workflow Behavior Evals](docs/spec-first-workflow-evals.md).

At each active macro phase, evaluate whether concrete, independent, bounded subagent lanes improve evidence or review independence. Use only useful lanes and keep sequential work local; record a local-only reason in an existing artifact or handoff instead of creating a gate file. The root owns synthesis, edits, and completion claims. Default to at most three concurrent lanes and no nested delegation.

Representative agents:

| Agent | Focus |
| --- | --- |
| `architecture-agent` | boundaries, ownership, interaction style |
| `api-agent` | client-visible contracts and HTTP semantics |
| `data-agent` | source of truth, schema, transactions, cache |
| `domain-agent` | invariants and state transitions |
| `security-agent` | trust boundaries, auth, tenant isolation, abuse |
| `reliability-agent` | timeouts, retries, overload, lifecycle |
| `qa-agent` | test obligations and validation readiness |
| `quality-agent` | idiomatic Go and structural simplification |
| `challenger-agent` | one focused plan/spec assumption challenge |

Representative workflow skills:

| Skill | Use when |
| --- | --- |
| `idea-refine` / `spec-first-brainstorming` | the outcome still needs product or engineering framing |
| `research-session` | evidence can change a decision |
| `specification-session` / `spec-document-designer` | behavior decisions need a durable record |
| `technical-design-session` / `go-design-spec` | mechanism or Go ownership is not obvious |
| `test-design-session` | proof needs a real scenario matrix |
| `planning-session` / `planning-and-task-breakdown` | implementation needs a durable ledger |
| `go-coder` / `go-qa-tester` | implement accepted Go behavior and tests |
| `go-systematic-debugging` | diagnose a bug, flake, hang, or build failure |
| `go-verification-before-completion` | map completion claims to fresh evidence |
| `workflow-status` | report status and next action from one task path |

Domain spec and review skills cover API/chi, data/cache, distributed consistency, domain invariants, security, reliability, concurrency, observability, performance, delivery, QA, and Go maintainability. Load only the skill that maps to the current decision or symptom.

## Orchestrator Model

The root agent owns framing, route selection, synthesis, implementation, and validation. Subagent and worker output is evidence, not authority.

For multi-session work, a compact task bundle may look like:

```text
specs/<feature-id>/
  workflow-plan.md   # only when resume/coordination needs it
  spec.md            # when decisions need to persist
  design/            # only when mechanism/ownership is non-obvious
  test-plan.md       # only when scenario design adds value
  tasks.md           # when execution needs a ledger
  research/          # only durable evidence
  rollout.md         # only non-trivial operational sequence
```

The short version: frame the outcome, persist only decisions another actor needs, implement from accepted behavior, and prove the changed surface.

## Quickstart

### Human Quickstart

```bash
make bootstrap
make template-init   # run when creating a new repository from the template
make check
make run
```

### Create Your Own Repository

Create an empty GitHub repository, then:

```bash
git clone https://github.com/Dankosik/go-service-template-rest.git my-service
cd my-service

git remote rename origin upstream
git remote add origin git@github.com:<your-user>/<your-repo>.git
# or: git remote add origin https://github.com/<your-user>/<your-repo>.git

make bootstrap
make template-init
make check

git add .
git commit -m "chore: initialize service from template"
git push -u origin main
```

`origin` should point to your repository and `upstream` to this template. If SSH fails, switch `origin` to HTTPS. Repositories created with GitHub's “Use this template” should still run `make bootstrap` and `make template-init`.

For production-style GitHub setup after the first push:

```bash
gh auth login
make gh-protect BRANCH=main
```

### Agent Quickstart

1. Read [AGENTS.md](AGENTS.md).
2. For non-trivial work, open [docs/spec-first-workflow.md](docs/spec-first-workflow.md) and only the current phase file.
3. Read [docs/repo-architecture.md](docs/repo-architecture.md) when design affects repository boundaries or generated-source ownership.
4. Use the [placement guide](docs/project-structure-and-module-organization.md#4-where-to-put-new-code) before choosing packages or tests.

Example:

```text
Add tenant-aware export jobs end to end.

Success means the API contract, durable job state, worker lifecycle, failure behavior,
tests, and relevant validation are complete. Preserve current tenant isolation and
generated-source ownership. Use research, design, a test plan, or subagents only where
they resolve a concrete uncertainty. Stop for a user decision or external action that
cannot be safely inferred; otherwise continue through implementation and proof.
```

## Repository Layout

- `cmd/service` — entrypoint and bootstrap lifecycle
- `internal/app` — use-case layer
- `internal/infra` — HTTP, PostgreSQL, telemetry, and other adapters
- `api/openapi/service.yaml` — REST API source of truth
- `internal/api` — generated OpenAPI artifacts
- `env/migrations` — SQL migrations
- `internal/infra/postgres/sqlcgen` — generated `sqlc` artifacts
- `specs/` — task decision and execution history
- `.agents/skills` — canonical skills
- `.codex/agents` — canonical project subagents

See [Project Structure & Module Organization](docs/project-structure-and-module-organization.md) and [Repository Architecture](docs/repo-architecture.md).

## Technology Stack

- Go `1.26`
- `chi`
- `kin-openapi` and `oapi-codegen`
- PostgreSQL `17`, `pgx/v5`, and `sqlc`
- `koanf`
- Prometheus and OpenTelemetry
- `testcontainers-go` and `goleak`
- Docker multi-stage builds and distroless runtime images
- GitHub Actions

See [`go.mod`](go.mod) and [`go.sum`](go.sum) for the full dependency graph.

## Quality Gates And Verification

Useful entry points:

- `make check` — quick local checks
- `make docker-check` — quick checks through pinned Docker tooling
- `BASE_REF=origin/main HEAD_REF=HEAD make check-full` — full pre-push baseline
- `make ci-local` — native CI-style flow
- `BASE_REF=origin/main HEAD_REF=HEAD make docker-ci` — Docker CI parity flow
- `make openapi-check` — OpenAPI generation, drift, runtime, lint, and schema checks
- `BASE_OPENAPI=<base> make openapi-breaking` — compatibility check
- `make sqlc-check` — SQL generation drift
- `make migration-validate` — migration rehearsal
- `make test-integration` — integration tests

Migration targets may skip when no `MIGRATION_DSN` is provided and Docker is unavailable; skip output is not migration proof. See [Build, Test, and Development Commands](docs/build-test-and-development-commands.md#everyday-pre-push-and-pr-parity) and `.github/workflows/` for exact gates.
