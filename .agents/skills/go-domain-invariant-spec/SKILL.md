---
name: go-domain-invariant-spec
description: "Design domain-invariant-first specifications for Go services. Use when behavior needs explicit business invariants, lifecycle transitions, acceptance semantics, violation outcomes, or replay rules before coding. Skip local fixes, transport/schema mechanics, and implementation planning."
---

# Go Domain Invariant Spec

## Outcome

Produce spec-ready domain decisions that make allowed behavior, forbidden states, and acceptance outcomes falsifiable without choosing transport, storage, or infrastructure mechanics.

## Method

1. Establish the domain terms, actors, business keys, and source-of-truth owner from current task and repository evidence.
2. Write each critical invariant as one owner-backed rule with type, enforcement point, and observable pass/fail condition.
3. Model nontrivial lifecycles as states and guarded transitions, including terminal, timeout, stuck, and forbidden paths.
4. Define violation, duplicate, replay, out-of-order, and concurrency outcomes only where the behavior permits them.
5. Emit only downstream constraints or proof obligations forced by the domain decision; hand off unresolved API, data, reliability, security, distributed, or QA choices to their owners.

## Decision Rules

- Treat the domain model, not transport or persistence shape, as the source of allowed behavior.
- Classify each invariant as `local_hard_invariant` or `cross_service_process_invariant`.
- Give every critical rule an owner, source of truth, enforcement point, and observable result.
- For each transition, name trigger, preconditions, postconditions, allowed next states, and forbidden next states.
- Map violations to a deterministic outcome: reject, deny, defer, compensate, forward-recover, manual intervention, or explicitly accepted risk.
- Never convert failed invariant checks, cancellation, timeout, or partial side effects into false success.
- Define domain sameness and effect boundaries before transport idempotency keys, queue dedupe, or storage mechanisms.
- Preserve invariant meaning across mixed-version rollout; record rollback limits and migration assumptions when relevant.
- Use `constraint_only`, `proof_only`, or `no new decision required in <domain>` when an adjacent domain does not need a new decision now.

## Reference Selector

Load at most one reference by default. Load a second only for an independent decision pressure.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| Terms, actors, authority, or “done” are ambiguous. | [domain-language-and-boundaries.md](references/domain-language-and-boundaries.md) | Define the local policy boundary before writing rules. |
| Rules are descriptive or lack ownership and enforcement. | [invariant-register-patterns.md](references/invariant-register-patterns.md) | Produce falsifiable owner-backed invariants. |
| Lifecycle, terminal, timeout, or forbidden paths matter. | [state-machine-and-transition-rules.md](references/state-machine-and-transition-rules.md) | Define legal movement instead of narrating event order. |
| Acceptance or edge behavior is too vague for proof. | [acceptance-criteria-and-corner-cases.md](references/acceptance-criteria-and-corner-cases.md) | Make positive, negative, and edge outcomes observable. |
| A rule lacks behavior for the false case. | [invariant-violation-semantics.md](references/invariant-violation-semantics.md) | Choose a deterministic violation outcome. |
| Retry, replay, duplicates, async work, or reconciliation matter. | [idempotency-replay-and-async-domain-rules.md](references/idempotency-replay-and-async-domain-rules.md) | Define sameness, effect boundaries, and replay policy. |
| Stable rules need downstream traceability. | [api-data-reliability-test-traceability.md](references/api-data-reliability-test-traceability.md) | Map invariant IDs to necessary constraints and proof. |

## Output

Return only the relevant sections: domain terms and scope; invariant register; state transitions; acceptance and corner cases; violation/replay semantics; forced downstream handoffs or proof obligations; assumptions and reopen conditions. Record viable options and rejection reasons only when a real live fork exists.

## Success And Stop

Success means planning and specialist design can consume the domain rules without inventing product meaning. Stop and reopen the owning decision when a critical term is undefined, invariants conflict, acceptance cannot be observed, a cross-service rule lacks an owner/recovery stance, or identity, tenant, replay, migration, or rollout semantics remain materially ambiguous.
