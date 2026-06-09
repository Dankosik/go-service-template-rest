# Handoff To Specification Review

## Behavior Change Thesis
When loaded after `spec.md` looks review-ready or when the next session route is being chosen, this file makes the model choose a clean `specification-review` handoff recorded in workflow artifacts instead of the likely mistake of starting review/design/planning work, hiding accepted assumptions in chat, or routing forward while the clarification gate is still blocked.

## When To Load
Load this after spec review-readiness or when deciding whether the specification session may set `Next session starts with: specification-review`.

## Decision Rubric
- Handoff requires review-ready `spec.md`, resolved inline or formal clarification gate, and agreement between triggered workflow-control artifacts.
- The handoff names accepted assumptions, blockers, and reopen conditions; it does not create design content.
- Accepted risk can pass forward only when it does not change scope, ownership, acceptance semantics, or validation proof; specification review may still challenge it.
- If a missing answer still changes a core decision, route to research or specification instead of specification review.
- The session stops before creating triggered `design/`, `tasks.md`, tests, or implementation changes unless an explicit direct/lean phase-collapse waiver was already recorded.

## Imitate
Ready handoff in `workflow-plan.md`:

```text
Current phase: specification
Current phase status: complete
Session boundary reached: yes
Ready for next session: yes
Next session starts with: specification-review
Artifacts: spec.md review-ready; specification-review missing; design/ missing; tasks.md missing
Clarification gate: complete and reconciled
Blockers: none
Reopen conditions: reopen specification if specification review finds a scope, decision, assumption, or validation contradiction.
```

Accepted risk that can move forward:

```text
Handoff status: ready with accepted risk
Accepted risk: exact retry backoff values are deferred to technical design under the constraint that retry budget remains bounded and observable.
Spec location: `Open Questions / Assumptions` and `Validation`
Next session starts with: specification-review
```

Copy the separation: risk constraints live in spec/workflow surfaces; specification review tests whether they are explicit enough for the next downstream phase.

## Reject
Premature design:

```text
Created `design/overview.md` with the known constraints so the handoff is concrete.
```

This fails because the handoff becomes technical design work.

Unsafe route:

```text
Next session starts with: specification-review
Clarification gate: blocked by idempotency semantics
```

This fails because the next phase would review from an unresolved decision record.

## Agent Traps
- Treating "no blockers mentioned in chat" as equivalent to `Blockers: none` in `workflow-plan.md`.
- Moving an approval-changing unknown forward as accepted risk.
- Writing a design checklist when a workflow handoff sentence is enough.
- Forgetting to mark `Session boundary reached: yes` when stopping at the phase boundary.
