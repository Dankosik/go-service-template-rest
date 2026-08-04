# Reference Selector

Each row names a place where this repository already made the decision and its
implementation is the constraint. State the expected behavior change before
loading. Both branches use this selector: the branch decides what you return,
the affected layer decides what you read.

Retry eligibility, backoff, jitter, `Retry-After`, and ambiguous outcomes have
no reference here — [`external-api-integration`](../../../../docs/universal-disciplines/external-api-integration/SKILL.md)
owns them, and `internal/infra/httpclient/retry.go` is the local implementation.
Degradation shape, rollout sequencing, and proof selection have none either:
decide them against the affected code, with `go-delivery-platform` owning
release mechanics and `go-test-strategy` owning proof level. Adding a reference
back requires a decision it would change.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| An outbound call, pooled acquire, query, or wait added to a request path; a handler that must fail before its caller does. | [request-budget.md](request-budget.md) | Spend a slice of the 8s budget the handler already carries, under the sub-budget relation config validation enforces, instead of choosing a timeout locally. |
| Concurrency, fan-out, queued work, a new pooled dependency, a rejection path, or a proposed limiter, bulkhead, or circuit breaker. | [admission-and-overload.md](admission-and-overload.md) | Bound the work in the listener, shedding, or rate-limit layer that already exists, and keep 503, 429, and 504 meaning what they already mean. |
| Health endpoints, what readiness depends on, signal handling, drain sequencing, or a step added to teardown. | [readiness-drain-shutdown.md](readiness-drain-shutdown.md) | Add a probe to the cached refresher and a stage to the one clamped shutdown deadline, instead of I/O in the probe handler and a timeout of its own. |
