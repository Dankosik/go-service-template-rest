# Retry Budget, Jitter, And Never-Retry Rules

## Behavior Change Thesis
When loaded for retry policy, this file makes the model choose retry eligibility, owner layer, jitter, and budget interaction instead of likely mistake "retry all errors three times."

## When To Load
Load when the spec needs retry eligibility, retry budgets, jitter, transient fault handling, idempotency, client throttling, async retry, or never-retry categories.

## Decision Rubric
- Default to no retry until the operation is retry-safe, idempotent, or protected by an idempotency key and deduplication contract.
- Never retry validation failure, authorization failure, business conflict, caller cancellation, known poison input, or a retry-unsafe mutation.
- Retry transient 502/503/timeouts only inside the caller deadline, with bounded attempts, randomized delay, and retry exhaustion signal.
- Assign one owner layer for retries. Nested independent retries require an explicit reason and combined attempt budget.
- Use a retry budget when many clients call the same dependency or failure can cascade; stop retries when primary and retry traffic cannot be measured separately.
- For durable async retry, state max attempts, backoff cap, poison classification, terminal state or DLQ, diagnostics, and reconciliation owner.

## Imitate
- "Retry inventory reads on transient 502/503/timeouts only while request deadline and retry budget remain; use exponential backoff with jitter; stop after `<max attempts>` or budget exhaustion."
- "Do not retry order creation unless the request carries an idempotency key and storage can deduplicate repeated attempts."
- "Async email delivery retries with capped backoff until `<max attempts>`; non-retryable template errors go directly to terminal failed state with diagnostics."

## Reject
- "Retry all errors three times." Validation, authorization, business conflict, caller cancellation, and overload can be made worse by retries.
- "Retry POST because it usually works." This lacks idempotency, deduplication, or compensating state.
- "Each layer retries independently." Nested retries can multiply load and contribute to cascading failure.
- "Fixed retry delay for all clients." Synchronized retries can create traffic spikes.

## Agent Traps
- Do not label a fault transient without saying which response classes or timeout conditions qualify.
- Do not use `Retry-After` as permission to retry after the caller deadline or after a local budget is exhausted.
- Do not put sync and async retry under the same rule; they have different caller-visible contracts.

## Validation Shape
- Given validation, authorization, not-found, business conflict, caller cancellation, or poison input, no retry is attempted.
- Given a transient failure and sufficient deadline/budget, retries occur within specified attempt and delay bounds.
- Given retry budget exhaustion, retries stop and the flow returns the selected fail-fast or degraded contract.
- Given an idempotency key is absent for a retry-unsafe mutation, the spec blocks automatic retry.
- Given async retry attempts are exhausted, the item enters a DLQ or terminal state with diagnostic context and owner.
