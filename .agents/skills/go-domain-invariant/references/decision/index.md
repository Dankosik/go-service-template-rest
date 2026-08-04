# Reference Selector

Choose the reference by the domain decision pressure it sharpens. Accepted cross-task terms and authority live in `docs/repo-architecture.md#domain-vocabulary` and `#source-of-truth-ownership`; read those before defining a term here.

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| Rules read as descriptive prose, or lack an owner, enforcement point, or false case. | [invariant-register-patterns.md](invariant-register-patterns.md) | Write falsifiable owner-backed rows, and choose application ordering or a database constraint by what a concurrent writer can break. |
| Lifecycle states, terminal states, reopen paths, or invalid moves decide what is legal. | [state-machine-and-transition-rules.md](state-machine-and-transition-rules.md) | Define forbidden movement and terminal policy, not the allowed list alone. |
| A rule has no stated behavior for its false case. | [invariant-violation-semantics.md](invariant-violation-semantics.md) | Pick one deterministic outcome and fail closed rather than reporting a scoped-empty success. |
| Retries, duplicates, replay, out-of-order arrival, or effect ordering are in scope. | [idempotency-replay-and-async-domain-rules.md](idempotency-replay-and-async-domain-rules.md) | Define domain sameness and the effect boundary before reaching for a key or dedupe. |

Downstream handoff needs no reference of its own: name the accepted rule, its violation outcome, and its proof, then hand representation to `go-api-contract`, storage authority and constraints to `go-data-architecture`, and durable coordination to `go-distributed`.
