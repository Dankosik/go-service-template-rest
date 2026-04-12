# Component Map

## `internal/infra/telemetry`

Expected changes:

- Add `startupRejections *prometheus.CounterVec` to `Metrics`.
- Register `startup_rejections_total` with label `reason`.
- Add `IncStartupRejection(reason string)`.
- Add reason normalization for bounded cardinality.
- Update metrics tests to include the new series and zero-value behavior.

Stable behavior:

- HTTP request metrics, telemetry init failure metrics, startup dependency status, and metrics handler behavior remain unchanged.
- Do not reintroduce removed series such as `network_policy_violation_total`.

## `cmd/service/internal/bootstrap`

Expected changes:

- In config failure path, keep config failure metric and add startup rejection metric.
- In policy/dependency/HTTP startup rejection helpers, switch from `IncConfigValidationFailure` to `IncStartupRejection`.
- Convert `serveHTTPRuntime` to accept a named unexported argument struct.
- Convert `recordDependencyProbeRejection` to use `startupDependencyProbeLabels`.
- Rename `parseOptionalBoolEnvWithPresence` to explicit declaration terminology.
- Rename `EmitEgressExceptionState` to validation terminology.
- Update focused bootstrap tests for metric assertions and refactored call sites.

Stable behavior:

- Startup and shutdown sequence remains unchanged.
- Span attributes and log event names remain unchanged unless a test forces a minor call-site update.
- Dependency probing, degraded-mode behavior, and policy enforcement remain unchanged.

## `internal/config`

Expected changes:

- Add `Config.PostgresReadinessProbeRequired()`.
- Add `Config.MongoReadinessProbeRequired()`.
- Use those predicates in readiness budget validation.
- Add tests mirroring the existing Redis readiness predicate tests.

Stable behavior:

- Redis readiness behavior remains unchanged.
- Config loading, parsing, validation ranges, and feature flag names remain unchanged.

## Documentation / Workflow Artifacts

Expected changes:

- Implementation may update this task's `tasks.md` progress.
- No repository docs are required unless implementation discovers dashboard compatibility or operator-facing docs that must mention the new metric.
