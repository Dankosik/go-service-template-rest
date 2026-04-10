# Go Service Template REST

Hello from claude code

AI-native Go REST template for solo developers who want coding agents that can work inside real Go constraints.

Generic AI-native repos are good at teaching agents how to spec, plan, and delegate. They are usually much weaker at teaching them how to operate inside idiomatic Go boundaries, preserve invariants, work with `context`, respect generated artifacts, reason about `chi` and `sqlc`, and ship code that survives review. `go-service-template-rest` is built around that exact gap.

This repository is for people who code with Codex, Claude Code, Cursor, Gemini CLI, and other LLM-assisted workflows, but do not want a generic process layer floating above the language. The workflow is agent-native. The instructions, skills, review surfaces, and validation loop are Go-native.

- **Orchestrator-first**: frame, delegate, synthesize, plan, implement, verify.
- **Go-native guidance**: the repository does not stop at language-agnostic workflow advice.
- **Project-scoped agents**: Codex agents live in `.codex/agents/`, Claude Code agents live in `.claude/agents/`.
- **Portable skills**: reusable workflow expertise lives in `.agents/skills` and is mirrored to compatibility/runtime directories.
- **Artifact-driven for non-trivial work**: master `workflow-plan.md` plus `workflow-plans/<phase>.md` separate cross-phase control from phase-local orchestration, while `spec.md`, `design/`, `plan.md`, and `tasks.md` keep decisions, technical design, execution strategy, and executable task state distinct. Pre-code phases produce that bundle; implementation and validation consume it and do not create new workflow/process artifacts.
- **Production stack underneath**: OpenAPI-first HTTP, PostgreSQL, `sqlc`, observability, tests, and CI gates are already wired.

## Why This Template Exists

Most AI-native coding today is solo. Most generic AI-native repos intentionally stay technology-independent. Most Go templates still stop at folder layout, Docker files, and a `Makefile`. That combination leaves a real hole:

- the workflow knows how to spec and delegate, but not how to reason in Go;
- the stack knows how to compile, but not how to guide an agent through non-trivial changes;
- the repo has commands, but no explicit ownership model for research, planning, implementation, review, and validation.

This template is built from the opposite assumption: if you want agents to be useful in a Go backend, the workflow and the language need to be wired together on purpose.

That is why this repository is opinionated in four places:

1. **The workflow is explicit.**
   Non-trivial work starts with optional idea refinement, then framing, workflow planning, research, synthesis, pre-spec challenge, specification with an autonomous clarification gate, technical design, implementation planning, implementation, review, and validation. The loop is visible, not implied, one session normally owns one phase, and session wrappers such as `workflow-planning-session`, `research-session`, `specification-session`, `technical-design-session`, `planning-session`, and `validation-closeout-session` keep those handoffs explicit.
2. **The specialists are real.**
   Subagents have narrow ownership areas like API, domain, data, reliability, performance, and security. They are not generic “helper” personas.
3. **The skills are Go-native.**
   The skill library does not stop at abstract design advice. It covers Go architecture, routing, DB/cache contracts, invariants, reliability, security, review, debugging, testing, and verification.
4. **The backend substrate is real.**
   OpenAPI, `chi`, PostgreSQL, `sqlc`, observability, tests, and CI gates are already in the template, so the workflow lands on an actual service baseline.

If you want a Go backend template that feels natural inside Codex or Claude Code and still respects how Go services are actually built, this repository is designed for that use case.

## How This Template Solves The Problem

The fix is not a single block of text. It shapes the whole repository:

