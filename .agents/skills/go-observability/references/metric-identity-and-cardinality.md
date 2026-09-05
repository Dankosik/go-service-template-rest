# Metric Identity And Cardinality

## Load When

Load when a metric, unit, label, resource attribute, or freshness signal changes.

## Decide

`internal/infra/telemetry/metrics_otel.go` sets a per-instrument cardinality
limit of 2000, leaving 1999 distinct attribute sets before overflow. Size the
product of all label values, not each label independently. Extra sets merge
silently into `otel.metric.overflow=true`, so alert on that series.

A bounded label uses one closed vocabulary plus fallback beside its owner. Every
producer maps through it; request, tenant, user, error text, and arbitrary IDs
do not become metric labels. Put entity pivots in structured logs/traces.

Prometheus export transforms names through underscore escaping and suffixes:
dots become underscores, units add suffixes (`s` -> `_seconds`, `By` ->
`_bytes`), and counters add `_total`. Unit `1` adds `_ratio`; use a brace unit
such as `{request}` for counts when the exporter should add no unit suffix.

`newResource` owns process-wide `service.name`, `service.version`,
`vcs.revision`, `service.instance.id`, and `deployment.environment.name`.
Resolve instance identity once; do not duplicate resource fields on every
instrument. Ambient resource attributes merge below explicit ones.

For freshness, publish source and observation timestamps and subtract at query
time. A precomputed age can freeze when observation stops. Exemplars provide
trace pivots only for sampled spans, so they are optional evidence rather than
the sole entity locator.

## Prove

State the label product against 1999, vocabulary/fallback and producer test,
exported Prometheus name, and the reader path that receives the instrument.
Reject any duplicate of existing otelhttp, otelgrpc, or otelpgx signals without
a distinct operator decision.
