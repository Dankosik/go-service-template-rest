# internal/infra Review Fixes Spec

Status: implemented and validated
Spec clarification gate: waived under the lightweight local phase-collapse recorded in `workflow-plan.md`.

## Context

The read-only review of `internal/infra` found bounded maintainability and correctness risks in the HTTP, Postgres, and telemetry adapters. This spec converts the accepted findings into implementation decisions without changing feature behavior or broad repository architecture.

The runtime architecture remains:
- `internal/config` owns validated runtime config defaults and constraints.
- `internal/infra/http` owns HTTP edge policy, route mapping, route observability, and Problem responses.
- `internal/infra/postgres` owns pgx/pgxpool lifecycle and SQLC-backed repository adapter code.
- `internal/infra/telemetry` owns Prometheus and OpenTelemetry adapters.
- `api/openapi/service.yaml` and `internal/api` remain the REST contract and generated bindings source of truth; generated code is not edited by hand.

Fresh review validation before this spec:
- `go vet ./internal/infra/...` passed.
- `go test -count=1 ./internal/infra/...` passed with 110 tests across 4 packages.

## Scope / Non-goals

In scope:
- Make Postgres idle-connection retention match the `MaxIdleConns` contract or make the contract explicit in code.
- Remove typed-nil and nil-receiver traps around the Postgres pool and SQLC sample repository.
- Make HTTP tracing cover the same edge policy surface as access logs and request metrics.
- Add route-template trace attributes, not only span names.
- Make `Metrics` safe for nil or zero-value use, matching its nil-safe method shape.
- Remove duplicated HTTP body-limit defaulting from the router.
- Make the generated strict `/metrics` handler explicitly non-runtime-owned because the root route owns `/metrics`.
- Include low-risk readability cleanups when they touch the same files and reduce future confusion.

Non-goals:
- No OpenAPI contract change.
- No hand edits under `internal/api` or `internal/infra/postgres/sqlcgen`.
- No schema, migration, or SQL query change.
- No new auth or network exposure policy for `/metrics`.
- No broad telemetry product redesign or SLO/SLA definition.
- No rewrite of the SQLC sample repository into production business behavior.

## Constraints

- Preserve existing public endpoint behavior for `/api/v1/ping`, `/health/live`, `/health/ready`, and the root `/metrics` route.
- Preserve fail-closed CORS and existing 404/405 Problem response behavior.
- Avoid nested HTTP server spans when moving OpenTelemetry instrumentation to the edge.
- Keep route labels bounded: use route templates or the existing `"<unmatched>"` fallback, never raw request paths as metric labels.
- Keep `MaxOpenConns` as the hard upper bound even when adjusting idle retention.
- Keep implementation local to handwritten infra/test files unless a compile-time signature change requires call-site updates.
- Treat existing unrelated worktree changes outside this task path as untouchable.

## Decisions

1. Postgres idle retention will be implemented as an explicit stateful limiter if `MaxIdleConns` remains a strict count contract.
   - Use a small package-local limiter that composes with `pgxpool.Config.AfterRelease`, `BeforeAcquire`, and `BeforeClose`.
   - Track retained idle connections by `*pgx.Conn` identity under a mutex or equivalent safe state.
   - Count connections accepted by `AfterRelease` as retained before returning `true`.
   - Remove tracked state on `BeforeAcquire` and `BeforeClose`.
   - Keep `MaxIdleConns <= 0` as "do not retain released connections".
   - Replace the current `pool.Stat().IdleConns()` check because pgxpool invokes `AfterRelease` asynchronously before returning the connection to the pool.

2. `(*Pool).DB` will become nil-safe.
   - Return nil when the receiver or inner pool is nil.
   - Keep `Close` and `Check` nil behavior consistent with the existing contract.

3. `PingHistoryRepository` construction will reject invalid DB dependencies at the exported boundary.
   - Prefer changing `NewPingHistoryRepository` to accept `*pgxpool.Pool` and return `(*PingHistoryRepository, error)`.
   - Keep a package-local constructor/helper for tests that need a `pingHistoryDB` fake.
   - Add method-level guards so a nil or zero-value repository returns `ErrPingHistoryRepository` instead of panicking.
   - Update integration and unit tests for the constructor signature.

4. HTTP router config will require an explicit `MaxBodyBytes`.
   - `NewRouter` should return an error when `RouterConfig.MaxBodyBytes <= 0`.
   - Tests should pass the config-owned default explicitly in their helper.
   - Do not keep `1 << 20` as a router-local fallback.

