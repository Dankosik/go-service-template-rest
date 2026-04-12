# Bootstrap Review Fixes Plan

## Execution Context

This plan prepares one implementation session that fixes the accepted bootstrap review findings without broadening the task into a bootstrap rewrite. The implementation should preserve current startup/shutdown behavior, existing error wrapping, and fail-closed network policy enforcement.

## Phase Plan

### Phase 1: Bootstrap maintainability fixes

Objective: Apply the four accepted findings in one bounded implementation slice.

Depends On: approved `spec.md` and `design/` in this directory.

Task Ledger Link / IDs: `tasks.md` T001-T008.

Acceptance Criteria:

- Redis and Mongo degraded-but-serving paths both update `startup_dependency_status` through one local policy path.
- Mongo degraded-but-serving path records `startup_dependency_status{dep="mongo",mode="degraded_read_only_or_stale"} 1`.
- `NETWORK_*` is documented as a deliberate operator-policy exception to ordinary `APP__...` config.
- `networkPolicyErrorLabels` is no longer production-defined and test-only.
- `dependencyInitFailure` no longer contains a branch that returns the same value as the default path.
- Existing startup rejection, error wrapping, and ingress/egress enforcement behavior remain intact.

Change Surface:

- `cmd/service/internal/bootstrap/startup_dependencies.go`
- `cmd/service/internal/bootstrap/startup_dependencies_additional_test.go`
- `cmd/service/internal/bootstrap/network_policy_parsing.go`
- `cmd/service/internal/bootstrap/startup_bootstrap.go`
- `cmd/service/internal/bootstrap/startup_common.go` only if needed for structured policy log fields
- `cmd/service/internal/bootstrap/startup_common_additional_test.go` only if policy log fields are wired
- `docs/configuration-source-policy.md`
- `docs/repo-architecture.md` and `docs/project-structure-and-module-organization.md` only for cross-links or missing egress policy note
- `env/.env.example`

Planned Verification:

- `go test ./cmd/service/internal/bootstrap`
- `go test ./internal/config ./cmd/service/internal/bootstrap`
- `go vet ./cmd/service/internal/bootstrap`
- `gofmt -l` on touched Go files

Review / Checkpoint:

- After implementation, review whether the helper actually reduced drift without hiding distinct Redis/Mongo semantics.
- Check that docs do not imply `NETWORK_*` participates in normal `APP__...` precedence.
- Check that env examples are commented and cannot silently change local startup behavior.

Exit Criteria:

- All planned verification passes.
- `tasks.md` checkboxes reflect only completed work.
- `workflow-plans/implementation-phase-1.md` is updated by the implementation session.

## Cross-Phase Validation Plan

The implementation session should run package tests before reporting success. If docs-only changes are made for network policy, no external service is required. If production logging fields are added for network policy classification, add a log assertion around the existing `bootstrapNetworkPolicyStage` parse-failure test or the policy violation helper test.

## Implementation Readiness

Status: `PASS`

Rationale: decisions and design are local, file surfaces are known, no migrations or API contract changes are required, and all remaining assumptions are accepted as implementation proof obligations.

Proof obligations:

- Prove Mongo degraded metric state.
- Prove network policy config parse failures still preserve root cause and `config.ErrDependencyInit`.
- Prove `NETWORK_*` documentation is explicit about source and precedence.

## Blockers / Assumptions

- Assumes `NETWORK_*` remains an operator policy channel outside typed app config.
- Assumes no new telemetry metric type is required.
- Assumes one implementation phase is sufficient.

## Handoffs / Reopen Conditions

Start the next session at `workflow-plans/implementation-phase-1.md`.

Reopen technical design if implementation pressure suggests migrating `NETWORK_*` into `internal/config`, changing telemetry metric contracts, or adding Redis/Mongo runtime adapter semantics.
