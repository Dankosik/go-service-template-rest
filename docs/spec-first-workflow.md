# Spec-First Workflow For Go Features With Multi-Skill Roles

## 1. Purpose

This workflow defines a process where:

- the main unit of work is the specification;
- architecture and contract decisions are closed before coding starts;
- `go-coder` implements only decisions that are already approved and operationalized into the detailed coder plan;
- reviewers evaluate code strictly within their area of expertise.

Result: less architectural drift, fewer coding-time assumptions, and higher predictability and implementation quality.

## 2. Core Principles

1. Spec-first always takes priority over code-first.
2. The spec is the Source of Truth for API, data, consistency, security, observability, and delivery decisions.
3. Code is written only after `Spec Sign-Off`.
4. Open questions are not carried into the coding phase.
5. Any `*-spec` skill can edit any spec file in `specs/<feature-id>/`.
6. `*-review` skills do not edit the specification and do not redesign the solution during code review.
7. After `Gate G2`, `Spec Freeze` is active: spec files can be changed only through `Spec Reopen`.
8. During code review, each reviewer is constrained to their domain and must not go outside their expertise.
9. Workflow control and technical expertise are separate concerns: this file defines the workflow, while each `*-spec` skill must contribute domain expertise, not only process or formatting control.
10. `*-spec` skills are expertise-first with explicit primary domains; cross-file edits are allowed, but specialist decisions must not be duplicated or overridden without rationale.
11. Every user turn must pass mandatory message-level routing (`M0`) before any response or action.
12. New feature/refactor/behavior-change requests must pass pre-spec brainstorming (`Phase -1`, `Gate B0`) before `Phase 0`.

## 3. Artifacts

For each feature, create this folder:

```text
specs/<feature-id>/
  00-input.md
  10-context-goals-nongoals.md
  15-domain-invariants-and-acceptance.md
  20-architecture.md
  30-api-contract.md
  40-data-consistency-cache.md
  50-security-observability-devops.md
  55-reliability-and-resilience.md
  60-implementation-plan.md
  65-coder-detailed-plan.md
  70-test-plan.md
  80-open-questions.md
  90-signoff.md
```

Additional review artifact:

```text
reviews/<feature-id>/
  code-review-log.md
```

## 4. Roles And Responsibilities

### 4.1 Skill Classes

- `PROCESS` class: message-level routing and pre-spec framing before specification phase starts.
- `SPEC` class: specification design and enrichment, open-question closure, and Definition of Ready formation.
- `REVIEW` class: code/diff review only, with validation against specs and domain quality criteria.

### 4.1.1 PROCESS Skills (Pre-Spec Control)

- `using-spec-first-superpowers` - mandatory pre-turn router (`M0`): classify intent/phase/gates, choose required/optional skills, and enforce route decision (`route_pass`/`route_lightweight`/`route_blocked`) before any work.
- `spec-first-brainstorming` - pre-spec framing for new feature/refactor/behavior-change requests: normalize problem, fix scope/non-goals/constraints, seed assumptions/open questions, and decide `Gate B0`.

### 4.2 SPEC Skills (Specification Phase)

All `*-spec` skills can edit any files in `specs/<feature-id>/`.
All `*-spec` skills are expertise roles first, not workflow-only roles.

- `go-architect-spec` - primary architecture expert and specification owner: boundaries, decomposition, interaction style (sync/async), consistency model frame, resilience architecture frame, plus loop orchestration.
- `api-contract-designer-spec` - API contracts and cross-cutting API semantics.
- `go-data-architect-spec` - SQL/data modeling/migrations/data reliability.
- `go-distributed-architect-spec` - cross-service workflows, saga, outbox/inbox, consistency.
- `go-db-cache-spec` - cache strategy and SQL-access risks in the specification.
- `go-observability-engineer-spec` - logs/metrics/traces, SLI/SLO, debuggability, telemetry cost/cardinality, async observability.
- `go-security-spec` - secure-by-default requirements and threat controls.
- `go-devops-spec` - CI gates, release safety, container/runtime hardening.
- `go-qa-tester-spec` - test strategy and test implementation planning (unit/integration/contract/e2e-smoke).
- `go-performance-spec` - perf budget decomposition, workload/hot-path normalization, benchmark/profile/trace protocol, latency/throughput/allocation risks, and measurable acceptance criteria.
- `go-domain-invariant-spec` - business invariants, state transitions, acceptance criteria, domain corner cases.
- `go-reliability-spec` - timeout/retry budgets, backpressure, graceful shutdown, degradation, rollout/rollback safety.
- `go-design-spec` - architecture integrity, simplicity, maintainability.
- `go-coder-plan-spec` - execution-grade coding plan design from approved specs (`65-coder-detailed-plan.md`) with atomic tasks, traceability, checkpoints, and clarification triggers.
- `go-chi-spec` - `go-chi` transport-specific design decisions (`Route`/`Mount` topology, middleware ordering, `404/405/OPTIONS` policy, route-template extraction) for routing-related scope.