- **Artifacts with clear jobs**: `specs/<feature-id>/workflow-plan.md` keeps master control, `specs/<feature-id>/workflow-plans/<phase>.md` holds phase-local orchestration, `specs/<feature-id>/spec.md` keeps final decisions, `specs/<feature-id>/design/` carries task-local technical design for non-trivial work, `specs/<feature-id>/plan.md` owns phase strategy, `specs/<feature-id>/tasks.md` owns the executable checkbox ledger, and the phase/session wrappers keep those handoffs explicit across sessions.
- **Go-aware subagents**: the agent portfolio is organized around real backend concerns instead of generic brainstorming personas.
- **Go-native skills**: the skill library gives the orchestrator and subagents concrete playbooks for Go design, implementation, review, and verification.
- **Verification as a first-class rule**: “done” is tied to fresh command evidence, not to confident prose from an LLM.
- **A serious service template underneath**: once the workflow moves into implementation, the repo already has OpenAPI-first HTTP, PostgreSQL, `sqlc`, telemetry, and CI guardrails.

## Workflow First

This repository treats delivery as an explicit loop, not as a single long chat and not as process theater:

```text
intake -> idea refine? -> workflow planning -> research -> synthesis -> pre-spec challenge -> specification + clarification challenge -> technical design -> planning -> implementation -> review -> validation
```

- `intake`: frame the change, scope it, and record assumptions.
- `idea refine`: when the request is still a raw concept, use `idea-refine` to make the user, problem, success criteria, MVP, and not-doing boundary explicit before engineering framing.
- `workflow planning`: choose the execution shape, decide whether work stays local or fans out, set the current phase in master `workflow-plan.md`, write the phase-local orchestration in `workflow-plans/<phase>.md`, and state whether later `design/`, `plan.md`, `tasks.md`, `test-plan.md`, or `rollout.md` artifacts will be required. Early checkpoints often use `workflow-plans/workflow-planning.md` or `workflow-plans/research.md`; later ones use files like `workflow-plans/specification.md`, `workflow-plans/technical-design.md`, or `workflow-plans/planning.md`. Do not optimize for a small lane count; optimize for coverage.
- `research`: keep simple work local or fan out only to read-only subagents, with enough lanes to cover the materially affected domains. When in doubt on a complex task, prefer more lanes over fewer.
- `synthesis`: compare specialist output and produce candidate decisions.
- `pre-spec challenge`: pressure-test candidate decisions before they harden into `spec.md`, and loop back to research if needed.
- `specification`: stabilize final decisions, constraints, and open questions in `spec.md`; for non-trivial work, run the autonomous `spec-clarification-challenge` gate through a read-only challenger before approval.
- `technical design`: for non-trivial work, turn approved decisions into a task-local `design/` bundle. Load [`docs/repo-architecture.md`](docs/repo-architecture.md) first when stable repository boundaries or runtime flows matter.
- `planning`: use `planning-and-task-breakdown` or equivalent discipline to turn approved `spec.md + design/` into phased, verifiable execution work; for non-trivial implementation, phase strategy lives in `plan.md` and the executable checkbox ledger lives in `tasks.md`, any later implementation/review/validation phase workflow files are created here before code starts, and the planning exit records implementation readiness as `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`.
- `implementation`: change the service in the main flow, not inside research agents. New code/test files are fine when the approved plan requires them; new workflow/process artifacts are not. Existing `tasks.md` checkbox/progress state may be updated; missing required `tasks.md` or implementation readiness of `FAIL` routes back to planning or the named earlier phase instead of being invented mid-code.
- `review`: run targeted review agents only where the risk justifies them.
- `validation`: do not claim "done" without fresh command evidence, and do not create new planning/process artifacts during closeout. Existing `tasks.md` may be progress-updated only when it already belongs to the task.

For non-trivial work, `one session = one phase` by default: master `workflow-plan.md` tracks the current phase, artifact status, next session, blockers, and links to phase workflow plans; the current `workflow-plans/<phase>.md` carries phase-local orchestration. Finish that phase, update both workflow-control files plus the owning phase artifacts, and stop before the next phase. Start the next phase in a new session unless an upfront `direct path` or `lightweight local` waiver was recorded. If later implementation/review/validation phase files are part of the plan, create them before implementation begins and let post-code sessions update them rather than inventing them mid-execution.

When a task benefits from explicit session boundaries, use the phase/session wrappers that match the current checkpoint:

