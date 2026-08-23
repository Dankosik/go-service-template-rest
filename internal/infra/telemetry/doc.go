// Package telemetry owns this process's OpenTelemetry setup: the tracer and
// meter providers, the OTLP exporters and their endpoint resolution, the shared
// resource, and the Prometheus scrape handler.
//
// It owns the SDK, not the policy that configures it. The values it installs
// come from internal/config, and the pure OTel policy those two must agree on
// lives in internal/observability/otelconfig — a leaf, because config may not
// import a runtime adapter.
//
// # Extending it
//
// A feature or adapter that needs its own metrics builds them directly from the
// OpenTelemetry meter in a telemetry.go of its own, beside the code they measure.
// Do not add a feature's instruments here — this package would then change every
// time an unrelated adapter gains a counter, and the instrument would sit a
// package away from the operation it counts.
//
// Add to this package only what the process shares: provider and exporter
// construction with no transport- or feature-owned series (tracing.go,
// metrics_otel.go), the ambient OTEL_EXPORTER_OTLP_* environment policy
// (exporter_env.go), endpoint resolution (otlp_endpoint.go), OTLP header parsing
// (otlp_headers.go), the resource identity every signal carries (resource.go),
// or the classifier that keeps exporter failures out of the logs verbatim
// (failure.go).
//
// Test support lives in telemetrytest, which owns the global-state install and
// restore that no test may do for itself.
package telemetry
