# Implementation Phase 2 Workflow Plan

## Scope

Implement Phase 2 config and HTTP guardrail remediation tasks from `tasks.md`: T016-T022.

Status: complete.

## Required Read Order

1. `workflow-plan.md`
2. `workflow-plans/implementation-phase-2.md`
3. `spec.md`
4. `design/overview.md`
5. `design/component-map.md`
6. `design/sequence.md`
7. `design/ownership-map.md`
8. `plan.md`
9. `tasks.md`
10. `research/finding-coverage.md`

## Allowed Writes

- `internal/config/config_test.go`
- `internal/config/load_koanf.go`
- `Makefile`
- `internal/infra/http/router.go`
- `internal/infra/http/router_test.go`
- `internal/infra/http/server_test.go`
- `internal/infra/http/doc.go`
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

- T016-T022 are checked with evidence in `tasks.md`.
- Targeted validation for `internal/config`, `internal/infra/http`, and `make openapi-runtime-contract-check` has fresh results.
- `make openapi-check` has fresh passing evidence after the OpenAPI drift check was narrowed to generated output.
- Skipped validation: none for Phase 2 targeted scope.
- Session boundary reached: yes.
- Ready for next session: yes.
- Next session starts with: `workflow-plans/implementation-phase-3.md`, then T023 in `tasks.md`.
- Stop after Phase 2. Do not start Phase 3 in the same session unless the user explicitly asks to continue and the phase-control state is updated first.
