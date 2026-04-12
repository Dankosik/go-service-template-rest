# Implementation Phase 1 Plan

## Phase Scope

- Phase: implementation-phase-1.
- Status: complete.
- Consume: `spec.md`, `design/`, `plan.md`, and `tasks.md`.
- Implement only the selected guidance and guardrail improvements.

## Allowed Writes

- `README.md`
- `docs/project-structure-and-module-organization.md`
- `docs/configuration-source-policy.md`
- `internal/api/README.md`
- `test/README.md`
- `scripts/ci/required-guardrails-check.sh`
- `internal/infra/http/router.go`
- `internal/infra/http/router_test.go`
- `cmd/service/internal/bootstrap/startup_common.go`
- focused bootstrap tests under `cmd/service/internal/bootstrap`
- existing workflow-control progress updates for this task

## Not Allowed Without Reopening Specification

- Fake business domain or runnable exemplar feature.
- Placeholder auth or real auth policy.
- `ping_history` migration/query/generated SQLC rename.
- Redis/Mongo runtime adapters.
- Generated OpenAPI or SQLC hand edits.

## Task Ledger

Use `tasks.md` T001-T012.

## Planned Verification

- `make guardrails-check`
- `go test ./cmd/service/internal/bootstrap ./internal/infra/http`
- `go test ./...`
- `make openapi-check` only if OpenAPI source/generated surfaces change.
- `make sqlc-check` only if migration/query/generated SQLC surfaces change.

## Stop Rule

Completed. `tasks.md` progress and this phase file were updated; final validation evidence is recorded in `workflow-plans/validation-phase-1.md`.

## Completion Notes

- T001-T011 completed inside the approved Phase 1 write set.
- No fake business domain, placeholder auth, `ping_history` rename, Redis/Mongo adapter, OpenAPI source change, migration change, SQL query change, or generated-code hand edit was made.
- Validation ran as T012 after implementation because the user asked to complete all tasks in the ledger.
