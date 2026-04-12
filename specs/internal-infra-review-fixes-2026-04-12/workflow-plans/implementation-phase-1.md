# Implementation Phase 1

## Scope

Implement the approved fixes from `tasks.md` without reopening specification or design unless a task exposes a material contradiction.

## Allowed Writes

- Code and tests under the task surfaces named in `tasks.md`.
- Existing control/progress surfaces for this task, especially this file and `tasks.md`.

Do not create new workflow, planning, design, or research artifacts during implementation.

## Start Here

1. Read `spec.md`, `design/overview.md`, `plan.md`, `tasks.md`, and `test-plan.md`.
2. Confirm the working tree state and avoid reverting unrelated user changes.
3. Execute tasks in dependency order.

## Stop / Reopen Rules

- Reopen planning if the new `internal/observability/otelconfig` package causes an unexpected import cycle or the selected dependency direction is not viable.
- Reopen technical design if preserving OTEL standard env behavior is requested instead of removing `resource.WithFromEnv()`.
- Reopen planning if deleting the Postgres transaction fixture reveals production or integration usage outside the current evidence set.

## Completion Marker

Implementation phase 1 is complete when all tasks in `tasks.md` are checked off and the planned verification commands in `test-plan.md` have fresh results or documented blockers.

## Status

Completed.

## Validation Evidence

- `go test ./internal/observability/... ./internal/config ./internal/infra/telemetry`: passed.
- `go test ./internal/infra/telemetry ./cmd/service/internal/bootstrap`: passed.
- `go test ./internal/infra/http -run 'TestServer'`: passed.
- `go test ./internal/infra/postgres`: passed.
- `rg createAndListRecentInTx internal/infra/postgres`: no matches; exit status 1 expected.
- `go test ./internal/infra/http -run 'ManualRootRoute|RootRouter|OpenAPIRuntimeContract'`: passed.
- `go test ./internal/infra/...`: passed.
- `go test ./internal/observability/... ./internal/config ./cmd/service/internal/bootstrap`: passed.
- `git diff --check`: passed.
- `go test -tags=integration ./test -run TestPingHistoryRepositorySQLCReadWrite`: passed.

Manual diff checks: generated SQLC diff was empty; no `common`, `shared`, `util`, or `utils` package was introduced under `internal`; `resource.WithFromEnv` had no matches under `internal`, `cmd`, `env`, or `docs`. The only remaining `OTEL_RESOURCE_ATTRIBUTES` / `OTEL_SERVICE_NAME` runtime references are the telemetry-local suppression shim that prevents the OTel SDK's `WithResource` env merge while preserving config-owned resource attributes.
