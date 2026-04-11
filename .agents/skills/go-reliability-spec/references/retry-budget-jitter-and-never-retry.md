# Retry Budget, Jitter, And Never-Retry Rules

## When To Load This
Load this file when the spec needs retry eligibility, retry budget math, jitter policy, transient fault handling, idempotency, client throttling, or never-retry categories.

## Contract Questions
- Is the failed operation retry-safe, idempotent, or protected by an idempotency key?
- Is the failure transient, capacity-related, caller-canceled, validation-related, or authorization-related?
- Which layer owns retries so nested retry loops do not multiply traffic?
- What budget disables retries when the system is overloaded?

## Option Comparisons
| Option | Use when | Contract shape | Reject when |
| --- | --- | --- | --- |
| No retry | Default for non-idempotent, user-blocking, or invariant-sensitive operations. | Return the classified failure and let the caller contract decide. | The operation is retry-safe and a transient failure is common enough to justify bounded retry. |
| Immediate retry | Rare single packet/connection anomalies with very low cost. | At most one immediate retry and only inside the caller budget. | The dependency may be overloaded or the retry can synchronize clients. |
| Exponential backoff with jitter | Transient remote failures or throttling responses. | Bounded attempts, randomized delay, deadline-aware stop, and operator-visible retry exhaustion. | It runs after caller cancellation or after the retry budget is exhausted. |
| Retry budget | Many clients call the same dependency or failure can cascade. | Cap extra retry traffic per client/process/window and fail fast when exhausted. | The spec cannot measure primary attempts and retry attempts separately. |
| Async retry with DLQ/reconciliation | Durable background work can converge later. | Max attempts, backoff cap, poison classification, DLQ or terminal state, and reconciliation owner. | The caller needs synchronous completion or immediate consistency. |

## Accepted Examples
- "Retry inventory reads on transient 502/503/timeouts only when the request deadline has budget remaining; use exponential backoff with jitter; stop after `<max attempts>` or retry-budget exhaustion."
- "Do not retry order creation unless the API request carries an idempotency key and the storage contract can deduplicate repeated attempts."
- "Async email delivery retries with capped backoff until `<max attempts>`; non-retryable template errors go directly to a terminal failed state with diagnostics."

## Rejected Examples
- "Retry all errors three times." Rejected because validation, authorization, business conflicts, caller cancellation, and overload can be made worse by retries.
- "Retry POST because it usually works." Rejected without idempotency, deduplication, or a compensating state model.
- "Each layer retries independently." Rejected because nested retries can multiply load and contribute to cascading failure.
- "Fixed retry delay for all clients." Rejected because synchronized retries can create traffic spikes.

## Testable Failure Contracts
- Given validation, authorization, not-found, business conflict, or caller cancellation, no retry is attempted.
- Given a transient failure and sufficient deadline/budget, retries occur within the specified attempt and delay bounds.
- Given retry budget exhaustion, retries stop and the flow returns the selected fail-fast/degraded contract.
- Given an idempotency key is absent for a retry-unsafe mutation, the spec blocks automatic retry.
- Given async retry attempts are exhausted, the item enters a DLQ or terminal state with diagnostic context and owner.

## Exa Source Links
- Microsoft Azure, Retry pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/retry
- Google SRE, Addressing Cascading Failures: https://sre.google/sre-book/addressing-cascading-failures/
- Google SRE, Production Services Best Practices: https://sre.google/sre-book/service-best-practices/
- Go `context` package cancellation/deadlines: https://pkg.go.dev/context
