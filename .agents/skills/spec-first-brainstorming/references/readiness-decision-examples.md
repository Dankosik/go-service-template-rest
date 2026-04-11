# Readiness Decision Examples

## When To Load
Load this when the pass/fail readiness decision is close, blocked, or drifting into a generic "ready enough" statement.

## Calibration Guardrail
Readiness answers only whether the problem frame can move into challenge/spec/design. It does not approve architecture, API, data, security, reliability, rollout, or implementation tasks.

## Raw Request Examples
Example A:

```text
Expose audit events to admins so they can investigate suspicious account changes.
```

Example B:

```text
Make uploads faster.
```

Example C:

```text
Add soft delete for projects so accidental deletes can be recovered.
```

## Final Framing Output
Example A:

```markdown
Readiness Decision
pass

Why
The frame names the operator, current gap, desired investigation outcome, high-risk seams, and open questions that specialist lanes can own. It is not yet an API or storage design, but it is specific enough for pre-spec challenge.

Next Handoff
Run pre-spec challenge with emphasis on authorization boundaries, tenant visibility, event completeness, retention expectations, and support-safe redaction.
```

Example B:

```markdown
Readiness Decision
fail

Why
"Make uploads faster" does not state the affected actor, upload type, current latency, desired threshold, dominant pain, or whether the problem is throughput, tail latency, reliability, UX feedback, or backend capacity.

Minimum Data Needed
- Which upload workflow is affected.
- Current observed latency or failure mode.
- Desired user or operator outcome.
- Whether the performance issue blocks product behavior, operational safety, or user satisfaction.
```

Example C:

```markdown
Readiness Decision
pass

Why
The frame can pass if it preserves the recovery problem and explicitly leaves delete-state modeling, API contract, retention, authorization, and cleanup policy for downstream design.

Accepted Risks
- Existing delete behavior and restore authorization remain open questions, but they are tracked as design blockers with owners and unblock conditions.

Next Handoff
Run pre-spec challenge because delete semantics are stateful and may hide authorization, retention, and consistency risks.
```

## Pass/Fail Readiness Examples
Pass:

```markdown
Readiness Decision
pass - The frame is specific enough to route downstream work: admin investigation of suspicious account changes, with authorization, event completeness, retention, and redaction marked as challenge seams. No final design is smuggled in.
```

Fail:

```markdown
Readiness Decision
fail - The frame says "implement soft delete with deleted_at and a restore endpoint." It jumped to implementation before proving delete/recover behavior, actor needs, authorization consequences, or retention constraints.
```

## Exa Source Links
Exa MCP was attempted before writing this reference, but search and fetch returned a 402 credit-limit error. Links below are fallback calibration sources, not repo-authoritative workflow rules.

- GOV.UK Service Manual, discovery finish criteria and stop/go decision framing: https://www.gov.uk/service-manual/agile-delivery/how-the-discovery-phase-works
- Product Talk, deciding what to build after testing risky assumptions: https://www.producttalk.org/opportunity-solution-trees/
- Atlassian Team Playbook, pre-mortem prompts for surfacing hidden blockers before work starts: https://www.atlassian.com/team-playbook/plays/pre-mortem
- NASA Systems Engineering Handbook appendix, validated requirements qualities and V&V distinction: https://www.nasa.gov/reference/system-engineering-handbook-appendix/
