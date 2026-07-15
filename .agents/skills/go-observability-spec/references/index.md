# Reference Selector

| Symptom | Load | Decision it sharpens |
| --- | --- | --- |
| Broad telemetry contract or component-by-component coverage | [signal-contract-matrix.md](signal-contract-matrix.md) | Build an operator-decision matrix instead of disconnected signal lists. |
| Service identity, resource attributes, semantic conventions, names, or instrumentation scope | [resource-identity-and-semantic-conventions.md](resource-identity-and-semantic-conventions.md) | Reuse stable resource and OpenTelemetry conventions and expose unstable status. |
| Metric labels, histograms, retention, cost, dashboards, IDs, paths, or error strings | [metrics-cardinality-and-cost.md](metrics-cardinality-and-cost.md) | Bound aggregation and move forensic detail out of metrics. |
| Structured events, redaction, PII/secrets, request/query data, or support pivots | [structured-logs-and-privacy.md](structured-logs-and-privacy.md) | Define allowlisted forensic logs and privacy controls. |
| W3C Trace Context, baggage, async linkage, retries, DLQ/redrive, or propagation | [trace-context-and-correlation.md](trace-context-and-correlation.md) | Preserve safe correlation and non-single-parent lineage. |
| SLIs/SLOs, error budgets, burn rates, event floors, alerts, dashboards, or runbooks | [sli-slo-error-budget-and-alerting.md](sli-slo-error-budget-and-alerting.md) | Define good/total events and proportional operator response. |
| Queues, retries, DLQs, backlog, lag, oldest age, idempotency, jobs, or reconcilers | [async-dlq-lag-and-reconciliation-observability.md](async-dlq-lag-and-reconciliation-observability.md) | Separate attempt, completion, freshness, redrive, and reconciliation visibility. |
| Health probes, shutdown, pprof/expvar, debug listeners, crash data, or telemetry flush | [runtime-diagnostics-and-debug-endpoints.md](runtime-diagnostics-and-debug-endpoints.md) | Separate orchestration and incident-debug decisions with access controls. |

When extending standards-sensitive examples, verify primary OpenTelemetry, W3C Trace Context, Prometheus, Go, Kubernetes, SRE, or cloud-provider sources; retain only the small freshness-sensitive pointer needed by the contract.