- `workflow-planning-session`: own the pre-research routing pass only.
- `research-session`: own evidence gathering and optional preserved `research/*.md` only.
- `specification-session`: own `spec.md` approval, the clarification gate, and `workflow-plans/specification.md` only.
- `technical-design-session`: own the task-local `design/` bundle and `workflow-plans/technical-design.md` only.
- `planning-session`: own `plan.md`, `tasks.md`, optional `test-plan.md` or `rollout.md`, any later phase workflow files the approved plan already requires, and `workflow-plans/planning.md` only.
- `validation-closeout-session`: own fresh proof, `spec.md` closeout updates, and existing validation-phase routing only.

Think of the workflow-control artifacts as complementary, not competing:

- `workflow-plan.md`: master cross-phase routing, artifact status, blockers, and next-session handoff.
- `workflow-plans/<phase>.md`: one phase only, with local orchestration, completion marker, stop rule, and next action.
- `plan.md`: execution strategy, phases, dependencies, checkpoints, validation plan, risk notes, and reopen conditions.
- `tasks.md`: executable task ledger with markdown checkboxes, stable IDs such as `T001`, phase labels, optional `[P]` only for safe parallel work, dependency markers when needed, concrete file/package surfaces, and proof expectations.
- Implementation readiness: a planning-phase gate. `PASS` allows implementation, `CONCERNS` requires named accepted risks and proof obligations, `FAIL` routes earlier, and `WAIVED` stays limited to explicit tiny/direct-path/prototype scope.

Use `workflow-status` when you only need a compact read-only status or next-action check from existing artifacts. It reports state; it does not repair artifacts, approve readiness, or replace the workflow-control files.

`pre-spec challenge` is a risk-driven checkpoint inside the synthesis boundary, not a separate approval authority.
`spec-clarification-challenge` is a required non-trivial approval gate inside `specification`: a read-only challenger surfaces high-impact questions for the orchestrator to answer from evidence, route to targeted research, accept as risk, defer with rationale, or mark `requires_user_decision` without inventing an answer.
For tiny or direct-path fixes, several of these stages can collapse into one short local pass instead of turning into mandatory ceremony.
Write-capable delegate agents are out of policy for this workflow; if a tool surface cannot reliably stay read-only, keep that track in the main flow instead of delegating it.

The full contract lives in [AGENTS.md](AGENTS.md) and the supporting workflow doc lives in [docs/spec-first-workflow.md](docs/spec-first-workflow.md).

## Agent Portfolio

This repository distinguishes between two different things:

- **Subagents** are read-only specialists you fan out to for focused research or review.
- **Skills** are portable workflow playbooks loaded on demand by the orchestrator or a subagent.

The repository ships with project-scoped, read-only subagents for focused reasoning and review.
Click an agent name to open its project-scoped instruction file in `.claude/agents`.

