# Test Design Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when reviewed behavior needs explicit test scenarios, proof levels, fail-before expectations, or quality gates before `tasks.md` is drafted.

## Read When

- Specification review has passed or returned `CONCERNS` with named accepted proof obligations.
- Required compact design, system/integration design, Go code ownership design, and technical design review are complete or explicitly not expected.
- Behavior proof is non-obvious, multi-layered, protected-domain-sensitive, regression-prone, or too dense to leave as prose in `tasks.md`.
- Planning would otherwise choose test scenario classes, proof levels, fail-before signals, quality gates, or coverage boundaries.
- The next phase is recorded as `test-design`, or planning/task review reopened because `tasks.md` invented weak or untraceable tests.

## Inputs

- Reviewed `spec.md` and specification-review verdict, including accepted risks and proof obligations.
- Compact design in `spec.md` or approved `design/` bundle, plus technical-design-review verdict when separate design depth was triggered.
- Existing `workflow-plan.md` and `workflow-plans/test-design.md` when phase-local routing exists.
- Existing `test-plan.md` when this is a repair pass.
- Repository command source, usually `docs/build-test-and-development-commands.md`, `Makefile`, and CI workflows.
- Nearby existing tests, fixtures, contract checks, migration checks, or generated drift checks when they affect executable proof.

## Outputs

- `test-plan.md` when triggered, with risk-based scenario IDs, selected proof levels, pass/fail observables, fail-before expectations, command or manual proof shape, residual risks, and reopen targets.
- Explicit `test-design: not expected` rationale when scenarios are small enough to live directly in `tasks.md`.
- Recorded `Test-design fan-out: complete | scoped_down | local_only | blocked | not_expected` with lane table or local-only rationale when the phase is non-trivial.
- Workflow-control updates that route next to `planning`, or to the smallest reopen target when behavior or design is not testable yet.

## Stop Rule

Stop when test design is approved, explicitly not expected, or blocked. Do not write `tasks.md`, production code, test code, migrations, generated artifacts, or implementation evidence in this phase.

## Ownership

Test design owns risk selection and proof design. It translates approved behavior into executable scenario obligations before planning slices tasks.

It owns:

- Scenario classes for happy path, fail path, edge case, abuse/negative, retry/idempotency, concurrency, data/cache/security/distributed, and API/contract behavior when relevant.
- Proof level selection: `unit`, `integration`, `contract`, or `e2e-smoke`.
- Pass/fail observables: response, persisted state, emitted message, generated output, log/metric expectation when part of the contract, or visible state transition.
- Fail-before expectation for proof-first tasks, unless a `Proof-first waiver:` is justified.
- Repository-executable command or manual proof shape for each material scenario.
- Residual risk, coverage limit, freshness or negative-proof requirement, and reopen target for missing or failing proof.

It does not own:

- Product semantics, domain invariants, public API contract, source-of-truth, data model, retry policy, rollout policy, package/file placement, cleanup ownership, or test code implementation.
- Task order, diff slicing, checkpoint placement, or implementation handoff wording.
- Broad coverage goals such as "increase coverage" without behavior-specific risk.

If this phase cannot write honest scenarios because behavior, contract, failure semantics, state ownership, test ownership, or executable command support is unclear, reopen specification, system/integration design, Go code ownership design, technical design review, or repository tooling work according to the missing owner.

## Trigger Rules

Create or repair `test-plan.md` when any of these are true:

- a public API, generated contract, event, migration, cache, security, auth, tenant, money, quota, concurrency, async, retry, lifecycle, rollout, or compatibility surface changes;
- accepted review `CONCERNS` include proof obligations that span more than one test level or failure mode;
- a bug fix or regression needs fail-before proof beyond one obvious unit or package test;
- behavior has meaningful negative, edge, abuse, idempotency, retry, concurrency, timeout, cancellation, or stale-data cases;
- final validation would otherwise become the first meaningful proof for behavior;
- planning would need to decide whether proof belongs in unit, integration, contract, or e2e smoke tests;
- existing test ownership or fixture strategy is unclear enough that Go code ownership design should reopen.

Do not create `test-plan.md` when:

- the work is direct path and one obvious proof command or manual check covers the claim;
- lean-local behavior has only one or two obvious scenarios that can be traced directly in `tasks.md`;
- the change is docs, mechanical config, generated drift, or cleanup-only and a targeted proof/negative search is enough;
- the artifact would be headings without scenario IDs, data shape, observables, proof commands, or reopen targets.

