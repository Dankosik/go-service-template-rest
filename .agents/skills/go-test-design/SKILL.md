---
name: go-test-design
description: "Use when accepted behavior needs risk-based scenarios, proof levels, deterministic oracles, fail-before expectations, invariant traceability, or executable gates before coding; Own test strategy and proof design; Skip when behavior is unresolved, tests must be implemented, a diff needs test review, or completion claims need validation."
---

# Go Test Design

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Outcome And Boundary

Produce the smallest honest strategy that makes accepted, non-obvious risks executable, deterministic, traceable, and ready for planning. Own risk-to-scenario mapping, proof-level choice, oracles, fixtures/controls, fail-before expectations, executable gates, and residual proof limits; do not implement tests, write `tasks.md`, sequence implementation, or invent product, API, data, cache, security, reliability, or distributed policy.

## Owned Core

- Treat strategy as risk selection and proof design, not a coverage checklist. Read accepted behavior/design, affected runtime boundaries, nearby tests/fixtures, and live repository commands. Disposition every material claim as sufficiently proved, needing stronger proof, or needing a new scenario; omission is not evidence that a risk is untriggered.
- Choose the smallest complementary set of `unit`, `integration`, `contract`, `component/process`, `e2e-smoke`, or repository-specific boundaries. Give each a distinct observable and reject broad e2e or duplicate levels that cannot establish the claim.
- Trace each selected claim/invariant to a scenario, deterministic oracle, executable command, proof boundary, residual gap, and reopen owner. In repository Test Design use the [canonical Test Design Outputs](../../../docs/spec-first-workflow/phases/test-design.md#outputs), not a competing artifact schema.
- Cover positive, negative, boundary, fail, and edge behavior; add abuse, retry/idempotency, concurrency, lifecycle, API, data/cache, tenant/security, migration, outbox/inbox, replay, redrive, and reconciliation scenarios only when the accepted change triggers them. A material fail path without proof or authorized residual risk blocks readiness.
- Make each scenario discriminate a plausible incorrect result or failure mechanism, and merge rows with the same risk, trigger, oracle, and reopen path. Derive its oracle from accepted behavior or an independent authority, never by copying production logic; reject existence, non-zero, or no-panic assertions unless that weak property is the accepted claim.
- Include the primary result plus relevant durable state, emitted effects, and forbidden or absent side effects. A command is evidence only when its result can establish that full oracle.
- Keep commands runnable in the named local, CI, or controlled target environment. Define controlled fixtures, isolation, cleanup, clocks/randomness, failure injection, and external-resource setup. Never depend on test order or sleep when a deterministic control exists; when only a bounded wait is feasible, target a named observable condition and record reason, limit, and mitigation.
- State the expected fail-before discriminator. If fail-first adds no useful discrimination, name why, the nearest falsifying signal, and the still-mandatory current-completion proof.

## Symptom-Driven References

References sharpen a triggered proof decision after accepted behavior is reconstructed; they neither define scope nor act as checklists.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| Proof level is unclear or broad e2e is proposed “for safety.” | [test-level-selection.md](references/test-level-selection.md) | Choose the smallest sufficient complementary boundaries. |
| The matrix is generic or happy-path-only. | [scenario-matrix-patterns.md](references/scenario-matrix-patterns.md) | Produce compact scenarios with observable outcomes. |
| Invariants or acceptance criteria lack traceable proof. | [invariant-and-acceptance-traceability.md](references/invariant-and-acceptance-traceability.md) | Map each claim to scenario and reopen trigger. |
| Timeout, retry, poison, backpressure, shutdown, or recovery matters. | [reliability-fail-path-test-obligations.md](references/reliability-fail-path-test-obligations.md) | Define deterministic failure triggers and lifecycle observables. |
| REST/OpenAPI, auth, validation, idempotency, or async `202` changed. | [api-contract-and-boundary-tests.md](references/api-contract-and-boundary-tests.md) | Choose boundary-observable contract proof. |
| SQL, cache, tenant storage, migration, outbox/inbox, replay, or reconciliation matters. | [data-cache-security-distributed-test-obligations.md](references/data-cache-security-distributed-test-obligations.md) | Select stateful and message observables rather than mocks. |
| Local/CI commands or proof limits are unclear. | [quality-gates-and-execution.md](references/quality-gates-and-execution.md) | Bind obligations to executable repository checks. |

## Return And Stop

Return only triggered scope, claim/invariant traceability, proof-level decisions, the risk/scenario matrix, deterministic fixtures and oracles, fail-before expectations, executable quality gates, proof limits, residual risk, and reopen conditions. Record only the forced owner and proof consequence for adjacent domains.

Success means planning can map reviewed scenarios into tasks without inventing coverage or commands. Stop on unresolved semantics, a critical untestable invariant, unavailable mandatory current-completion proof, a mandatory gate without an executable path, or omitted fail, retry/idempotency, concurrency, contract, abuse, or stateful proof required by the accepted change.