| Agent | Owns | Use when | Returns |
|---|---|---|---|
| [`architecture-agent`](.claude/agents/architecture-agent.md) | boundaries, ownership, interaction style, failure-domain shape | a feature or refactor may change module or service shape | boundary call, interaction recommendation, handoffs |
| [`api-agent`](.claude/agents/api-agent.md) | client-visible contract behavior and transport semantics | endpoints, statuses, errors, idempotency, or async acknowledgment change | contract recommendation, compatibility notes |
| [`concurrency-agent`](.claude/agents/concurrency-agent.md) | goroutine, channel, cancellation, and shutdown correctness | a diff touches worker pools, goroutines, shared state, or race-prone code | concurrency findings, validation gaps |
| [`challenger-agent`](.claude/agents/challenger-agent.md) | pre-spec challenge, spec clarification challenge, hidden assumptions, corner cases, and planning-risk pressure tests | candidate decisions exist but need an independent challenger before planning or before non-trivial `spec.md` approval | discriminating questions, blocker calls, next actions |
| [`data-agent`](.claude/agents/data-agent.md) | source of truth, schema evolution, transaction and cache rules | schema, query, migration, or cache behavior changes | data contract, rollout implications |
| [`delivery-agent`](.claude/agents/delivery-agent.md) | CI/CD gates, rollout policy, runtime hardening, release trust | release controls, deployment policy, or platform constraints change | delivery policy, gating recommendations |
| [`design-integrator-agent`](.claude/agents/design-integrator-agent.md) | cross-domain reconciliation and simplification | multiple specialist outputs conflict or the design feels over-layered | integrated path, contradictions, reopen conditions |
| [`distributed-agent`](.claude/agents/distributed-agent.md) | cross-service consistency, outbox/inbox, replay, reconciliation | the workflow crosses service boundaries or depends on eventual consistency | flow model, recovery stance |
| [`domain-agent`](.claude/agents/domain-agent.md) | business invariants, state transitions, acceptance semantics | behavior changes touch lifecycle, rules, duplicates, or forbidden paths | invariant set, corner cases, handoffs |
| [`performance-agent`](.claude/agents/performance-agent.md) | performance budgets, bottleneck hypotheses, proof strategy | the change is hot-path sensitive or justified mainly by speed | performance stance, proof obligations |
| [`qa-agent`](.claude/agents/qa-agent.md) | test obligations, proving levels, validation readiness | a non-trivial behavior change needs a real regression plan | scenario matrix, validation strategy |
| [`quality-agent`](.claude/agents/quality-agent.md) | idiomatic Go review and simplification | the diff feels noisy, over-abstracted, or hard to maintain | maintainability findings, cleanup guidance |
| [`reliability-agent`](.claude/agents/reliability-agent.md) | timeouts, retries, overload, startup, shutdown, degradation | failure behavior, degraded mode, or lifecycle semantics change | reliability contract, residual risks |
| [`security-agent`](.claude/agents/security-agent.md) | trust boundaries, auth, tenant isolation, abuse resistance | changed paths handle untrusted input or cross security boundaries | threat/control map, verification expectations |

All of these agents stay advisory and read-only. Write-capable delegates are not part of this subagent model. Final decisions always stay with the orchestrator in the main flow.

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
- Pre-spec pressure-test on ambiguous work: `challenger-agent` + the specialist whose decision still feels under-evidenced
- Spec approval clarification: `challenger-agent` with exactly one skill, `spec-clarification-challenge`
- Storage, cache, or migration change: `data-agent` + `reliability-agent`
- Cross-service or async workflow: `architecture-agent` + `distributed-agent` + `security-agent`
- Pre-merge cleanup on a larger diff: `quality-agent` + the domain reviewer that matches the risk

## Skill Library

`.agents/skills` is the canonical repository skill set. These skills are procedural building blocks, not autonomous owners of the workflow.
Click a skill name to open its canonical instruction file.

The catalog has two layers:

- phase/session wrappers that keep one session bounded to one checkpoint and update `workflow-plan.md` plus the matching `workflow-plans/<phase>.md`
- deeper skills that do framing, design, planning, implementation, review, or validation work inside those boundaries when needed

### Session-Bounded Phase Skills

| Skill | What it does | Load when |
|---|---|---|
| [`workflow-planning-session`](.agents/skills/workflow-planning-session/SKILL.md) | owns the workflow-planning checkpoint only and writes or repairs `workflow-plan.md` plus `workflow-plans/workflow-planning.md` | non-trivial or agent-backed work needs explicit routing, research mode, lane planning, and artifact expectations before research starts |
| [`research-session`](.agents/skills/research-session/SKILL.md) | owns the research checkpoint only and keeps evidence gathering, optional `research/*.md`, and routing updates separate from spec writing | the task already has framing and workflow routing, but one bounded research session is needed before specification |
| [`specification-session`](.agents/skills/specification-session/SKILL.md) | owns the specification checkpoint only, runs or reconciles the non-trivial clarification gate, and updates `spec.md`, `workflow-plan.md`, and `workflow-plans/specification.md` without drifting into design or planning | research or bounded local analysis is strong enough that the next honest step is finalizing the decision record |
| [`technical-design-session`](.agents/skills/technical-design-session/SKILL.md) | owns the technical-design checkpoint only and turns approved `spec.md` into a planning-ready `design/` bundle plus `workflow-plans/technical-design.md` | non-trivial work needs task-local technical design before implementation planning |
| [`planning-session`](.agents/skills/planning-session/SKILL.md) | owns the planning checkpoint only and produces `plan.md`, `tasks.md`, optional `test-plan.md` or `rollout.md`, and any later phase workflow files the approved plan already requires, while updating `workflow-plans/planning.md` | approved `spec.md + design/` are ready to turn into ordered, coder-facing execution work |
| [`validation-closeout-session`](.agents/skills/validation-closeout-session/SKILL.md) | owns final validation and closeout only, refreshes `spec.md` `Validation` and `Outcome`, and updates existing validation-phase routing honestly | implementation is finished and you need fresh proof before saying a phase or task is complete |

