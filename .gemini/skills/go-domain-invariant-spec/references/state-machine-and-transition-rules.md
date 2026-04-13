# State Machine And Transition Rules

## Behavior Change Thesis
When loaded for symptom "a feature has lifecycle states, phase boundaries, or invalid transitions", this file makes the model define legal movement, guards, terminal states, and forbidden paths instead of likely mistake "describe event sequence or implementation progress as if that were a state model."

## When To Load
Load this when lifecycle states, phase boundaries, workflow ownership, terminal states, invalid transitions, stuck-state behavior, transition guards, or reopen rules change the allowed domain behavior.

## Decision Rubric
- Model a lifecycle only when allowed behavior differs by state. A one-step command with no legal-state difference usually belongs in acceptance criteria instead.
- Separate domain states from implementation statuses. `specification_blocked` can be domain-meaningful in this repo workflow; `file_written` usually is not unless it changes what is legal next.
- For each transition, name trigger, guard/preconditions, postconditions, allowed next state, forbidden next states, and violation outcome.
- Name terminal states and whether reopening is legal. If reopening is legal, make it an explicit transition.
- Include idempotent repeat, duplicate event, timeout, and stuck-state behavior when those can change correctness.
- Avoid invented states that only mirror an implementation queue, table, or handler unless the business policy depends on that state.

## Imitate
```text
Lifecycle: non_trivial_task
Owner: orchestrator workflow contract
State: planning
Trigger: expected `tasks.md` is ready for implementation handoff
Preconditions:
- `spec.md` is approved or eligible for explicit direct-path waiver
- required `design/` artifacts are approved or design-skip rationale exists
- validation/proof path is explicit
Allowed next states:
- implementation when readiness = PASS
- implementation_with_accepted_risks when readiness = CONCERNS and risks/proof obligations are named
- upstream_reopen when readiness = FAIL
- implementation when readiness = WAIVED and scope/rationale are eligible
Forbidden next states:
- implementation when readiness = FAIL
- implementation when an unresolved high-impact open question could change correctness or ownership
Violation outcome: block implementation and route to the named earlier phase
```

Copy the shape: exact current state, trigger, guard, legal next state, forbidden movement, and consequence.

```text
| From | Trigger | Guard | To | Violation outcome |
| --- | --- | --- | --- | --- |
| `specification` | `spec.md` candidate complete | clarification gate reconciled or eligible waiver recorded | `technical_design` | keep `spec.md` draft or blocked |
| `implementation` | proof exposes missing required `tasks.md` | none | `planning_reopen` | stop coding; do not invent task ledger mid-code |
```

Copy the contrast: the table records only transitions that change allowed behavior, not every file edit.

## Reject
```text
When planning is mostly done, start coding.
```

Failure: "mostly done" does not say which gate was reached, which questions remain, or which failure path applies.

```text
States: creating, saving, logging, returning
```

Failure: these are implementation steps unless the domain gives different permissions or outcomes in each state.

## Agent Traps
- Do not call a status enum "documented" if forbidden transitions are still missing.
- Do not hide ambiguous external outcomes behind "retry later"; name pending, ambiguous, reconciliation, manual intervention, or reject.
- Do not treat `WAIVED`, `CONCERNS`, and `PASS` as synonyms when the domain policy distinguishes them.
- Do not create terminal states without saying whether support, replay, reconciliation, or admin action can reopen them.
- Do not convert every downstream async step into a domain state; keep only states that change allowed behavior or proof obligations.

## Validation Shape
For a nontrivial lifecycle, proof should include at least one allowed transition, one forbidden transition, one terminal or reopen case when applicable, and one duplicate/retry/stuck-state case when applicable.
