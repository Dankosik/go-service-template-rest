# Reference Selector

The examples are review lenses, not reusable business rules. Select the one that changes the finding, then cite local authority or state the bounded inference.

| Symptom | Load | Distinction preserved |
| --- | --- | --- |
| Construction, mutation, save, guard, or direct field update may admit impossible state. | [invariant-preservation-review.md](invariant-preservation-review.md) | Prove a local bypass, not generic aggregate reshaping. |
| Status, lifecycle guard, terminal state, transition table, or event-driven state update changed. | [state-transition-review.md](state-transition-review.md) | Prove an illegal or missing move, not demand a formal state machine. |
| A command, error, no-op, duplicate, event, or validation path changes accepted/rejected/ignored/already-applied meaning. | [acceptance-and-rejection-semantics.md](acceptance-and-rejection-semantics.md) | Preserve exact business acceptance semantics, not error-style taste. |
| An external or durable effect can outlive a rejected or partially completed operation. | [preconditions-side-effects-and-partial-failure.md](preconditions-side-effects-and-partial-failure.md) | Prove guard/effect ordering or a forbidden mixed outcome before escalating design. |
| Retry, replay, idempotency, stale input, backfill, optimistic concurrency, or reordered delivery changed. | [retry-duplicate-and-reorder-domain-risks.md](retry-duplicate-and-reorder-domain-risks.md) | Tie duplicate/order handling to one concrete business consequence. |
| A rename changes states, obligations, ownership, eligibility, totals, or lifecycle terms. | [domain-language-and-meaning-drift.md](domain-language-and-meaning-drift.md) | Separate semantic drift from readability taste. |
| Changed business behavior lacks a falsifying negative-path assertion. | [domain-test-traceability.md](domain-test-traceability.md) | Name the regression that can pass, not generic coverage work. |
