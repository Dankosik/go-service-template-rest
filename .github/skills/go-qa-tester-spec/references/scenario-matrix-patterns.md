# Scenario Matrix Patterns

## Behavior Change Thesis
When loaded for symptom "the test plan is becoming happy-path-only or checklist-shaped", this file makes the model write discriminating scenario rows with observables instead of likely mistake "list generic cases without proof value."

## When To Load
Load this when approved behavior needs a compact scenario matrix, especially for fail, edge, abuse, retry, or concurrency rows.

## Decision Rubric
- Start from the changed behavior, not from a universal category list.
- Every row needs preconditions, data/input shape, selected proof level, expected observable, and pass/fail rule.
- Add a fail row for each meaningful rejection, rollback, denial, retry stop, or degraded-mode behavior.
- Add an edge row only when a named boundary value, terminal state, empty set, ordering tie, limit, or malformed shape changes the proof.
- Add an abuse row when caller identity, tenant, object ID, cursor, limit, key, or payload size is caller-controlled.
- Add retry/concurrency rows when duplicate suppression, conflict, ordering, cancellation, worker lifecycle, or shared state matters.
- Rows that cannot name an observable are not "future test ideas"; they are underspecified behavior to escalate.

## Imitate
| Requirement | Compact Rows | Observable To Copy |
| --- | --- | --- |
| Request creates one resource | valid payload; invalid field; unknown field if strict; duplicate idempotency key; oversized body if limit changed | status, response body, `Location` or resource state, no partial side effect |
| State transition | allowed transition; forbidden transition; stale version; concurrent transition attempt; terminal-state repeat | persisted state, conflict class, emitted event count |
| Cache-backed read | hit; miss; stale entry; corrupt entry; cache timeout; tenant key mismatch; parallel miss | returned value, origin call count, cache write/delete/bypass |
| Async processing | accepted; retryable failure; non-retryable failure; poison message; duplicate replay; restart replay | terminal state, retry count, DLQ/escalation signal, idempotent replay |

## Reject
- "Happy path, invalid input, edge case" with no named data shape. The model has not said what makes the edge interesting.
- "Expect an error" for a negative row. The observable must name the error class, status, persisted state, message state, or side-effect absence that proves the contract.
- "Concurrency test" without duplicate, conflict, ordering, or race-sensitive observable.
- "Security test" that uses only an authorized actor. Boundary proof needs the wrong actor, missing credential, wrong tenant, or caller-controlled identifier path when relevant.

## Agent Traps
- Do not add every row type to every requirement. A small matrix is better when it proves the risk honestly.
- Do not let examples invent behavior. If a status, terminal state, retry policy, or concealment policy is not approved, mark it as a blocker.
- Do not use "edge" as a bucket for cases the strategy cannot explain.
- Do not bury pass/fail rules in prose; make them visible enough for implementation to encode later.

## Validation Shape
The matrix is ready when each row can be converted into a deterministic test name without asking what input, expected state, or proof level was intended.
