# Implementation Phase 1 Workflow Plan

## Scope

Implement Phase 1 docs/onboarding remediation tasks from `tasks.md`: T001-T015.

## Phase Status

- Status: complete.
- Completed tasks: T001-T015.
- Validation evidence: recorded in `tasks.md` under `Phase 1 validation`.
- Code/test/Makefile validation skipped because this phase changed docs and task metadata only.

## Required Read Order

1. `workflow-plan.md`
2. `workflow-plans/implementation-phase-1.md`
3. `spec.md`
4. `design/overview.md`
5. `design/component-map.md`
6. `design/sequence.md`
7. `design/ownership-map.md`
8. `plan.md`
9. `tasks.md`
10. `research/finding-coverage.md`

## Allowed Writes

- `docs/project-structure-and-module-organization.md`
- `docs/build-test-and-development-commands.md`
- `docs/configuration-source-policy.md`
- `README.md`
- `internal/api/README.md`
- `specs/template-readiness-remediation-2026-04-11/tasks.md`
- `specs/template-readiness-remediation-2026-04-11/research/finding-coverage.md`
- Related nearby files only if a task cannot be completed correctly without them, and only after recording the reason in `tasks.md`.

## Prohibited Writes

- Do not restore, modify, or delete unrelated `specs/template-readiness-*` files already dirty in the worktree.
- Do not add Redis/Mongo adapter packages.
- Do not add generic `common`, `util`, repository framework, migration framework, or service registry packages.
- Do not hand-edit generated files under `internal/api` or `internal/infra/postgres/sqlcgen`.
- Do not rename the `internal/infra/http` package unless the implementation session deliberately reopens design; prefer documenting the `httpx` convention.

## Completion Marker

- T001-T015 are checked with evidence in `tasks.md`.
- Phase 1 targeted validation commands have fresh results recorded in `tasks.md`.
- Any skipped validation has an explicit reason.
- No implementation drift outside the approved files remains unexplained.
- Stop after Phase 1. Do not start Phase 2 in the same session unless the user explicitly asks to continue and the phase-control state is updated first.

## Session Boundary

- Session boundary reached: yes.
- Ready for next session: yes.
- Next session starts with: `workflow-plans/implementation-phase-2.md`, then `tasks.md`.
