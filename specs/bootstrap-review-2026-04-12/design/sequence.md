# Sequence

## Runtime Startup Sequence After Fix

1. `Run` parses load options and creates startup contexts as today.
2. `bootstrapRuntime` loads config and configures logging as today.
3. Before creating an OTLP exporter, telemetry target admission runs:
   - telemetry describes the effective configured exporter target using the same precedence as exporter option construction;
   - bootstrap loads/applies network policy through the existing bootstrap-owned policy code;
   - bootstrap calls `EnforceEgressTarget(target, scheme)` for the described OTLP target when configured.
4. If the target is allowed, `telemetry.SetupTracing` may create the exporter.
5. If the target is denied, tracing setup degrades to feature-off and no exporter is created. Startup continues unless the underlying `NETWORK_*` policy configuration is invalid and the normal network-policy stage later rejects startup.
6. Bootstrap continues through startup span/reporting, network policy ingress/exception validation, dependency probes, router construction, and HTTP serve/shutdown as today.

## Failure Semantics

- Invalid OTLP endpoint syntax: telemetry setup error, feature-off tracing, startup continues.
- OTLP endpoint denied by egress policy: telemetry setup error/degraded status, no exporter, startup continues.
- Invalid `NETWORK_*` configuration: startup rejects through existing policy violation path.
- Postgres DSN parse in probe-address extraction: return a secret-safe `errDependencyInit`-classified error; do not expose raw DSN content in logs or final errors.

## Implementation Order

1. Add telemetry target-description behavior and tests.
2. Gate telemetry setup in bootstrap using the target description and network policy.
3. Refactor config stage duration mapping.
4. Resolve Postgres parse diagnostic safety.
5. Clarify dependency probe stage labels.
6. Run focused tests and package validation.
