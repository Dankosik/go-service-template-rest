# planning workflow

## Phase scope

- Phase: `planning`.
- Status: complete.
- Objective: convert the accepted review findings into implementation-ready pre-code artifacts without editing production code.
- Completion marker: `spec.md`, required `design/` files, `plan.md`, `tasks.md`, and `workflow-plans/implementation-phase-1.md` exist and route the next implementation session.
- Stop rule: stop after writing the planning handoff; do not implement in this session.

## Phase-collapse rationale

This is a lightweight-local follow-up to an already completed agent-backed review. Specification, technical design, and planning are collapsed because:

- all accepted findings are concrete code review findings;
- repository docs already answer the `MongoProbeAddress` ownership question;
- no API, data, migration, deployment, or product decision is required;
- implementation can be driven from a short spec, compact design bundle, and task ledger.

## Work completed

- Wrote `spec.md` with scope, decisions, assumptions, and validation obligations.
- Wrote `design/overview.md`, `design/component-map.md`, `design/sequence.md`, and `design/ownership-map.md`.
- Wrote `plan.md` and `tasks.md`.
- Wrote `workflow-plans/implementation-phase-1.md` for the next session.
- Wrote `research/review-coverage.md` to preserve the accepted, filtered, deferred, and residual review points.

## Implementation readiness

- Status: `PASS`.
- Proof obligations:
  - `go test ./internal/config ./cmd/service/internal/bootstrap`;
  - `rg -n "ErrDependencyInit" internal/config cmd/service/internal/bootstrap`;
  - focused regression tests listed in `tasks.md`.

## Blockers and reopen conditions

- Blockers: none.
- Reopen specification or design if service-name ports, local symlink policy changes, `MongoProbeAddress` ownership changes, or non-bootstrap use of `config.ErrDependencyInit` becomes required.

## Handoff

- Session boundary reached: yes.
- Next session starts with `workflow-plans/implementation-phase-1.md` and `tasks.md`.
