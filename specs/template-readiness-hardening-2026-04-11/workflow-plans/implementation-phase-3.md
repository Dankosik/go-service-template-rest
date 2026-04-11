# Implementation Phase 3 Workflow Plan

## Phase Control

- Phase: implementation-phase-3.
- Phase status: complete.
- Entry condition: Phase 2 complete or explicitly skipped by the user.
- Scope: HTTP security, metrics, generated route ownership, and request error details.
- Tasks: T014 through T018.
- Stop rule: do not start Phase 4 until OpenAPI/runtime route behavior is validated.

## Expected Work

- Remove or reconcile unused OpenAPI `bearerAuth`; prefer removal without fake auth.
- Add endpoint security decision guidance.
- Make `/metrics` route ownership explicit and guarded.
- Sanitize strict-handler request error details.
- Preserve fail-closed CORS behavior.

## Proof

- `go test ./internal/infra/http -count=1`: passed.
- `make check`: passed.
- `make openapi-check`: passed with a temporary git index containing the regenerated OpenAPI source/generated pair. The first plain run failed at `openapi-drift-check` because `internal/api/openapi.gen.go` is intentionally changed in the uncommitted working tree; the real git index was left unchanged.

## Completion

- Completion marker: T014 through T018 are implemented and checked in `tasks.md`.
- Stop rule status: satisfied for route behavior and OpenAPI contract checks; Phase 4 may start in the next implementation session.
- Session boundary reached: yes.
- Ready for next session: yes.
- Next session starts with: implementation-phase-4.

## Reopen Conditions

- Reopen specification if the user decides to implement real auth or move metrics to a separate listener in this task.
