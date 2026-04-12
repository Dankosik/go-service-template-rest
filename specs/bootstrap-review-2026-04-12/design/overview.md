# Design Overview

## Chosen Approach

Keep the fix local to bootstrap and telemetry:

- `internal/infra/telemetry` should own OTLP endpoint selection/parsing and expose a pure, SDK-free target description for bootstrap.
- `cmd/service/internal/bootstrap` should own network-policy admission and decide whether a telemetry target is allowed before calling `telemetry.SetupTracing`.
- Tracing remains optional fail-open: denied telemetry egress disables tracing and records degraded telemetry status, while invalid global `NETWORK_*` policy still blocks startup through the existing network-policy path.

The implementation should also clean up two maintainability seams in bootstrap: one config-stage duration mapping for metrics/spans, and clearer dependency probe stage/label ownership. The Postgres parse-error item should be resolved as a secret-safe diagnostic guard, not as raw cause wrapping by default.

## Artifact Index

- `component-map.md`: package/file surfaces and stable areas.
- `sequence.md`: startup sequence and implementation ordering.
- `ownership-map.md`: source-of-truth and dependency ownership.

No data model, contract, rollout, or dependency-graph artifact is triggered.

## Important Design Notes

- The OTLP exporter can be configured by `OTLPEndpoint`, `OTLPTracesEndpoint`, or currently by headers alone causing SDK default/env endpoint behavior. The fix must close all of these target-selection paths, not only explicit endpoint strings.
- Do not move network policy ownership into `internal/infra/telemetry`; that would invert the current boundary.
- Do not duplicate the telemetry endpoint parser in bootstrap; that would create another source-of-truth seam.
- Do not change the global network-policy rule that scheme denial happens before host classification.
- The Postgres parse-address error currently losing the parser cause is likely a secret-safety decision even if it is not documented. Treat it as intentional until proven safe to expose.

## Readiness

Planning is ready. Implementation can start from `tasks.md` in a separate session.
