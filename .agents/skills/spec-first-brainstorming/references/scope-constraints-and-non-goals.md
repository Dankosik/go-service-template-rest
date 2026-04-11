# Scope Constraints And Non-Goals

## When To Load
Load this when scope is expanding, non-goals are vague, constraints may shape downstream design, or the request mixes engineering framing with delivery commitments.

## Calibration Guardrail
Use this file to make boundaries visible. Do not convert scope into architecture, API, schema, security, reliability, rollout, or task decisions.

## Raw Request Examples
Example A:

```text
Add tenant-aware export limits, plus maybe cache export files and show admin analytics while we are in there.
```

Example B:

```text
We need account deletion to be GDPR-compliant. Delete all user data everywhere and make it fast.
```

Example C:

```text
Can we make background jobs observable and maybe also rewrite the worker pool?
```

## Final Framing Output
Example A:

```markdown
Scope
In scope:
- Frame the tenant-aware export-limit behavior and how users or operators should experience limit enforcement.
- Identify which tenant/account boundaries matter to the limit decision.
- Capture the success signal: repeatable enforcement that prevents one tenant from exhausting export capacity for others.

Non-Goals
- No export-file cache design.
- No admin analytics feature.
- No storage or queue implementation choice.
- No specific rate-limit algorithm.

Constraints
- Tenant isolation is a hard constraint.
- The framing must preserve fairness across tenants without deciding the enforcement mechanism.
- Existing export semantics must not be redefined unless later specification explicitly approves it.
```

Example B:

```markdown
Scope
In scope:
- Frame the account-deletion behavior from user request through externally visible completion or rejection.
- Identify data-retention and audit constraints that could prevent immediate deletion.
- Separate user-visible deletion semantics from internal cleanup mechanics.

Non-Goals
- No final legal policy decision.
- No database deletion strategy.
- No cross-service saga design.
- No performance SLO selection.

Constraints
- Privacy and retention requirements may conflict and must be surfaced as planning-critical questions.
- "Fast" is not a usable requirement until a stakeholder defines the acceptance threshold.
```

Example C:

```markdown
Scope
In scope:
- Frame the operator problem: background job failures are hard to diagnose and correlate with user-visible outcomes.
- Identify which job lifecycle states or failure classes must be visible to support later design.

Non-Goals
- No worker-pool rewrite.
- No tracing library choice.
- No metric names or log schema.
- No queue replacement.

Constraints
- The frame must preserve existing job semantics unless later specification approves behavior change.
- Observability signals should be tied to operator questions, not generic telemetry expansion.
```

## Pass/Fail Readiness Examples
Pass:

```markdown
Readiness Decision
pass - Scope, non-goals, and constraints isolate the export-limit problem from adjacent cache and analytics ideas. Downstream design has enough boundary information to proceed without treating the extra ideas as approved work.
```

Fail:

```markdown
Readiness Decision
fail - The frame says "make account deletion GDPR-compliant" but does not separate deletion, retention, audit, user notification, and external dependency boundaries. It would force downstream specialists to invent policy or implementation scope.
```

## Exa Source Links
Exa MCP was attempted before writing this reference, but search and fetch returned a 402 credit-limit error. Links below are fallback calibration sources, not repo-authoritative workflow rules.

- GOV.UK Service Manual, defining the problem and constraints before building: https://www.gov.uk/service-manual/agile-delivery/how-the-discovery-phase-works
- Product Talk, outcome/opportunity scope and assumption testing: https://www.producttalk.org/opportunity-solution-trees/
- Atlassian Team Playbook, pre-mortem prompts that expose missing resources and risks: https://www.atlassian.com/team-playbook/plays/pre-mortem
- NASA Systems Engineering Handbook appendix, scope, boundary, and requirement validation cues: https://www.nasa.gov/reference/system-engineering-handbook-appendix/
