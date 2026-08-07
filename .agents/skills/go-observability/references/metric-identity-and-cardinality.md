# Metric Identity And Cardinality

## Behavior Change Thesis
When loaded for a metric name, unit, label, or resource attribute, this file makes the model size the label product against this repository's enforced per-instrument limit and name the instrument in the form the Prometheus reader will actually export, instead of the likely mistake: judging a label "bounded enough" from today's value count and assuming an over-budget instrument fails loudly.

## When To Load
Load this when a change adds or widens a metric label, introduces an instrument, chooses a unit, or touches resource identity.

## Decision Rubric

**The limit is real, per-instrument, and fails silently.** `internal/infra/telemetry/metrics_otel.go` sets `sdkmetric.WithCardinalityLimit(2000)` on the meter provider. The budget is per instrument over the *product* of its label values, not per service and not per label. The SDK admits `aggLimit-1` distinct attribute sets, so the effective budget is 1999.

Exceeding it does not reject the new series and logs nothing. Every attribute set past the limit is merged into one stream carrying only `otel.metric.overflow=true`, so the instrument keeps reporting and its numbers stop meaning anything — a `rate()` over the overflow stream sums an arbitrary mix of label combinations. Size a new label as `existing series x new values` against 1999, and treat `otel.metric.overflow` as a series worth alerting on.

**A finite label is a closed vocabulary, not merely a cardinality bound.** Keep the accepted values and one fallback beside the code that bounds the label. Every current producer must select from that vocabulary, and one test must enumerate or drive those producers through a shared corpus so an unknown value proves it collapses to the fallback. Capping arbitrary strings without proving their semantic mapping leaves the instrument bounded but unreadable.

**The exported name is not the name you wrote.** Metrics reach a Prometheus registry through `otlptranslator.UnderscoreEscapingWithSuffixes`, so the instrument name is transformed: dots become underscores, the unit is appended (`s` -> `_seconds`, `By` -> `_bytes`), and counters gain `_total`. Two units are traps: unit `1` appends `_ratio` to gauges, which misnames a boolean state; a unit in braces (`{request}`, `{rpc}`) appends nothing and is how this repo spells "count of things". Pick the unit for the name it produces.

**Resource identity is set once, in one place.** `newResource` in `internal/infra/telemetry/resource.go` publishes `service.name`, `service.version`, `vcs.revision`, `service.instance.id`, and `deployment.environment.name` — the renamed convention, not the older `deployment.environment`. `service.instance.id` comes from `ResolveInstanceID` and must be resolved once per process, because traces and metrics resolving separately attribute one replica to two instances. Ambient `OTEL_RESOURCE_ATTRIBUTES` merges underneath these and is deliberately preserved. Resource attributes are already on every series; copying them into instrument labels multiplies the product for nothing.

**Publish a timestamp, subtract at query time.** For freshness, `postgresoutbox` exports `outbox.relay.oldest.timestamp` (unit `s`) beside `outbox.relay.observation.timestamp`. A pre-computed age freezes at its last value when the observer stalls and reads as a healthy backlog; a timestamp pair makes a stalled observer and a genuinely old backlog different queries.

**Forensic pivots already exist.** `exemplar.TraceBasedFilter` is configured, so exemplars carry trace IDs for *sampled* spans only — under a ratio sampler, or when trace export is degraded and the sampler falls back to `NeverSample`, there are none. An entity-level pivot belongs in logs, which already carry `trace_id`.

## Reject
- A label whose values come from tenants, users, request data, or error text, because the product is unbounded and the failure is the silent overflow merge rather than a rejected write.
- A new instrument duplicating one otelhttp, otelgrpc, or otelpgx already emits, because the repo pins `semconv/v1.37.0` while otelhttp v0.69.0 emits v1.41.0 and a hand-rolled twin drifts from both.
- Reading the Prometheus scrape as proof that identity is right: the scraper supplies `instance` itself, which is what hid a missing `service.instance.id` until this service pushed over OTLP.

## Validation Shape
- State each new label's value source and the resulting series count against 1999 for that instrument.
- Name the closed vocabulary, fallback, current producers, and the test that proves their mapping.
- State the exported Prometheus name the chosen unit produces.
- Confirm a deployment reached only through the OTLP collector still gets the signal: the Prometheus **process** collector is scrape-only, while Go runtime metrics are registered on the meter provider and reach both readers.
