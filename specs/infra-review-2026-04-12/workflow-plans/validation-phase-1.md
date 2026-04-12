# Validation Phase 1 Plan: internal/infra Review Fixes

Phase: validation-phase-1
Status: complete

## Entry Criteria

- Implementation phase 1 has completed T001-T008 or recorded explicit deferrals.
- No required generated artifact, migration, or OpenAPI change is pending.

## Required Validation

- `go test -count=1 ./internal/infra/...`
- `go vet ./internal/infra/...`
- `git status --short`

## Targeted Evidence To Inspect

- Postgres tests prove max-idle limiter accounting and nil-safe pool/repository behavior.
- HTTP tests prove edge-wide tracing, bounded route span names, and `http.route` attributes.
- HTTP tests prove root `/metrics` still wins over the generated route.
- Telemetry tests prove zero-value `Metrics` methods and handler do not panic.

## Optional Validation

Run when Docker is available or constructor call-site changes make it useful:
- `go test -tags=integration ./test/... -run 'TestPingHistoryRepositorySQLCReadWrite' -count=1`

Run only if generated inputs or generated outputs changed unexpectedly:
- `make sqlc-check`
- `make openapi-check`

## Exit Criteria

- Required validation passes, or failures are classified with a reopen target.
- `spec.md` `Outcome` and `Validation` are updated with actual evidence.
- `workflow-plan.md`, this file, and `tasks.md` reflect final status.

## Completion

- Required validation passed:
  - `go test -count=1 ./internal/infra/...`
  - `go vet ./internal/infra/...`
  - `git status --short`
- Optional Docker-backed integration validation passed:
  - `go test -tags=integration ./test/... -run 'TestPingHistoryRepositorySQLCReadWrite' -count=1`
- Additional limiter race sanity check passed:
  - `go test -race ./internal/infra/postgres -run TestMaxIdleConnLimiterConcurrentReleases -count=1`
- Generated surfaces inspected with no dirty files:
  - `internal/api`
  - `internal/infra/postgres/sqlcgen`
- T009 complete.
- Session boundary reached: yes.
- Ready for next session: no; task is done.
