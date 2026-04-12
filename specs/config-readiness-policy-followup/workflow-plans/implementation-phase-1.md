# Implementation Phase 1 Plan

Phase: implementation-phase-1.
Status: complete.

## Goal

Implement the approved config-readiness follow-up in one bounded pass while preserving compatibility and avoiding new adapter behavior.

## Allowed Writes

- `cmd/service/internal/bootstrap/startup_dependencies.go`
- `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go`
- `internal/config/load_koanf.go`
- `internal/config/validate.go`
- `internal/config/config_test.go` if focused coverage needs to move with the readability changes
- `docs/configuration-source-policy.md`
- `docs/project-structure-and-module-organization.md`
- Existing task-local closeout surfaces in `specs/config-readiness-policy-followup/`

Do not create Redis or Mongo adapter packages in this phase.

## Task Ledger

Use `tasks.md` as the executable ledger.

Primary tasks:
- `T001`: bound runtime Postgres readiness by `postgres.healthcheck_timeout`.
- `T002`: add focused Postgres readiness wrapper coverage.
- `T003`: document Redis/Mongo reserved extension keys.
- `T004`: rename and restructure config-file policy selection.
- `T005`: keep Redis mode validation on the local normalized mode and simplify the Mongo branch.
- `T006`: run focused and cross-package validation and closeout updates in Validation Phase 1.

## Stop Rule

Stop and reopen `spec.md` if implementation requires:
- Removing existing Redis/Mongo config keys.
- Changing readiness probe ordering in `internal/app/health.Service`.
- Changing API, schema, migration, OpenAPI generation, or dependency criticality.
- Making `postgres.Pool.Check` globally own healthcheck timeouts instead of the bootstrap readiness wrapper.

## Completion Marker

Implementation phase is complete when:
- All `tasks.md` implementation tasks are checked off or explicitly blocked.
- Focused tests and the cross-package validation command pass.
- `workflow-plans/validation-phase-1.md` is ready to consume the evidence.

Completion evidence:
- `T001` through `T005` are implemented and checked off in `tasks.md`.
- `go test ./cmd/service/internal/bootstrap -run 'Test.*Postgres.*Readiness|TestRunDependencyProbe|TestInitStartupDependenciesAllDisabled' -count=1`: passed.
- `go test ./internal/config -run 'Test(LocalAllowsSymlinkConfig|ConfigFileWithoutEnvironmentHintFailsClosed|NonLocalRejectsSymlinkConfig|RedisStoreGuard|RedisModePolicyHelpers|MongoProbeAddress)' -count=1`: passed.
- `go test ./internal/config ./cmd/service/internal/bootstrap ./internal/infra/postgres -count=1`: passed.
