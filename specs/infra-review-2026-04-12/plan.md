# Implementation Plan: internal/infra Review Fixes

Status: complete

## Execution Context

This plan consumes `spec.md` plus the approved `design/` bundle in this directory. It is intentionally local: no generated code, OpenAPI contract, migrations, or app-layer behavior should change.

Execution shape: lightweight local, single implementation phase plus validation.

Phase-collapse rationale: the work is bounded to accepted review findings, all affected surfaces are handwritten infra/test files, and implementation will happen in a later session. Specification, technical design, and planning are collapsed here to give the next implementation session a complete handoff without starting code now.

## Phase Plan

Phase: implementation-phase-1
Objective: land all accepted infra review fixes in one cohesive infra maintenance change.
Depends On: approved `spec.md`, `design/overview.md`, `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md`.
Task Ledger Link / IDs: `tasks.md` T001-T009.
Acceptance Criteria:
- Postgres idle retention no longer relies on `pool.Stat().IdleConns()` as an async release guard.
- SQLC sample repository construction and zero-value use return classified errors instead of panicking.
- HTTP edge tracing covers router fallback, body/framing rejection, and recovery surfaces without nested normal-route server spans.
- Matched-route spans get bounded route names and `http.route` attributes.
- `Metrics` zero-value use is safe.
- Router body limit consumes explicit config instead of owning a fallback default.
- Runtime `/metrics` remains root-route-owned.
Change Surface:
- `internal/infra/postgres/postgres.go`
- `internal/infra/postgres/postgres_test.go`
- `internal/infra/postgres/ping_history_repository.go`
- `internal/infra/postgres/ping_history_repository_test.go`
- `test/postgres_sqlc_integration_test.go`
- `internal/infra/http/router.go`
- `internal/infra/http/middleware.go`
- `internal/infra/http/handlers.go`
- `internal/infra/http/router_test.go`
- `internal/infra/http/openapi_contract_test.go`
- `internal/infra/telemetry/metrics.go`
- `internal/infra/telemetry/metrics_test.go`
- `internal/infra/telemetry/tracing.go` and `tracing_test.go` only for the low-risk rename if touched.
Planned Verification:
- `go test -count=1 ./internal/infra/...`
- `go vet ./internal/infra/...`
- Optional: `go test -tags=integration ./test/... -run 'TestPingHistoryRepositorySQLCReadWrite' -count=1` when Docker is available.
Review / Checkpoint:
- Self-review after T006 before low-risk cleanup, checking for nested spans, generated-file edits, and constructor signature fallout.
Exit Criteria:
- All implementation tasks T001-T008 completed or explicitly deferred with rationale.
- Required validation commands pass or any unavailable optional command is reported honestly.

## Cross-Phase Validation Plan

Required:
- Run focused infra unit tests with `-count=1`.
- Run `go vet` for `internal/infra`.
- Inspect `git status --short` to confirm generated files and unrelated task artifacts were not modified unexpectedly.

Conditional:
- Run the Postgres integration test if Docker is available or if constructor signature updates touch integration call sites.
- Run generated drift checks only if generated inputs or generated outputs changed unexpectedly.

## Implementation Readiness

Status: PASS

Accepted risks:
- `SetupTracing` fallback ownership is intentionally left out of implementation scope.
- The exact implementation of the Postgres max-idle limiter may choose mutex state rather than atomic state if that is clearer and safer.

Proof obligations:
- Add tests that fail against the old stat-based Postgres limiter or otherwise prove concurrent accounting at the limiter level.
- Add repository transaction-helper coverage for `createAndListRecentInTx` success and commit-error paths while touching the Postgres repository tests.
- Add HTTP span attribute coverage; span-name-only tests are not enough.
- Add zero-value and nil-path tests for exported helpers whose contract changes.

## Blockers / Assumptions

No blocker currently.

Assumptions:
- Constructor signature change is acceptable for the internal SQLC sample repository.
- The next session may implement all tasks in one phase because the change surfaces are bounded and share validation commands.
- Docker-backed integration proof is useful but not mandatory if unavailable.

## Handoffs / Reopen Conditions

Next session starts with `workflow-plans/implementation-phase-1.md` and `tasks.md`.

Reopen specification or design instead of coding if:
- `MaxIdleConns` should no longer be strict.
- `NewPingHistoryRepository` cannot change its signature.
- Edge-wide OpenTelemetry wrapping causes an unacceptable trace-shape change.
- Implementation requires editing generated files, OpenAPI, migrations, or app-layer behavior.
