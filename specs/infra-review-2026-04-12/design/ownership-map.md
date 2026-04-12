# Ownership Map

Status: approved

## Config Ownership

Owner: `internal/config`

Rules:
- Defaults and validation for `http.max_body_bytes` remain in `internal/config`.
- `internal/infra/http` consumes a validated value and does not define a fallback default.
- Postgres option validation remains split as today: config validates the runtime snapshot, and `internal/infra/postgres.New` protects direct adapter use with local validation.

## HTTP Transport Ownership

Owner: `internal/infra/http`

Rules:
- The HTTP package owns middleware order, route fallback policy, route labels, and Problem response mapping.
- `/metrics` root route ownership stays manual and documented.
- The generated strict `Metrics` method must not become a second runtime route owner.
- OpenTelemetry edge instrumentation belongs to the HTTP adapter because it is transport observability.

## OpenAPI And Generated Code Ownership

Owner: `api/openapi/service.yaml` and generated `internal/api`

Rules:
- Do not edit generated `internal/api/openapi.gen.go`.
- Do not change the OpenAPI contract for this task.
- Runtime route ownership is tested through handwritten HTTP adapter tests, not by patching generated handlers.

## Postgres Adapter Ownership

Owner: `internal/infra/postgres`

Rules:
- pgxpool lifecycle, idle retention, healthcheck behavior, and concrete SQLC repository adapters are owned here.
- `env/migrations` remains the schema source of truth; this task does not change schema.
- SQLC generated files remain derived output; this task should not edit `sqlcgen`.
- The sample `PingHistoryRepository` remains a template fixture, not production ping behavior.

## Telemetry Adapter Ownership

Owner: `internal/infra/telemetry`

Rules:
- Prometheus registry and collector construction remain inside `telemetry.New`.
- Zero-value safety is a defensive adapter contract for exported methods.
- `SetupTracing` fallback defaults remain unchanged in this batch; config/telemetry ownership can be reopened separately if a stricter contract is desired.

## Test Ownership

Owners:
- HTTP behavior: `internal/infra/http/*_test.go`
- Postgres unit behavior: `internal/infra/postgres/*_test.go`
- Postgres container-backed behavior: `test/postgres_sqlc_integration_test.go`
- Telemetry behavior: `internal/infra/telemetry/*_test.go`

Rules:
- Add tests near the owning package first.
- Use integration-tag tests only if constructor signature changes affect real pgxpool usage or if unit tests cannot observe the behavior honestly.