Primary-domain rule for SPEC skills:

- each `*-spec` skill must deliver decisions in its primary expertise domain;
- no `*-spec` skill should be reduced to workflow management or formatting-only control;
- no `*-spec` skill should duplicate another skill's specialized decision unless it records rationale and cross-domain impact in spec artifacts.

### 4.3 Roles During Implementation

- `go-coder-plan-spec` - prepares `65-coder-detailed-plan.md` after `G2` and before coding (`G2.5` readiness gate).
- `go-coder` - implementation strictly against `65-coder-detailed-plan.md`, preserving strategic intent and constraints from `60` plus approved contracts.
- `go-qa-tester` - test implementation strictly against `70-test-plan.md` and spec requirements.

### 4.4 REVIEW Skills (Code Review Phase)

- `go-idiomatic-review`
- `go-design-review`
- `go-qa-review`
- `go-domain-invariant-review`
- `go-language-simplifier-review`
- `go-performance-review`
- `go-concurrency-review`
- `go-db-cache-review`
- `go-reliability-review`
- `go-security-review`
- `go-chi-review`

## 5. Phase Sequence

## Pre-Phase Control. Message-Level Routing (`M0`)

Owner: `using-spec-first-superpowers`.

Actions:

1. Classify request intent.
2. Determine current workflow phase and gate state.
3. Select required and optional skills with explicit execution order.
4. Emit routing decision:
   - `route_pass`
   - `route_lightweight`
   - `route_blocked`

Gate M0:

- every turn has a routing record with explicit decision;
- no response/action is produced before routing decision;
- `route_blocked` stops execution until unblock condition is resolved.

## Phase -1. Pre-Spec Brainstorming

Owner: `spec-first-brainstorming`.

Actions:

1. Normalize request into one clear problem statement.
2. Fix scope, non-goals, constraints, and success criteria.
3. Record explicit assumptions and risks.
4. Seed prioritized open questions with owner/unblock condition.
5. Prepare handoff to `go-architect-spec` for `Phase 0`.

Gate B0 (entry to Phase 0):

- problem and expected behavior change are unambiguous;
- scope/non-goals/constraints are explicitly fixed;
- critical assumptions are explicit;
- open questions are prioritized and actionable;
- no hidden architecture decisions are made in brainstorming.

## Phase 0. Intake And Spec Initialization

Owners: `go-architect-spec`, `go-domain-invariant-spec`, `go-reliability-spec`.

Entry conditions:

- Gate M0 passed for the active turn;
- for new feature/refactor/behavior-change requests, Gate B0 passed.

Actions:

1. Take user input and map it into `00-input.md`.
2. Fill `10-context-goals-nongoals.md`.
3. Create a skeleton of files `15..90`.
4. `go-domain-invariant-spec` creates the initial invariant register in `15-domain-invariants-and-acceptance.md`.
5. `go-reliability-spec` creates the initial reliability baseline in `55-reliability-and-resilience.md`.
6. Create the initial list of uncertainties in `80-open-questions.md`.

Gate G0 (entry to design):

- goal, scope, and non-goals are fixed;
- assumptions are explicitly listed;
- the baseline list of business invariants is created;
- the baseline list of reliability-risk scenarios is created;
- the open questions list is created.

## Phase 1. Baseline Architecture Frame

Owners: `go-architect-spec`, `go-domain-invariant-spec`, then `go-design-spec`.

Actions:

1. `go-architect-spec` defines the baseline design in `20-architecture.md`.
2. Adds an initial implementation plan in `60-implementation-plan.md`.
3. `go-domain-invariant-spec` refines invariants and acceptance criteria in `15-domain-invariants-and-acceptance.md`.
4. `go-design-spec` performs architecture sanity-check.
5. If there are findings, repeat the loop until resolved.

Gate G1:

- a consistent architecture frame exists;
- boundaries and ownership are defined;
- invariants and acceptance criteria are written in a verifiable form;
- implementation is broken into clear steps without hidden complexity.

## Phase 2. Spec Enrichment Loops (Main Loop)

This is the core phase. It can repeat N times until `80-open-questions.md` is empty.

Skill execution order inside one loop:

