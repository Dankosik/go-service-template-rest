# Reference Selector

Load one reference for the pressure the change actually creates, and another only for an independent one. Each file states the accepted policy and the repository surface that enforces it, so the same file serves a Decision and a Review.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| A metric label, instrument, unit, or resource attribute is added or widened | [metric-identity-and-cardinality.md](metric-identity-and-cardinality.md) | Sizes the label product against the enforced per-instrument limit and names the instrument as the Prometheus reader will export it, rather than trusting today's value count. |
| Correlation crosses a service boundary, or a value must reach a downstream | [trace-context-and-correlation.md](trace-context-and-correlation.md) | Picks an explicit egress policy for the target, rather than an allowlisted-baggage design that propagates nothing here. |
| A log event or field is added, or request, response, or query content is proposed for logging | [structured-logs-and-privacy.md](structured-logs-and-privacy.md) | Adds the field that discriminates between failures sharing a status, rather than re-adding correlation the logger already publishes. |
| A readiness check, drain behavior, or debug surface changes | [health-probes-and-diagnostics.md](health-probes-and-diagnostics.md) | Extends the cached readiness verdict, rather than probing a dependency per request. |

Valid as of 2026-08-04, claims here are pinned to `go.mod`: OpenTelemetry Go 1.44.0, `prometheus/client_golang` v1.24.1, `prometheus/otlptranslator` v1.0.0, `otelpgx` v0.11.1, `otelhttp`/`otelgrpc` v0.69.0, repository `semconv/v1.37.0`. When a name, unit, or convention status decides the answer, verify against the primary OpenTelemetry, W3C Trace Context, or Prometheus source for the pinned version rather than the latest published one.
