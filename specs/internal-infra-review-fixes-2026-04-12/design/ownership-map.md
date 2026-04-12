# Ownership Map

## Sources Of Truth

| Concern | Owner | Notes |
| --- | --- | --- |
| Runtime config keys, defaults, snapshot, validation | `internal/config` | Uses `otelconfig` vocabulary but still owns config loading and error semantics. |
| OTel config vocabulary | `internal/observability/otelconfig` | Narrow pure package; no SDK imports and no config loading. |
| OTel SDK setup | `internal/infra/telemetry` | Builds resources, samplers, exporters, propagators, tracer provider. |
| Startup composition | `cmd/service/internal/bootstrap` | Passes config snapshot into telemetry and records startup dependency metrics. |
| HTTP server wrapper | `internal/infra/http` | Owns nil/zero-value behavior of `Server`. |
| SQLC fixture behavior | `internal/infra/postgres` plus `internal/infra/postgres/queries` | Fixture remains replaceable and non-production. |

## Dependency Direction

Allowed:

- `internal/config -> internal/observability/otelconfig`
- `internal/infra/telemetry -> internal/observability/otelconfig`
- `cmd/service/internal/bootstrap -> internal/config`
- `cmd/service/internal/bootstrap -> internal/infra/telemetry`
- `cmd/service/internal/bootstrap -> internal/infra/http`
- `internal/infra/postgres -> internal/infra/postgres/sqlcgen`

Not allowed:

- `internal/config -> internal/infra/telemetry`
- `internal/infra/telemetry -> internal/config`
- new generic `internal/common`, `internal/shared`, or `internal/util`
- generated SQLC code edits by hand

## Label Ownership

- Startup dependency names and mode label strings stay in bootstrap because they describe bootstrap lifecycle policy.
- Startup dependency status encoding stays in telemetry because it maps semantic status to Prometheus gauge values.
- Telemetry init failure reason constants move to telemetry because they are metric label vocabulary; bootstrap may classify errors into those constants.
