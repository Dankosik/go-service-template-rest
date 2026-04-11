# Retry Budget And Idempotency Review

## When To Load
Load this reference when a Go diff adds, changes, or removes retries, backoff, jitter, retry classification, HTTP status retry handling, message redrive, idempotency keys, duplicate suppression, or operation replay behavior.

Keep findings local: review whether this changed call is safe to retry and bounded. Hand off API-visible idempotency contracts to `api-contract-designer-spec`, broader distributed replay design to `go-distributed-architect-spec`, and DB uniqueness or transaction mechanics to `go-db-cache-review`.

## Review Smells
- Retries are added for all errors, all HTTP 5xx statuses, or all `net.Error`s without checking the operation class.
- Mutating operations retry without an idempotency key, conditional write, dedup key, or safe natural idempotency.
- Retries ignore `ctx.Err()` and keep going after cancellation.
- Backoff has no cap, no jitter, or no maximum attempt count.
- A new retry wraps a client library that already retries, creating layered retry amplification.
- Retry loops include validation, auth, not-found, conflict, or caller-canceled failures.
- The retry budget resets inside a loop, goroutine, or recursive helper.
- Logs report every failed attempt at error level and can flood during partial outages.

## Failure Modes
- A brief downstream overload becomes a retry storm.
- A payment, email, webhook, order submission, or state transition executes twice.
- A non-transient error burns latency budget and delays the caller without improving success odds.
- Multiple retry layers multiply attempts beyond the intended budget.
- A service under load spends most resources rejecting or processing retry traffic instead of fresh work.

## Review Examples

Bad: the loop retries a mutating request for every failure and ignores cancellation.

```go
func (c *Client) CreateInvoice(ctx context.Context, in Invoice) error {
	for attempt := 0; attempt < 5; attempt++ {
		err := c.postInvoice(context.Background(), in)
		if err == nil {
			return nil
		}
		time.Sleep(time.Second << attempt)
	}
	return ErrCreateInvoice
}
```

Review finding shape:

```text
[high] [go-reliability-review] internal/billing/client.go:88
Issue: The new retry loop retries a mutating invoice create with a detached context and no idempotency protection.
Impact: A lost response or caller cancellation can create duplicate invoices, while exponential sleeps continue after the request has already been abandoned.
Suggested fix: Preserve the caller context, retry only explicitly transient classes, cap attempts and backoff with jitter, and require an idempotency key or conditional create before retrying the mutation.
Reference: Azure Retry pattern and Google SRE retry-budget guidance.
```

Good: keep attempts bounded and tied to retry-safe classes.

```go
func (c *Client) CreateInvoice(ctx context.Context, in Invoice) error {
	key := in.IdempotencyKey
	if key == "" {
		return errors.New("missing invoice idempotency key")
	}

	backoff := 100 * time.Millisecond
	for attempt := 0; attempt < 3; attempt++ {
		err := c.postInvoice(ctx, in, key)
		if err == nil || !isTransient(err) {
			return err
		}
		if err := sleepWithJitter(ctx, backoff); err != nil {
			return err
		}
		backoff = min(backoff*2, time.Second)
	}
	return ErrRetryBudgetExceeded
}
```

Do not require this exact helper shape. The review requirement is that retries have a clear eligibility rule, a budget, cancellation, and duplicate-effect protection where effects can be repeated.

## Smallest Safe Fix
- Add or tighten retry classification so permanent, caller-canceled, validation, auth, not-found, and conflict failures do not retry.
- Bound attempts and total elapsed time within the caller context.
- Add jitter and cap the backoff.
- Remove duplicate retry layers or make the outer layer fail fast when an inner library already retries.
- Require idempotency protection before retrying a mutating operation.
- Log interim failures at debug/info level and emit error only after final failure, unless local logging policy says otherwise.

## Validation Commands
- `go test ./... -run 'Test.*(Retry|Backoff|Jitter|Idempot|Duplicate|Replay)'`
- `go test ./... -run 'Test.*(NoRetry|Permanent|Canceled|Conflict|Unauthorized|NotFound)'`
- `go test ./... -count=20 -run 'Test.*Retry'` for retry timing or jitter tests that might be flaky.
- `go test -race ./...` if retry state, budget counters, or dedup maps are shared.

Prefer deterministic fake clocks or stub sleepers over real sleeps in tests.

## Exa Source Links
- Azure Retry pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/retry
- Azure Circuit Breaker pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker
- Google SRE Handling Overload, including retry budgets: https://sre.google/sre-book/handling-overload/
- Google SRE Production Services Best Practices, including retry amplification and jitter: https://sre.google/sre-book/service-best-practices/
- AWS transactional outbox notes on duplicate messages and idempotent consumers: https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html

