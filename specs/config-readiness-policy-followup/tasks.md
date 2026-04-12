# Task Ledger

- [x] T001 [Implementation Phase 1] Add a bootstrap-owned Postgres runtime readiness wrapper in `cmd/service/internal/bootstrap/startup_dependencies.go` so the registered Postgres readiness probe derives a child context with `cfg.Postgres.HealthcheckTimeout` before calling `pg.Check`. Depends on: none. Proof: focused bootstrap readiness test plus `go test ./cmd/service/internal/bootstrap -run 'Test.*Postgres.*Readiness|TestRunDependencyProbe|TestInitStartupDependenciesAllDisabled' -count=1`.

- [x] T002 [Implementation Phase 1] Add focused coverage in `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go` proving the Postgres readiness wrapper caps the context deadline and does not extend a shorter parent deadline. Depends on: T001. Proof: same focused bootstrap test command as T001.

- [x] T003 [Implementation Phase 1] Clarify Redis/Mongo extension-key policy in `docs/configuration-source-policy.md` and `docs/project-structure-and-module-organization.md`, distinguishing active guard/probe controls from reserved future adapter/cache/store settings. Depends on: none. Proof: docs review plus cross-package tests in T006.

- [x] T004 [Implementation Phase 1] Replace `isLocalEnvironmentHint(hasConfigFiles bool)` in `internal/config/load_koanf.go` with a policy-named helper that returns `configFilePolicy` and preserves the existing fail-closed behavior for explicit config files without a local env hint. Depends on: none. Proof: `go test ./internal/config -run 'Test(LocalAllowsSymlinkConfig|ConfigFileWithoutEnvironmentHintFailsClosed|NonLocalRejectsSymlinkConfig)' -count=1`.

- [x] T005 [Implementation Phase 1] In `internal/config/validate.go`, use the local normalized `mode` for the Redis store-mode guard and, if already editing the file, simplify the redundant bare-IPv6 predicate after Mongo bracket handling. Depends on: none. Proof: `go test ./internal/config -run 'Test(RedisStoreGuard|RedisModePolicyHelpers|MongoProbeAddress)' -count=1`.

- [x] T006 [Validation Phase 1] Run required validation and update task-local closeout surfaces. Depends on: T001, T002, T003, T004, T005. Proof: `go test ./internal/config ./cmd/service/internal/bootstrap ./internal/infra/postgres -count=1`, then update `spec.md` `Validation` and `Outcome`, `workflow-plan.md`, and `workflow-plans/validation-phase-1.md`.
