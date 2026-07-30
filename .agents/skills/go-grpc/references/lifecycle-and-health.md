# Lifecycle And Health

Use this reference when startup, serving state, drain, or shutdown can diverge
from the process lifecycle.

## Decision

- Define observable states from disabled/configured through listener bound,
  standard health registered, `NOT_SERVING`, admitted `SERVING`, draining
  `NOT_SERVING`, transport stop, and cleanup complete.
- Bind required listeners and validate credentials before publishing readiness.
  Align gRPC health with the accepted application admission source.
- On drain, publish `NOT_SERVING` before rejecting new application work and
  beginning transport shutdown. Preserve standard health behavior during the
  transition.
- Start `GracefulStop` for in-flight RPCs and invoke `Stop` when the accepted
  budget expires. The public shutdown method must return within that budget
  rather than joining an uncooperative handler indefinitely.
- Treat handler cancellation as cooperative. State how dependencies remain safe
  for any application work that can outlive forced transport stop, and require
  every owned handler to release work on context cancellation.
- Make repeated shutdown and partial-startup cleanup idempotent, including bind
  failure after another listener or resource was acquired.

## Review

Report `SERVING` before dependency admission, health left serving during drain,
new work admitted after drain begins, `GracefulStop` without a force deadline,
`Stop` followed by an unbounded join, dependencies closed while reachable
handler work still owns them, leaked partial-start resources, or shutdown tests
whose handler always cooperates.

## Proof

Prove disabled no-bind, startup failure cleanup, `NOT_SERVING → SERVING →
NOT_SERVING`, cooperative in-flight completion, cancellation-aware release,
an intentionally uncooperative handler, bounded shutdown return, repeated
shutdown, race/liveness, and production-process signal handling. Release parked
test handlers explicitly after asserting the public deadline.

Vendor authority:
[Graceful shutdown](https://grpc.io/docs/guides/server-graceful-stop/),
[Health checking](https://grpc.io/docs/guides/health-checking/), and
[Cancellation](https://grpc.io/docs/guides/cancellation/).
