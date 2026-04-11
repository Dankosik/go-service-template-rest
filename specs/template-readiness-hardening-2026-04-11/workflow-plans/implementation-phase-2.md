# Implementation Phase 2 Workflow Plan

## Phase Control

- Phase: implementation-phase-2.
- Phase status: complete.
- Entry condition: Phase 1 complete or explicitly skipped by the user.
- Scope: composition root, readiness, lifecycle, and bootstrap helper ownership.
- Tasks: T005 through T013.
- Stop rule: do not start Phase 3 until router construction, readiness behavior, and bootstrap tests are green.

## Expected Work

- Remove production fallback construction from HTTP router setup.
- Keep concrete wiring in bootstrap.
- Resolve readiness probe interface ownership.
- Add external readiness admission gating without deadlocking startup admission.
- Make readiness/shutdown budgets explicit.
- Improve dependency cleanup/status handling.
- Extract narrow same-package bootstrap helpers.

## Proof

- `go test ./cmd/service/internal/bootstrap ./internal/app/health ./internal/infra/http -count=1`: passed.
- `go test ./internal/config -count=1`: passed.
- `make check`: passed.

## Completion

- Completion marker: router construction, readiness behavior, bootstrap lifecycle ownership, and Phase 2 config/docs updates are implemented and targeted tests are green.
- Session boundary reached: yes.
- Ready for next session: yes.
- Next session starts with: implementation-phase-3.

## Reopen Conditions

- Reopen design if the implementation needs to gate all non-health app traffic, not only external readiness.
