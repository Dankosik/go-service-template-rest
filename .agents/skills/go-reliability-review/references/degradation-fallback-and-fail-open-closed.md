# Degradation, Fallback, And Fail-Open/Closed

## When To Load
Load this reference when a diff changes fallback behavior, cache or stale-data fallback, optional dependency handling, degraded response shape, feature-disable paths, circuit-breaker fallback, fail-open or fail-closed behavior, or recovery from a dependency outage.

Keep findings local: ask whether the changed fallback is bounded, observable, and contract-safe. Hand off primary security policy for fail-open access decisions to `go-security-review`, DB/cache consistency to `go-db-cache-review`, and product/API contract changes to `api-contract-designer-spec`.

## Review Smells
- A fallback silently returns zero values, empty lists, defaults, or stale data where correctness matters.
- The code falls back to a slower or more expensive path without a timeout or overload guard.
- A noncritical dependency still blocks the whole critical response until its full deadline expires.
- Fail-open behavior is added for auth, authorization, tenant isolation, payment, or data integrity without an explicit policy.
- Fail-closed behavior is added for optional enrichment and turns a partial outage into full request failure.
- Fallback activation and recovery are not logged or metered.
- A circuit-breaker open state returns data with different semantics than the caller contract expects.
- Recovery from degraded mode depends on process restart or manual cleanup without being stated.

## Failure Modes
- Users receive plausible but wrong data without any indication of degradation.
- A dependency outage shifts load to an origin or alternate dependency and causes a second outage.
- Optional features consume the entire latency budget and starve critical response work.
- Fail-open behavior creates security or correctness exposure that reliability review should not own alone.
- Fail-closed behavior makes a noncritical dependency a hard availability dependency.

## Review Examples

Bad: an optional recommendation dependency can block the order response.

```go
func (s *Service) OrderSummary(ctx context.Context, id string) (Summary, error) {
	order, err := s.orders.Get(ctx, id)
	if err != nil {
		return Summary{}, err
	}
	recs, err := s.recommendations.ForOrder(ctx, id)
	if err != nil {
		return Summary{}, err
	}
	return Summary{Order: order, Recommendations: recs}, nil
}
```

Review finding shape:

```text
[medium] [go-reliability-review] internal/orders/summary.go:67
Issue: The changed summary path fails the whole response when the optional recommendations dependency fails.
Impact: A recommendations outage becomes a full order-summary outage and can consume the caller's latency budget before returning.
Suggested fix: Bound the optional call with a smaller child deadline and return a documented degraded summary when recommendations are unavailable.
Reference: Azure self-healing guidance and Google SRE overload/degraded-mode guidance.
```

Good: bound optional work and make degradation explicit.

```go
func (s *Service) OrderSummary(ctx context.Context, id string) (Summary, error) {
	order, err := s.orders.Get(ctx, id)
	if err != nil {
		return Summary{}, err
	}

	recCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	recs, err := s.recommendations.ForOrder(recCtx, id)
	if err != nil {
		s.metrics.RecommendationsDegraded.Inc()
		return Summary{Order: order, Degraded: true}, nil
	}
	return Summary{Order: order, Recommendations: recs}, nil
}
```

Do not invent user-visible response semantics in review. If degraded output changes API or domain behavior, flag the local risk and hand off.

## Smallest Safe Fix
- Bound fallback work with a child deadline shorter than the primary request budget.
- Fail closed for correctness-critical or security-sensitive dependencies unless an explicit approved policy says otherwise.
- Fail open or degrade for optional enrichment only when the response contract allows it.
- Add a visible degraded marker, metric, or log that does not leak sensitive data.
- Prevent fallback from storming a cache origin or alternate dependency.
- Keep recovery automatic where possible; if manual recovery is required, say so in the finding as a residual risk or escalation.

## Validation Commands
- `go test ./... -run 'Test.*(Fallback|Degraded|Degradation|FailOpen|FailClosed)'`
- `go test ./... -run 'Test.*(OptionalDependency|Stale|CircuitOpen|Origin)'`
- `go test ./... -run 'Test.*(Timeout|Deadline)'` for fallback budget enforcement.
- `go test -race ./...` when degraded-mode state is shared.

Use failure-injection tests that make the optional dependency fail or hang and assert the critical response remains bounded.

## Exa Source Links
- Azure Design for self-healing, graceful degradation and self-preservation: https://learn.microsoft.com/en-us/azure/architecture/guide/design-principles/self-healing
- Azure Circuit Breaker pattern: https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker
- Google SRE Handling Overload, degraded responses and resource errors: https://sre.google/sre-book/handling-overload/
- Google SRE Production Services Best Practices, fail sanely and overload behavior: https://sre.google/sre-book/service-best-practices/
- Google SRE Addressing Cascading Failures, noncritical backend blackhole tests: https://sre.google/sre-book/addressing-cascading-failures/