### Core Workflow, Implementation, And Verification Skills

| Skill | What it does | Load when |
|---|---|---|
| [`idea-refine`](.agents/skills/idea-refine/SKILL.md) | turns a raw idea into one concrete direction with explicit user problem, assumptions, MVP boundary, and not-doing list | the request is still product- or solution-ambiguous and is not ready for engineering framing yet |
| [`spec-first-brainstorming`](.agents/skills/spec-first-brainstorming/SKILL.md) | turns a refined idea or rough change request into an engineering-ready problem frame with scope, constraints, assumptions, and design-readiness | the task is close to spec work but still needs crisp framing before challenge or deeper design |
| [`pre-spec-challenge`](.agents/skills/pre-spec-challenge/SKILL.md) | pressure-tests candidate decisions with discriminating questions before planning | research is done but hidden assumptions or edge cases could still change the spec |
| [`spec-clarification-challenge`](.agents/skills/spec-clarification-challenge/SKILL.md) | surfaces non-obvious spec-approval questions for orchestrator reconciliation before non-trivial `spec.md` is marked approved | candidate decisions exist inside `specification` and the orchestrator needs a read-only clarification gate before approval |
| [`spec-document-designer`](.agents/skills/spec-document-designer/SKILL.md) | designs and normalizes repository-native `spec.md` decision records with the right section depth, decision placement, and handoff into design and planning | framing or research is already in place and the orchestrator needs a clean decision record instead of a PRD, research dump, or task list |
| [`planning-and-task-breakdown`](.agents/skills/planning-and-task-breakdown/SKILL.md) | turns approved `spec.md + design/` into `plan.md` phase strategy plus a `tasks.md` checkbox ledger with checkpoints, acceptance criteria, and verification steps | the decisions and task-local technical design are stable and implementation needs real planning artifacts instead of ad hoc execution |
| [`go-coder`](.agents/skills/go-coder/SKILL.md) | implements approved Go changes without semantic drift or new workflow-artifact sprawl | the implementation plan is explicit, readiness allows code work, and code work is next |
| [`go-qa-tester`](.agents/skills/go-qa-tester/SKILL.md) | writes deterministic Go tests from approved test obligations as implementation work, not new planning | test code itself needs to be added or upgraded |
| [`go-systematic-debugging`](.agents/skills/go-systematic-debugging/SKILL.md) | drives root-cause-first debugging with reproducible evidence | a bug, flaky test, build failure, or incident needs diagnosis |
| [`go-verification-before-completion`](.agents/skills/go-verification-before-completion/SKILL.md) | maps completion claims to fresh command evidence without inventing missing process artifacts | you are about to say “fixed”, “ready”, or “done” |
| [`workflow-status`](.agents/skills/workflow-status/SKILL.md) | reports the current task path, phase, blockers, allowed writes, next action, stop rule, and implementation-start status from existing artifacts only | you need a compact read-only workflow status or next-action check without creating a new source of truth |

### Prompt Composition And Tooling

