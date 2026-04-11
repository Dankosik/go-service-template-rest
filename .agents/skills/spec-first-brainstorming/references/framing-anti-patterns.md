# Framing Anti-Patterns

## When To Load
Load this when the brainstorming output starts smuggling architecture/API/data/security/reliability decisions, task breakdowns, implementation design, or vague stakeholder theater into the frame.

## Calibration Guardrail
This file helps reject bad framing. It does not authorize downstream design decisions or task plans.

## Raw Request Examples
Example A:

```text
We need Redis-backed dedupe middleware for webhook delivery.
```

Example B:

```text
Make search better. Use Elasticsearch and add a new /search endpoint.
```

Example C:

```text
Add audit logging because enterprise customers need compliance.
```

## Final Framing Output
Example A:

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
- No handler implementation plan.

Anti-Pattern Avoided
Do not restate the request as "build Redis-backed dedupe middleware." That converts a behavior problem into an unapproved implementation.
```

Example B:

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

Example C:

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

## Pass/Fail Readiness Examples
Pass:

```markdown
Readiness Decision
pass - The output rejects Redis, middleware, and endpoint choices as non-goals while preserving the duplicate-side-effect behavior that downstream design must solve.
```

Fail:

```markdown
Readiness Decision
fail - The output contains "Tasks: add Redis table, write middleware, add tests, update docs." That is task breakdown and implementation design, not spec-first brainstorming.
```

## Exa Source Links
Exa MCP was attempted before writing this reference, but search and fetch returned a 402 credit-limit error. Links below are fallback calibration sources, not repo-authoritative workflow rules.

- GOV.UK Service Manual, reframing predefined solutions into problems and non-problem boundaries: https://www.gov.uk/service-manual/agile-delivery/how-the-discovery-phase-works
- Product Talk, separating outcomes/opportunities/solutions and testing assumptions: https://www.producttalk.org/opportunity-solution-trees/
- Atlassian Team Playbook, prompts that expose plan risk before execution: https://www.atlassian.com/team-playbook/plays/pre-mortem
- NASA Systems Engineering Handbook appendix, avoiding unverifiable language in requirement-like framing: https://www.nasa.gov/reference/system-engineering-handbook-appendix/
