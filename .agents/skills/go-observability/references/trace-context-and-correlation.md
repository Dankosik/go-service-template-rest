# Trace Context And Correlation

## Load When
Load this when a change crosses a service boundary with correlation attached: a new outbound client, a value that must reach a downstream, async lineage, or log-to-trace correlation.

## Decide

**Baggage does not travel here.** `SetupTracing` registers `propagation.TraceContext{}` alone — no composite, no `propagation.Baggage`. The fixed HTTP client strips `traceparent`, `tracestate`, `baggage`, and the request-ID header and injects nothing; gRPC strips them before applying its explicit policy. A value that must reach a downstream travels as an explicit field of the request, or the policy is extended deliberately by its owner.

**gRPC egress is a per-target policy and fail-closed.** `PropagationPolicy` has three values and the zero value is `PropagationNone`. `PropagationTraceContext` emits W3C trace context; `PropagationTrustedService` adds the request ID and is the choice that asserts the target is trusted with it. Fixed HTTP provider calls emit no remote correlation.

**Correlation on logs is already automatic.** `internal/observability/logctx` publishes `request_id`, `trace_id`, and `span_id` on every record from the context it was logged with. Adding them by hand duplicates them; logging without a context drops them, which is what the repository's sloglint `context: scope` setting catches. The pivot from an alert to a trace to a log runs on these keys and needs nothing added.

**Trace IDs stay valid when tracing is off.** When no exporter resolves, the sampler falls back to `NeverSample` rather than disabling tracing, deliberately: span contexts stay valid so propagation and log correlation keep working with no span recorded. Correlation surviving is therefore not evidence that spans are being exported — `RecordTraceExporterState` publishes that separately.

**Lineage that is not single-parent uses links.** Batch, fan-in, retry, and redrive processing take span links rather than electing one message as parent. Links known at span start belong in the creation options; added later they miss the sampler decision.

## Reject
- Baggage as a transport for tenant, user, or plan values, because nothing registers a baggage propagator and the sanitizer removes the header regardless.
- A fixed HTTP provider call that expects distributed traces without a separate accepted telemetry policy, because the shared client deliberately emits no remote correlation.
- Re-adding `trace_id` or `request_id` to a log call, because `logctx` already published them and the second copy is what drifts.

## Prove
- Name the `PropagationPolicy` chosen for each new target and what it asserts about that target's trust.
- For async work, state whether lineage is parent-child or linked, and why.
- Confirm one alert-to-trace-to-log pivot works from the keys `logctx` already publishes.
