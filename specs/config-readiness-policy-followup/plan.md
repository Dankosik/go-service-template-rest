# Implementation Plan

## Execution Context

This plan consumes `spec.md` and the approved `design/` bundle. The next session should make code/docs changes only; no new workflow, spec, or design artifacts should be needed unless a reopen condition triggers.

## Phase Plan

### Phase

Implementation Phase 1: config-readiness policy repair.

### Objective

Make the runtime behavior match the config readiness-budget contract, clarify Redis/Mongo reserved extension keys, and land the local readability cleanups.

### Depends On

- Approved `spec.md`.
- Approved `design/overview.md`, `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md`.

### Task Ledger Link / IDs

Use `tasks.md`: `T001` through `T006`.

### Acceptance Criteria

- Postgres runtime readiness is capped by `cfg.Postgres.HealthcheckTimeout` while still respecting any shorter parent readiness deadline.
- Redis/Mongo docs clearly distinguish active guard/probe controls from reserved future adapter/cache/store settings.
- Config-file policy selection reads as policy selection, not as a local-environment predicate with hidden fail-closed semantics.
- `validateRedis` uses the local normalized mode value for its store-mode branch.
- Optional Mongo branch cleanup, if implemented, does not change `TestMongoProbeAddress` behavior.

### Change Surface

- `cmd/service/internal/bootstrap/startup_dependencies.go`
- `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go`
- `internal/config/load_koanf.go`
- `internal/config/validate.go`
- `internal/config/config_test.go` only if focused coverage needs adjustment
- `docs/configuration-source-policy.md`
- `docs/project-structure-and-module-organization.md`

### Planned Verification

Focused:
- `go test ./cmd/service/internal/bootstrap -run 'Test.*Postgres.*Readiness|TestRunDependencyProbe|TestInitStartupDependenciesAllDisabled' -count=1`
- `go test ./internal/config -run 'Test(LocalAllowsSymlinkConfig|ConfigFileWithoutEnvironmentHintFailsClosed|NonLocalRejectsSymlinkConfig|RedisStoreGuard|RedisModePolicyHelpers|MongoProbeAddress)' -count=1`

Cross-package:
- `go test ./internal/config ./cmd/service/internal/bootstrap ./internal/infra/postgres -count=1`

Optional broad check if implementation touches more files than planned:
- `go test ./...`

### Review / Checkpoint

Before validation, check that:
- No Redis/Mongo adapter package was added.
- No config keys were removed without reopening `spec.md`.
- `internal/app/health.Service` remains unchanged.
- `postgres.Pool.Check` remains context-driven unless design was reopened.

### Exit Criteria

- All `tasks.md` items are completed or explicitly blocked.
- Required validation commands pass.
- `spec.md` `Validation` and `Outcome` are updated during validation closeout.

## Cross-Phase Validation Plan

Validation Phase 1 consumes `workflow-plans/validation-phase-1.md`. It should record command evidence and close out the task-local artifacts.

## Review Finding Coverage

| Review finding | Decision/design coverage | Executable coverage |
| --- | --- | --- |
| Runtime readiness budget mismatch: config counts `postgres.healthcheck_timeout`, but runtime Postgres readiness used only the outer readiness context. | `spec.md` decisions 1-3; `design/sequence.md`; `design/component-map.md`; `design/ownership-map.md`. | `T001`, `T002`, `T006`. |
| Redis/Mongo guard-only extension drift: existing config keys can look like hidden cache/store/database adapter policy. | `spec.md` decision 4; `design/ownership-map.md` Redis/Mongo extension ownership; `design/component-map.md` docs row. | `T003`, `T006`. |
| Config-file policy readability: `isLocalEnvironmentHint` hides fail-closed file-policy selection behind an environment predicate. | `spec.md` decision 5; `design/component-map.md` `internal/config/load_koanf.go` row. | `T004`, `T006`. |
| Redis mode validation drift: `validateRedis` computes local `mode` and then calls `cfg.StoreMode()` for the same branch. | `spec.md` decision 6; `design/component-map.md` `internal/config/validate.go` row. | `T005`, `T006`. |
| Mongo probe-address cleanup: redundant bare-IPv6 branch predicate after bracket handling. | `spec.md` decision 7; `design/component-map.md` `internal/config/validate.go` row. | `T005`, `T006`. |
| Idiomatic Go lane reported no blocking findings. | No implementation task required. | Validation remains scoped to changed packages in `T006`. |

## Implementation Readiness

Status: PASS.

Rationale:
- The selected technical approach is local and bounded.
- No API, schema, migration, or generated-code changes are required.
- Redis/Mongo compatibility and no-adapter behavior are explicit.
- Validation commands are concrete.

## Blockers / Assumptions

Blockers: none.

Assumptions:
- Existing Redis/Mongo keys remain compatible reserved extension API.
- A helper-level bootstrap test can prove the Postgres readiness deadline without needing a live Postgres instance.

## Handoffs / Reopen Conditions

Reopen specification if the implementation should remove Redis/Mongo keys or change dependency criticality.

Reopen technical design if implementation requires changing `internal/app/health.Service`, globally changing `postgres.Pool.Check`, or introducing a shared readiness-budget abstraction.
