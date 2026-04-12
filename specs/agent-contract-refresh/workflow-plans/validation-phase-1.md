# Agent Contract Refresh Validation Phase 1

Phase: validation-phase-1
Status: pending

## Scope

Run final proof for the instruction-only agent contract refresh and update existing closeout surfaces.

Task IDs: T900-T903.

## Consumes

- `spec.md`
- `design/`
- `plan.md`
- `tasks.md`
- `workflow-plan.md`
- all `workflow-plans/implementation-phase-*.md`

## Allowed Future Writes

- `spec.md` `Validation` and `Outcome`
- `workflow-plan.md`
- `tasks.md` progress state
- this existing `workflow-plans/validation-phase-1.md`

## Entry Condition

Phase 7 is complete.

## Stop Rule

Do not create new workflow/process artifacts during validation. If proof exposes a missing decision or missing plan, record the reopen target in existing control artifacts and reopen the appropriate earlier phase.

## Completion Marker

T900-T903 are complete and the task is either marked done with fresh evidence or routed to a named reopen phase.
