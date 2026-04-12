# Component Map

Status: approved

## `internal/infra/postgres/postgres.go`

Changes:
- Replace `shouldKeepReleasedConn(maxIdleConns, idleConnsBeforeRelease)` with a stateful limiter that tracks retained idle connections by connection identity.
- Install limiter hooks on `pgxpool.Config`:
  - `AfterRelease`: decide whether a released healthy connection may return to idle.
  - `BeforeAcquire`: untrack a retained idle connection when it leaves idle use.
  - `BeforeClose`: untrack a retained idle connection when pgxpool closes it.
- Make `(*Pool).DB` nil-safe.

Stable:
- `Pool.Close`, `Pool.Name`, and `Pool.Check` behavior.
- `ErrConfig`, `ErrConnect`, and `ErrHealthcheck` classifications.
- `MaxOpenConns`, `ConnectTimeout`, `HealthcheckTimeout`, and `ConnMaxLifetime` validation semantics.

## `internal/infra/postgres/ping_history_repository.go`

Changes:
- Reject invalid DB dependencies in `NewPingHistoryRepository`.
- Prefer exported constructor signature `NewPingHistoryRepository(db *pgxpool.Pool) (*PingHistoryRepository, error)`.
- Keep an unexported constructor/helper for tests that need `pingHistoryDB` fakes.
- Add repository readiness guards for nil/zero-value repository methods.

Stable:
- SQLC query source files.
- Generated `sqlcgen` code.
- `PingHistoryRecord` shape.
- `ErrPingHistoryRepository` classification.

## `internal/infra/postgres/*_test.go` And `test/postgres_sqlc_integration_test.go`

Changes:
- Update constructor call sites for the new return error.
- Add tests for nil constructor input, zero-value repository behavior, nil-safe `Pool.DB`, and limiter accounting.
- Improve fake query callbacks when touching the same unit tests.

Stable:
- Container-backed integration test behavior.
- Migration application helper behavior.

## `internal/infra/http/router.go`

Changes:
- Require `RouterConfig.MaxBodyBytes > 0` and return an explicit router config error otherwise.
- Create one edge-wide OpenTelemetry middleware wrapper in the main handler stack.
- Remove route-local `otelMiddleware` from generated handler and `/metrics` wrapper construction.

Stable:
- Root `/metrics` registration before mounted generated routes.
- Manual root route exception map and route-tree tests.
- `RequestCorrelation`, `SecurityHeaders`, `AccessLog`, `RequestBodyLimit`, `RequestFramingGuard`, and `Recover` as the primary HTTP policy stack.

## `internal/infra/http/middleware.go`

Changes:
- Separate route path template extraction from display label formatting.
- Make route capture update span name, `http.route` span attribute, and `otelhttp` labeler attribute.
- Use deferred route capture around downstream handling so panics still get route metadata where chi has resolved a route.

Stable:
- Request ID context behavior.
- Bounded route display labels.
- Access log fields and custom Prometheus metric labels.

## `internal/infra/http/handlers.go`

Changes:
- Remove production use of `httptest.NewRecorder`.
- Make strict generated `Metrics` return an internal error because runtime `/metrics` is owned by the root route.

Stable:
- `Ping`, `HealthLive`, and `HealthReady` strict handlers.
- Readiness timeout and gate behavior.

## `internal/infra/telemetry/metrics.go`

Changes:
- Add field-level nil guards for each exported method that uses collectors.
- Make `Handler` return `http.NotFoundHandler()` when `registry` is nil.

Stable:
- Metric names and label sets.
- `normalizeTelemetryFailureReason`.
- Registry creation through `telemetry.New`.

## `internal/infra/telemetry/tracing.go`

Changes:
- Optional low-risk rename: `isSafeOTLPHeaderKey` -> `canEchoOTLPHeaderKeyInError`.

Stable:
- `SetupTracing` defaults and exporter option parsing.
- OTLP endpoint/header parsing behavior.
