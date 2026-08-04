# Request Budget

## Behavior Change Thesis
When loaded for symptom `an outbound call, query, or wait added to a request path`, this file makes the model spend a slice of a budget the handler already has instead of likely mistake `add a context.WithTimeout with a locally chosen duration and call the call bounded`.

## When To Load
Load when a change adds or re-bounds waiting on a request path: a dependency call, a pooled acquire, a query, a poll, or a handler that must fail before its caller does.

## What Already Owns This
`RequestTimeout` in `internal/infra/http/middleware_timeout.go` installs `http.request_timeout` (8s) on `r.Context()` for every handler. A new call is therefore already bounded. What it may need is a *smaller* bound — and only when it must fail before the whole request does, leaving the rest of the budget to something else.

Expiry answers 504, not the 503 this service uses for not-ready, so an operator can tell a slow dependency from a draining instance. `http.request_timeout` sits below `http.write_timeout` (10s) so the budget expires while the connection can still carry that 504. The middleware writes it only if the handler committed nothing; a strict-server operation returning its expired context is mapped to the same problem in `generatedStrictServerOptions`. A handler that observes `ctx.Err()` should return, not invent a status.

## The Arithmetic Is Enforced, Not Conventional
`internal/config/validate.go` rejects a config where `postgres.acquire_timeout >= http.request_timeout` or `postgres.statement_timeout > http.request_timeout`. Sub-budgets are validated against the enclosing one at startup, so a number chosen locally either fits the relation or the service refuses to boot.

`Pool.Acquire` in `internal/infra/postgres/postgres.go` is the shape to copy: a 1s acquire budget inside an 8s request, and on expiry a distinct `ErrSaturated` rather than a generic deadline error — but only when the sub-budget expired while the caller's context was still live. A caller whose own context is already done is reporting a spent request budget or a disconnected client, and calling that saturation would hide both. That test is what keeps the two failures separable in metrics.

## Cancellation Does Not Reach Every Dependency
pgx delivers client-side cancellation as a separate `CancelRequest` on a separate connection — exactly what fails to arrive when the network is what went wrong, which is the common case for a slow dependency. `statement_timeout` and `idle_in_transaction_session_timeout` are published as session defaults for that reason. A new pooled dependency needs the same two-sided bound; a context alone is a bound only when the transport is healthy.

## What The Gate Already Catches, And The One It Does Not
`.golangci.yml` enables `noctx`, `contextcheck`, and `govet`. Measured against a probe file in `internal/infra/http`: `http.NewRequest` where the context-carrying form exists fails on `noctx`; a discarded `CancelFunc` fails on `lostcancel`; `context.Background()` inside a function that takes a `ctx` parameter fails on `contextcheck`. Those need no reviewer.

The gap is the one that matters here. `contextcheck` did **not** flag `context.Background()` inside an `http.HandlerFunc`, because the parent there is `r.Context()` rather than a parameter — so a handler that roots its own deadline is the one context defect that reaches review intact, and the one worth spending a finding on.

## Reject
- `http.TimeoutHandler` as the way to bound a handler: it returns early while leaking the handler goroutine anyway, and buffers every response to do it.
- A per-hop timeout at or above the enclosing budget. It can never fire, and it reads as a bound.

## Proof
`go test ./internal/infra/http/... -run 'Timeout|Deadline|Cancel'` and `go test ./internal/config/...` for a changed budget relation. Assert the caller's own deadline stops the work, not that a timer exists.
