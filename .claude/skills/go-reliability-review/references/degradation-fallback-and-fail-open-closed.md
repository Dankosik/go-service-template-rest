# Degradation, Fallback, And Fail-Open/Closed

## Behavior Change Thesis
When loaded for symptom `fallback, stale data, degraded response, optional dependency, feature-disable path, or fail-open/fail-closed behavior changed`, this file makes the model check whether degradation is bounded, observable, and contract-safe instead of likely mistake `accept silent defaults, origin storms, or fail-open access behavior as availability improvements`.

## When To Load
Load when a diff changes fallback behavior, cache or stale-data fallback, optional dependency handling, degraded response shape, feature-disable paths, circuit-breaker fallback, fail-open or fail-closed behavior, or recovery from a dependency outage.

Keep findings local: ask whether the changed fallback is bounded, observable, and contract-safe. Hand off primary security policy for fail-open access decisions to `go-security-review`, DB/cache consistency to `go-db-cache-review`, and product/API contract changes to `api-contract-designer-spec`.

## Decision Rubric
- A fallback silently returns zero values, empty lists, defaults, or stale data where correctness matters.
- The code falls back to a slower or more expensive path without a timeout or overload guard.
- A noncritical dependency still blocks the whole critical response until its full deadline expires.
- Fail-open behavior is added for auth, authorization, tenant isolation, payment, or data integrity without an explicit policy.
- Fail-closed behavior is added for optional enrichment and turns a partial outage into full request failure.
- Fallback activation and recovery are not logged or metered.
- A circuit-breaker open state returns data with different semantics than the caller contract expects.
- Recovery from degraded mode depends on process restart or manual cleanup without being stated.

## Imitate

Bad finding shape to copy: optional dependency failure should not silently become critical-path failure.

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

```text
[medium] [go-reliability-review] internal/orders/summary.go:67
Issue: The changed summary path fails the whole response when the optional recommendations dependency fails.
Impact: A recommendations outage becomes a full order-summary outage and can consume the caller's latency budget before returning.
Suggested fix: Bound the optional call with a smaller child deadline and return a documented degraded summary when recommendations are unavailable.
Reference: Azure self-healing guidance and Google SRE overload/degraded-mode guidance.
```

Good correction shape: bound optional work and make degradation explicit.

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

## Reject

```go
if err != nil {
	return Result{}, nil // use defaults during dependency outage
}
```

Reject when defaults are plausible but wrong or indistinguishable from real data.

```go
if cacheErr != nil {
	return origin.Load(ctx, key)
}
```

Reject when cache outage fallback can storm the origin without its own deadline, limiter, or circuit behavior.

```go
if authzErr != nil {
	return allow(), nil
}
```

Reject as reliability-owned "availability" unless an approved security policy exists; hand off to `go-security-review`.

## Agent Traps
- Do not equate fail-open with reliability. For security, tenant isolation, payment, and data integrity, fail-open is usually a security/domain decision.
- Do not invent degraded response contracts. Review may ask for bounded local behavior and escalate product/API semantics.
- Do not accept stale data without a freshness window and caller-visible or operator-visible degradation signal when correctness matters.
- Do not let fallback work spend the entire primary request budget if the dependency is optional.
- Do not treat a circuit breaker as sufficient if open-state output changes semantics silently.

## Validation Shape
- `go test ./... -run 'Test.*(Fallback|Degraded|Degradation|FailOpen|FailClosed)'`
- `go test ./... -run 'Test.*(OptionalDependency|Stale|CircuitOpen|Origin)'`
- `go test ./... -run 'Test.*(Timeout|Deadline)'` for fallback budget enforcement.
- `go test -race ./...` when degraded-mode state is shared.

Use failure-injection tests that make the optional dependency fail or hang and assert the critical response remains bounded.
