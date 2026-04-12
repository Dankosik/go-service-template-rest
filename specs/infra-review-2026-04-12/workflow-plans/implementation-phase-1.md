# Implementation Phase 1 Plan: internal/infra Review Fixes

Phase: implementation-phase-1
Status: complete

## Entry Criteria

- Read `workflow-plan.md`.
- Read this file.
- Read `spec.md`, `design/overview.md`, `design/component-map.md`, `design/sequence.md`, `design/ownership-map.md`, `plan.md`, and `tasks.md`.
- Confirm no generated files are dirty before editing unless a task explicitly requires generated output.

## Execution Order

1. Implement T001-T003 Postgres safety fixes.
2. Implement T004 telemetry metrics zero-value safety.
3. Implement T005-T007 HTTP router/tracing/metrics-route ownership fixes.
4. Implement T008 low-risk colocated cleanup only if it remains local and behavior-preserving.
5. Stop before validation closeout and hand off to `validation-phase-1`.

## Guardrails

- Do not edit `internal/api` generated files.
- Do not edit `internal/infra/postgres/sqlcgen`.
- Do not change OpenAPI, migrations, or app-layer behavior.
- Do not keep both edge-wide and route-local `otelhttp.NewMiddleware` wrappers active for normal matched routes.
- Do not weaken fail-closed CORS, Problem fallback behavior, or `/metrics` root priority.

## Expected Progress Updates

Update existing `tasks.md` checkboxes as tasks complete.
Update this file and `workflow-plan.md` only for real phase status changes, blockers, or handoff state.

## Exit Criteria

- T001-T008 complete or explicitly deferred with rationale.
- Implementation compiles at least through focused package tests.
- No new workflow/process artifacts are created during implementation.

## Completion

- T001-T008 complete.
- Focused package tests passed:
  - `go test -count=1 ./internal/infra/postgres`
  - `go test -count=1 ./internal/infra/telemetry`
  - `go test -count=1 ./internal/infra/http`
- Session boundary reached: yes.
- Validation handoff was completed in `workflow-plans/validation-phase-1.md`.
