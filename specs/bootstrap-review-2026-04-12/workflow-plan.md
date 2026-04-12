# Bootstrap Review Fixes Workflow Plan

Task: fix the accepted `cmd/service/internal/bootstrap` review findings correctly.

Execution shape: lightweight local phase-collapse after completed review fan-out.
Current phase: done.
Phase status: implementation phase 1 complete.
Session boundary reached: yes.
Ready for next session: no; implementation and focused validation are complete.
Next session starts with: no planned follow-up; reopen only if review finds a regression or a recorded reopen condition is hit.

## Scope

In scope:
- Preserve decisions and technical context for all accepted review findings.
- Create repo-native `spec.md`, `design/`, `plan.md`, and `tasks.md` artifacts.
- Reconcile the Postgres DSN diagnostic finding against secret-source policy before tasking implementation.
- Implement and validate `tasks.md` T001-T009.

Out of scope:
- API, schema, data migration, rollout, or generated-code changes.

## Artifact Status

- `spec.md`: implemented; validation passed.
- `design/`: approved for planning handoff; core bundle present.
- `plan.md`: approved.
- `tasks.md`: complete.
- `test-plan.md`: not expected.
- `rollout.md`: not expected.
- `workflow-plans/review-phase-1.md`: complete.
- `workflow-plans/specification.md`: superseded by planning phase-collapse.
- `workflow-plans/planning.md`: complete.

## Routing

Prior review research mode: fan-out.

Completed review lanes:
- `go-idiomatic-review`: Go semantics, stdlib contracts, error/context/lifetime/resource idioms.
- `go-language-simplifier-review`: readability, helper economics, control flow, naming, maintainability drift.
- `go-design-review`: bootstrap boundary fit, composition-root ownership, dependency direction, source-of-truth seams.

Workflow plan adequacy challenge: completed for the review phase. Planning adequacy challenge waived under lightweight-local phase-collapse because the implementation scope is bounded to accepted review findings, no subagents are used in this planning session, and no implementation starts here.

Spec clarification challenge: waived under lightweight-local phase-collapse. The planning-critical decision is captured in `spec.md` and must reopen only if a future security/reliability review rejects the selected telemetry fail-open behavior.

## Blockers And Assumptions

Blockers: none.
Assumptions:
- The review targets current working tree contents, not a git diff only.
- Subagents remain read-only and advisory; final findings belong to the orchestrator.
- Findings must be prioritized by merge/maintenance risk, not style preference.
- Telemetry exporter egress should respect bootstrap network policy without making optional tracing a startup-critical dependency.
- Postgres DSN parse errors must remain secret-safe even when diagnostics improve.

## Validation And Closeout

Fresh evidence:
- Inspected target package files and directly relevant tests.
- Fan-in subagent outputs and reconciled conflicts.
- Verified file and line references with local commands before final response.
- Ran `go test ./cmd/service/internal/bootstrap`: passed.
- Ran `go test -race ./cmd/service/internal/bootstrap`: passed.
- Ran `go vet ./cmd/service/internal/bootstrap`: no issues found.
- Re-read `docs/repo-architecture.md`, `docs/configuration-source-policy.md`, bootstrap network-policy code, telemetry exporter parsing, and OTel exporter v1.19.0 defaults before writing pre-implementation context.
- Ran `go test ./internal/infra/telemetry`: passed.
- Ran `go test ./cmd/service/internal/bootstrap`: passed.
- Ran `go test ./internal/infra/telemetry ./cmd/service/internal/bootstrap`: passed.
- Ran `go test -race ./cmd/service/internal/bootstrap`: passed.
- Ran `go vet ./internal/infra/telemetry ./cmd/service/internal/bootstrap`: no issues found.

Completion marker:
- Adequacy challenge reconciled for review.
- Review lanes returned.
- Final review findings delivered with verified line references.
- Pre-implementation `spec.md`, `design/`, `plan.md`, and `tasks.md` written and ready for a separate implementation session.
- Implementation phase 1 tasks T001-T009 completed.
- Focused validation passed.
