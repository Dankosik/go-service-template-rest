# Invariant Violation Semantics

## When To Load
Load this when a rule says what must be true but not what happens when it is false. This is especially important for authorization, tenant isolation, invalid transitions, dependency ambiguity, process failures, and cross-service inconsistencies.

Use repo artifacts first for behavior. In this repo, examples include read-only subagent policy, implementation-readiness blocking rules, source-of-truth path rules, and validation-before-completion rules.

## Violation Outcome Catalog
Use explicit outcomes:

- `reject`: do not accept the command/event; caller may correct and retry.
- `deny`: fail closed because identity, tenant, ownership, or authorization does not allow the action.
- `defer_async`: accept only an honest async or pending state, not fake immediate success.
- `compensate`: apply a modeled corrective action after a prior allowed step.
- `forward_recover`: continue from a durable intermediate state without replaying harmful side effects.
- `manual_intervention`: route to a human or operational queue because automation is unsafe or not worth the complexity.
- `accepted_risk`: document the consequence and reopen trigger; do not pretend the invariant is enforced.

## Example Invariant Statements
- `SubagentMutationViolation`: if a read-only subagent mutates repository files or git state, the mutation is not accepted as authoritative and the orchestrator must reconcile or discard it.
- `ReadinessFailViolation`: if implementation readiness is `FAIL`, starting implementation is a workflow violation; the only legal outcome is routing to the named earlier phase.
- `ProjectionTruthViolation`: if a read model or mirror differs from the source-of-truth artifact, the source of truth wins and the derived surface must be repaired before closeout.
- `TenantOwnershipViolation`: if an actor cannot prove tenant or object ownership, the domain outcome is deny/fail-closed, not best-effort filtering.
- `AmbiguousCommitViolation`: if a dependency timeout leaves commit status unknown, the domain must expose an ambiguous, pending, reconciliation, or forward-recovery path rather than report business success.

## Good And Bad State Transition Specs
Good invalid-transition spec:

```text
State: implementation
Trigger: orchestrator discovers a required planning artifact is missing
Invalid transition attempted: implementation -> done
Reason: proof path and task ledger are not stable
Violation outcome: reject closeout; reopen planning in the next eligible session
User-facing meaning: no completion claim is made
Downstream handoff: planning must create or waive the missing artifact before coding resumes
```

Bad transition violation spec:

```text
If something goes wrong, handle the error and continue.
```

Why it fails: it allows false success and hides whether the correct response is reject, compensate, recover, or route to a human.

Good violation table:

| Violation | Outcome | Why |
| --- | --- | --- |
| Same idempotency key, different domain intent | reject or conflict | It is not a replay of the same logical operation. |
| Duplicate async event already applied | no-op or replay equivalent outcome | The invariant is one domain effect per logical event. |
| Source-of-truth mirror drift | forward-repair derived surface | The canonical artifact remains authority. |
| Authorization or tenant mismatch | deny | Correctness and data isolation require fail-closed behavior. |
| Cross-service correction cannot be guaranteed online | compensate or manual intervention | The invariant is process-level, not local hard consistency. |

## Edge-Case Prompts
- Which violations are allowed to return success, and why? Usually the answer should be "none" unless success means idempotent replay of the same already-accepted operation.
- If compensation fails, is the system in a terminal failed state, a retryable state, or a manual-intervention state?
- Is this a user correction problem or a system reconciliation problem?
- Can a stale projection cause a command to be accepted or rejected incorrectly?
- What should happen when cancellation arrives after the side effect may have committed?
- Does the same outcome apply to user-driven commands and automated policy commands?
- Is the violation externally visible, internally audited, or both?

## Downstream Handoff Notes
- API handoff: after choosing violation outcome, encode stable external error/async semantics without leaking internal implementation details.
- Data handoff: DB constraints and transaction boundaries should enforce only the domain decisions they can actually protect.
- Distributed handoff: process-level violations need outbox/inbox, idempotency, compensation, and reconciliation stance after the business outcome is chosen.
- Reliability handoff: dependency failure modes must not masquerade as business success.
- QA handoff: invalid-transition, duplicate, timeout, and authorization tests should assert the chosen violation outcome directly.

## Exa Source Links
- [Domain-Driven Design Reference](https://www.domainlanguage.com/wp-content/uploads/2016/05/DDD_Reference_2015-03.pdf) for aggregate responsibilities and assertions/postconditions.
- [Event Store: Counterexamples Regarding Consistency](https://eventstore.com/blog/counterexamples-regarding-consistency-in-event-sourced-solutions-part-3/) for projection truth, policy/process-manager, and idempotent checkpoint examples.
- [Spec Coding: Edge Case Checklist](https://spec-coding.dev/guides/edge-case-checklist) for invalid transition and dependency-failure prompts.
- [Cosmic Python: Aggregates and Consistency Boundaries](http://www.cosmicpython.com/book/chapter_07_aggregate.html) for rejecting changes that would break invariants.
