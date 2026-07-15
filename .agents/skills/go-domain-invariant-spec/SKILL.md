---
name: go-domain-invariant-spec
description: "Use when business terms, invariants, state transitions, acceptance semantics, violation outcomes, or replay meaning must be decided before coding; Own domain policy and effect boundaries; Skip when the primary decision is API transport, schema/cache mechanics, distributed mechanism, or implementation planning."
---

# Go Domain Invariant Spec

Load the [shared specialist contract](../specialist-contract.md), then apply this business-policy boundary.

## Outcome And Boundary

Produce falsifiable domain decisions for allowed behavior, forbidden states, and observable acceptance outcomes. Own business terms and exact-value meaning; invariants; lifecycle/state transitions; authorization-independent business acceptance; side-effect eligibility and ordering; violation outcomes; and business idempotency identity.

Use accepted data authority without choosing schema, indexes, cache, or SQL mechanics. State durable-flow consequences without selecting saga, outbox/inbox, replay topology, or reconciliation; leave transport representation, security enforcement, reliability budgets, and repository placement to their owners.

## Domain Core

1. **Fix the language before the mechanism.** Define actors, business keys, units, exact values, time meanings, authority, and what “accepted,” “done,” and terminal mean from task and repository evidence; persistence or transport shape cannot define allowed behavior.
2. **Make every critical invariant owner-backed and observable.** Classify it as `local_hard_invariant` or `cross_service_process_invariant`, then name its owner, accepted source of truth, enforcement point, and pass/fail result.
3. **Model nontrivial lifecycles as guarded transitions.** For each transition, name trigger, preconditions, postconditions, allowed next states, forbidden next states, and terminal, timeout, stuck, or manual-intervention paths.
4. **Separate request acceptance from business effect.** State when side effects become eligible, which effects must precede others, what partial progress means, and which result is safe to expose; never turn a failed invariant, cancellation, timeout, or partial effect into false success.
5. **Define every false case.** Map violations deterministically to reject, deny, defer, compensate, forward-recover, manual intervention, or an explicit accepted risk; do not leave “cannot happen” as policy.
6. **Define business sameness before dedup mechanics.** Name the stable business action or effect boundary and the result of duplicate, replayed, late, out-of-order, or concurrent attempts before transport keys, queue dedup, or storage implementation.
7. **Preserve meaning through mixed versions.** Keep invariant and transition semantics stable during rollout; record compatibility assumptions, rollback limits, and any forced data or distributed consequence without designing that neighbor’s mechanism.

## Reference Selector

Choose references by the domain decision pressure they sharpen.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| Terms, actors, authority, or “done” are ambiguous. | [domain-language-and-boundaries.md](references/domain-language-and-boundaries.md) | Define the local policy boundary before writing rules. |
| Rules are descriptive or lack ownership and enforcement. | [invariant-register-patterns.md](references/invariant-register-patterns.md) | Produce falsifiable owner-backed invariants. |
| Lifecycle, terminal, timeout, or forbidden paths matter. | [state-machine-and-transition-rules.md](references/state-machine-and-transition-rules.md) | Define legal movement instead of narrating event order. |
| Acceptance or edge behavior is too vague for proof. | [acceptance-criteria-and-corner-cases.md](references/acceptance-criteria-and-corner-cases.md) | Make positive, negative, and edge outcomes observable. |
| A rule lacks behavior for the false case. | [invariant-violation-semantics.md](references/invariant-violation-semantics.md) | Choose a deterministic violation outcome. |
| Retry, replay, duplicates, async work, or reconciliation matter. | [idempotency-replay-and-async-domain-rules.md](references/idempotency-replay-and-async-domain-rules.md) | Define sameness, effect boundaries, and replay policy. |
| Stable rules need downstream traceability. | [api-data-reliability-test-traceability.md](references/api-data-reliability-test-traceability.md) | Map invariant IDs to necessary constraints and proof. |

## Return And Stop

Return only relevant domain terms and exact meanings; invariant register; state transitions; acceptance and corner cases; side-effect ordering; violation, duplicate, and replay semantics; forced neighbor constraints or proof; assumptions; and reopen conditions.

The domain decision is ready when downstream specialists can proceed without inventing product meaning. Stop when a critical term or business identity is undefined, invariants conflict, acceptance cannot be observed, effect ordering is ambiguous, or tenant, replay, migration, rollout, or cross-service ownership meaning is unsettled. Name the product/domain decision owner and do not choose schema/cache mechanics or durable recovery topology to hide the gap.
