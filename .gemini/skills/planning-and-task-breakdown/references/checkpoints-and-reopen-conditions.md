# Checkpoints And Reopen Conditions

## Behavior Change Thesis
When loaded for handoff or blocker wording, this file makes the model name executable checkpoints and exact reopen targets instead of telling the implementation session to "figure it out" or create missing workflow artifacts after coding starts.

## When To Load
Load this when writing review checkpoints, validation checkpoints, implementation-readiness handoff, blockers, stop rules, or reopen conditions for specification, technical design, planning, or validation.

## Decision Rubric
- A checkpoint is useful only if it changes go/no-go state, review scope, validation scope, or the next-session handoff.
- Reopen conditions should name the earlier phase and the trigger that proves planning is no longer safe.
- Required named review or validation phase-control files must be planned before implementation begins; if they are missing later, reopen planning. Do not create such files for already-small tasks when `tasks.md` is sufficient.
- Readiness `FAIL` is honest when it prevents unsafe implementation; do not soften it into optimistic handoff text.
- Conditional reopen rules are not blockers when they clearly describe "if discovered later" stop conditions.

## Imitate
```markdown
## Handoffs / Reopen Conditions

Implementation may start only when readiness is `PASS` or eligible `CONCERNS`.

Reopen `technical design` if:
- an implementation task needs a file/package ownership decision not present in `design/ownership-map.md`
- a task requires ordering that contradicts `design/sequence.md`
- a conditional artifact such as `test-plan.md` or `rollout.md` becomes necessary but was not created during planning

Reopen `specification` if:
- acceptance criteria require a product or behavior decision not already recorded in `spec.md`
```

Copy the named reopen targets and precise triggers. Keep the triggers tied to this task's artifact chain.

## Reject
```markdown
## Risks

If something is missing, figure it out during coding. Add extra docs as needed.
```

This fails because it invites post-code artifact invention and hides the phase that owns the missing decision.

## Agent Traps
- Do not require a checkpoint after every task; place checkpoints where risk or phase boundaries change.
- Do not make `T040 [Checkpoint] Start implementation`; planning checkpoints must not start the next phase.
- Do not create new `workflow-plans/validation-phase-N.md` after code starts; if it was required but missing, reopen planning.
- Do not turn every conditional reopen into a current blocker; distinguish "blocked now" from "stop if discovered later."
