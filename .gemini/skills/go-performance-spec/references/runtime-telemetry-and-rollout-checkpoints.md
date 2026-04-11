# Runtime Telemetry And Rollout Checkpoints

## Behavior Change Thesis
When loaded for symptom "the decision needs canary, rollback, runtime telemetry, or production validation," this file makes the model tie budget metrics to release actions instead of likely mistake ending at local benchmarks or saying "watch the dashboard."

## When To Load
Load when the performance spec needs runtime telemetry alignment, SLI/SLO rollout guardrails, canary acceptance, rollback limits, production profiling, or post-release validation.

## Decision Rubric
- Allow benchmark-only validation only when production shape cannot materially change the conclusion.
- Use staging load validation when representative dependencies and data volume can be simulated before release.
- Use canary validation when real traffic, tenant skew, DB/cache behavior, runtime scheduling, or fleet shape can change the result.
- Use SLI/SLO guardrails when user-visible latency, error, throughput, or availability governs release risk.
- Use runtime-metrics guardrails when goroutines, scheduler latency, GC, memory, DB pool, cache, or dependency saturation can regress without immediate request failures.
- Bound production profile or trace sampling by scope, duration, overhead, and access controls.

## Imitate
- Canary advances from 5% to 25% to 100% only if checkout `p95 <= 120ms`, `p99 <= 300ms`, error rate stays within baseline plus 0.1 percentage points, DB wait `p95 <= 5ms`, cache timeouts do not rise, and goroutine count returns to baseline after traffic drains. Copy the metric-threshold-action shape.
- GC-sensitive allocation reduction requires runtime telemetry for memory used, allocation rate, GC pause or scheduler latency, and request latency. Copy the refusal to let a benchmark alone prove production GC impact.
- Trace-based checkpoint captures a short trace from one canary instance only after p99 crosses the investigation threshold, then rolls back first if the SLO guardrail is burning too quickly. Copy the rollback-before-diagnosis rule.
- Cache-down fallback breach disables the cache feature flag or rolls back before deeper diagnosis. Copy the feature-disable action.

## Reject
- "Watch the dashboard" with no metric names, aggregation window, threshold, owner, or action.
- Canary acceptance based only on average latency while the risk is tail latency and tenant skew.
- Fleet-wide production CPU profiling without overhead expectation, sampling policy, or access control.
- Metrics with raw request IDs, tenant IDs, or cache keys as high-cardinality attributes.

## Agent Traps
- Reusing benchmark thresholds as canary thresholds without defining traffic class and aggregation window.
- Adding telemetry that cannot distinguish warm, cold, cache-down, degraded dependency, or shed modes when those modes matter.
- Treating rollback as an operator judgment call rather than a named action tied to a threshold.
- Measuring only request latency when the likely regression is saturation, memory, scheduler, DB pool, or cache behavior.

## Validation Shape
Local proof can seed the plan, but runtime validation needs repository-specific metrics and actions:

```bash
go test -run='^$' -bench='BenchmarkCheckout/(baseline|candidate)$' -benchmem -count=20 ./internal/checkout > checkout.txt
benchstat checkout.txt
curl -o cpu.pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
go tool pprof -top ./bin/service cpu.pprof
```

For telemetry systems, require metric names, route labels, aggregation windows, baseline comparison, threshold, owner, and action for request latency, error rate, saturation, goroutines, scheduler latency, memory/GC, DB pool wait, cache timeouts, and dependency errors as applicable.
