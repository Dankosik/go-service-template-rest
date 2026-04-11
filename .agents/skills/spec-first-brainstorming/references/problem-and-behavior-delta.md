# Problem And Behavior Delta

## When To Load
Load this when the request needs a sharper problem statement, actor, current behavior, desired behavior, or smallest behavior delta before specialist design can begin.

## Calibration Guardrail
Use these examples to shape engineering framing only. Do not choose architecture, API shape, storage model, security policy, reliability strategy, rollout plan, or implementation tasks.

## Raw Request Examples
Example A:

```text
We should add idempotency keys to POST /orders because retries sometimes create duplicate orders.
```

Example B:

```text
Make the health endpoint more useful for deploys. It should tell us if dependencies are okay.
```

Example C:

```text
Users keep asking why their export is stuck. Add better export status.
```

## Final Framing Output
Example A:

```markdown
Problem
Clients that retry order creation after a timeout can create more than one order for the same user intent, leaving operators and customers with duplicate state to reconcile.

Behavior Delta
- Current behavior: A retry of the same order-create intent is treated like a new create request.
- Desired behavior: The service can recognize repeat submissions of the same create intent and return a stable outcome instead of creating a duplicate.
- Affected actor: API clients and downstream order operators.
- Boundary: This frame covers duplicate create-intent semantics only. It does not choose the idempotency-key header contract, storage design, transaction strategy, or retry policy.
```

Example B:

```markdown
Problem
Deployment automation and operators cannot tell whether the service is ready to receive traffic or merely process-alive, so deploy gates may advance with hidden dependency failure.

Behavior Delta
- Current behavior: Health reporting is too coarse to distinguish process liveness from traffic readiness.
- Desired behavior: The service exposes enough readiness signal for deploy automation to avoid routing traffic to an instance that cannot serve its expected dependencies.
- Affected actor: deploy automation and on-call operators.
- Boundary: This frame names the readiness problem only. It does not decide probe routes, dependency checks, failure thresholds, caching, or Kubernetes configuration.
```

Example C:

```markdown
Problem
Users cannot distinguish an export that is still progressing from one that is blocked or failed, so they retry or ask support before the system has a clear next action.

Behavior Delta
- Current behavior: Export progress is opaque after the request is accepted.
- Desired behavior: Users and support can see a clear export lifecycle state and know whether to wait, retry, or investigate.
- Affected actor: export requesters and support operators.
- Boundary: This frame covers status transparency. It does not decide event storage, polling vs push, background worker design, or UI layout.
```

## Pass/Fail Readiness Examples
Pass:

```markdown
Readiness Decision
pass - The problem, affected actor, current behavior, desired behavior, and design boundary are explicit. Downstream design can now decide contract, data, reliability, and validation details without reopening whether the work is about duplicate create-intent semantics.
```

Fail:

```markdown
Readiness Decision
fail - "Add idempotency keys" is still a mechanism, not a problem frame. The output does not state which operation duplicates, who is affected, whether repeat submissions should return the original outcome or reject, or what is out of scope.
```

## Exa Source Links
Exa MCP was attempted before writing this reference, but search and fetch returned a 402 credit-limit error. Links below are fallback calibration sources, not repo-authoritative workflow rules.

- GOV.UK Service Manual, discovery and problem framing: https://www.gov.uk/service-manual/agile-delivery/how-the-discovery-phase-works
- Product Talk, opportunity and assumption-test framing: https://www.producttalk.org/opportunity-solution-trees/
- Atlassian Team Playbook, pre-mortem prompts for risk discovery: https://www.atlassian.com/team-playbook/plays/pre-mortem
- NASA Systems Engineering Handbook appendix, requirements quality and validation language: https://www.nasa.gov/reference/system-engineering-handbook-appendix/
