# Runtime Sequence

Status: approved

## HTTP Request Flow After Change

1. `cmd/service/internal/bootstrap` passes validated `HTTP.MaxBodyBytes` to `httpx.NewRouter`.
2. `NewRouter` rejects missing or non-positive `MaxBodyBytes`.
3. The handler stack is assembled with `RequestCorrelation` outermost and one edge-wide `otelhttp` middleware inside it.
4. Request execution flows through security headers, access logging, request body limit, request framing guard, panic recovery, then the chi root router.
5. Matched generated routes and the manual `/metrics` route run route-template capture after chi resolves the route.
6. Route capture sets:
   - route holder value for access logs and Prometheus labels,
   - active span name as `METHOD template`,
   - `http.route` span attribute and `otelhttp` labeler attribute using the path template.
7. If a downstream handler panics, deferred route capture still attempts to set route metadata before outer recovery writes the 500 Problem response.
8. Edge-wide `otelhttp` records the final status and metrics after the full HTTP policy handler returns.

Failure points:
- Unknown route: no route-local capture runs; access log and custom metrics keep `"<unmatched>"`; edge trace still exists.
- 405/OPTIONS/CORS fallback: edge trace exists; route may remain unmatched unless chi exposes a route pattern.
- Body/framing rejection: edge trace exists and final status is captured.
- Panic: `Recover` owns sanitized logging and Problem response; edge trace records 500.

## `/metrics` Route Ownership

1. Root router registers manual `GET /metrics` before mounting the generated API subrouter.
2. Manual `/metrics` serves `strict.metrics.Handler()` directly.
3. Generated strict `Metrics` remains only to satisfy the generated interface and returns an error if called.
4. Existing route priority and route-tree tests prove root ownership.

## Postgres Pool Release Flow After Change

1. `postgres.New` validates config and builds `pgxpool.Config`.
2. It installs a stateful max-idle limiter when configuring pgxpool.
3. On healthy connection release, pgxpool calls `AfterRelease` asynchronously before returning the connection to the pool.
4. The limiter records the connection as retained only when doing so stays within `MaxIdleConns`.
5. If the limit is full, `AfterRelease` returns false and pgxpool destroys the connection.
6. When an idle connection is acquired, `BeforeAcquire` removes it from the retained-idle set.
7. When pgxpool closes a retained connection, `BeforeClose` removes it from the retained-idle set.

Failure points:
- `MaxIdleConns == 0`: released healthy connections are not retained.
- Concurrent releases: limiter state serializes the retention decision and avoids the old duplicate pre-release idle count.
- Connection close while idle: `BeforeClose` prevents stale accounting.

## SQLC Repository Construction Flow After Change

1. Runtime callers pass `*pgxpool.Pool` to `NewPingHistoryRepository`.
2. Constructor rejects nil pool and returns `ErrPingHistoryRepository`.
3. Tests that need fakes use a package-local constructor/helper that accepts `pingHistoryDB`.
4. Repository methods check readiness before calling generated SQLC queries.
5. Invalid zero-value repository use returns `ErrPingHistoryRepository` instead of panicking.

## Metrics Method Flow After Change

1. `telemetry.New` still creates the real registry and collectors.
2. Collector methods return immediately if the receiver or required collector field is nil.
3. `Handler` returns NotFound if receiver or registry is nil.
4. Normal runtime metrics behavior is unchanged for instances built with `telemetry.New`.
