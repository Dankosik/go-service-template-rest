# Timeout And Deadline Budgets

## When To Load This
Load this file when the spec needs inbound deadlines, outbound per-hop budgets, context propagation, fail-fast thresholds, server timeout policy, or shutdown deadlines.

## Contract Questions
- What is the end-to-end user-visible budget for the flow?
- Which work must stop when the caller cancels or the deadline expires?
- How much budget is reserved for response write, cleanup, telemetry flush, and rollback-safe exit?
- Which background work is allowed to detach from the inbound request, and what durable tracking proves it?

## Option Comparisons
| Option | Use when | Contract shape | Reject when |
| --- | --- | --- | --- |
| Propagated inbound deadline | Synchronous work is done on behalf of an incoming request. | Outbound calls derive from the inbound context and cannot exceed remaining budget. | Work is intentionally async and durably accepted before detaching. |
| Per-hop cap | A dependency has known latency and resource cost. | `min(<per-hop cap>, <remaining inbound budget> - <reserved cleanup>)`. | A fixed timeout would exceed the caller budget or cause late writes. |
| Fail-fast floor | Remaining budget is too small to do useful work. | Skip the outbound call and return the named timeout/degraded response. | The flow has a safe cached/stale response already inside budget. |
| Detached async deadline | Work continues after the response by contract. | Create a new bounded deadline, durable tracking ID, and reconciliation/expiry rule. | It hides unfinished work behind a fake synchronous success. |
| Shutdown deadline | Server is draining. | Stop new work, allow in-flight work until `<drain window>`, then exit within the platform grace period. | Long-lived or hijacked connections are not covered by the shutdown policy. |

## Accepted Examples
- "The request budget is `<end-to-end budget>`; downstream inventory calls use `min(<inventory cap>, remaining request budget - <write/cleanup reserve>)`; if remaining budget is below `<fail-fast floor>`, skip the call and return the specified timeout response."
- "The export endpoint returns `202 Accepted` with a tracking reference before starting background work; the worker has its own `<worker deadline>` and expiry policy."
- "Shutdown sets readiness false, stops accepting new traffic, waits up to `<drain window>` for in-flight HTTP requests, and records how long draining took."

## Rejected Examples
- "Use `context.Background()` for outbound calls from a request handler." Rejected because request cancellation and deadline propagation are lost unless the spec explicitly defines an async handoff.
- "Set every downstream timeout to 5s." Rejected because it can exceed the caller budget and amplify slow failures.
- "Let shutdown wait until requests finish." Rejected because an unbounded wait can exceed the platform termination window.

## Testable Failure Contracts
- Given caller cancellation, synchronous dependency work observes cancellation and stops before starting new outbound work.
- Given remaining inbound budget below the fail-fast floor, no outbound dependency call is attempted.
- Given a dependency exceeds its per-hop timeout, the returned contract is timeout/degraded/fail-closed as specified and no later write is attempted.
- Given shutdown begins, readiness changes before new work is rejected, in-flight work is drained or timed out, and the process exits within the grace window.

## Exa Source Links
- Go `context` package: https://pkg.go.dev/context
- Go `net/http` package and server controls: https://pkg.go.dev/net/http
- Go `Server.Shutdown`: https://pkg.go.dev/net/http#Server.Shutdown
- Google SRE, Managing Load: https://sre.google/workbook/managing-load/
- Google SRE, Production Services Best Practices: https://sre.google/sre-book/service-best-practices/
