# Specialist Arbitration

Use this non-triggerable file only when multiple domain axes plausibly own one defect. Route by the violated contract, not the shared symptom; select the decision branch for unset or changing policy and the review branch for conformance to accepted policy.

| Decisive axis | Domain skill |
| --- | --- |
| Go/stdlib semantics: panic, nil/zero, aliasing, receivers, context API/lifetime, errors, or resources | `go-idiomatic` |
| Correct local behavior whose control flow, predicates, naming, or helper shape is hard to change | `go-language-simplifier` |
| Explicit harsh whole-diff review finds structural overbuild, mixed responsibility, or missed deletion | `go-structural-quality` |
| Package/file owner, dependency direction, source of truth, or implementation seam | `go-implementation-ownership` |
| Happens-before, shared state, goroutine/channel/timer, cancellation-unblock, or join protocol | `go-concurrency` |
| Deadline, retry, overload, degradation, readiness, startup, drain, shutdown, recovery, or rollout | `go-reliability` |
| Durable-boundary replay, partial failure, ordering, compensation, redrive, or reconciliation | `go-distributed` |

Replacing caller context on a blocking dependency path is reliability when it breaks an accepted end-to-end budget; storing call-scoped context or accepting invalid nil context is idiomatic Go. A goroutine that cannot observe cancellation and join is concurrency; readiness and rolling shutdown are reliability; replay after durable process loss is distributed. Local readability remains simplification, explicit whole-diff overbuild remains structural quality, and wrong ownership remains implementation ownership.
