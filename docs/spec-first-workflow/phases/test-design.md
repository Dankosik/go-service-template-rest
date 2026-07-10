# Test Design Phase

Detailed macro-phase companion for `docs/spec-first-workflow.md`. Read this when reviewed behavior needs explicit test scenarios, proof levels, fail-before expectations, quality gates, and independent QA review before `tasks.md` is drafted.

## Read When

- Specification review has passed or returned `CONCERNS` with named accepted proof obligations.
- Required compact design, system/integration design, Go code ownership design, and technical design review are complete or explicitly not expected.
- Behavior proof is non-obvious, multi-layered, protected-domain-sensitive, regression-prone, or too dense to leave as prose in `tasks.md`.
- Planning would otherwise choose test scenario classes, proof levels, fail-before signals, quality gates, or coverage boundaries.
- The next phase is recorded as `test-design`, or planning/task review reopened because `tasks.md` invented weak or untraceable tests.

## Inputs

- Reviewed `spec.md` and specification-review verdict, including accepted risks and proof obligations.
- Compact design in `spec.md` or approved `design/` bundle, plus technical-design-review verdict when separate design depth was triggered.
- Existing current durable `workflow-plan.md`, when present, and `workflow-plans/test-design.md` only when `ROUTING-PHASE-CONTROL` resolves `phase_control=required`.
- Existing `test-plan.md` when this is a repair pass.
- Repository command source, usually `docs/build-test-and-development-commands.md`, `Makefile`, and CI workflows.
- Nearby existing tests, fixtures, contract checks, migration checks, or generated drift checks when they affect executable proof.

## Outputs

- `test-plan.md` when triggered, with risk-based scenario IDs, selected proof levels, assertion atoms for broad protected-domain scenarios, pass/fail observables, fail-before expectations, command or manual proof shape, residual risks, and reopen targets.
- Typed no-plan decision when scenarios are small enough to live directly in `tasks.md`: `test-plan.md artifact_expectation=not_expected`, `artifact_state=absent`, `record_validity=current`, and `waiver_disposition=none`, with trigger rationale and reopen condition in the current owning artifact.
- Recorded `Test-design fan-out: complete | scoped_down | local_only | blocked | not_expected` with lane table or local-only rationale when the phase is non-trivial.
- A current independent test-design review verdict with exact reviewed revision, model route, findings, repair closure, and planning-readiness consequence when `test-plan.md` is expected.
- Existing durable master updates that route next to `planning`, or to the smallest reopen target when behavior or design is not testable yet; create or update `workflow-plans/test-design.md` only when `phase_control=required`.

When the no-plan decision is reached without durable master control, it must already be recorded in a current reviewed owning `spec.md` or design/review handoff. Test design does not edit an approved upstream artifact or create a new control root merely to store the decision; if the carrier is missing, reopen the smallest upstream owner before planning.

## Stop Rule

Do not self-approve an authored `test-plan.md`. Mark it review-ready, launch a fresh read-only QA review, repair every actionable test-design finding, and obtain fresh re-review. Stop only when the current verdict permits `artifact_state=approved`, the typed expectation is `not_expected`, or the macro phase is honestly blocked. Do not write `tasks.md`, production code, test code, migrations, generated artifacts, or implementation evidence in this phase.

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
artifact_expectation: not_expected
artifact_state: absent
record_validity: current
waiver_disposition: none
Trigger test: <why scenario matrix cannot change planning or proof quality>
Proof carrier: <tasks.md direct proof | inline direct path proof | other>
Reopen if: <condition that would make scenario design necessary>
```

## Test Plan Shape

Use the smallest shape that makes planning executable:

```markdown
# Test Plan

artifact_expectation: expected
artifact_state: draft | approved | blocked
record_validity: current

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

When a scenario covers multiple state-machine, money, security, lifecycle, retry, idempotency, duplicate, optional-identifier, or side-effect branches, split it or list assertion atoms before approval: source branch -> invariant -> forbidden regression or side effect -> proof signal. A broad scenario row is not planning-ready if `tasks.md` would need to decide which states, terminal groups, nullable identifiers, retry classes, duplicate/conflict cases, or no-side-effect branches actually require tests.

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

`local_only` is valid for authoring fan-out when the scenario work is one sequential reasoning chain and no concrete independent bounded question would materially improve scenario completeness, proof level, planning readiness, or implementation safety. It does not waive the independent test-design review. Repository-standing authorization covers read-only lanes; block only after both primary and configured fallback review surfaces are unavailable.

## Independent Test-Design Review And Repair

After authoring reaches `artifact_state=review_ready`, launch a fresh read-only `qa-agent` semantic review. Use `critical-reviewer-agent` only for one named approval-critical proof question whose protected-domain blast radius justifies high effort. The reviewer checks scenario completeness, source traceability, proof level, pass/fail observables, fail-before expectations or waivers, determinism, quality gates, and reopen ownership. It returns advisory `PASS`, `CONCERNS`, or `FAIL` and never edits `test-plan.md`.

The test-design root reconciles findings and repairs every actionable test-design-owned defect in the same session. A missing behavior, mechanism, ownership, or approval decision reopens the appropriate earlier macro phase instead of being invented here. After repair, mark the prior verdict stale and launch a fresh reviewer context at the same or stronger tier against the changed revision. `CONCERNS` may close only when it contains named accepted risks or downstream proof obligations, not an omitted in-scope scenario repair.

Record:

```text
Test-design review procedural_gate_state: pending | complete | blocked
Test-design review_verdict: pending | PASS | CONCERNS | FAIL
Reviewed revision / cycle: <artifact anchor and attempt>
Reviewer / model route: <fresh agent thread or process and effective profile>
Finding closure: <finding id -> repair/evidence anchor -> fresh status>
Planning readiness: <ready with obligations | blocked and reopen target>
```

## Handoff To Planning

Planning may start when:

- `test-plan.md` is `approved`, or test design is explicitly `not expected` with trigger rationale;
- the independent test-design review verdict is current `PASS` or eligible `CONCERNS` when a test plan exists;
- every material scenario has a stable ID, source anchor, selected proof level, pass/fail observable, proof command or manual proof shape, and reopen target;
- broad protected-domain scenarios are split or carry assertion atoms granular enough for planning to map into `Implementation obligations` without inventing coverage;
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
