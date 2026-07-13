---
name: go-qa-tester-spec
description: "Design risk-based test strategy for Go services before coding. Use in test design when behavior needs stable TD-* scenarios, proof-level selection, fail-before expectations, invariant traceability, and executable quality gates. Skip writing tests, reviewing implementation diffs, and deciding unresolved product/API/data/security/reliability behavior."
---

# Go QA Tester Spec

## Outcome

Produce the smallest honest test strategy that makes non-obvious risks executable, deterministic, traceable, and ready for planning without inventing behavior during implementation.

## Method

1. Read approved behavior/design, the affected runtime boundary, nearby tests/fixtures, and live repository commands; for each material claim record whether existing proof is sufficient, must be strengthened, or needs a new scenario.
2. Choose the smallest set of complementary proof boundaries that jointly proves each claim: `unit`, `integration`, `contract`, `component/process`, `e2e-smoke`, or a repository-specific proof type.
3. In repository Test Design, use the [canonical Test Design Outputs](../../../docs/spec-first-workflow/phases/test-design.md#outputs); do not create a competing artifact schema here.
4. Cover fail and edge behavior; add abuse, retry, concurrency, reliability, data/cache/security, or distributed scenarios only when triggered by the accepted change.
5. Route unresolved semantics back to their owner instead of encoding them as test assumptions.

## Decision Rules

- Treat test strategy as risk selection and proof design, not a coverage checklist.
- Disposition every affected material accepted claim; omission is not evidence that a risk is untriggered. For each claim selected for proof, name a traceable oracle. An accepted change that triggers a material fail path with no proof or authorized residual-risk disposition is a blocker.
- Give each selected proof boundary a distinct observable; reject a boundary when it cannot establish its claim or merely duplicates another boundary.
- Every scenario must distinguish a material behavior or failure mechanism and name a plausible incorrect observable behavior or regression that its oracle would reject. Merge rows with the same risk, trigger, oracle, and reopen path.
- Derive the oracle from approved behavior or an independent reference, not from production logic. Reject existence, non-zero, or no-panic assertions unless that weak property is itself the accepted claim.
- An oracle includes the primary result plus relevant durable state, emitted effects, and forbidden or absent side effects. A command is proof only when its result can establish that oracle.
- A fail-before reason unavailable must name why fail-first adds no useful discrimination and the nearest existing falsifying signal; it never waives mandatory current-completion proof.
- Keep triggers deterministic and commands runnable in the named local, CI, or controlled target environment. Name isolation, cleanup, and deterministic controls for shared state, external resources, or nondeterminism. Never rely on test order. Do not use sleeps when a deterministic control exists; if deterministic control is infeasible, any bounded wait must target a named observable condition and record the reason, limitation, and mitigation.
- Do not write test code, `tasks.md`, implementation sequencing, or new domain/API/data/security/reliability policy.

## Reference Selector

References sharpen a triggered proof decision; they do not define scope or act as checklists. Reconstruct accepted behavior first. Load one reference by default and add another only for a materially independent unresolved proof decision.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| Proof level is unclear or broad e2e is proposed “for safety.” | [test-level-selection.md](references/test-level-selection.md) | Choose the smallest sufficient set of complementary boundaries. |
| The matrix is generic or happy-path-only. | [scenario-matrix-patterns.md](references/scenario-matrix-patterns.md) | Produce compact scenarios with observable outcomes. |
| Invariants or acceptance criteria lack traceable proof. | [invariant-and-acceptance-traceability.md](references/invariant-and-acceptance-traceability.md) | Map each claim to scenario and reopen trigger. |
| Timeout, retry, poison, backpressure, shutdown, or recovery matters. | [reliability-fail-path-test-obligations.md](references/reliability-fail-path-test-obligations.md) | Define deterministic failure triggers and lifecycle observables. |
| REST/OpenAPI, auth, validation, idempotency, or async 202 changed. | [api-contract-and-boundary-tests.md](references/api-contract-and-boundary-tests.md) | Choose boundary-observable contract proof. |
| SQL, cache, tenant storage, migration, outbox/inbox, replay, or reconciliation matters. | [data-cache-security-distributed-test-obligations.md](references/data-cache-security-distributed-test-obligations.md) | Select stateful and message observables instead of mocks. |
| Local/CI commands or proof limits are unclear. | [quality-gates-and-execution.md](references/quality-gates-and-execution.md) | Bind obligations to executable repository checks. |

## Output

Return only triggered scope, proof-boundary decisions, traceability, scenario matrix, executable quality gates, proof limits, residual risk, and reopen conditions. Omit adjacent domains that do not affect the strategy; when one blocks proof, name the exact unresolved owner and checkpoint.

## Success And Stop

Success means planning can map reviewed scenarios into tasks without inventing coverage or commands. Stop and reopen the owning decision when a critical invariant is untestable, semantics are unresolved, proof required for current completion is unavailable, a mandatory current-completion check has no executable path, or the strategy omits fail, retry/idempotency, concurrency, contract, or stateful proof required by the accepted change.
