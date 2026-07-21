# Go Service Template REST

Agent-native Go REST template for solo developers who want Codex or Claude Code to work inside real Go, API, data, and operational constraints without carrying a heavyweight process prompt. Both harnesses are pre-wired: clone the repository and either agent picks up the same instructions, specialist subagents, and skills with no setup.

The repository combines:

- outcome-first workflow guidance with risk-proportional artifacts;
- project-scoped read-only specialist agents;
- reusable Go design, implementation, review, and verification skills;
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

Structured and orchestrated work evaluates every phase boundary in order. The owning macro phases are specification, technical design, test design, planning, and implementation/validation/closeout; intake and research support the owning phase unless the user names `research only`, which makes the fixed synthesis its own independently reviewed macro-phase outcome. Research, design, or test design may be scoped down when its question is already closed, with a concrete reason; specification, planning, and their independent review gates remain required. One authorized request may cross several phases without collapsing their ownership or gates. An explicit boundary such as `research only`, `planning only`, `read-only`, or `docs-only` stops the work there. Required non-implementation reviews need a fresh `PASS`; `CONCERNS` stays for disposition/re-review and `FAIL` for repair/reopen. Implementation retains the current local/direct and optional Worker contract in [AGENTS.md](AGENTS.md#implementation-and-evidence) and [Implementation / Validation / Closeout](docs/spec-first-workflow/phases/implementation-validation-closeout.md). An explicitly requested independent review of completed implementation is a separate read-only request. A next-session prompt is reserved for an intentional next macro phase or an honest blocker the current root cannot resolve.

Before each applicable non-implementation review, the root runs one autonomous read-only grilling probe against the completed candidate, records material dispositions in that candidate, and then uses a different child for the required review. This applies once to Specification, combined Technical Design, Test Design, Planning, and explicit `research only`; it does not add probes to supporting steps, direct work, or Implementation. Explicit user-requested grilling remains a root-to-user dialogue. See [Autonomous Pre-Review Challenge](docs/spec-first-workflow/shared/autonomous-pre-review-challenge.md).

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

`.codex/agents/*.toml` and `.claude/agents/*.md` contain the same project-scoped read-only specialist roles for the Codex App and Claude Code respectively. `.agents/skills` is the canonical skill set, exposed to Claude Code through per-skill symlinks in `.claude/skills/` (`make claude-skills-sync` resyncs them). `CLAUDE.md` imports `AGENTS.md`, so both harnesses load one contract; [docs/agent-harness.md](docs/agent-harness.md) owns the mapping between their native controls (workers, subagents, models, reasoning effort).

The repository ships the runtime agent and skill guidance used by Codex and
Claude Code. It does not ship a model-evaluation runner, fake adapters, or a
second CI system for validating that guidance.

In non-implementation macro phases, evaluate whether concrete, independent, bounded research or review lanes improve evidence or review independence. Use only useful read-only subagent lanes and keep tightly coupled reasoning local; record a local-only reason in an existing artifact or handoff instead of creating a gate file. [Subagents And Handoff](docs/spec-first-workflow/shared/subagents-and-handoff.md) owns those lanes; the [implementation phase](docs/spec-first-workflow/phases/implementation-validation-closeout.md#optional-worker-execution) retains its current optional Worker and root-review boundary.

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
| `challenger-agent` | focused challenge or internal pre-review grilling probe |

Representative workflow skills:

| Skill | Use when |
| --- | --- |
| `idea-refine` / `spec-first-brainstorming` | the outcome still needs product or engineering framing |
| `research-session` | evidence can change a decision |
| `specification-session` / `spec-document-designer` | behavior decisions need a durable record |
| `technical-design-session` / `go-implementation-ownership` | mechanism or Go ownership is not obvious |
| `test-design-session` | proof needs a real scenario matrix |
| `planning-session` / `planning-and-task-breakdown` | implementation needs a durable ledger |
| `go-coder` / `go-test-implementation` | implement accepted Go behavior and tests |
| `go-systematic-debugging` | diagnose a bug, flake, hang, or build failure |
| `go-verification-before-completion` | map completion claims to fresh evidence |

Domain spec and review skills cover API/chi, data/cache, distributed consistency, domain invariants, security, reliability, concurrency, observability, performance, delivery, QA, and Go maintainability. Load only the skill that maps to the current decision or symptom.

## Orchestrator Model

The root owns orchestration and final claims under [AGENTS.md](AGENTS.md); the [implementation phase](docs/spec-first-workflow/phases/implementation-validation-closeout.md#optional-worker-execution) owns optional Worker execution and the root acceptance boundary.

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
make template-init CODEOWNER=@your-user-or-org/team
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

make template-init CODEOWNER=@your-user-or-org/team
make check

git add .
git commit -m "chore: initialize service from template"
git push -u origin main
```

`origin` should point to your repository and `upstream` to this template. If SSH fails, switch `origin` to HTTPS. Repositories created with GitHub's “Use this template” should still run `make template-init CODEOWNER=@...`.

After the first push, configure a GitHub Ruleset or organization policy that
requires the blocking jobs currently defined in `.github/workflows/ci.yml`.
GitHub settings are administrative state and are not mutated by this template.

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

- `make check` — broad local baseline (fmt, full lint, full tests)
- `make ci-local` — deterministic native CI aggregate
- `make check-full` — native aggregate plus Docker-backed integration, migration, and image scan
- `BASE_REF=origin/main make pr-check` — full local checks plus OpenAPI breaking analysis
- `make openapi-check` — OpenAPI generation, drift, runtime, lint, and schema checks
- `BASE_OPENAPI=<base> make openapi-breaking` — compatibility check
- `make sqlc-check` — SQL generation drift
- `make migration-validate` — migration rehearsal
- `make test-integration` — integration tests
- `make bench` / `make bench-compare` — repeatable Go benchmark capture and `benchstat` comparison
- `make bench-db BENCH_DB_WORKLOAD_ID=<fixture-state>` / `make bench-db-compare` — real PostgreSQL benchmarks with pinned server/schema/workload provenance
- `make bench-http` — thresholded steady, stress, spike, or soak HTTP load with digest-pinned k6 and `.env.bench`
- `make benchmark-infra-check` — executable smoke for Go capture, benchstat, PostgreSQL provenance, and both k6 executor modes
- `make benchmark-remote-check` — read-only DigitalOcean/doctl, SSH key, region, image, size, and current-price preflight
- `make benchmark-remote-image` — explicitly paid one-time build of a reusable DigitalOcean snapshot for faster Droplet startup
- `scripts/dev/benchmark-remote.sh` — ephemeral CPU-Optimized Droplet execution for private local source, including same-host comparison and two-host HTTP load

DigitalOcean is the preferred benchmark environment only when `doctl` is
already installed and authorized. Otherwise do not install it, start login, or
create an account automatically; use the matching local `make bench*` command.
Local execution is the supported fallback for template users without a
DigitalOcean account. Missing local prerequisites such as Docker remain an
explicit blocker for the benchmark level that needs them.

Users who explicitly opt in can follow the tested
[DigitalOcean account, `doctl`, SSH/Keychain, Tier 1 smoke, and cleanup
onboarding](.agents/skills/digitalocean-benchmark-runner/references/digitalocean.md#account-and-workstation-onboarding).
The guide separates human-only token/passphrase entry from commands an agent
may run, and never treats Shared-CPU smoke numbers as performance evidence.
Its optional [golden snapshot
path](.agents/skills/digitalocean-benchmark-runner/references/digitalocean.md#reusable-golden-snapshot)
preinstalls the stable OS dependencies once. The normal public Ubuntu image
remains the fallback; building or retaining a snapshot is never required for
local users and is never performed without explicit paid-write authorization.

`make migration-validate` requires either `MIGRATION_DSN` or a reachable Docker
daemon; it never reports a missing proof path as a successful skip. See
[Benchmarking](docs/benchmarking.md), [Build, Test, and Development
Commands](docs/build-test-and-development-commands.md), and
`.github/workflows/` for exact gates.
