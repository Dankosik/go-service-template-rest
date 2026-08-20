# Framing Anti-Patterns

## Load When
Load this when the brainstorming output starts smuggling architecture/API/data/security/reliability decisions, task breakdowns, implementation design, or vague stakeholder theater into the frame. When a narrower positive reference matches, load that instead — this file is smell triage.

## The Move
Preserve the real behavior concern while stripping the unapproved mechanism the user proposed. Remove task ledgers, schema and API choices, rollout plans, and telemetry names from brainstorming output; replace stakeholder theater with a named actor, behavior, and success signal. A "non-goal" that is actually a planning-critical unknown routes as an open question instead. When the frame cannot be rescued without product or repository evidence, mark readiness `fail`.

## Inspect
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
Restating the request as "build Redis-backed dedupe middleware" would convert a behavior problem into an unapproved implementation.
```

Copy: Redis and middleware are removed while duplicate side-effect safety stays alive.

## Reject
```markdown
Tasks
- Add Redis table.
- Write dedupe middleware.
- Add tests.
- Update docs.
```

Task breakdown belongs after approved spec/design/planning, not in brainstorming.
