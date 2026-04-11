# Assumptions And Open Questions

## When To Load
Load this when the frame depends on implied facts, missing owner decisions, unvalidated risky assumptions, or a question list that is not prioritized by unblock value.

## Calibration Guardrail
Use assumptions and questions to expose uncertainty. Do not answer product policy, architecture, API, data, security, reliability, or rollout questions unless repository evidence or the user has already supplied the answer.

## Raw Request Examples
Example A:

```text
Add a retry-safe webhook receiver. Partners sometimes send duplicates but it should be fine because events have IDs.
```

Example B:

```text
Let's require email verification before users can create projects. That should stop abuse.
```

Example C:

```text
Move report generation async. Reports are taking too long.
```

## Final Framing Output
Example A:

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

Example B:

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

Example C:

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

## Pass/Fail Readiness Examples
Pass:

```markdown
Readiness Decision
pass - Critical assumptions are labeled with risk and validation path, and blocking questions have owners plus unblock conditions. Downstream work can either answer them in specialist spec lanes or mark the frame blocked without losing the uncertainty.
```

Fail:

```markdown
Readiness Decision
fail - The output lists "dedupe events" as a given and asks "what database should we use?" It skips the planning-critical assumption about partner event identity and jumps into implementation territory.
```

## Exa Source Links
Exa MCP was attempted before writing this reference, but search and fetch returned a 402 credit-limit error. Links below are fallback calibration sources, not repo-authoritative workflow rules.

- Product Talk, assumption testing as solution-evaluation evidence: https://www.producttalk.org/opportunity-solution-trees/
- GOV.UK Service Manual, assumptions and questions during problem reframing: https://www.gov.uk/service-manual/agile-delivery/how-the-discovery-phase-works
- Atlassian Team Playbook, pre-mortem prompts for hidden worries and missing inputs: https://www.atlassian.com/team-playbook/plays/pre-mortem
- NASA Systems Engineering Handbook appendix, validation and explicitly stated assumptions: https://www.nasa.gov/reference/system-engineering-handbook-appendix/
