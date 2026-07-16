# Invariant Violation Semantics

## Behavior Change Thesis
When loaded for symptom "a rule says what must be true but not what happens when it is false", this file makes the model choose a deterministic violation outcome instead of likely mistake "say handle the error, continue, or return success despite an invariant failure."

## When To Load
Load this when violation behavior is unspecified for authorization, tenant isolation, invalid transitions, dependency ambiguity, partial side effects, process failures, cross-service inconsistencies, stale projections, or source-of-truth drift.

## Decision Rubric
- Pick exactly one primary outcome for each violation: `reject`, `deny`, `defer_async`, `compensate`, `forward_recover`, `manual_intervention`, or `accepted_risk`.
- Use `reject` when the command/event is not accepted and the caller can correct or retry.
- Use `deny` for identity, tenant, ownership, or authorization failures. Fail closed; do not best-effort filter.
- Use `defer_async` only when the accepted state is honestly pending or asynchronous, not fake immediate success.
- Use `compensate` or `forward_recover` only after naming the durable state and side effects that already happened.
- Use `manual_intervention` when automation cannot prove safe correction. Use `accepted_risk` only with consequence and reopen trigger.
- Treat idempotent replay of the same already-accepted operation as replayed success, not as a fresh success after a failed invariant check.

## Imitate
```text
State: implementation
Trigger: orchestrator discovers a required planning artifact is missing
Invalid transition attempted: implementation -> done
Reason: proof path and task ledger are not stable
Violation outcome: reject closeout; reopen planning in the next eligible session
User-facing meaning: no completion claim is made
Downstream handoff: planning must create or waive the missing artifact before coding resumes
```

Copy the shape: attempted violation, reason, deterministic outcome, user-facing meaning, and handoff.

```text
| Violation | Outcome | Why |
| --- | --- | --- |
| Same idempotency key, different domain intent | reject or conflict | It is not a replay of the same logical operation. |
| Duplicate async event already applied | no-op or replay equivalent outcome | The invariant is one domain effect per logical event. |
| Source-of-truth mirror drift | forward-repair derived surface | The canonical artifact remains authority. |
| Authorization or tenant mismatch | deny | Correctness and data isolation require fail-closed behavior. |
| Cross-service correction cannot be guaranteed online | compensate or manual intervention | The invariant is process-level, not local hard consistency. |
```

Copy the mapping: outcome differs because the violated rule and repair authority differ.

## Reject
```text
If something goes wrong, handle the error and continue.
```

Failure: allows false success and hides whether the correct response is reject, deny, compensate, recover, or route to a human.

```text
If ownership cannot be verified, return an empty list.
```

Failure: this can mask an authorization or tenant invariant violation as success. The safer domain outcome is deny unless the product explicitly defines scoped emptiness as the rule.

## Agent Traps
- Do not let success mean "we logged a warning" after a critical invariant failed.
- Do not use `accepted_risk` to avoid designing the failure path; it needs consequence, owner, and reopen trigger.
- Do not classify dependency timeout as business rejection if the side effect may already have committed; model ambiguous, pending, reconciliation, or forward recovery.
- Do not make compensation sound magical. Name the prior allowed step and the corrective action.
- Do not leak transport details into the domain decision. Choose the outcome before choosing status codes or problem shapes.

## Validation Shape
Proof should assert the chosen violation outcome directly: invalid transition rejects, tenant mismatch denies, ambiguous commit enters pending/reconciliation, duplicate replay returns equivalent outcome, and failed compensation routes to the stated terminal or manual state.