1. `go-architect-spec` -> opens the loop, sets pass goals and open-question priority.
2. `go-domain-invariant-spec` -> reviews the full spec package and edits any files with focus on invariants and acceptance criteria.
3. `api-contract-designer-spec` -> reviews the full spec package and edits any files.
4. `go-data-architect-spec` -> reviews the full spec package and edits any files.
5. `go-distributed-architect-spec` -> reviews the full spec package and edits any files.
6. `go-reliability-spec` -> reviews the full spec package and edits any files with focus on resilience/reliability contracts.
7. `go-db-cache-spec` -> reviews the full spec package and edits any files.
8. `go-observability-engineer-spec` -> reviews the full spec package and edits any files.
9. `go-security-spec` -> reviews the full spec package and edits any files.
10. `go-devops-spec` -> reviews the full spec package and edits any files.
11. `go-qa-tester-spec` -> reviews the full spec package and edits any files with focus on completeness of `70-test-plan.md`.
12. `go-performance-spec` -> reviews the full spec package and edits any files with focus on perf budget, measurable evidence thresholds, and explicit performance blockers/open questions.
13. `go-design-spec` -> performs an integrated pass on the full spec package and edits as needed.
14. `go-architect-spec` -> consolidates decisions, enforces architecture coherence across domains, closes/reframes questions in `80-open-questions.md`, and updates `90-signoff.md`.

Loop rules:

- No file-ownership lock: any `*-spec` skill may edit any spec file.
- Any new risk/uncertainty must be added to `80-open-questions.md`.
- Every closed question must be recorded in `90-signoff.md` as an accepted decision.
- Unresolved business invariants and unresolved reliability contracts are blockers for exiting the loop.

Gate G2 (Spec Sign-Off, Definition of Ready for coding):

- `80-open-questions.md` is empty;
- all decisions have an owner and rationale;
- API, data, distributed, cache, security, observability, and devops requirements are finalized;
- `15-domain-invariants-and-acceptance.md` contains a full invariant register and acceptance criteria;
- `55-reliability-and-resilience.md` contains timeout/retry/backpressure/degradation/shutdown/rollback policy;
- `60-implementation-plan.md` has no architecture-level TODOs;
- `70-test-plan.md` includes a test matrix for unit/integration/contract and expected critical scenarios;
- `70-test-plan.md` includes coverage for invariants and reliability fail-path scenarios;
- perf budget and perf acceptance criteria are fixed (latency/throughput/allocations for affected hot paths);
- `90-signoff.md` includes confirmation from all `*-spec` skills;
- `Spec Freeze` is active until code review is complete.

## Phase 2.5. Detailed Coder Plan Design

Owner: `go-coder-plan-spec`.

Actions:

1. Load approved feature artifacts (`15/20/30/40/50/55/60/70/80/90`) after `G2`.
2. Create `65-coder-detailed-plan.md` as execution-grade plan for coding:
   - atomic task graph;
   - task-level traceability to approved decisions/invariants/test obligations;
   - checkpoint contracts and clarification triggers;
   - outcome-oriented sequencing without low-level coding prescriptions.
3. Preserve coder autonomy in technical realization details:
   - no hard file-path lock-in;
   - no low-level code-mechanics prescriptions in the plan.

Gate G2.5 (Detailed Plan Ready):

- `65-coder-detailed-plan.md` exists and is complete;
- critical approved obligations are mapped to executable tasks with expected evidence;
- checkpoints and clarification contract are explicitly defined;
- no contradiction with frozen approved decisions.

## Phase 3. Code-Only Implementation

Owners: `go-coder`, `go-qa-tester`.

Rules:

1. `go-coder` writes production code strictly according to `65-coder-detailed-plan.md`, while preserving strategic intent and constraints from `60-implementation-plan.md`.
2. `go-qa-tester` writes test code strictly according to `70-test-plan.md`:
   - unit tests
   - integration tests
   - contract tests (when applicable)
3. Roles do not make new architecture decisions independently.
4. If an unresolved ambiguity is found during implementation:
   - stop implementation;
   - create a `Spec Clarification Request`;
   - return to Phase 2.
5. Update only technical implementation details without changing contract/architecture intent.
6. Implementation must preserve invariants from `15-domain-invariants-and-acceptance.md` and reliability contracts from `55-reliability-and-resilience.md`.
7. During `Spec Freeze`, spec files cannot be changed except through `Spec Reopen`.

Gate G3:

- code is implemented according to `65` and remains aligned with approved strategic constraints from `60` and contracts from `15/30/40/50/55`;
- unit/integration/contract tests are implemented according to `70-test-plan.md`;
- critical invariants and reliability fail-path scenarios are covered by tests;
- local tests/lint/vet pass;
- no unresolved spec clarification requests remain.

## Phase 4. Domain-Scoped Code Review

Review is domain-scoped (not "find anything"), and is executed only by `*-review` skills.

Recommended order:

