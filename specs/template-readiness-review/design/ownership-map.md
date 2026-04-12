# Ownership Map

## Source Of Truth

| Concern | Owner | Implementation Rule |
| --- | --- | --- |
| Business use-case behavior | `internal/app/<feature>` | Keep feature-local types and app-owned ports here first. |
| Shared domain contracts | `internal/domain` | Promote only after a stable shared contract exists. |
| REST API contract | `api/openapi/service.yaml` | Edit contract first, regenerate `internal/api`, do not hand-edit generated code. |
| HTTP mapping and route policy | `internal/infra/http` | Map generated request objects to app calls and app errors to contract-shaped responses. |
| Schema | `env/migrations` | Treat `ping_history` as a template fixture unless deliberately replaced. |
| SQL queries | `internal/infra/postgres/queries` | Regenerate `sqlcgen`; do not hand-edit generated output. |
| App-facing persistence port | `internal/app/<feature>` | Keep the port beside the consuming feature. |
| Postgres adapter | `internal/infra/postgres` | Map `sqlcgen` rows into app-facing types; do not leak generated types into app code. |
| Composition root | `cmd/service/internal/bootstrap` | Wire concrete adapters and app services here. |
| Runtime config snapshot | `internal/config` | Own typed config/defaults/validation, not feature behavior or integration runtime semantics. |
| Shared telemetry runtime | `internal/infra/telemetry` | Keep shared HTTP/bootstrap instruments here; feature metrics start with the owning feature or adapter. |

## Boundary Rules To Preserve

- `internal/app` must not import `internal/infra/*`, generated SQLC packages, or concrete DB drivers.
- `internal/infra/http` may import `internal/api` and `internal/app/*`.
- `cmd/service/internal/bootstrap` may import concrete adapters because it is the composition root.
- Redis/Mongo cache/store semantics must not become config/bootstrap behavior before a real feature and adapter own them.
- Mongo/Redis probe helpers in config/bootstrap remain guard-only until a real adapter ownership decision exists.
- Protected endpoint behavior must be designed from a real security policy, not from placeholder middleware.