| Skill | What it does | Load when |
|---|---|---|
| [`ru-agent-prompt-composer`](.agents/skills/ru-agent-prompt-composer/SKILL.md) | turns messy Russian task descriptions into a strong English prompt for coding agents working in this repository | rough Russian or mixed-language notes need intent reconstruction, repo-aware context selection, and a downstream-agent-ready prompt instead of plain translation |

### System Design And Control Surfaces

| Skill | Focus | Load when |
|---|---|---|
| [`go-architect-spec`](.agents/skills/go-architect-spec/SKILL.md) | service boundaries, ownership, sync vs async interaction style | system shape or module ownership may change |
| [`go-design-spec`](.agents/skills/go-design-spec/SKILL.md) | integrated technical-design-bundle assembly and reconciliation across domains | approved decisions exist, but the task-local `design/` bundle still feels contradictory, layered, or not yet stable enough for task breakdown |
| [`go-devops-spec`](.agents/skills/go-devops-spec/SKILL.md) | CI/CD policy, rollout controls, runtime hardening, release trust | delivery or release behavior is part of the change |
| [`go-observability-engineer-spec`](.agents/skills/go-observability-engineer-spec/SKILL.md) | logs, metrics, traces, correlation, telemetry cost | observability behavior needs an explicit contract |
| [`go-performance-spec`](.agents/skills/go-performance-spec/SKILL.md) | latency, throughput, contention, benchmark strategy | performance budgets or hot paths drive the design |
| [`go-reliability-spec`](.agents/skills/go-reliability-spec/SKILL.md) | timeouts, retries, degradation, lifecycle behavior | failure handling or operational resilience changes |
| [`go-security-spec`](.agents/skills/go-security-spec/SKILL.md) | trust boundaries, auth, tenant isolation, abuse resistance | the change touches security-critical surfaces |
| [`go-qa-tester-spec`](.agents/skills/go-qa-tester-spec/SKILL.md) | test levels, scenario coverage, proof strategy | you need an explicit verification plan before coding |

### API, Routing, Domain, Data, And Distributed Semantics

| Skill | Focus | Load when |
|---|---|---|
| [`api-contract-designer-spec`](.agents/skills/api-contract-designer-spec/SKILL.md) | resources, methods, statuses, errors, idempotency, async contracts | client-visible API behavior is changing |
| [`go-chi-spec`](.agents/skills/go-chi-spec/SKILL.md) | chi router topology, middleware ordering, fallback and CORS semantics | routing shape or HTTP middleware policy changes |
| [`go-data-architect-spec`](.agents/skills/go-data-architect-spec/SKILL.md) | source of truth, schema ownership, migration and rollback shape | schema or persistence model changes |
| [`go-db-cache-spec`](.agents/skills/go-db-cache-spec/SKILL.md) | query discipline, transaction rules, cache strategy and staleness | runtime DB or cache behavior needs an explicit contract |
| [`go-domain-invariant-spec`](.agents/skills/go-domain-invariant-spec/SKILL.md) | business invariants, state transitions, acceptance rules | lifecycle or core domain behavior changes |
| [`go-distributed-architect-spec`](.agents/skills/go-distributed-architect-spec/SKILL.md) | saga shape, outbox/inbox, replay safety, reconciliation | a flow crosses service boundaries or depends on eventual consistency |

### Review Skills

