# Challenge Recommendation Examples

## When To Load
Load this when deciding whether the pre-spec challenge pass is `required`, `recommended`, or `skippable`, or when choosing the seams a challenger should pressure-test.

## Calibration Guardrail
The recommendation routes risk; it does not decide design. Do not use this file to answer architecture, API, data, security, reliability, rollout, or implementation-plan questions.

## Raw Request Examples
Example A:

```text
Make password reset tokens single-use and expire faster.
```

Example B:

```text
Add a query parameter to filter projects by archived=true.
```

Example C:

```text
Queue invoice finalization so checkout responds faster.
```

## Final Framing Output
Example A:

```markdown
Challenge Recommendation
required

Why
Password reset touches identity and account recovery. Hidden assumptions about token reuse, existing sessions, replay windows, audit needs, and user lockout could materially change the spec.

Seams To Pressure-Test
- Security semantics: What abuse case is being prevented, and what recovery behavior must remain usable?
- Domain semantics: What happens to already-issued tokens when expiration or single-use behavior changes?
- Observability and support: What signals are needed without leaking token or account details?
```

Example B:

```markdown
Challenge Recommendation
skippable

Why
The request is local and low risk if research confirms `archived` is already an existing project state and the filter does not change authorization, pagination, or default visibility semantics.

Seams To Pressure-Test If Skipping Is Reconsidered
- Whether archived projects have distinct authorization or tenant visibility behavior.
- Whether adding the filter changes default result ordering or pagination expectations.
```

Example C:

```markdown
Challenge Recommendation
required

Why
Invoice finalization likely touches money, external side effects, consistency, retries, and user-visible checkout state. Queueing could change correctness even if it improves latency.

Seams To Pressure-Test
- Domain invariant: When is an invoice considered final from the customer's perspective?
- Distributed consistency: What side effects must be atomic, idempotent, or recoverable?
- Reliability: What happens if queued finalization fails after checkout responds?
```

## Pass/Fail Readiness Examples
Pass:

```markdown
Readiness Decision
pass - The challenge recommendation is tied to concrete risk seams: identity, issued-token migration, and support-safe observability. The frame is ready for a pre-spec challenge because the challenger has specific uncertainty to attack.
```

Fail:

```markdown
Readiness Decision
fail - The output says "challenge recommended because challenges are good" and names no seam. It cannot guide a useful pre-spec challenge and may create ritual review instead of risk reduction.
```

## Exa Source Links
Exa MCP was attempted before writing this reference, but search and fetch returned a 402 credit-limit error. Links below are fallback calibration sources, not repo-authoritative workflow rules.

- Atlassian Team Playbook, pre-mortem prompts for project failure causes and missing inputs: https://www.atlassian.com/team-playbook/plays/pre-mortem
- GOV.UK Service Manual, deciding whether discovery should continue based on evidence and constraints: https://www.gov.uk/service-manual/agile-delivery/how-the-discovery-phase-works
- Product Talk, testing riskiest assumptions before deciding what to build: https://www.producttalk.org/opportunity-solution-trees/
- NASA Systems Engineering Handbook appendix, validation questions and requirement verifiability: https://www.nasa.gov/reference/system-engineering-handbook-appendix/