5. HTTP OpenTelemetry server instrumentation will move to one edge-wide wrapper.
   - The edge wrapper should sit inside `RequestCorrelation` so extracted/created trace context is available to downstream middleware and access logging.
   - It should wrap the security headers, access log, body limit, framing guard, panic recovery, and root router surfaces.
   - Remove route-local `otelMiddleware` around generated operations and manual `/metrics` to avoid nested server spans.
   - Keep post-routing route-template capture as route-local middleware.

6. Route capture will set both span name and route attribute.
   - Derive a route path template separately from the existing `METHOD path` display label.
   - Set span name to the bounded `METHOD template` label.
   - Set `http.route` on the active span and, when available, on the `otelhttp` labeler before the edge middleware records metrics.
   - Use `defer` in route capture so route metadata is attempted even when a downstream handler panics and outer recovery handles the response.

7. The generated strict `Metrics` method will no longer buffer metrics through `httptest.NewRecorder`.
   - Keep the method only to satisfy the generated strict interface.
   - Return an internal error with a short comment that `/metrics` is served by the documented root-router exception.
   - Preserve existing root route priority and route-tree tests.

8. `Metrics` methods will be nil and zero-value tolerant.
   - Keep no-op behavior for nil receivers.
   - Add field-level guards so direct zero-value use does not panic.
   - `Handler` should return `http.NotFoundHandler()` when the registry is not initialized.

9. Low-risk readability cleanups are allowed when colocated with the above work.
   - Rename `isSafeOTLPHeaderKey` to a redaction-specific name such as `canEchoOTLPHeaderKeyInError`.
   - Make `fakePingHistoryQuerier` callbacks optional in tests and return targeted unexpected-call errors instead of nil-function panics.

10. `SetupTracing` default fallback behavior is not changed in this batch.
   - Runtime config still passes validated values through bootstrap.
   - Tightening `SetupTracing` to reject zero values would be a separate config/telemetry ownership decision because the adapter currently has useful defensive direct-use defaults.

## Open Questions / Assumptions

- Assumption: keeping `MaxIdleConns` as a strict count contract is preferred over deprecating or softening the config key.
- Assumption: changing the exported `NewPingHistoryRepository` signature is acceptable because this is an internal package and current known callers can be updated in the same change.
- Assumption: trace route attributes should use `http.route` directly if the semconv helper is not convenient from this package.
- Assumption: no separate review phase is required before implementation; the implementation session can run focused self-review plus validation.

Reopen specification if:
- `MaxIdleConns` should become best-effort or be replaced by a `MaxConnIdleTime`-style contract instead of a strict limiter.
- The `NewPingHistoryRepository` signature must remain source-compatible.
- Edge-wide tracing is rejected for telemetry cardinality, overhead, or trace-shape reasons.
- `SetupTracing` fallback defaults are pulled back into scope.

## Plan Summary / Link

Implementation followed `plan.md` and `tasks.md` in this directory. The work completed as one implementation phase plus validation because the changes were local, bounded, and shared test surfaces.

## Validation

Minimum proof for implementation:
- `go test -count=1 ./internal/infra/...`
- `go vet ./internal/infra/...`

Additional targeted proof expected:
- HTTP route policy and observability tests under `internal/infra/http`.
- Postgres pool limiter unit tests under `internal/infra/postgres`.
- SQLC repository constructor and zero-value guard tests under `internal/infra/postgres`.
- `createAndListRecentInTx` success-path and commit-error tests under `internal/infra/postgres`.
- Metrics zero-value tests under `internal/infra/telemetry`.

Optional, if Docker/tooling is available and the implementation touches integration call sites:
- `go test -tags=integration ./test/... -run 'TestPingHistoryRepositorySQLCReadWrite' -count=1`

Not required unless generated artifacts or migrations change:
- `make sqlc-check`
- `make migration-validate`
- `make openapi-check`

Actual evidence on 2026-04-12:
- `go test -count=1 ./internal/infra/...` passed.
- `go vet ./internal/infra/...` passed.
- `go test -tags=integration ./test/... -run 'TestPingHistoryRepositorySQLCReadWrite' -count=1` passed with Docker available.
- `go test -race ./internal/infra/postgres -run TestMaxIdleConnLimiterConcurrentReleases -count=1` passed.
- `git status --short` inspected the worktree.
- `internal/api` and `internal/infra/postgres/sqlcgen` had no dirty generated-file changes.

## Outcome

Implemented and validated. T001-T009 are complete in `tasks.md`.
