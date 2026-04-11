# Planning Phase Workflow Plan

## Phase Control

- Phase: planning.
- Phase status: complete.
- Research mode: local synthesis from completed subagent-backed review.
- Completion marker: review findings reconciled into `spec.md`, required design context written, implementation plan and task ledger written, validation strategy written, and implementation-readiness status recorded.
- Stop rule: do not implement code in this session.
- Next action: implementation-phase-1 in a later session, starting from `tasks.md`.

## Local Lanes

- `review-synthesis`: compare architecture, quality, HTTP/API, data, reliability, QA, and security review findings against current repository evidence.
- `spec`: record implementation decisions and non-goals in `spec.md`.
- `design`: map decisions to affected packages and runtime sequence in `design/`.
- `planning`: write `plan.md`, `tasks.md`, and `test-plan.md`.

## Explicit Waivers

- No new workflow adequacy challenger was run in this turn because new subagent work was not requested. The prior review pass already used read-only subagents and an adequacy challenge for the review workflow.
- The specification, technical design, and planning phases are collapsed into this local artifact-writing pass because the user explicitly asked for a complete implementation handoff before a later implementation session and no code is being changed now.

## Out Of Scope

- Code edits.
- Generated OpenAPI/sqlc/mock/stringer output changes.
- Migration rewrites.
- Creating new runtime auth, tenant, CSRF, or dependency-manager abstractions.

## Validation

- Artifact consistency check: this phase writes all files needed for the future implementation entry point.
- Command validation: not run; no runtime behavior changed.

## Handoff

- Implementation starts with Phase 1 in `plan.md` and tasks `T001` through `T006`.
- If implementation discovers a decision that changes security model, public ingress policy, or readiness aggregation strategy, stop and reopen planning rather than improvising in code.
