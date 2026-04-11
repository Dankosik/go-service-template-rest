# Template Readiness Follow-Up Workflow Plan

## Master Control

- Task: prepare an implementation handoff for the remaining template-readiness gaps found by `specs/template-readiness-review-2026-04-11`.
- Execution shape: lightweight local artifact synthesis with an upfront phase-collapse waiver.
- Phase-collapse waiver: the previous review pass was already subagent-backed and covered architecture, quality, HTTP/API, data, reliability, QA, and security. The current user request asks to think through the fixes and write the implementation context into files, with implementation explicitly deferred to a later session. No new subagent fan-out is authorized or needed for this handoff.
- Current phase: validation.
- Phase status: complete.
- Session boundary reached: yes.
- Ready for next session: no.
- Next session starts with: N/A.

## Scope

- Planning handoff scope: convert review findings into implementation-ready decisions, design context, ordered plan, task ledger, and validation strategy.
- Completed implementation scope: `tasks.md` T001 through T027.
- Completed validation scope: `tasks.md` T028 through T032.
- Out of scope for completed task: migrations, SQLC surfaces, migration-backed runtime behavior, real auth, browser session runtime, CSRF middleware, and metrics listener/auth redesign.
- Allowed phase-1 writes used: `api/openapi/service.yaml`, generated `internal/api/openapi.gen.go`, `internal/infra/http`, `docs/project-structure-and-module-organization.md`, and existing workflow/task progress artifacts.
- Allowed phase-2 writes used: `cmd/service/internal/bootstrap`, `internal/infra/http`, `internal/infra/telemetry`, `internal/config`, `docs/configuration-source-policy.md`, `docs/project-structure-and-module-organization.md`, `docs/repo-architecture.md`, and existing workflow/task progress artifacts.
- Allowed phase-3 writes used: `internal/config`, `cmd/service/internal/bootstrap`, `docs/configuration-source-policy.md`, `docs/project-structure-and-module-organization.md`, `docs/repo-architecture.md`, and existing workflow/task progress artifacts.
- Allowed phase-4 writes used: `internal/domain/doc.go`, `README.md`, `docs/project-structure-and-module-organization.md`, `docs/build-test-and-development-commands.md`, `Makefile`, and existing workflow/task progress artifacts.
- Validation closeout writes used: existing `spec.md`, `tasks.md`, and this `workflow-plan.md`. No dedicated validation phase file was used or created.

## Source Inputs

- `specs/template-readiness-review-2026-04-11/workflow-plan.md`
- `specs/template-readiness-review-2026-04-11/workflow-plans/review-phase-1.md`
- Final review synthesis delivered in chat for the repository-template readiness review.
- Stable repository baseline: `docs/repo-architecture.md`.
- Structure guidance: `docs/project-structure-and-module-organization.md`.
- Current code evidence from `api/openapi/`, `cmd/service/internal/bootstrap/`, `internal/config/`, `internal/infra/http/`, `internal/infra/postgres/`, `internal/infra/telemetry/`, `test/`, `README.md`, and `Makefile`.

## Artifact Status

- `workflow-plan.md`: complete.
- `workflow-plans/planning.md`: approved.
- `spec.md`: complete; `Validation` and `Outcome` refreshed during closeout.
- `design/overview.md`: approved.
- `design/component-map.md`: approved.
- `design/sequence.md`: approved.
- `design/ownership-map.md`: approved.
- `design/contracts/http-security-and-generated-errors.md`: approved.
- `plan.md`: approved.
- `tasks.md`: complete.
- `test-plan.md`: approved.
- `rollout.md`: not expected; this is template hardening in the repository, not a live production rollout.

## Implementation Readiness

- Status: PASS.
- Rationale: remaining review findings are translated into bounded implementation phases with explicit ownership, non-goals, and proof expectations.
- Required implementation entry files: read `spec.md`, `design/overview.md`, `plan.md`, `tasks.md`, and `test-plan.md` before editing code.
- Accepted residual risk: no new adequacy subagent was run for this follow-up because the user did not request new subagents in this turn; this bundle consumes the already completed subagent-backed review and records a phase-collapse waiver.

## Implementation Progress

- Phase 1: complete.
- Phase 2: complete.
- Phase 3: complete.
- Phase 4: complete.
- Final validation: complete.
- Progress source: `tasks.md` T001 through T032.
- Phase-control note: no per-implementation phase files were pre-created during planning; this lightweight handoff pointed implementation at `tasks.md`, so phase progress is recorded in this master plan and task ledger.
- Verification: `go test ./internal/infra/http -count=1` passed. `make openapi-check` passed with a temporary git index containing the current generated `internal/api` output, leaving the real staging area unchanged.
- Phase 2 verification: `go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/telemetry -count=1` passed.
- Phase 3 verification: `go test ./internal/config ./internal/app/health ./cmd/service/internal/bootstrap -count=1` passed. `make check` passed.
- Phase 4 verification: `make help` was checked and updated for migration validation and generated-helper drift discoverability. `go test ./... -count=1` passed. `make check` passed.
- Final validation verification: `go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/telemetry ./internal/app/... -count=1` passed. `make check` passed. `make openapi-check` passed using a temporary git index for the current generated `internal/api` output, leaving the real staging area unchanged. SQLC/migration/integration checks were not run because no `env/migrations`, `internal/infra/postgres/queries`, or `internal/infra/postgres/sqlcgen` files changed and no migration-backed runtime behavior changed.

## Validation Expectations

- Final validation is complete in `tasks.md` T028 through T032 and `spec.md` `Validation`.
- Required closeout commands passed: targeted package tests, `make check`, and `make openapi-check`.
- Conditional SQLC/migration/integration checks are not applicable because no SQLC, migration, or migration-backed runtime behavior changed.

## Blockers

- None.

## Resume Order

Task is complete. No next session is required for this follow-up.
