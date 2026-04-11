# Planning Workflow Plan

## Phase Control

- Phase: planning.
- Phase status: complete.
- Research mode: local synthesis from the completed orchestrated review in `specs/template-readiness-review-2026-04-11`.
- Completion marker: decisions, design, ordered plan, task ledger, and validation expectations are written without code changes.
- Stop rule: do not begin implementation in this session.
- Next action: a later implementation session starts at `tasks.md` Phase 1.

## Local Inputs

- Prior review workflow: `specs/template-readiness-review-2026-04-11/workflow-plan.md`
- Prior review phase plan: `specs/template-readiness-review-2026-04-11/workflow-plans/review-phase-1.md`
- Stable architecture baseline: `docs/repo-architecture.md`
- Workflow model: `AGENTS.md`, `docs/spec-first-workflow.md`
- Direct code/docs evidence cited in the prior review report.

## Planning Decisions

- Use a phased implementation: config/clone correctness first, then ownership seams, then HTTP/security/metrics, then persistence and docs/tests.
- Prefer real source-of-truth fixes over comments that only explain drift.
- Do not add fake business auth or fake domain behavior. Where the template lacks a real identity model, remove misleading contract hints and document the required security decision path.
- Do not make `ping` a persistence-backed business feature. Treat `ping_history` as a template sample/tooling issue, not as production behavior.
- Keep generated-code sources authoritative: OpenAPI edits start in `api/openapi/service.yaml`; sqlc edits start in migrations and query files; generated folders are regenerated, not hand-edited.

## Implementation Readiness Gate

- Status: PASS.
- Proof obligations:
  - `make check`
  - `make openapi-check` if OpenAPI or HTTP generated-boundary docs/tests change
  - `make sqlc-check`, or `make docker-sqlc-check` if native sqlc remains blocked
  - targeted `go test` packages listed in `test-plan.md`
  - `make test-integration` only if the implementation changes integration helpers, migrations, or runtime Postgres behavior
- Later implementation phase-control files were pre-created for phases 1 through 4.

## Session Handoff

- Session boundary reached: yes.
- Ready for next session: yes.
- Next session starts with: implementation-phase-1 from `tasks.md`.
