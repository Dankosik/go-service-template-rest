# implementation-phase-1 workflow

## Phase scope

- Phase: `implementation-phase-1`.
- Status: complete.
- Objective: implement the accepted `internal/config` fixes and bootstrap dependency-init ownership cleanup from `tasks.md`.
- Completion marker: tasks `T001` through `T008` are complete, verification commands pass, and no reopen condition is triggered.
- Stop rule: if a planned assumption is invalidated, stop and reopen `spec.md` or `design/` instead of inventing a new policy during coding.

## Inputs

- `spec.md`
- `design/overview.md`
- `design/component-map.md`
- `design/sequence.md`
- `design/ownership-map.md`
- `plan.md`
- `tasks.md`

## Execution order

1. Implement parsing and sampler finite-value fixes: `T001`, `T002`.
2. Implement numeric TCP port validation: `T003`.
3. Implement readability/source-of-truth cleanups: `T004`, `T005`, `T006`.
4. Implement bootstrap dependency-init ownership cleanup: `T007`.
5. Run final verification and update task state: `T008`.

## Parallelism

- `T001`/`T002`, `T003`, `T004`/`T005`/`T006`, and `T007` touch mostly separate surfaces, but a single implementation session should keep them sequential unless multiple workers are explicitly requested later.

## Verification

- Required command: `go test ./internal/config ./cmd/service/internal/bootstrap`.
- Required ownership check: `rg -n "ErrDependencyInit" internal/config cmd/service/internal/bootstrap`.
- Expected ownership after implementation: no `ErrDependencyInit` declaration or dependency-init classification in `internal/config`; bootstrap owns the sentinel and call sites.
- Result: implementation completed; final fresh verification is recorded in `tasks.md`/session handoff.

## Reopen conditions

- A non-bootstrap package depends on `config.ErrDependencyInit`.
- Service-name ports must be supported.
- Local symlink behavior must change.
- Moving `MongoProbeAddress` becomes an intentional requirement.

## Session handoff

- Completion marker: satisfied for `T001` through `T008`.
- Session boundary reached: yes.
- Ready for next session: yes, no reopen condition triggered.
- Next session starts with: optional post-implementation review or closeout only if the user wants an additional review pass.
