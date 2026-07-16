# Acceptance Criteria And Corner Cases

## Behavior Change Thesis
When loaded for symptom "domain rules exist but acceptance behavior or proof obligations are vague", this file makes the model write observable positive, negative, and edge-case outcomes instead of likely mistake "happy-path prose, implementation hints, or generic 'handle gracefully' statements."

## When To Load
Load this when domain decisions are known but acceptance behavior, edge cases, proof obligations, or QA handoff are still too vague for planning.

## Decision Rubric
- Write acceptance as externally observable domain behavior: `Given [state and actor] / When [command, event, or decision] / Then [allowed outcome] / And [invariant remains true]`.
- Cover happy path, forbidden path, fail path, and the smallest corner cases that can change the business outcome.
- Include duplicate, replay, out-of-order, timeout, stale-read, and actor-boundary cases only when they are relevant to the rule.
- Say whether invalid input or state is rejected, denied, treated as idempotent replay, routed to reconciliation, or accepted as risk.
- Avoid implementation hints until the business outcome is explicit. "Insert a row" is not acceptance behavior unless the rule is about durable record authority.
- Make each criterion convertible into a test name, setup, trigger, expected outcome, and invariant assertion.

## Imitate
```text
Given a non-trivial task is in `implementation`
And the implementation discovers that required `tasks.md` is missing
When the orchestrator needs task ordering to continue safely
Then the workflow transitions to `planning_reopen`
And coding stops
And no new task ledger is invented during implementation
And the reopen target is recorded in existing control artifacts when they exist
```

Copy the shape: state, trigger, allowed outcome, forbidden side effect, and proof target.

```text
| Category | Prompt | Expected domain decision |
| --- | --- | --- |
| Duplicate | What if the same user request is submitted twice? | Same logical operation yields one domain effect or an explicit conflict. |
| Invalid transition | What if validation tries to create a new planning artifact? | Reject and reopen the correct earlier phase. |
| Permission | What if a read-only subagent attempts a write? | Treat as policy violation; do not accept the mutation. |
| Stale read | What if a projection disagrees with the source of truth? | Source of truth wins unless the spec has an explicit freshness contract. |
```

Copy the category table when a compact set of edge prompts will improve planning more than long prose.

## Reject
```text
Handle missing planning files gracefully.
```

Failure: no state, trigger, allowed outcome, rejected outcome, or proof path is defined.

```text
When the command is valid, call `Save()` and return success.
```

Failure: it describes mechanics and assumes validity; it does not state the business outcome or failed-invariant behavior.

## Agent Traps
- Do not treat "green tests" as acceptance criteria unless the rule, trigger, and expected outcome are named.
- Do not write only the happy path when a forbidden transition or fail-closed actor boundary is the actual risk.
- Do not collapse retry, duplicate, and replay into one generic "idempotent" sentence; name the same-intent and different-intent outcomes when relevant.
- Do not use "eventually consistent" without saying what stale reads may and may not decide.
- Do not invent exhaustive edge-case lists; pick the few cases that can change the domain decision.

## Validation Shape
A strong acceptance criterion gives QA enough material for: test name, setup state, actor, trigger, expected outcome, invariant assertion, and any side-effect or rejection assertion.
