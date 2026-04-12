# Validation Phase 1 Plan

Phase: validation-phase-1.
Status: complete.

## Goal

Prove the implementation satisfies the approved readiness and config-policy decisions without reopening design.

## Required Evidence

Run after implementation:
- `go test ./cmd/service/internal/bootstrap -run 'Test.*Postgres.*Readiness|TestRunDependencyProbe|TestInitStartupDependenciesAllDisabled' -count=1`
- `go test ./internal/config -run 'Test(LocalAllowsSymlinkConfig|ConfigFileWithoutEnvironmentHintFailsClosed|NonLocalRejectsSymlinkConfig|RedisStoreGuard|RedisModePolicyHelpers|MongoProbeAddress)' -count=1`
- `go test ./internal/config ./cmd/service/internal/bootstrap ./internal/infra/postgres -count=1`

Optional broader proof if the implementation touches more than the planned files:
- `go test ./...`

## Closeout Updates

Completed:
- `tasks.md` checkbox state updated for `T001` through `T006`.
- `spec.md` `Validation` and `Outcome` updated with actual evidence and outcome.
- `workflow-plan.md` phase status updated to done/complete.
- This file's status updated to complete.

Evidence:
- `go test ./cmd/service/internal/bootstrap -run 'Test.*Postgres.*Readiness|TestRunDependencyProbe|TestInitStartupDependenciesAllDisabled' -count=1`: passed.
- `go test ./internal/config -run 'Test(LocalAllowsSymlinkConfig|ConfigFileWithoutEnvironmentHintFailsClosed|NonLocalRejectsSymlinkConfig|RedisStoreGuard|RedisModePolicyHelpers|MongoProbeAddress)' -count=1`: passed.
- `go test ./internal/config ./cmd/service/internal/bootstrap ./internal/infra/postgres -count=1`: passed.

## Reopen Conditions

Reopen technical design if:
- Runtime Postgres readiness cannot be bounded without changing `internal/app/health.Service`.
- The reserved Redis/Mongo key policy conflicts with current docs or user intent.
- Focused tests show existing readiness or config-file policy semantics changed unexpectedly.
