# Retry Budget And Idempotency Review

## Behavior Change Thesis
When loaded for symptom `retry, redrive, idempotency, duplicate suppression, or replay behavior changed`, this file makes the model require retry eligibility, a bounded budget, cancellation, and duplicate-effect protection instead of likely mistake `treat retries as generic resilience or add backoff without proving the operation is retry-safe`.

## When To Load
Load when a Go diff adds, changes, or removes retries, backoff, jitter, retry classification, HTTP status retry handling, message redrive, idempotency keys, duplicate suppression, or operation replay behavior.

Keep findings local: review whether this changed call is safe to retry and bounded. Hand off API-visible idempotency contracts to `api-contract-designer-spec`, broader distributed replay design to `go-distributed-architect-spec`, and DB uniqueness or transaction mechanics to `go-db-cache-review`.

## Decision Rubric
- Retries are added for all errors, all HTTP 5xx statuses, or all `net.Error`s without checking the operation class.
- Mutating operations retry without an idempotency key, conditional write, dedup key, or safe natural idempotency.
- Retries ignore `ctx.Err()` and keep going after cancellation.
- Backoff has no cap, no jitter, or no maximum attempt count.
- A new retry wraps a client library that already retries, creating layered retry amplification.
- Retry loops include validation, auth, not-found, conflict, or caller-canceled failures.
- The retry budget resets inside a loop, goroutine, or recursive helper.
- Logs report every failed attempt at error level and can flood during partial outages.

## Imitate

Bad finding shape to copy: the issue is unsafe replay of an effect, not merely "missing jitter."

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

```text
[high] [go-reliability-review] internal/billing/client.go:88
Issue: The new retry loop retries a mutating invoice create with a detached context and no idempotency protection.
Impact: A lost response or caller cancellation can create duplicate invoices, while exponential sleeps continue after the request has already been abandoned.
Suggested fix: Preserve the caller context, retry only explicitly transient classes, cap attempts and backoff with jitter, and require an idempotency key or conditional create before retrying the mutation.
Reference: Azure Retry pattern and Google SRE retry-budget guidance.
```

Good correction shape: keep attempts bounded and tied to retry-safe classes.

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

Copy the review move: do not require this exact helper shape. Require a clear eligibility rule, a budget, cancellation, and duplicate-effect protection where effects can repeat.

## Reject

```go
if err != nil {
	return retry.Do(func() error { return c.CreatePayment(ctx, p) })
}
```

Reject because retrying a mutating operation without an idempotency key or conditional write can duplicate the effect after a lost response.

```go
for {
	if err := c.Call(ctx); err == nil {
		return nil
	}
	time.Sleep(backoff)
}
```

Reject because the loop has no attempt or elapsed-time budget and ignores caller cancellation while sleeping.

```go
return outerRetry(ctx, func() error {
	return c.httpClient.Do(req) // client already retries 3 times
})
```

Reject when the nested client retry makes the actual attempt budget multiplicative and invisible to the caller.

## Agent Traps
- Do not recommend retries before proving the error class and operation are retry-safe.
- Do not treat all `5xx`, `net.Error`, or timeout errors as retryable; caller cancellation and overloaded dependencies can be made worse by retry.
- Do not ask for idempotency keys on naturally idempotent reads; focus the finding on duplicate effects.
- Do not over-focus on jitter if the larger defect is replay safety or an unbounded budget.
- Do not report every verbose per-attempt log as reliability risk unless partial outages can flood logs or alerts.

## Validation Shape
- `go test ./... -run 'Test.*(Retry|Backoff|Jitter|Idempot|Duplicate|Replay)'`
- `go test ./... -run 'Test.*(NoRetry|Permanent|Canceled|Conflict|Unauthorized|NotFound)'`
- `go test ./... -count=20 -run 'Test.*Retry'` for retry timing or jitter tests that might be flaky.
- `go test -race ./...` if retry state, budget counters, or dedup maps are shared.

Prefer deterministic fake clocks or stub sleepers over real sleeps in tests.
