# Idempotency, Replay, And Async Domain Rules

## Decide
- Define sameness before mechanism: which fields carry business intent, and which are incidental metadata. Caller key, business key, event ID, aggregate version, and tenant scope are candidates; a transport key or a payload hash is one only if the domain says so.
- Same identity plus **different** intent is a conflict, not a replay. That case is the one an idempotency key alone answers wrongly, because the key matches and the meaning does not.
- Name the effect boundary: which effects may be observed before acceptance commits, and which must not exist until it does. An effect that is externally visible or irreversible belongs after the last guard.
- Out-of-order arrival is a decision, not a default: no-op, reject, compensate, forward-recover, or reconcile. Stale-overwrite is only acceptable where a rule says so.
- An unknown commit outcome is its own case, distinct from success and failure. Say what the next arrival of the same operation does while it is unresolved.
- This repository's outbox is at-least-once and says so (`docs/postgres-transactional-outbox.md`): a lease that expires after broker acknowledgement republishes the same ID and bytes. It orders and retries delivery; it does not make a business effect once-only. The rule must name what does — an existing state guard, a processed-event key, or a naturally idempotent effect.
- Say whether reconciliation issues corrective domain actions or only repairs a derived surface. `go-distributed` owns the durable coordination mechanism; this file owns what "the same operation" means.

## Reject
```text
The consumer is idempotent because it checks whether the message ID was seen.
```
Failure: message identity is not business identity. The same logical operation can arrive under another message, another retry path, or another producer, and the effect repeats.

```text
Replay the events through the handlers.
```
Failure: replay is not a rerun until each side effect is classified recomputable, sandbox-only, forbidden, or policy-controlled.

## Prove
Proof should cover same identity with same intent, same identity with different intent, a concurrent or in-progress duplicate, an unknown commit, an out-of-order arrival, and whichever reconciliation path the rules chose.
