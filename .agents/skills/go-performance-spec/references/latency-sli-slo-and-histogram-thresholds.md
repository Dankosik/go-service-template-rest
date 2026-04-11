# Latency SLI SLO And Histogram Thresholds

## When To Load
Load this when the spec must connect latency budgets to SLIs, SLOs, percentiles, histogram buckets, aggregation windows, synthetic versus real traffic, or error-budget rollout decisions.

Keep this pre-coding. Do not invent product SLOs; mark proposed thresholds as assumptions when the user or repo has not supplied them.

## Option Comparisons
- Operation-local threshold: use for implementation acceptance when no formal SLO exists, but the affected path still needs a measurable budget.
- SLI-linked threshold: use when runtime telemetry must map directly to a service-level indicator such as request latency or throughput.
- Multi-percentile threshold: use when tail latency shape matters, such as `p50`, `p95`, `p99`, or `p99.9`.
- Workload-specific SLO: use when interactive and bulk users have different latency or throughput needs.
- Synthetic-check threshold: use for smoke and black-box detection, but do not treat it as a substitute for real traffic if user experience differs.
- Blocked SLO: choose when business or product ownership must set the objective.

## Accepted Examples
Accepted example: an interactive read path uses `p95 <= 100ms` and `p99 <= 250ms` over five-minute windows for real traffic, while synthetic checks only prove availability and gross latency. The spec maps runtime telemetry to `http.server.request.duration` and requires low-cardinality route labels.

Accepted example: a batch ingestion endpoint has a throughput SLI and end-to-end completion latency target, not a strict request-response p99. The spec avoids forcing interactive SLOs on bulk work.

Accepted example: a rollout gate uses a tighter internal threshold than the published SLO to preserve safety margin during canary.

## Rejected Examples
Rejected example: using average latency as the only SLI for a path with long-tail user-visible pain.

Rejected example: copying current p99 as a permanent SLO without asking whether it matches user expectations or cost trade-offs.

Rejected example: metric labels include raw URL paths or tenant IDs, making histogram data high-cardinality and less useful.

Rejected example: changing histogram bucket boundaries during rollout without a compatibility plan for dashboards and alerts.

## Pass/Fail Rules
Pass when:
- latency thresholds name percentile, aggregation window, traffic class, route or operation, and measurement source
- SLO-linked metrics measure behavior users or dependent systems care about
- histogram buckets can represent the threshold without hiding the decision boundary
- telemetry labels are low-cardinality and stable enough for comparison
- rollout guardrails state how SLO or error-budget risk changes go/no-go decisions

Fail when:
- thresholds are averages, dashboard vibes, or invented absolute values without assumptions
- synthetic and real-user traffic are mixed without explanation
- high-cardinality labels make percentile or histogram analysis unreliable
- aggregation windows hide bursts that the spec claims to protect against
- SLO/error-budget implications are omitted for a user-visible regression risk

## Validation Commands
Use local commands only for acceptance evidence; runtime SLI proof still needs telemetry:

```bash
go test -run='^$' -bench='BenchmarkReadPath/(p50_shape|p95_shape|p99_shape)$' -benchmem -count=20 ./internal/readpath > latency.txt
benchstat latency.txt
go test -run='^$' -bench='BenchmarkReadPath/p99_shape$' -trace trace.out ./internal/readpath
go tool trace trace.out
```

For runtime validation, require repository-specific dashboard links or queries for route-level latency histograms, status/error rate, request volume, saturation signals, and canary baseline comparison.

## Exa Source Links
- [Google SRE: Service Level Objectives](https://sre.google/sre-book/service-level-objectives/)
- [Google SRE: Embracing Risk](https://sre.google/sre-book/embracing-risk/)
- [OpenTelemetry HTTP metrics semantic conventions](https://opentelemetry.io/docs/specs/semconv/http/http-metrics/)
- [OpenTelemetry metrics supplementary guidelines](https://opentelemetry.io/docs/specs/otel/metrics/supplementary-guidelines)
- [Go diagnostics](https://go.dev/doc/diagnostics)
