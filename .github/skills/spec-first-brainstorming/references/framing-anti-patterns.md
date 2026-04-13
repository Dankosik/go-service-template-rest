# Framing Anti-Patterns

Behavior Change Thesis: When loaded for the symptom "the draft starts smuggling downstream design, task breakdown, or stakeholder theater into brainstorming," this file makes the model remove or reframe the drift instead of the likely mistake of treating a broad anti-pattern list as primary design guidance.

## When To Load
Load this when the brainstorming output starts smuggling architecture/API/data/security/reliability decisions, task breakdowns, implementation design, or vague stakeholder theater into the frame.

## Decision Rubric
- Use this as challenge/smell triage, not the default reference when a narrower positive reference matches.
- Preserve the real behavior concern while stripping unapproved mechanisms.
- Reject task ledgers, schema/API choices, rollout plans, and telemetry names inside brainstorming output.
- Replace vague stakeholder theater with actor, behavior, and success signal.
- If the frame cannot be rescued without product or repository evidence, mark readiness `fail`.

## Imitate
Input:
```text
We need Redis-backed dedupe middleware for webhook delivery.
```

Output:
```markdown
Problem
Webhook processing may perform duplicate side effects when the same delivery intent arrives more than once.

Behavior Delta
- Current behavior: Duplicate delivery handling is unclear or unsafe.
- Desired behavior: Repeat deliveries can be recognized or safely absorbed according to the accepted webhook semantics.

Non-Goals
- No Redis decision.
- No middleware decision.
- No storage design.
- No handler task breakdown.

Anti-Pattern Avoided
Do not restate the request as "build Redis-backed dedupe middleware." That converts a behavior problem into an unapproved implementation.
```

Copy: Redis and middleware are removed while duplicate side-effect safety stays alive.

Input:
```text
Make search better. Use Elasticsearch and add a new /search endpoint.
```

Output:
```markdown
Problem
Users cannot find the records they expect with the current search behavior, but the failure mode is not yet specific enough to design.

Behavior Delta
- Current behavior: Search quality, latency, filter expressiveness, or result explainability may be insufficient, but the exact gap is unknown.
- Desired behavior: The relevant search failure is named and measurable before solution design.

Readiness Decision
fail - The frame needs the affected actor, search workflow, current failure examples, and success signal before choosing an index, endpoint, ranking behavior, or query model.

Anti-Pattern Avoided
Do not accept "Use Elasticsearch" as the problem statement.
```

Copy: solution choice is rejected because the search failure mode is still unknown.

Input:
```text
Add audit logging because enterprise customers need compliance.
```

Output:
```markdown
Problem
Enterprise operators need enough account or system-change history to investigate compliance-relevant actions, but the required events, actors, retention, and visibility boundaries are not yet defined.

Behavior Delta
- Current behavior: Compliance-relevant changes may not be inspectable by the right operator.
- Desired behavior: The service can support investigation of named compliance-relevant actions with appropriate actor and boundary context.

Open Questions
- [blocks design] Which actions are compliance-relevant for the first pass?
- [blocks security design] Who may view audit history, and across which tenant/account boundary?
- [blocks data design] What retention expectation applies?

Anti-Pattern Avoided
Do not turn "enterprise compliance" into a generic logging feature or event schema before the compliance question is framed.
```

Copy: "compliance" is decomposed into event, actor, visibility, and retention questions without designing the log schema.

## Reject
Bad:
```markdown
Tasks
- Add Redis table.
- Write dedupe middleware.
- Add tests.
- Update docs.
```

Why: task breakdown belongs after approved spec/design/planning, not brainstorming.

Bad:
```markdown
Problem
Enterprise customers need compliance.
```

Why: it is stakeholder theater unless it names the compliance-relevant actions, actors, and success signal.

## Agent Traps
- Do not use this file to produce a broad "avoid all bad things" checklist.
- Do not discard the user's implementation idea without preserving the behavior concern it hints at.
- Do not call something a non-goal if it is actually a planning-critical unknown; route it as an open question instead.
