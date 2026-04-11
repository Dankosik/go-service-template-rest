# Allowed Writes And Prohibited Actions

## Behavior Change Thesis
When loaded for a planning session with scope pressure, this file makes the model write only planning-phase artifacts instead of editing code, tests, migrations, `spec.md`, `design/`, or surprise phase-control files.

## When To Load
Load immediately before writes when the requested action may cross from planning into implementation, review, validation, specification, or technical design.

## Decision Rubric
- Allowed writes are `plan.md`, `tasks.md`, triggered `test-plan.md`, triggered `rollout.md`, `workflow-plan.md`, `workflow-plans/planning.md`, and later phase-control files already required by the approved phase structure.
- A later `workflow-plans/<phase>.md` file is allowed only when the approved phase structure names that phase or planning explicitly creates it for a named future checkpoint.
- `spec.md` and `design/` are read-only in this session. Missing decisions route upstream.
- Code, tests, migrations, generated output, runtime config, review execution, validation execution, rollout execution, and closeout are out of scope.
- When scope is ambiguous, choose the narrower planning-only write and record a blocker or next-session handoff.

## Imitate
```markdown
Allowed writes used:
- `plan.md`
- `tasks.md`
- `workflow-plan.md`
- `workflow-plans/planning.md`
- `workflow-plans/implementation-phase-1.md` because `plan.md` names that phase

Out of scope and untouched: `spec.md`, `design/`, code, tests, migrations, generated artifacts, runtime config.
```

Copy this shape: it records both the positive write set and the surfaces deliberately left untouched.

```markdown
Planning is ready, but coding T001 is outside this session.
Recorded handoff: `Next session starts with: implementation-phase-1`.
Stop rule: implementation begins in a later session.
```

Copy this shape: it answers a bundled planning-and-coding request without starting implementation.

## Reject
```markdown
I created `workflow-plans/review-phase-1.md` in case review is useful later.
```

Failure: just-in-case phase files create control artifacts not called for by the approved phase structure.

```markdown
I updated `design/sequence.md` with the missing migration order so the plan can be approved.
```

Failure: planning exposed a technical-design gap and must reopen that phase instead of editing design.

## Agent Traps
- Treating "only a tiny test" or "only a quick migration" as harmless during a planning session.
- Editing `spec.md` or `design/` to make a task ledger easier to write.
- Changing `workflow-plan.md` to `Current phase: implementation` before the planning session has stopped.
- Marking speculative tasks approved when their facts belong upstream.
