# Reference Selector

Choose references by the domain decision pressure they sharpen.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| Terms, actors, authority, or “done” are ambiguous. | [domain-language-and-boundaries.md](domain-language-and-boundaries.md) | Define the local policy boundary before writing rules. |
| Rules are descriptive or lack ownership and enforcement. | [invariant-register-patterns.md](invariant-register-patterns.md) | Produce falsifiable owner-backed invariants. |
| Lifecycle, terminal, timeout, or forbidden paths matter. | [state-machine-and-transition-rules.md](state-machine-and-transition-rules.md) | Define legal movement instead of narrating event order. |
| Acceptance or edge behavior is too vague for proof. | [acceptance-criteria-and-corner-cases.md](acceptance-criteria-and-corner-cases.md) | Make positive, negative, and edge outcomes observable. |
| A rule lacks behavior for the false case. | [invariant-violation-semantics.md](invariant-violation-semantics.md) | Choose a deterministic violation outcome. |
| Retry, replay, duplicates, async work, or reconciliation matter. | [idempotency-replay-and-async-domain-rules.md](idempotency-replay-and-async-domain-rules.md) | Define sameness, effect boundaries, and replay policy. |
| Stable rules need downstream traceability. | [api-data-reliability-test-traceability.md](api-data-reliability-test-traceability.md) | Map invariant IDs to necessary constraints and proof. |
