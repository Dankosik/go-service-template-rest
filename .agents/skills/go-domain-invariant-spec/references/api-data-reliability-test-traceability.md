# API, Data, Reliability, And Test Traceability

## When To Load
Load this after the domain rules are explicit and the spec needs downstream handoff into API contract, data ownership, distributed consistency, reliability, security, observability, or tests.

Do not use this file to jump directly into transport, SQL, infrastructure, or implementation choices. It exists to preserve traceability from business decisions to downstream obligations.

## Traceability Pattern
Use a matrix after the invariant and violation semantics are stable:

| Domain rule | API handoff | Data handoff | Reliability handoff | Security handoff | Test proof | Reopen trigger |
| --- | --- | --- | --- | --- | --- | --- |
| Rule ID and statement | External behavior only | Source of truth and consistency expectation | Timeout, retry, recovery, reconciliation expectation | Identity/tenant/authorization expectation | Positive, negative, edge, replay proof | What would change the domain decision |

Keep handoff cells short. If a cell starts making detailed design decisions, route to the appropriate specialist spec skill after the domain decision is stable.

## Example Invariant Statements
- `TraceEveryCriticalInvariant`: every critical invariant must map to downstream API, data, reliability, security, and test obligations or explicitly mark the handoff as not applicable.
- `DomainBeforeTransport`: HTTP status, schema shape, DB constraint, and infrastructure choice are derived decisions; they must not replace the business rule they encode.
- `SourceOfTruthBeforeCache`: cache or projection behavior may accelerate reads but cannot become authority for invariant-sensitive decisions without a freshness contract.
- `AuthContextAsDomainRule`: tenant, actor, and object ownership constraints are domain invariants when they affect allowed behavior.
- `ProofMatchesClaimScope`: validation evidence must exercise the same domain rule and boundary that the completion claim names.

Example traceability row:

| Domain rule | API handoff | Data handoff | Reliability handoff | Security handoff | Test proof | Reopen trigger |
| --- | --- | --- | --- | --- | --- | --- |
| `SubagentReadOnly`: subagents are advisory and read-only. | User-facing orchestration docs must not promise delegated write authority. | No code/data mutation is accepted from delegated research lanes. | Review fan-in cannot claim coverage for interrupted or abandoned lanes. | N/A unless delegated lane has access to sensitive data. | Review transcript or workflow artifact shows lane scope and read-only result. | A future tool surface can no longer reliably enforce read-only behavior. |

## Good And Bad State Transition Specs
Good downstream transition spec:

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

Bad transition spec:

```text
Add an endpoint, a table, retries, and tests.
```

Why it fails: it skips the domain rule and makes downstream artifacts compete as implicit sources of truth.

## Edge-Case Prompts
- Which downstream surfaces need to know this invariant, and which should stay unaware?
- Does the API need to expose a domain conflict, or is the violation internal only?
- Is there a source-of-truth entity or artifact, and can any derived view be stale?
- What retry, timeout, or cancellation can make the domain outcome ambiguous?
- What tenant or actor boundary changes the allowed behavior?
- Which test proves a rejected transition, not just the happy path?
- What validation would catch a mirror, projection, cache, generated-code, or documentation drift?

## Downstream Handoff Notes
- API: state the external contract consequence after the domain outcome is stable; keep method/status/schema details in API design.
- Data: state ownership, consistency, constraint, and migration compatibility needs; keep physical schema details in data design.
- Reliability: state retry budget, deadline, degradation, forward recovery, and reconciliation expectations; keep tooling and deployment mechanics out of the domain spec.
- Security: state fail-closed identity, tenant, and ownership semantics; hand off detailed threat controls separately.
- QA: map each invariant to positive, negative, duplicate/replay, invalid-transition, timeout, and compatibility tests as applicable.
- Observability: trace domain-significant transitions and violation outcomes with stable vocabulary; keep low-cardinality metric design for observability work.

## Exa Source Links
- [Domain-Driven Design Reference](https://www.domainlanguage.com/wp-content/uploads/2016/05/DDD_Reference_2015-03.pdf) for bounded contexts, model language, and aggregate authority.
- [Cosmic Python: Aggregates and Consistency Boundaries](http://www.cosmicpython.com/book/chapter_07_aggregate.html) for domain tests and aggregate consistency boundaries.
- [EventSourcingDB: Building Event Handlers](https://docs.eventsourcingdb.io/best-practices/building-event-handlers/) for handler responsibility, replay, and idempotency handoff.
- [Spec Coding: Edge Case Checklist](https://spec-coding.dev/guides/edge-case-checklist) for mapping edge cases into testable expectations.