| Skill | Focus | Load when |
|---|---|---|
| [`go-design-review`](.agents/skills/go-design-review/SKILL.md) | architecture alignment, boundary integrity, accidental complexity | a diff may hide broader design drift |
| [`go-chi-review`](.agents/skills/go-chi-review/SKILL.md) | router ownership, middleware order, HTTP fallback semantics | chi routing or transport behavior changed |
| [`go-db-cache-review`](.agents/skills/go-db-cache-review/SKILL.md) | SQL safety, transaction scope, cache correctness, fallback risk | DB or cache code changed |
| [`go-domain-invariant-review`](.agents/skills/go-domain-invariant-review/SKILL.md) | business-invariant preservation and side-effect safety | behavior changes carry semantic risk |
| [`go-idiomatic-review`](.agents/skills/go-idiomatic-review/SKILL.md) | idiomatic Go, error handling, context flow, naming | you want merge-risk review on Go code quality |
| [`go-language-simplifier-review`](.agents/skills/go-language-simplifier-review/SKILL.md) | lower cognitive complexity and cleaner control flow | the code works but feels noisy or over-abstracted |
| [`go-concurrency-review`](.agents/skills/go-concurrency-review/SKILL.md) | goroutines, channels, cancellation, shutdown safety | concurrent behavior changed or races are suspected |
| [`go-performance-review`](.agents/skills/go-performance-review/SKILL.md) | hot-path regression, allocation and contention risk | performance is a review concern |
| [`go-qa-review`](.agents/skills/go-qa-review/SKILL.md) | coverage quality, assertion strength, determinism | review depends on test quality and proof strength |
| [`go-reliability-review`](.agents/skills/go-reliability-review/SKILL.md) | retries, backpressure, startup, shutdown, degraded mode | failure-path behavior changed |
| [`go-security-review`](.agents/skills/go-security-review/SKILL.md) | authz, isolation, injection/SSRF, secret handling | changed paths accept untrusted input or cross trust boundaries |

### Skill Locations Across Runtimes

These repository-native skill locations keep the workflow portable:

- `.agents/skills`
- `.claude/skills`
- `.cursor/skills`
- `.gemini/skills`
- `.github/skills`
- `.opencode/skills`

The source of truth stays in `.agents/skills`, so you do not have to hand-maintain separate skill instructions per tool.
Refresh the runtime mirrors with `bash ./scripts/dev/sync-skills.sh` or `make skills-sync`, and verify them with `bash ./scripts/dev/sync-skills.sh --check` or `make skills-check`.

## This Is An Orchestrator Project

The repository is designed so the main agent acts like an orchestrator, not like a single monolithic coder.

- The orchestrator owns framing, scope, synthesis, planning, implementation, reconciliation, and validation.
- Subagents own narrow research or review tracks only.
- Skills are tools, not the workflow itself.
- `spec.md` is the canonical decisions artifact.
- `workflow-plan.md` is the master control artifact for the whole task.
- `workflow-plans/<phase>.md` is the phase workflow artifact for one phase only.
- `design/` is the task-local technical design bundle for non-trivial work.
- `plan.md` is the execution strategy artifact, not a second spec.
- `tasks.md` is the executable task ledger, not a second spec, second design bundle, or competing plan.
- Implementation readiness is the planning exit gate, not a phase. It is recorded in `workflow-plan.md`, with the result and stop or handoff rule in `workflow-plans/planning.md`.
- `research/*.md` is optional supporting evidence, not a competing source of truth.

For non-trivial implementation work, the artifact shape is intentionally simple:

```text
specs/<feature-id>/
  workflow-plan.md
  workflow-plans/
  spec.md
  design/
  plan.md
  tasks.md
  research/
```

If you want the short version: frame first, keep cross-phase control in `workflow-plan.md`, keep current-phase orchestration in `workflow-plans/<phase>.md`, use the session wrappers when a checkpoint needs a dedicated session, keep approved decisions in `spec.md`, write task-local technical design in `design/`, plan strategy in `plan.md`, track executable work in `tasks.md`, and move phase by phase with review and validation between increments. For tiny fixes, keep it lighter and skip the extra artifacts when the change is obviously local and the waiver is explicit.

## Quickstart

### Human Quickstart

```bash
make bootstrap
make template-init   # run this when you create a new repo from the template
make check
make run
```

### Create Your Own Repository From This Template

Recommended flow:

1. Create a new empty GitHub repository under your account or organization. It may be `private` or `public`, but do not initialize it with `README`, `.gitignore`, or `LICENSE`.
2. Clone this template into the directory you want to use for the new service.
3. Rename the template remote to `upstream` and point `origin` to your repository.
4. Run template initialization before the first push.

