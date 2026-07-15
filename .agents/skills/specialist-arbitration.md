# Specialist Arbitration

Use this non-triggerable file only when multiple specialist axes plausibly own
one defect. Route by the violated contract, not the shared symptom; an unset or
changed policy goes to the named specification owner and is never invented in
review.

| Decisive defect axis | Primary review owner | Missing or changed decision owner |
| --- | --- | --- |
| Go/stdlib semantics: panic, nil/zero, aliasing, receiver/method set, context API/lifetime, error, or resource behavior | `go-idiomatic-review` | nearest affected domain spec; `go-implementation-ownership-spec` only for placement |
| Correct local behavior whose control flow, predicates, naming, or helper shape is hard to understand/change | `go-language-simplifier-review` | owning behavior spec if simplification changes behavior |
| Explicit harsh whole-diff review finds structural overbuild, abstraction cost, mixed responsibility, or missed deletion | `go-structural-quality-review` | `go-implementation-ownership-spec` for responsibility/placement; otherwise owning behavior spec |
| Package/file owner, dependency direction, source of truth, or implementation seam is wrong | `go-implementation-ownership-review` | `go-implementation-ownership-spec`; `go-system-architecture-spec` if topology or system authority changes |
| Happens-before, shared state, goroutine/channel/timer, cancellation-unblock, or join protocol | `go-concurrency-review` | `go-reliability-spec` for lifecycle policy; `go-implementation-ownership-spec` only for placement |
| Accepted deadline, retry, overload, degradation, readiness, startup, drain, shutdown, recovery, or rollout behavior | `go-reliability-review` | `go-reliability-spec` |
| Durable-boundary replay, partial failure, ordering, compensation, redrive, or reconciliation | `go-distributed-review` | `go-distributed-spec` |

Replacing caller context on a blocking dependency path is reliability when it
breaks the accepted end-to-end budget; storing call-scoped context or accepting
an invalid nil context is idiomatic Go. A goroutine that cannot observe
cancellation and join, including a WaitGroup/channel shutdown hang, is
concurrency; readiness, drain, and rolling shutdown are reliability; resume or
replay after durable process loss is distributed. Local readability remains
simplification, explicit whole-diff overbuild remains structural quality, and
wrong ownership/dependency direction remains implementation ownership.
