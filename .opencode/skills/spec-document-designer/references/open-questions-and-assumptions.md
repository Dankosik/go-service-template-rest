# Open Questions And Assumptions

## Behavior Change Thesis
When loaded for unresolved uncertainty, this file makes the model label and route the uncertainty by unblock path instead of inventing certainty, hiding blockers in `Decisions`, or leaving decorative `TBD`s.

## When To Load
Load this when a spec has unknowns, `TODO`, `TBD`, blocked clarification items, external policy decisions, soft assumptions, or risks that could change planning.

## Decision Rubric
- Use `[assumption]` when the orchestrator can proceed and the assumption is safe enough to test or revisit.
- Use `[accepted_risk]` when the risk is known, deliberately accepted, and does not block planning.
- Use `[requires_user_decision]` when repository evidence cannot decide a product, business, or policy question and the answer changes behavior or validation.
- Use `[targeted_research]` when evidence is missing and research can answer the question.
- Use `[defer_to_design]` when the spec-level decision is stable but the exact component, sequence, or ownership detail belongs in `design/`.
- Use `[reopen_spec_if_false]` when downstream design may invalidate a spec-level assumption.
- Do not keep an item in `Open Questions / Assumptions` if it is merely "future work"; make it a non-goal or remove it.

## Imitate

```markdown
## Open Questions / Assumptions
- [assumption] The existing tenant ID remains the isolation boundary for this change; no new tenant hierarchy is introduced.
- [accepted_risk] The first implementation will not backfill historical records because the feature only affects new writes.
- [defer_to_design] The exact package boundary for the reload coordinator belongs in `design/ownership-map.md`; the spec only decides that config remains the source of truth.
```

Copy this: each item says how to treat the uncertainty and why it does not need to become a fake decision.

```markdown
## Open Questions / Assumptions
- [requires_user_decision] Product must choose whether expired invites should be hidden or shown as disabled. Planning is blocked because this changes API response semantics and tests.
```

Copy this: the spec refuses to invent product policy when the answer changes acceptance semantics.

## Reject

```markdown
## Open Questions / Assumptions
- TBD
- Maybe performance?
- Need to check stuff.
```

Failure: the items do not identify the decision, impact, owner, or unblock path.

```markdown
## Decisions
- Expired invites probably stay visible, but this can change later.
```

Failure: the uncertainty changes API behavior, so it belongs in `Open Questions / Assumptions` until resolved or explicitly accepted.

## Agent Traps
- Asking the user every question instead of answering from repository evidence first.
- Treating `[defer_to_design]` as permission to defer behavior semantics.
- Marking a spec approved while `[requires_user_decision]` still changes correctness or validation.
- Leaving a vague "future work" list instead of drawing a non-goal.
