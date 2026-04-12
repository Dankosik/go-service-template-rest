# Implementation Plan

## Execution Context

This plan consumes the approved review-fix spec and design bundle in this directory. The goal is a small, reviewable cleanup/fix pass with no runtime API, schema, or generated-code changes.

## Phase Plan

### Phase 1: Shared OTel Vocabulary And Config Boundary

Objective: remove hidden OTEL env configuration and make OTel vocabulary ownership explicit.

Depends on: none.

Task ledger: `T001` through `T004`.

Change surface:

- `internal/observability/otelconfig`
- `docs/repo-architecture.md`
- `docs/project-structure-and-module-organization.md`
- `internal/config/defaults.go`
- `internal/config/validate.go`
- `internal/config/*_test.go`
- `internal/infra/telemetry/tracing.go`
- `internal/infra/telemetry/tracing_test.go`

Acceptance criteria:

- `resource.WithFromEnv()` is removed.
- OTel sampler/protocol strings come from `otelconfig` where shared across config and telemetry.
- `SetupTracing` no longer applies fallback resource identity defaults that belong to config.
- Non-finite sampler ratio inputs return errors at telemetry runtime boundary.
- Config validation behavior remains equivalent.

Planned verification:

- `go test ./internal/observability/... ./internal/config ./internal/infra/telemetry`

### Phase 2: Metrics Intent And Bootstrap Call Sites

Objective: remove raw boolean dependency status call sites while preserving Prometheus label output.

Depends on: Phase 1 only if telemetry constants or package shape are used in both phases.

Task ledger: `T005` and `T006`.

Change surface:

- `internal/infra/telemetry/metrics.go`
- `internal/infra/telemetry/metrics_test.go`
- `cmd/service/internal/bootstrap/*.go`
- `cmd/service/internal/bootstrap/*_test.go`

Acceptance criteria:

- Startup dependency ready/blocked status is expressed by intent-named methods.
- Existing metric names and labels remain unchanged.
- Telemetry init failure reasons are constant-owned by telemetry and still normalized.

Planned verification:

- `go test ./internal/infra/telemetry ./cmd/service/internal/bootstrap`

### Phase 3: HTTP Server Guard

Objective: make HTTP `Server` nil/zero-value misuse return inspectable errors.

Depends on: none; may run after Phase 1/2 to keep review smaller.

Task ledger: `T007`.

Change surface:

- `internal/infra/http/server.go`
- `internal/infra/http/server_test.go`

Acceptance criteria:

- `(*Server)(nil).Run`, zero-value `Server.Run`, `Serve`, and `Shutdown` do not panic.
- `Serve(nil)` still returns `ErrNilListener` for initialized servers.

Planned verification:

- `go test ./internal/infra/http -run 'TestServer'`

### Phase 4: Postgres Fixture Trim

Objective: remove unused transaction policy from the replaceable SQLC fixture.

Depends on: none; keep after other phases if minimizing simultaneous test churn.

Task ledger: `T008`.

Change surface:

- `internal/infra/postgres/ping_history_repository.go`
- `internal/infra/postgres/ping_history_repository_test.go`

Acceptance criteria:

- `createAndListRecentInTx` and transaction-only fake scaffolding are gone.
- `PingHistoryRepository` no longer stores a transaction-only `db` field.
- Existing `Create`, `ListRecent`, and SQLC integration behavior remain.

Planned verification:

- `go test ./internal/infra/postgres`
- `go test ./test -run TestPingHistoryRepositorySQLCReadWrite` if integration dependencies are available.

### Phase 5: Route Metadata Cleanup

Objective: collapse manual root route declarations and reason metadata into one local source of truth.

Depends on: none.

Task ledger: `T009`.

Change surface:

- `internal/infra/http/router.go`
- `internal/infra/http/router_test.go`

Acceptance criteria:

- Manual root routes still reject `/api/...` manual additions.
- `/metrics` remains documented and root-owned.
- No route behavior changes.

Planned verification:

- `go test ./internal/infra/http -run 'ManualRootRoute|RootRouter|OpenAPIRuntimeContract'`

## Cross-Phase Validation Plan

Run the focused commands after their phase, then run:

- `go test ./internal/infra/...`
- `go test ./internal/observability/... ./internal/config ./cmd/service/internal/bootstrap`
- `git diff --check`
- docs diff inspection for the new `internal/observability/otelconfig` boundary

See `test-plan.md` for the full validation list.

## Implementation Readiness

Status: `PASS`.

Proof obligations:

- Keep dependency direction exactly as described in `design/dependency-graph.md`.
- Preserve metric labels and HTTP route behavior.
- Do not edit generated SQLC code.
- Do not preserve `resource.WithFromEnv()` unless the spec is reopened.

## Blockers / Assumptions

No known blockers.

Assumptions are recorded in `spec.md`.

## Handoffs / Reopen Conditions

Reopen technical design if `internal/observability/otelconfig` creates an import-cycle problem or if a future requirement demands OTEL standard env support.

Reopen planning if the Postgres transaction fixture has non-test consumers not found in this pass.
