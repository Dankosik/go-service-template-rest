---
name: go-domain-invariant-review
description: "Use when changed Go can alter business rules, lifecycle transitions, acceptance outcomes, violation behavior, or side effects; Own preservation of accepted domain invariants and meaning; Skip when the primary defect is transport, data/cache mechanics, security enforcement, or test structure."
---

# Go Domain Invariant Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Target And Boundary

Review changed behavior against approved business rules from task artifacts, domain docs, tests, fixtures, callers, and adjacent code, in that order. If no approved artifact exists, label the smallest rule inferred from code-visible evidence; never import a generic DDD rule as local authority.

Own business invariants, legal transitions, acceptance/rejection meaning, and business side-effect safety. Do not redesign the model, accept eventual repair for a hard invariant without an explicit process contract, or duplicate transport, DB/cache, security, reliability, architecture, or test-strategy findings.

## Domain Defect Invariants

1. Every affected invariant has an enforcement point on every accepting, mutating, constructing, or persistence path; no path may accept, persist, or expose forbidden state.
2. Changed lifecycle code permits only approved transitions, preserves terminal states and postconditions, and keeps success, rejection, no-op, and already-applied semantics deterministic. Commands and events retain their distinct accepted handling.
3. All domain preconditions run before irreversible, durable, or externally visible effects. A rejected operation produces no prohibited charge, refund, inventory, entitlement, event, webhook, email, or save; state and effects cannot split into a forbidden mixed outcome.
4. Retry, replay, duplicate, stale, delayed, dependency-failure, side-channel, and out-of-order paths preserve the same transition rules and idempotency meaning as the first attempt. Tie any defect to the exact repeated, skipped, or overwritten business effect rather than requesting generic deduplication or ordering.
5. Success, failure, and corner cases preserve approved domain errors and observable business meaning. Treat vocabulary drift as a defect only when it changes state, obligation, ownership, eligibility, amount meaning, audit interpretation, or another locally distinct concept.
6. Require focused negative proof when a changed invariant, transition, rejection, duplicate, or side-effect rule could regress without detection; identify the business failure that the missing assertion would allow. Leave broad test strategy and test shape to `go-test-review`.

## Symptom-Driven References

The examples are review lenses, not reusable business rules. Select the one that changes the finding, then cite local authority or state the bounded inference.

| Symptom | Load | Distinction preserved |
| --- | --- | --- |
| Construction, mutation, save, guard, or direct field update may admit impossible state. | [invariant-preservation-review.md](references/invariant-preservation-review.md) | Prove a local bypass, not generic aggregate reshaping. |
| Status, lifecycle guard, terminal state, transition table, or event-driven state update changed. | [state-transition-review.md](references/state-transition-review.md) | Prove an illegal or missing move, not demand a formal state machine. |
| A command, error, no-op, duplicate, event, or validation path changes accepted/rejected/ignored/already-applied meaning. | [acceptance-and-rejection-semantics.md](references/acceptance-and-rejection-semantics.md) | Preserve exact business acceptance semantics, not error-style taste. |
| An external or durable effect can outlive a rejected or partially completed operation. | [preconditions-side-effects-and-partial-failure.md](references/preconditions-side-effects-and-partial-failure.md) | Prove guard/effect ordering or a forbidden mixed outcome before escalating design. |
| Retry, replay, idempotency, stale input, backfill, optimistic concurrency, or reordered delivery changed. | [retry-duplicate-and-reorder-domain-risks.md](references/retry-duplicate-and-reorder-domain-risks.md) | Tie duplicate/order handling to one concrete business consequence. |
| A rename changes states, obligations, ownership, eligibility, totals, or lifecycle terms. | [domain-language-and-meaning-drift.md](references/domain-language-and-meaning-drift.md) | Separate semantic drift from readability taste. |
| Changed business behavior lacks a falsifying negative-path assertion. | [domain-test-traceability.md](references/domain-test-traceability.md) | Name the regression that can pass, not generic coverage work. |

## Findings, Escalation, And Stop

Each finding names the violated invariant, transition, acceptance rule, or side-effect contract; the approved or explicitly inferred evidence; the exact business impact; and the smallest owner-preserving correction. `critical` means a confirmed invariant violation, forbidden transition, silent corruption/loss, prohibited side effect, or forbidden mixed outcome; `high` means strong evidence of a material failure-path or corner-case break.

Stop and hand off rather than changing policy: invariant, transition, or acceptance meaning to `go-domain-invariant-spec`; API-visible behavior or errors to `go-api-contract-spec`; transaction/cache/consistency design to `go-db-cache-spec`; retry/degradation to `go-reliability-spec`; durable replay/reconciliation to `go-distributed-spec`; and placement or responsibility drift to `go-implementation-ownership-spec`. For accepted supporting contracts, hand transaction/query/cache mechanics to `go-db-cache-review`, timeout/retry/degradation enforcement to `go-reliability-review`, authn/authz/tenant/object enforcement to `go-security-review`, and test-shape depth to `go-test-review` without duplicating their findings.
