# Trace Context And Correlation

## Behavior Change Thesis
When loaded for propagation, correlation identifiers, or "pass this value to the downstream service", this file makes the model choose an explicit egress policy for the target instead of the likely mistake: designing an allowlisted-baggage solution, which in this repository propagates nothing — baggage is never injected and is stripped on the way out, so the design silently no-ops and the value never arrives.

## When To Load
Load this when a change crosses a service boundary with correlation attached: a new outbound client, a value that must reach a downstream, async lineage, or log-to-trace correlation.

## Decision Rubric

**Baggage does not travel here.** `SetupTracing` registers `propagation.TraceContext{}` alone — no composite, no `propagation.Baggage`. Both outbound clients go further: `internal/infra/httpclient/propagation.go` and `internal/infra/grpcclient/propagation.go` strip `traceparent`, `tracestate`, `baggage`, and the request-ID header from every attempt before injecting. A value that must reach a downstream travels as an explicit field of the request, or the policy is extended deliberately by its owner.

**Egress is a per-target policy and fail-closed.** `PropagationPolicy` has three values and the zero value is `PropagationNone`. `PropagationTraceContext` emits W3C trace context; `PropagationTrustedService` adds the request ID and is the choice that asserts the target is trusted with it. A new client picks one explicitly — the default emits no remote correlation at all, so an unset policy is a silent loss of end-to-end tracing rather than a leak.

**Correlation on logs is already automatic.** `internal/observability/logctx` publishes `request_id`, `trace_id`, and `span_id` on every record from the context it was logged with. Adding them by hand duplicates them; logging without a context drops them, which is what the repository's sloglint `context: scope` setting catches. The pivot from an alert to a trace to a log runs on these keys and needs nothing added.

**Trace IDs stay valid when tracing is off.** When no exporter resolves, the sampler falls back to `NeverSample` rather than disabling tracing, deliberately: span contexts stay valid so propagation and log correlation keep working with no span recorded. Correlation surviving is therefore not evidence that spans are being exported — `RecordTraceExporterState` publishes that separately.

**Lineage that is not single-parent uses links.** Batch, fan-in, retry, and redrive processing take span links rather than electing one message as parent. Links known at span start belong in the creation options; added later they miss the sampler decision.

## Reject
- Baggage as a transport for tenant, user, or plan values, because nothing registers a baggage propagator and the sanitizer removes the header regardless.
- A new outbound client that leaves `Propagation` at its zero value while expecting distributed traces, because the zero value is `PropagationNone` and the trace ends at this service.
- Re-adding `trace_id` or `request_id` to a log call, because `logctx` already published them and the second copy is what drifts.

## Validation Shape
- Name the `PropagationPolicy` chosen for each new target and what it asserts about that target's trust.
- For async work, state whether lineage is parent-child or linked, and why.
- Confirm one alert-to-trace-to-log pivot works from the keys `logctx` already publishes.