When not creating `test-plan.md`, record a compact rationale:

```text
Test design: not expected
Trigger test: <why scenario matrix cannot change planning or proof quality>
Proof carrier: <tasks.md direct proof | inline direct path proof | other>
Reopen if: <condition that would make scenario design necessary>
```

## Test Plan Shape

Use the smallest shape that makes planning executable:

```markdown
# Test Plan

Status: draft | approved | blocked

## Inputs

Reviewed spec:
Specification review:
Technical design/review:
Accepted proof obligations:
Repository command source:

## Strategy

Chosen proof levels:
Rejected weaker or broader levels:
Quality gates:
Residual coverage limits:

## Scenario Matrix

| ID | Source | Risk / invariant | Level | Setup / data | Action | Expected observable | Fail-before signal | Proof command | Reopen target |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| TD-001 | <spec/review/design anchor> | <risk> | unit/integration/contract/e2e-smoke | <preconditions> | <action> | <pass/fail signal> | <expected failing assertion or waiver> | <command/read/manual proof> | <phase/artifact> |

## Coverage Obligations

Happy path:
Fail paths:
Edge cases:
Abuse / negative:
Retry / idempotency / concurrency:
Reliability / lifecycle:
API / contract:
Data / cache / security / distributed:
Generated / mirrored / cleanup proof:

## Planning Handoff

Tasks must reference scenario IDs:
Checkpoint gates:
Proof-first waivers:
Residual risks:
Reopen target:
```

Use only sections that carry real obligations. Delete or mark irrelevant coverage categories `not_applicable` with one-line rationale; do not leave empty headings as proof of coverage.

## Fan-Out

Before approving non-trivial test design, identify independent proof-risk questions. Use read-only lanes when a specialist could change scenario selection, proof level, quality gate, or reopen target.

Typical lanes:

- QA/test strategy with `go-qa-tester-spec`.
- API/contract proof when HTTP, generated, idempotency, status, problem details, or compatibility behavior changed.
- Data/cache/security/distributed proof when state, tenant isolation, cache correctness, replay, or ordering behavior changed.
- Reliability/concurrency proof when deadlines, cancellation, retry, async recovery, shutdown, or parallel requests matter.
- Performance proof when hot-path, allocation, contention, or capacity behavior is part of the accepted scope.

Record the gate in this shape:

```text
Test-design fan-out: complete | scoped_down | local_only | blocked | not_expected
Candidate proof seams: <scenario/proof-level seams considered>
Lane table: <lane id, lens/domain, proof-risk question, artifact section it could change, planning blocker if unanswered, skill/no-skill, inspect-first target, read-only enforcement, status>
Collapsed seams: <duplicate or consequence-only seams folded into the integrated pass>
Fan-in outcome: <orchestrator reconciliation that changes or confirms the test plan>
Readiness consequence: <ready for planning | blocked | reopen specification/system-integration-design/go-code-ownership-design/technical-design-review>
```

`local_only` is valid only with candidate-lane analysis proving no omitted lane can change scenario completeness, proof level, planning readiness, or implementation safety. Missing explicit subagent authorization is not a valid local-only rationale when a required read-only lane would otherwise run.

## Handoff To Planning

Planning may start when:

- `test-plan.md` is `approved`, or test design is explicitly `not expected` with trigger rationale;
- every material scenario has a stable ID, source anchor, selected proof level, pass/fail observable, proof command or manual proof shape, and reopen target;
- fail-before expectations or waivers are explicit for proof-first behavior tasks;
- quality gates are executable in repository-supported tooling or honestly marked as unavailable with residual risk;
- no scenario requires planning or implementation to decide product semantics, source of truth, package/file ownership, rollout policy, or test ownership.

The handoff must tell planning:

- which scenario IDs must become proof-first or test tasks;
- which scenario IDs can be grouped under one reviewable diff story and why;
- which checkpoint gates are needed before later tasks rely on earlier proof;
- which residual risks or accepted concerns must map into `tasks.md`, `rollout.md`, or a non-task rationale;
- which failing or missing proof reopens test design versus an upstream specification/design owner.

If planning would still need to choose scenario classes, proof levels, fail-before signals, quality gates, or reopen targets, test design is not complete.