1. `go-idiomatic-review`
2. `go-qa-review` (test quality and alignment with `70-test-plan.md`)
3. `go-domain-invariant-review` (preservation of business invariants and acceptance behavior from spec)
4. `go-language-simplifier-review`
5. `go-performance-review` (perf risks, benchmark/profile evidence for hot paths)
6. `go-concurrency-review` (when concurrency changes exist)
7. `go-db-cache-review` (when DB/cache changes exist)
8. `go-reliability-review` (timeout/retry/degradation/shutdown/rollback correctness)
9. `go-security-review`
10. `go-chi-review` (when `go-chi` transport behavior changes: route topology/order/policy/labels)
11. `go-design-review`

Each reviewer must:

- leave findings only in their domain;
- reference concrete file/line;
- provide practical fixes, not abstract advice;
- not edit spec files during code review.

Each reviewer is forbidden to:

- review business logic outside their domain;
- challenge already approved architecture without an explicit spec conflict;
- block a PR with non-actionable "just don't like it" comments.

If a spec-level mismatch is found:

- the reviewer creates a `Spec Reopen` record in `reviews/<feature-id>/code-review-log.md`;
- the task returns to Phase 2;
- after spec updates, a new implementation/review cycle starts.

Gate G4 (Code Quality Sign-Off):

- critical findings from all reviewer roles are resolved;
- no open `high`/`critical` findings remain;
- no open `Spec Reopen` remains;
- CI quality gates are green.

## Phase 5. Merge And Post-Fact Documentation

Owner: `go-architect-spec` (or task owner).

Actions:

1. Update `90-signoff.md` with links to PR/commit.
2. Record deviations from spec (if any) and rationale.
3. Create follow-up tasks for tech debt/improvements when identified.

## 6. Reviewer Focus Matrix

### `go-idiomatic-review`

- Focus: idiomatic Go, package boundaries, error/context handling, readability.
- Not focus: business rules and product scenarios.

### `go-design-review`

- Focus: code alignment with approved architecture, maintainability, complexity control.
- Not focus: deep domain correctness outside spec.

### `go-language-simplifier-review`

- Focus: code structure simplification, reduced cognitive load, naming/clarity.
- Not focus: architecture or contract changes.

### `go-qa-review`

- Focus: completeness and quality of unit/integration/contract tests, suite stability, traceability to `70-test-plan.md`.
- Not focus: architecture changes without spec escalation.

### `go-domain-invariant-review`

- Focus: correctness of domain invariants, state transitions, acceptance criteria, and corner cases against `15-domain-invariants-and-acceptance.md`.
- Not focus: micro-optimizations and low-level style unless they affect invariants.

### `go-performance-review`

- Focus: performance correctness of changes, measurement evidence (bench/profile), control of hot paths, allocations, and latency/throughput regressions.
- Not focus: functional business logic when there is no performance impact.

### `go-concurrency-review`

- Focus: goroutine lifecycle, cancellation, race/deadlock/leaks, bounded concurrency.
- Not focus: endpoint business meaning.

### `go-reliability-review`

- Focus: deadline/timeout propagation, retry budgets, jitter, backpressure, graceful startup/shutdown, degradation modes, rollout/rollback safety.
- Not focus: business rules unless reliability/failure behavior is affected.

### `go-db-cache-review`

- Focus: query discipline, transaction boundaries, N+1, cache correctness/invalidation/stampede.
- Not focus: general UI/API product design.

### `go-security-review`

- Focus: input validation, authz, injection/SSRF/path traversal, secrets, abuse resistance.
- Not focus: style/readability comments without security impact.

### `go-chi-review`

- Focus: `go-chi` transport behavior (`Route`/`Mount` topology, middleware ordering, `/metrics` conflict policy, `404/405/OPTIONS`, route-template extraction for logs/metrics/traces).
- Not focus: broader business/domain correctness and non-transport architecture topics.

## 7. Review Findings Format

Standard format for `reviews/<feature-id>/code-review-log.md`:

```text
[severity] [skill] [file:line]
Issue:
Impact:
Suggested fix:
Spec reference:
```

Severity:

- `critical`
- `high`
- `medium`
- `low`

## 8. Readiness Definitions

### Definition of Ready (to start coding)

- Gate G2 passed;
- Gate G2.5 passed;
- specification has no open questions;
- `Spec Freeze` is active;
- `65-coder-detailed-plan.md` is present and execution-ready.

### Definition of Done (for merge)

- Gate G3 and Gate G4 passed;
- no open `Spec Reopen`;
- mandatory CI quality gates are green;
- spec artifacts and code are synchronized.

## 9. Why This Workflow Works

- architecture decisions are made before coding;
- the coding agent is not distracted by high-level design;
- review becomes expert-driven and actionable;
- late-stage surprises are significantly reduced;
- maintainability and scalability improve through a predictable process.
