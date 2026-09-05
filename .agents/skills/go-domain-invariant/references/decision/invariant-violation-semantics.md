# Invariant Violation Semantics

## Decide
- Exactly one primary outcome per violation: `reject`, `deny`, `defer_async`, `compensate`, `forward_recover`, `manual_intervention`, or `accepted_risk`.
- `reject` when the command is not accepted and the caller can correct it. `deny` for identity, tenant, ownership, and authorization failures — fail closed rather than returning a filtered or empty result, because an empty list is indistinguishable from a legitimate empty scope at every layer above.
- `defer_async` only when the accepted state is honestly pending. `compensate` and `forward_recover` only after naming the durable state and side effects that already happened. `accepted_risk` only with its consequence and its reopen trigger.
- A dependency timeout after a possible commit is an ambiguous outcome, not a business rejection: model pending, reconciliation, or forward recovery.
- Replay of an already-accepted operation returns the equivalent accepted outcome; that is a satisfied invariant, not a fresh success after a failed check.
- State the outcome in domain terms. `internal/problem` owns the client-visible status and body, and `go-api-contract` owns which outcome maps to which response.

## Reject
```text
If ownership cannot be verified, return an empty list.
```
Failure: an authorization or tenant violation is reported as success. The caller, the audit trail, and every downstream consumer read it as "there is nothing", which is the reading that survives into a support ticket.

```text
Log a warning and continue.
```
Failure: the operation now succeeds with the invariant false. No caller can distinguish that from a clean accept.

## Prove
Proof asserts the chosen outcome directly: the forbidden transition rejects, the tenant mismatch denies, the ambiguous commit lands in pending or reconciliation, the duplicate returns the equivalent outcome, and a failed compensation reaches the stated terminal or manual state.
