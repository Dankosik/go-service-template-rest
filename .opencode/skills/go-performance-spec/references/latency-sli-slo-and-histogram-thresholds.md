# Latency SLI SLO And Histogram Thresholds

## Behavior Change Thesis
When loaded for symptom "the spec must connect latency to SLI/SLO, histograms, aggregation windows, or error-budget risk," this file makes the model choose percentile, window, and label-aware thresholds instead of likely mistake averages, copied dashboard values, or high-cardinality telemetry.

## When To Load
Load when the spec must connect latency budgets to SLIs, SLOs, percentiles, histogram buckets, aggregation windows, synthetic versus real traffic, or error-budget rollout decisions.

## Decision Rubric
- Use operation-local thresholds for implementation acceptance when no formal SLO exists but the path still needs a measurable budget.
- Use SLI-linked thresholds when runtime telemetry must map directly to service-level request latency, throughput, or completion-lag indicators.
- Use multiple percentiles when tail shape matters; avoid averages as primary acceptance for interactive user pain.
- Use workload-specific SLOs when interactive, admin, bulk, and background users have different objectives.
- Use synthetic-check thresholds for smoke and black-box detection only; do not substitute them for real traffic when user experience differs.
- Mark product SLOs as blocked or assumed when business ownership must set the objective.

## Imitate
- Interactive read path uses `p95 <= 100ms` and `p99 <= 250ms` over five-minute windows for real traffic; synthetic checks only prove availability and gross latency. Copy the real-versus-synthetic split.
- Batch ingestion uses throughput SLI and end-to-end completion latency rather than strict request-response p99. Copy the workload-specific SLI selection.
- Canary uses a tighter internal threshold than the published SLO to preserve safety margin. Copy the guardrail margin.

## Reject
- Average latency as the only SLI for a path with long-tail user-visible pain.
- Copying current p99 as a permanent SLO without checking user expectations and cost trade-offs.
- Metric labels with raw URL paths or tenant IDs that make histogram data high-cardinality.
- Changing histogram bucket boundaries during rollout without compatibility for dashboards and alerts.

## Agent Traps
- Mixing synthetic and real-user traffic in the same threshold without saying why.
- Choosing a threshold that falls between histogram buckets, making the decision boundary hard to observe.
- Using an aggregation window that hides bursts the spec claims to protect against.
- Omitting SLO or error-budget consequences for a user-visible regression risk.

## Validation Shape
Local acceptance commands can prove candidate behavior, but runtime SLI proof needs metric queries:

```bash
go test -run='^$' -bench='BenchmarkReadPath/(p50_shape|p95_shape|p99_shape)$' -benchmem -count=20 ./internal/readpath > latency.txt
benchstat latency.txt
go test -run='^$' -bench='BenchmarkReadPath/p99_shape$' -trace trace.out ./internal/readpath
go tool trace trace.out
```

For runtime validation, require repository-specific route-level latency histogram queries, status/error rate, request volume, saturation signals, aggregation window, bucket compatibility, and canary baseline comparison.
