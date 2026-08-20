# Specialist Arbitration

Use only when multiple domain axes plausibly own one defect. Route by the
violated contract, not the shared symptom; select Decision for unset/changing
policy and Review for conformance to accepted policy.

| Decisive axis | Domain skill |
| --- | --- |
| Go/stdlib semantics, context, errors, aliasing, nil/zero, or resources | `go-idiomatic` |
| Readability, control flow, predicates, names, or helper shape | `go-language-simplifier` |
| Whole-diff overbuild, mixed responsibility, or missed deletion | `go-structural-quality` |
| Package/file owner, dependency direction, source, or seam | `go-implementation-ownership` |
| Happens-before, shared state, goroutine/channel/timer, cancellation, or join | `go-concurrency` |
| Deadline, retry, overload, readiness, startup, drain, shutdown, or rollout | `go-reliability` |
| Durable replay, partial failure, ordering, compensation, redrive, or reconciliation | `go-distributed` |

Choose the owner whose contract is violated; symptom vocabulary does not create
another discipline.
