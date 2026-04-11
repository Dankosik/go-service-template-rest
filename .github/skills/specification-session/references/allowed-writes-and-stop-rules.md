# Allowed Writes And Stop Rules

## Behavior Change Thesis
When loaded for a specification session that is about to edit files or respond to a bundled downstream request, this file makes the model choose a specification-only boundary update instead of the likely mistake of creating `design/`, `plan.md`, `tasks.md`, tests, migrations, or implementation "just to keep momentum."

## When To Load
Load this immediately before file edits when the prompt, artifact state, or user request pressures the session to touch anything beyond `spec.md`, `workflow-plan.md`, and `workflow-plans/specification.md`.

## Decision Rubric
- Writable surfaces are only task-local `spec.md`, task-local `workflow-plan.md`, task-local `workflow-plans/specification.md`, and `workflow-plans/` only when needed to hold the phase file.
- If a requested downstream file would be useful, record the next-session route; do not create a starter version.
- If the user bundles specification with technical design, planning, tests, or implementation, either finish the specification checkpoint or block the session with the boundary issue recorded.
- Do not treat tests, migrations, commits, staging, or implementation diffs as proof of spec quality.
- Do not update later phase-control files from this session unless the approved specification-session artifact explicitly owns them, which should be rare.

## Imitate
Boundary refusal:

```text
Cannot write `design/overview.md` in this session.
Reason: specification-session may update only `spec.md`, `workflow-plan.md`, and `workflow-plans/specification.md`.
Action: finish or block specification, then route the next session to `technical-design` if `spec.md` is approved.
```

Allowed writes note for `workflow-plans/specification.md`:

```text
Allowed writes confirmed: spec.md, workflow-plan.md, workflow-plans/specification.md.
Out of scope: design/, plan.md, tasks.md, tests, migrations, implementation files.
Stop rule: stop after specification artifacts agree on state and next session routing.
```

Copy the specific writable surface names and the explicit stop rule. The value is the boundary decision, not the wording.

## Reject
Bad boundary handling:

```text
Created `design/overview.md` as a placeholder so the next session has somewhere to start.
```

This fails because a placeholder design file is still a downstream artifact and can become false authority.

Bad proof handling:

```text
Ran tests to prove the spec is good.
```

This fails because specification readiness is proved by decision completeness and gate reconciliation, not implementation validation.

## Agent Traps
- Writing a "handoff note" into `design/overview.md` instead of updating `workflow-plan.md`.
- Treating `tasks.md` as harmless because it is only a checklist.
- Editing files first and then trying to justify the phase boundary afterward.
- Recording out-of-scope work only in chat, leaving the workflow artifacts stale.
