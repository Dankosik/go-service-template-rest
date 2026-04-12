# Bootstrap Maintainability Fixes Plan

## Execution Context

This is a bounded maintainability and observability-contract cleanup. It should land as one implementation phase because the metric contract, tests, and bootstrap call sites are tightly coupled.

No API, database, migration, rollout, or generated-code work is expected.

## Phase Plan

### Phase 1: Bootstrap Maintainability Fixes

- Objective: implement the approved metric split, source-of-truth helpers, named lifecycle arguments, and rename-only clarity fixes.
- Depends on: approved `spec.md` and `design/` in this task folder.
- Task ledger: `tasks.md` T001-T009.
- Acceptance criteria:
  - `startup_rejections_total{reason}` exists with bounded reason normalization.
  - Config failures still increment `config_validation_failures_total`.
  - Non-config startup failures no longer increment `config_validation_failures_total`.
  - Startup outcome and dependency status metrics remain intact.
  - Postgres, Redis, and Mongo readiness participation share config-level predicate helpers.
  - `serveHTTPRuntime` call sites use named fields.
  - Dependency rejection logging/spans use `startupDependencyProbeLabels` as the label source of truth.
  - Explicit declaration and egress exception helper renames do not change behavior.
- Change surface:
  - `internal/infra/telemetry/metrics.go`
  - `internal/infra/telemetry/metrics_test.go`
  - `internal/config/*`
  - `cmd/service/internal/bootstrap/*`
- Planned verification:
  - `gofmt` or repository fmt check on touched Go files.
  - `go test -count=1 ./cmd/service/internal/bootstrap`
  - `go test -count=1 ./internal/config ./internal/infra/telemetry`
  - Prefer `make test` or `go test ./...` if runtime budget allows.
- Review checkpoint:
  - Confirm no new behavior or policy decision was introduced outside the metric contract correction.
- Exit criteria:
  - All tasks in `tasks.md` complete.
  - Required tests pass or a reopen target is recorded.

## Cross-Phase Validation Plan

Focused validation must prove:

- New telemetry series and reason labels are correct and low-cardinality.
- Existing config-specific metric remains available for config failures.
- Bootstrap test expectations move to the correct metric without losing rejected outcome assertions.
- Readiness budget validation and runtime readiness probe inclusion use the same predicates.
- Existing shutdown/startup tests continue to pass after the args struct refactor.
- Network policy tests continue to pass after rename-only helpers.

## Implementation Readiness

- Status: `CONCERNS`.
- Accepted risk: metric contract changes by adding `startup_rejections_total` and moving non-config failure reasons away from `config_validation_failures_total`.
- Proof obligations:
  - Update telemetry and bootstrap tests around the metric split.
  - Preserve startup outcome metric behavior.
  - Do not remove `config_validation_failures_total` in this task.
- Handoff: implementation may start in the next session.

## Blockers / Assumptions

- Assumption: no external dashboard compatibility contract blocks the new metric.
- Assumption: implementation can keep the change within the listed package surfaces.
- No blocker is currently known.

## Handoffs / Reopen Conditions

Reopen specification or design before coding further if:

- callers require non-config failure reasons to remain on `config_validation_failures_total`;
- a dashboard/alert/runbook compatibility requirement appears;
- readiness predicate semantics need to change rather than only centralize;
- network policy parsing/enforcement behavior must change rather than only rename.
