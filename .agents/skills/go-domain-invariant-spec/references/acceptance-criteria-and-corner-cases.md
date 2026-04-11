# Acceptance Criteria And Corner Cases

## When To Load
Load this when the domain decisions are known but acceptance behavior, edge cases, or proof obligations are still too vague for planning or QA handoff.

Start from the current task's product/spec artifacts. In this repo, useful anchors include `spec.md` `Decisions`, `Validation`, `Outcome`, and plan acceptance criteria. Use external sources only to calibrate testable wording and edge-case categories.

## Acceptance Pattern
Write acceptance criteria as observable domain behavior:

```text
Given [domain state and actor]
When [command/event/decision occurs]
Then [allowed outcome]
And [invariant remains true]
And [side effect or violation behavior is explicit]
```

Avoid implementation hints until the business outcome is explicit. "Insert a row" is not acceptance behavior unless the user-visible or domain-significant rule is about a durable record.

## Example Invariant Statements
- `NoSuccessOnFailedInvariant`: a workflow action must not be marked successful when a critical invariant check failed.
- `ValidationEvidenceBeforeDone`: a task may not claim done without fresh validation evidence matching the claim scope.
- `CanonicalPathAcceptance`: after source-of-truth skill cleanup, active docs and tooling must not treat removed `skills/` paths as canonical.
- `DirectPathWaiverScope`: a direct-path waiver applies only to the stated scope and does not authorize later artifact invention in implementation or validation.
- `AcceptanceCriteriaTraceability`: every critical domain rule must have at least one acceptance criterion that a reviewer can convert into a test name, input, trigger, and expected outcome.

## Good And Bad State Transition Specs
Good transition-backed acceptance spec:

```text
Given a non-trivial task is in `implementation`
And the implementation discovers that required `tasks.md` is missing
When the orchestrator needs task ordering to continue safely
Then the workflow transitions to `planning_reopen`
And coding stops
And no new task ledger is invented during implementation
And the reopen target is recorded in existing control artifacts when they exist
```

Bad transition-backed acceptance spec:

```text
Handle missing planning files gracefully.
```

Why it fails: no state, trigger, allowed outcome, or proof path is defined.

Good corner-case table:

| Category | Prompt | Expected domain decision |
| --- | --- | --- |
| Duplicate | What if the same user request is submitted twice? | Same logical operation yields one domain effect or an explicit conflict. |
| Invalid transition | What if validation tries to create a new planning artifact? | Reject and reopen the correct earlier phase. |
| Boundary | What is the smallest task that may skip the design bundle? | Only tiny/direct-path work with explicit rationale. |
| Permission | What if a subagent attempts a write? | Treat as policy violation; do not accept the mutation. |
| Timeout | What if an external dependency outcome is unknown? | Model ambiguous state, retry, reconciliation, or manual intervention explicitly. |
| Stale read | What if a projection disagrees with the source of truth? | Source of truth wins; projection cannot drive invariant-sensitive decisions without a freshness contract. |

## Edge-Case Prompts
- What invalid input or state should be rejected rather than repaired silently?
- What duplicate or replay should be idempotent, and what duplicate should conflict?
- What out-of-order event could arrive, and which state is allowed afterward?
- Which boundary values change the business outcome?
- What happens when cancellation or timeout occurs after a side effect may have committed?
- Which actor is not allowed to perform this transition?
- Which "rare" case would force support/manual intervention, and is that acceptable?

## Downstream Handoff Notes
- QA handoff: include enough detail for test name, setup, trigger, expected outcome, and invariant assertion.
- API handoff: once acceptance semantics are stable, encode the external success, conflict, rejection, or async acknowledgement behavior.
- Data handoff: identify any persistence proof needed for the acceptance criteria, but do not lead with schema mechanics.
- Reliability handoff: timeout and retry criteria should preserve the same domain outcome under repeated attempts.
- Rollout handoff: if mixed-version behavior can change acceptance results, define the compatibility expectation before implementation planning.

## Exa Source Links
- [Spec Coding: Edge Case Checklist](https://spec-coding.dev/guides/edge-case-checklist) for edge-case categories and testable spec wording.
- [Spec Coding: Designing Idempotent Workflows with Specs](https://spec-coding.dev/blog/designing-idempotent-workflows-with-specs) for retry-safe acceptance criteria examples.
- [Cosmic Python: Aggregates and Consistency Boundaries](http://www.cosmicpython.com/book/chapter_07_aggregate.html) for using high-level tests as living documentation of domain rules.
