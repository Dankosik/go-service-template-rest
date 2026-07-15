# Reference Selector

| Symptom | Load |
| --- | --- |
| Several runtime paths or signal types lack an operator-decision contract. | [signal-contract-matrix.md](../../go-observability-spec/references/signal-contract-matrix.md) |
| Structured logs, redaction, sensitive fields, raw payloads, or log-based alerting change. | [structured-logs-and-privacy.md](../../go-observability-spec/references/structured-logs-and-privacy.md) |
| Trace propagation, baggage, request IDs, async lineage, retry, batch, or redrive correlation changes. | [trace-context-and-correlation.md](../../go-observability-spec/references/trace-context-and-correlation.md) |
| Metric labels, histograms, raw identifiers, cardinality, retention, or telemetry cost changes. | [metrics-cardinality-and-cost.md](../../go-observability-spec/references/metrics-cardinality-and-cost.md) |
| Resource attributes, semantic conventions, names, units, or cross-signal identity drift. | [resource-identity-and-semantic-conventions.md](../../go-observability-spec/references/resource-identity-and-semantic-conventions.md) |
| SLI/SLO math, logical outcomes, error budgets, alert noise, dashboards, or runbooks change. | [sli-slo-error-budget-and-alerting.md](../../go-observability-spec/references/sli-slo-error-budget-and-alerting.md) |
| Probes, pprof/expvar, debug exposure, shutdown diagnostics, or exporter flush changes. | [runtime-diagnostics-and-debug-endpoints.md](../../go-observability-spec/references/runtime-diagnostics-and-debug-endpoints.md) |
| Producers, consumers, retries, DLQ/redrive, backlog age, jobs, or reconciliation change. | [async-dlq-lag-and-reconciliation-observability.md](../../go-observability-spec/references/async-dlq-lag-and-reconciliation-observability.md) |
