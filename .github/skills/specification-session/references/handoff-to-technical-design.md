# Handoff To Technical Design

## Behavior Change Thesis
When loaded after `spec.md` looks approved or when the next session route is being chosen, this file makes the model choose a clean `technical-design` handoff recorded in workflow artifacts instead of the likely mistake of starting design work, hiding accepted assumptions in chat, or routing forward while the clarification gate is still blocked.

## When To Load
Load this after spec approval or when deciding whether the specification session may set `Next session starts with: technical-design`.

## Decision Rubric
- Handoff requires approved `spec.md`, resolved or explicitly waived clarification gate, and agreement between `workflow-plan.md` and `workflow-plans/specification.md`.
- The handoff names accepted assumptions, blockers, and reopen conditions; it does not create design content.
- Accepted risk can pass forward only when it does not change scope, ownership, acceptance semantics, or validation proof.
- If a missing answer still changes a core decision, route to research or specification instead of technical design.
- The session stops before creating `design/`, `tasks.md`, optional `plan.md`, tests, or implementation changes.

## Imitate
Ready handoff in `workflow-plan.md`:

```text
Current phase: specification
Current phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: technical-design
Artifacts: spec.md approved; design/ missing; tasks.md missing; plan.md not expected unless later justified
Clarification gate: complete and reconciled
Blockers: none
Reopen conditions: reopen specification if technical design finds a scope or acceptance contradiction.
```

Accepted risk that can move forward:

```text
Handoff status: ready with accepted risk
Accepted risk: exact retry backoff values are deferred to technical design under the constraint that retry budget remains bounded and observable.
Spec location: `Open Questions / Assumptions` and `Validation`
Next session starts with: technical-design
```

Copy the separation: risk constraints live in spec/workflow surfaces; the actual design choice is left for the next session.

## Reject
Premature design:

```text
Created `design/overview.md` with the known constraints so the handoff is concrete.
```

This fails because the handoff becomes technical design work.

Unsafe route:

```text
Next session starts with: technical-design
Clarification gate: blocked by idempotency semantics
```

This fails because the next phase would design from an unapproved decision record.

## Agent Traps
- Treating "no blockers mentioned in chat" as equivalent to `Blockers: none` in `workflow-plan.md`.
- Moving an approval-changing unknown forward as accepted risk.
- Writing a design checklist when a workflow handoff sentence is enough.
- Forgetting to mark `Session boundary reached: yes` when stopping at the phase boundary.
