# Implementation Phase 3 Workflow Plan

## Scope

Implement Phase 3 Postgres/bootstrap/artifact cleanup and validation tasks from `tasks.md`: T023-T028.

## Status

- Phase status: complete.
- Completion marker: met.
- Session boundary reached: yes.
- Next action: none unless the user requests review, commit, or follow-up work.

## Required Read Order

1. `workflow-plan.md`
2. `workflow-plans/implementation-phase-3.md`
3. `spec.md`
4. `design/overview.md`
5. `design/component-map.md`
6. `design/sequence.md`
7. `design/ownership-map.md`
8. `plan.md`
9. `tasks.md`
10. `research/finding-coverage.md`

## Allowed Writes

- `internal/infra/postgres/ping_history_repository.go`
- `internal/infra/postgres/ping_history_repository_test.go`
- `test/postgres_sqlc_integration_test.go`
- `cmd/service/internal/bootstrap/startup_common.go`
- `cmd/service/internal/bootstrap/startup_dependencies.go`
- `cmd/service/internal/bootstrap/*_test.go`
- `.gitignore`
- `.artifacts/test/junit.xml`
- `.artifacts/test/test2json.json`
- `specs/template-readiness-remediation-2026-04-11/tasks.md`
- `specs/template-readiness-remediation-2026-04-11/research/finding-coverage.md`
- Related nearby files only if a task cannot be completed correctly without them, and only after recording the reason in `tasks.md`.

## Prohibited Writes

- Do not restore, modify, or delete unrelated `specs/template-readiness-*` files already dirty in the worktree.
- Do not add Redis/Mongo adapter packages.
- Do not add generic `common`, `util`, repository framework, migration framework, or service registry packages.
- Do not hand-edit generated files under `internal/api` or `internal/infra/postgres/sqlcgen`.
- Do not wire `ping_history` into `ping`.

## Completion Marker

- T023-T028 are checked with evidence in `tasks.md`.
- Targeted validation for Postgres, bootstrap, artifact cleanup, and final broader checks has fresh results.
- Every row in `research/finding-coverage.md` is mapped to a completed task, explicit no-op, or accepted residual risk.
- Any skipped validation has an explicit reason.
