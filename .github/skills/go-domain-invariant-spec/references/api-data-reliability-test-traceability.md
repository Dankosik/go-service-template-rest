# API, Data, Reliability, And Test Traceability

## Behavior Change Thesis
When loaded for symptom "stable domain rules need downstream handoff", this file makes the model preserve traceability from invariant IDs to API, data, reliability, security, observability, and test obligations instead of likely mistake "jump into endpoint, schema, retry, or test details that become competing sources of truth."

## When To Load
Load this after domain rules, violation outcomes, and duplicate/replay behavior are explicit, and the spec needs downstream handoff into API contract, data ownership, distributed consistency, reliability, security, observability, or tests.

## Decision Rubric
- Use this only after the domain decision is stable. If owner, source of truth, pass/fail signal, or violation outcome is missing, reopen the domain spec first.
- Keep handoff cells short and consequence-focused. If a cell starts making detailed design decisions, route to the appropriate specialist spec skill.
- State only what downstream surfaces must preserve: external behavior, source-of-truth authority, consistency expectation, retry/recovery expectation, identity/tenant rule, and proof obligation.
- Explicitly mark a handoff as not applicable when no downstream surface needs to know the rule.
- Add a reopen trigger for facts that would change the domain decision, not every implementation concern.

## Imitate
| Domain rule | API handoff | Data handoff | Reliability handoff | Security handoff | Test proof | Reopen trigger |
| --- | --- | --- | --- | --- | --- | --- |
| `SubagentReadOnly`: subagents are advisory and read-only. | User-facing orchestration docs must not promise delegated write authority. | No code/data mutation is accepted from delegated research lanes. | Review fan-in cannot claim coverage for interrupted or abandoned lanes. | N/A unless delegated lane has access to sensitive data. | Review transcript or workflow artifact shows lane scope and read-only result. | A future tool surface can no longer reliably enforce read-only behavior. |

Copy the shape: start from the domain rule and make each downstream cell a consequence, not a design takeover.

```text
Domain state: invariant_decided
Trigger: spec needs API/data/reliability/test handoff
Preconditions:
- invariant has owner, source of truth, pass/fail signal, and violation outcome
- duplicate/replay behavior is stated if retries or async work are possible
Allowed transitions:
- invariant_decided -> api_contract_design when external behavior changes
- invariant_decided -> data_design when source-of-truth or persistence consistency changes
- invariant_decided -> reliability_design when timeout/retry/reconciliation semantics change
- invariant_decided -> qa_strategy when proof obligations need expansion
Forbidden transition:
- invariant_ambiguous -> implementation_mechanics
Violation outcome: reopen domain specification before downstream design
```

Copy the boundary: handoff happens after the domain invariant is decided, not before.

## Reject
```text
Add an endpoint, a table, retries, and tests.
```

Failure: skips the domain rule and lets downstream artifacts compete as implicit sources of truth.

```text
API: return 409. Data: add a unique index. Reliability: retry three times.
```

Failure: details may be correct later, but the handoff is missing the domain conflict, source-of-truth boundary, retry-safe outcome, and proof claim.

## Agent Traps
- Do not use this reference as primary design guidance; it is a traceability and handoff rubric after domain rules are stable.
- Do not smuggle API status codes, physical schema, retry budgets, or observability label design into the domain spec unless the user explicitly asked for that downstream spec.
- Do not treat cache, mirrors, projections, or generated code as authority unless the domain rule includes a freshness or source-of-truth contract.
- Do not hand off security only as "auth required"; name the tenant, actor, or object-ownership behavior that changes allowed actions.
- Do not write test proof as "add tests"; name the positive, negative, edge, duplicate/replay, invalid-transition, or timeout proof that matches the rule.

## Validation Shape
The traceability matrix is ready when every critical invariant row has either a concise downstream consequence or an explicit `N/A`, plus a proof shape and a reopen trigger tied to the domain decision.
