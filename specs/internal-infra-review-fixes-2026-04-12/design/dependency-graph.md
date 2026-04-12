# Dependency Graph

## Target Graph

```text
internal/observability/otelconfig
  -> standard library only

internal/config
  -> internal/observability/otelconfig
  -> config dependencies

internal/infra/telemetry
  -> internal/observability/otelconfig
  -> Prometheus / OpenTelemetry SDK packages

cmd/service/internal/bootstrap
  -> internal/config
  -> internal/infra/telemetry
  -> other app/infra packages
```

## Guardrails

- `internal/observability/otelconfig` must not import `internal/config`, `internal/infra/telemetry`, or OTel SDK packages.
- `internal/config` must not import `internal/infra/telemetry`.
- `internal/infra/telemetry` must not import `internal/config`.
- If these guardrails cannot be kept, reopen technical design before implementation continues.
