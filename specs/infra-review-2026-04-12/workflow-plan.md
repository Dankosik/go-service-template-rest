# Workflow Plan: internal/infra Review Fixes

Current phase: validation-phase-1
Phase status: complete
Session boundary reached: yes
Ready for next session: no
Next session starts with: done; no next phase is pending.

## Task Frame

Goal: fix the accepted `internal/infra` review findings correctly.
Scope: pre-implementation decision, design, and planning context for `internal/infra/http`, `internal/infra/postgres`, and `internal/infra/telemetry`, including directly relevant tests and generated-code boundaries.
Non-goals: no generated-code edits, no OpenAPI or migration changes, no auth/network policy redesign for `/metrics`.
Success check: T001-T009 are implemented, validated, and recorded in the existing task artifacts.

## Execution Shape

Execution shape: lightweight local phase-collapse for pre-code artifacts, building on the completed full orchestrated read-only review.
Research mode: local synthesis from completed review fan-out.
Current phase plan: `workflow-plans/validation-phase-1.md`.

Phase-collapse rationale: the findings are bounded to handwritten infra packages/tests, and the requested implementation plus validation used the pre-created implementation and validation phase-control files.

## Artifact Status

- `workflow-plan.md`: complete
- `workflow-plans/review.md`: complete
- `workflow-plans/planning.md`: complete
- `workflow-plans/implementation-phase-1.md`: complete
- `workflow-plans/validation-phase-1.md`: complete
- `spec.md`: complete
- `design/`: approved
- `research/review-findings-traceability.md`: complete
- `plan.md`: complete
- `tasks.md`: complete
- `test-plan.md`: not expected.
- `rollout.md`: not expected.

## Completed Review Lanes

- Workflow adequacy challenge: `challenger-agent`, skill `workflow-plan-adequacy-challenge`, read-only.
- Idiomatic Go review: `quality-agent`, skill `go-idiomatic-review`, read-only.
- Readability and simplification review: `quality-agent`, skill `go-language-simplifier-review`, read-only.
- Design and maintainability review: `architecture-agent`, skill `go-design-review`, read-only.
- Chi transport review: `api-agent`, skill `go-chi-review`, read-only.
- Postgres data-access review: `data-agent`, skill `go-db-cache-review`, read-only.

## Planned Implementation Scope

- Postgres max-idle limiter and nil-safe pool/repository behavior.
- HTTP edge-wide tracing, route attributes, explicit body-limit config, and root-owned `/metrics`.
- Telemetry `Metrics` zero-value safety.
- Low-risk colocated readability cleanup.

## Blockers And Risks

- Existing unrelated worktree changes are present under `specs/template-readiness-review`; do not modify or restore them.
- Other untracked review directories may exist under `specs/`; do not modify them as part of this task.
- `internal/infra/postgres/sqlcgen` is generated code and should be treated as derived output unless a generated-source boundary issue is found.
- `SetupTracing` fallback defaults are explicitly out of scope for the planned implementation unless reopened.

## Adequacy Challenge

Review artifact status: complete; no blocking adequacy gaps in initial pass or second pass after adding chi and Postgres data-access lanes.
Planning bundle status: waived under lightweight local phase-collapse; no new subagent fan-out was requested for the follow-up planning turn.

## Implementation Readiness

Status: PASS
Rationale: accepted review findings had approved spec, design, plan, task ledger, and phase-control files before implementation started. The gate was consumed by the completed implementation and validation work.

## Handoff Rule

No next phase is pending. Future sessions should reopen the appropriate earlier phase only if new evidence changes the accepted spec, design, or validation result.
