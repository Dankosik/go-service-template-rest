# Validation Phase 1

## Scope

Validate the implementation of the bootstrap maintainability fixes after code changes land.

## Required Evidence

- `gofmt -l` or repository `fmt-check` equivalent shows no unformatted touched Go files.
- `go test -count=1 ./cmd/service/internal/bootstrap`
- `go test -count=1 ./internal/config ./internal/infra/telemetry`
- Prefer `make test` or `go test ./...` if runtime budget allows.

## Required Assertions

- Non-config startup rejections increment `startup_rejections_total{reason=...}` and no longer rely on `config_validation_failures_total{reason="dependency_init"|"policy_violation"|"startup_error"}`.
- Config load/parse/validation failures still increment the config failure metric and also report a startup rejection reason when startup is rejected.
- Readiness predicate helpers in `internal/config` match the bootstrap probe inclusion rules.
- HTTP runtime refactor keeps cancellation, readiness admission, shutdown delay, and terminal error precedence stable.
- Rename-only helpers preserve network policy behavior.

## Closeout Writes

- Update `tasks.md` checkboxes.
- Update `spec.md` `Validation` and `Outcome` after fresh evidence exists.
- Update `workflow-plan.md` readiness/current-phase status only after validation passes or a reopen target is identified.

## Status

Complete for the task-local bootstrap maintainability scope on 2026-04-12.

Evidence:

- `gofmt -l` on touched Go files: no output.
- `git diff --check` on touched Go files: passed.
- `go test -count=1 ./cmd/service/internal/bootstrap`: 91 passed in 1 package.
- `go test -count=1 ./internal/config`: 119 passed in 1 package.
- `go test -count=1 ./internal/infra/telemetry -run 'TestNormalizeStartupRejectionReason|TestCoreMetricsHandlerExposesExpectedSeries|TestMetricsNilAndZeroValueMethodsAreNoops'`: 13 passed in 1 package.
- `go test -count=1 ./internal/config ./internal/infra/telemetry`: failed in the current workspace because unrelated tracing work fails `TestSetupTracingUsesConfigResourceAttributesOnly`.
- `go test ./...`: failed in the current workspace because unrelated Postgres repository test edits fail to build and the unrelated tracing test above fails.

No task-local reopen target remains. Full package/repo validation should be rerun after the unrelated workspace changes are reconciled.