```bash
git clone https://github.com/Dankosik/go-service-template-rest.git my-service
cd my-service

git remote rename origin upstream
git remote add origin git@github.com:<your-user>/<your-repo>.git
# or: git remote add origin https://github.com/<your-user>/<your-repo>.git

git remote -v

make bootstrap
make template-init
make check

git add .
git commit -m "chore: initialize service from template"
git push -u origin main
```

What this does:

- `origin` becomes your repository, so normal `git push` goes to your project.
- `upstream` keeps a reference to the original template repository in case you want to compare or pull template updates later.
- `make template-init` rewires the Go module path, `CODEOWNERS`, and skill mirrors for the new repository.
- `git push -u origin main` publishes the first `main` branch to your repository and makes future plain `git push` / `git pull` work against `origin/main`.

If `git push` says `Everything up-to-date` but your GitHub repository is still empty, your local branch is probably still tracking the template branch instead of your own repository. Check:

```bash
git remote -v
git branch -vv
```

Expected state:

- `origin` points to your repository.
- `upstream` points to `go-service-template-rest`.
- `main` tracks `origin/main`, not `upstream/main`.

If needed, publish the branch explicitly:

```bash
git push -u origin main
```

If SSH push fails with `Permission denied (publickey)`, either configure your GitHub SSH key or switch `origin` to HTTPS:

```bash
git remote set-url origin https://github.com/<your-user>/<your-repo>.git
git push -u origin main
```

If you use GitHub's **Use this template** button instead of the manual clone flow, clone your generated repository normally and still run:

```bash
make bootstrap
make template-init
```

For production-style GitHub setup after the first push:

```bash
gh auth login
make gh-protect BRANCH=main
```

Typical next steps:

1. Copy `env/.env.example` to `.env` if `make bootstrap` did not already do it.
2. Run `make template-init` after cloning into a new service repository to rewire module path, `CODEOWNERS`, and skill mirrors.
3. Use `make check-full` before larger changes or before opening a PR.

### Agent Quickstart

1. Open the repository in Codex or Claude Code.
2. Read [AGENTS.md](AGENTS.md). Claude-facing compatibility is mirrored in [CLAUDE.md](CLAUDE.md).
3. For non-trivial or agent-backed work, open [docs/spec-first-workflow.md](docs/spec-first-workflow.md) before workflow planning or subagent fan-out.
4. If the task reaches technical design, load [docs/repo-architecture.md](docs/repo-architecture.md) before writing task-local `design/`.
5. Start with an artifact-driven, phase-bounded prompt, not with direct code generation.

Example kickoff prompt:

```text
Use `workflow-planning-session` if this is non-trivial enough to need dedicated workflow control.
Use `idea-refine` only if the request is still too raw.
Frame a change to add tenant-aware export jobs.
Fan out to `architecture-agent`, `data-agent`, and `qa-agent` only if needed.
Run `challenger-agent` before `specification-session` if material assumptions remain.
During `specification-session`, run `challenger-agent` with `spec-clarification-challenge` before approving non-trivial `spec.md`.
Load `docs/repo-architecture.md` before `technical-design-session` if repository boundaries matter.
Write master control to `specs/tenant-export-jobs/workflow-plan.md`.
Start the current checkpoint in `specs/tenant-export-jobs/workflow-plans/workflow-planning.md`, then advance one session-bounded phase at a time through `research.md`, `specification.md`, `technical-design.md`, `planning.md`, and any needed post-code phase files.
Write decisions to `specs/tenant-export-jobs/spec.md`, task-local technical design to `specs/tenant-export-jobs/design/`, phase strategy to `specs/tenant-export-jobs/plan.md`, and the executable task ledger to `specs/tenant-export-jobs/tasks.md` before coding.
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
- `.agents/skills` - canonical skill definitions

More detail: [docs/project-structure-and-module-organization.md](docs/project-structure-and-module-organization.md), plus the stable architecture baseline in [docs/repo-architecture.md](docs/repo-architecture.md)

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
