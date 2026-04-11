# Runtime Telemetry And Rollout Checkpoints

## When To Load
Load this when the performance spec needs runtime telemetry alignment, SLI/SLO rollout guardrails, canary acceptance, rollback limits, production profiling, or post-release validation.

Keep this as delivery-facing performance contract work. Do not invent the deployment process; specify what the rollout must prove and hand off delivery mechanics when necessary.

## Option Comparisons
- Benchmark-only validation: allow only for local, non-runtime-critical changes where production shape cannot materially change the conclusion.
- Staging load validation: use when representative dependencies and data volume can be simulated before release.
- Canary validation: use when real traffic, tenant skew, DB/cache behavior, or runtime scheduling can change the result.
- SLO-linked guardrail: use when a user-visible latency, error, throughput, or availability objective governs release risk.
- Runtime-metrics guardrail: use when goroutines, scheduler latency, GC, memory, DB pool, or cache behavior can regress without immediate request failures.
- Production profile or trace sampling: use when a representative runtime bottleneck needs post-release evidence, with capture overhead and access controls stated.

## Accepted Examples
Accepted example: a canary plan advances from 5% to 25% to 100% only if `checkout p95 <= 120ms`, `checkout p99 <= 300ms`, error rate does not exceed baseline by 0.1 percentage points, DB wait duration p95 stays below 5ms, cache timeout count does not rise, and goroutine count returns to baseline after traffic drains.

Accepted example: a GC-sensitive allocation reduction requires runtime telemetry for heap allocation rate, `go.memory.used`, GC pause or scheduler latency, and request latency, because the benchmark alone cannot prove production impact.

Accepted example: a trace-based validation checkpoint captures a short trace from one canary instance only after p99 crosses the investigation threshold, then rolls back first if the SLO guardrail is burning too quickly.

Accepted example: a rollout plan states that if cache-down fallback breaches DB pool wait thresholds, rollback or disable the cache feature flag before diagnosing.

## Rejected Examples
Rejected example: "watch the dashboard" with no metric names, aggregation window, threshold, owner, or rollback rule.

Rejected example: canary acceptance based only on average latency while the risk is tail latency and tenant skew.

Rejected example: production CPU profiling enabled fleet-wide without overhead expectation, sampling policy, or access control.

Rejected example: adding OpenTelemetry metrics with high-cardinality request IDs, tenant IDs, or raw cache keys as attributes.

## Pass/Fail Rules
Pass when:
- rollout checkpoints name metric, aggregation window, baseline comparison, threshold, owner, and action
- telemetry maps to the same SLI or budget used in the spec, not a loosely related internal counter
- runtime metrics cover goroutines, scheduler latency, memory/GC, DB pool, cache, and dependency saturation when those are plausible regressions
- canary and rollback rules are explicit before rollout
- production profile or trace capture includes scope and overhead controls

Fail when:
- validation stops at local benchmarks for a production-shape risk
- telemetry is too high-cardinality, missing, or unrelated to the stated budget
- no rollback or feature-disable checkpoint exists for a risky performance change
- SLO/error-budget impact is described qualitatively but not tied to a threshold
- runtime validation cannot distinguish cache-up, cache-down, warm, cold, and degraded dependency modes when those modes matter

## Validation Commands
Use local command patterns where possible:

```bash
go test -run='^$' -bench='BenchmarkCheckout/(baseline|candidate)$' -benchmem -count=20 ./internal/checkout > checkout.txt
benchstat checkout.txt
curl -o cpu.pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
go tool pprof -top ./bin/service cpu.pprof
curl -o trace.out 'http://127.0.0.1:6060/debug/pprof/trace?seconds=5'
go tool trace trace.out
```

For OpenTelemetry or SLO systems, require repository-specific query commands or dashboard links in the plan. The spec should name the metric streams, such as request latency histograms, `go.goroutine.count`, `go.schedule.duration`, `go.memory.used`, DB pool wait duration, cache timeout count, and saturation signals.

## Exa Source Links
- [OpenTelemetry Go instrumentation](https://opentelemetry.io/docs/languages/go/instrumentation/)
- [OpenTelemetry Go runtime metrics semantic conventions](https://opentelemetry.io/docs/specs/semconv/runtime/go-metrics/)
- [runtime/metrics package](https://pkg.go.dev/runtime/metrics)
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [Google SRE: Service Level Objectives](https://sre.google/sre-book/service-level-objectives/)
- [Google SRE: Embracing Risk](https://sre.google/sre-book/embracing-risk/)
- [Google SRE: Production Services Best Practices](https://sre.google/sre-book/service-best-practices/)
