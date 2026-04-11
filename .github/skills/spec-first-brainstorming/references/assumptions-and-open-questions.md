# Assumptions And Open Questions

Behavior Change Thesis: When loaded for the symptom "the frame depends on implied facts, missing owner decisions, or an unprioritized question pile," this file makes the model choose labeled assumptions and owner-routed blocking questions instead of the likely mistake of inventing answers or producing a generic TODO list.

## When To Load
Load this when the request says "should be fine," "obviously," "just," "we know," or similar, or when the open questions do not say who owns the answer and what answer would unblock design.

## Decision Rubric
- Mark uncertain facts as `[assumption]`; do not disguise them as decisions.
- Each assumption needs risk and validation path. If it has no risk, it probably does not belong here.
- Each blocking question needs owner and unblock condition.
- Use tags such as `[blocks design]`, `[blocks API design]`, `[blocks data/reliability design]`, or `[nice to know]`.
- Do not answer product policy, architecture, API, data, security, reliability, or rollout questions unless the user or repository evidence already supplied the answer.

## Imitate
Input:
```text
Add a retry-safe webhook receiver. Partners sometimes send duplicates but it should be fine because events have IDs.
```

Output:
```markdown
Assumptions
- [assumption] Partner event IDs are stable across duplicate deliveries.
  Risk: If event IDs change per retry, dedupe framing will fail.
  Validation path: Confirm partner event identity semantics from docs, contracts, or observed payloads before API/data design.
- [assumption] Duplicate delivery should not create duplicate side effects.
  Risk: Some event types may be intentionally repeated with different meaning.
  Validation path: Classify event types and side effects during specification.

Open Questions
- [blocks design] Who owns the duplicate-event semantics: this service contract, a partner contract, or an upstream event gateway?
  Owner: API/domain spec lane.
  Unblock condition: A source-of-truth decision for event identity and repeat handling.
- [blocks data/reliability design] What is the required duplicate-recognition window?
  Owner: data/reliability spec lanes.
  Unblock condition: A retention or replay horizon that downstream design can preserve.
- [nice to know] Which partner produces the highest duplicate volume?
  Owner: research or operations.
  Unblock condition: Evidence for prioritization; not required to frame semantics.
```

Copy: the output targets event identity semantics before any datastore or middleware choice.

Input:
```text
Let's require email verification before users can create projects. That should stop abuse.
```

Output:
```markdown
Assumptions
- [assumption] Unverified email is a material abuse path.
  Risk: The change may add user friction without reducing the dominant abuse mode.
  Validation path: Check abuse reports or logs for correlation with unverified accounts.
- [assumption] Project creation is the right enforcement point.
  Risk: Abuse may happen before or after project creation, or require different controls.
  Validation path: Map the user action where abuse becomes costly before security design.

Open Questions
- [blocks design] What abuse behavior must change?
  Owner: product/security framing.
  Unblock condition: A concrete abuse scenario and success signal.
- [blocks API/domain design] What should happen to existing unverified users with projects?
  Owner: product/domain spec lane.
  Unblock condition: Acceptance semantics for existing accounts.
```

Copy: the proposed control is not accepted until the abuse behavior and enforcement point are proven.

Input:
```text
Move report generation async. Reports are taking too long.
```

Output:
```markdown
Assumptions
- [assumption] Users can tolerate delayed report completion if they get clear status.
  Risk: Async behavior may solve server latency but worsen user workflow.
  Validation path: Confirm user-facing acceptance expectations before design.
- [assumption] The slow path is report generation, not request validation, data access, or export delivery.
  Risk: Async work may hide the bottleneck instead of addressing it.
  Validation path: Gather timing evidence before performance design.

Open Questions
- [blocks design] Which report types exceed acceptable latency, and by how much?
  Owner: performance/research lane.
  Unblock condition: Measured or accepted latency threshold.
- [blocks product/API design] What user-visible lifecycle states are required?
  Owner: API/domain spec lane.
  Unblock condition: Accepted status semantics before contract design.
```

Copy: performance suspicion is separated from user-visible async behavior.

## Reject
Bad:
```markdown
Assumptions
- Events have IDs, so dedupe is safe.
```

Why: the risky part is whether the IDs are stable across duplicate delivery, not whether an `id` field exists.

Bad:
```markdown
Open Questions
- What database?
- What endpoint?
- What tests?
```

Why: implementation questions arrived before the unknown behavior semantics were owned.

## Agent Traps
- Do not ask every possible specialist question. Ask the questions that change framing or route the next spec lanes.
- Do not let `[nice to know]` questions block readiness.
- Do not say "owner: team" unless the routing is meaningful enough for the orchestrator to act on it.
