# Assumptions And Open Questions

Behavior Change Thesis: When loaded for the symptom "the frame depends on implied facts, missing owner decisions, or an unprioritized question pile," this file makes the model choose labeled assumptions and decided technical defaults instead of the likely mistake of inventing answers, producing a generic TODO list, or exporting agent-owned branches to the user as questions.

## When To Load
Load this when the request says "should be fine," "obviously," "just," "we know," or similar, or when the open questions do not say who owns the answer and what answer would unblock design.

## Decision Rubric
- Mark uncertain facts as `[assumption]`; do not disguise them as decisions.
- Each assumption needs risk and validation path. If it has no risk, it probably does not belong here.
- Decide architecture, API, data, security, reliability, and rollout branches yourself from current evidence and record each as `[decided]` with its reopen condition.
- Route a branch forward as `[blocks design]` only when a later phase owns that decision and the frame genuinely cannot fix it. The branch stays agent-owned there; it never becomes a user question on the way.
- Tag `[user-owned]` only for an item that survives [Decision Ownership](../../../../AGENTS.md#decision-ownership), and give it an owner and unblock condition.
- Use `[nice to know]` for anything that cannot block readiness.

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

Decisions
- [decided] This service's receiver contract owns duplicate-event semantics; partner payloads are untrusted input, not an authority.
  Reopen condition: A partner contract or upstream event gateway already defines dedupe as its guarantee.
- [decided] Recognize a duplicate for the longest partner retry horizon in current evidence, and treat a later repeat as a new event.
  Reopen condition: Measured partner retry behavior exceeds that horizon, or retention cost makes it infeasible.

Open Questions
- [nice to know] Which partner produces the highest duplicate volume?
  Owner: research or operations.
  Unblock condition: Evidence for prioritization; not required to frame semantics.
```

Copy: event identity semantics are decided before any datastore or middleware choice, not handed back as a question.

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

Decisions
- [decided] Existing unverified users keep their projects and are prompted, not blocked; enforcement applies to new project creation only.
  Reopen condition: The abuse evidence shows existing unverified accounts are the dominant source.

Open Questions
- [user-owned] What abuse behavior must change?
  Why it survives: Both "stop throwaway-account signup floods" and "stop shared-credential resale" are defensible, they lead to different controls, and neither can be assumed honestly from repository evidence.
  Owner: product/security framing.
  Unblock condition: A concrete abuse scenario and success signal.
```

Copy: the one branch that changes the accepted outcome reaches the user; the compatibility branch is decided.

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

Decisions
- [decided] Expose `queued`, `running`, `succeeded`, `failed`, and `expired` as the user-visible lifecycle, with a terminal failure reason.
  Reopen condition: An existing client contract already fixes different status semantics.

Open Questions
- [blocks design] Which report types exceed acceptable latency, and by how much?
  Owner: performance/research lane.
  Unblock condition: Measured latency per report type; measure rather than ask.
```

Copy: the measurable branch becomes an agent-owned evidence obligation, and the contract shape is decided rather than surveyed.

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

Bad:
```markdown
Open Questions
- [blocks design] Should dedupe live in middleware, the handler, or a unique index?
  Owner: user.
```

Why: it hands the user a technical menu. Mechanism is agent-owned; decide it and record the reopen condition.

## Agent Traps
- Do not ask every possible specialist question. Ask the questions that change framing or route the next spec lanes.
- Do not say "owner: team" unless the routing is meaningful enough for the orchestrator to act on it.
