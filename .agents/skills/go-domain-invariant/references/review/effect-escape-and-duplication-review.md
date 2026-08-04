# Effect Escape And Duplication Review

## Behavior Change Thesis
When loaded for symptom "an external or durable effect can happen before acceptance, or happen twice", this file makes the model prove the escaped or repeated effect instead of likely mistake "prescribe a saga, an outbox, or dedupe before any partial or duplicate outcome is shown."

## Decision Rubric
- One test, two directions: an effect the domain never accepted, or an effect the domain accepted once that can happen twice. Both are the same defect — the count of effects does not match the count of accepted operations.
- A finding names the rejected or replayed operation **and** the lasting effect together. One without the other is not yet this lane.
- Review the entity key, the current-state guard, the processed-event key, the version check, and the effect's own idempotency as one picture. Stale input that overwrites newer business state counts as a repeated effect on the state.
- Duplicate intent comes from an idempotency token, a message ID, a version, or a current-state guard. Identical payloads, timestamps, or field hashes are evidence of similarity, and treating them as identity both suppresses real duplicates and merges two genuine operations into one.
- Prefer moving the guard before the effect, making the effect conditional on accepted state, or using an idempotency token or state guard the service already establishes, before proposing new storage.
- An outbox is not the protection. This repository's is at-least-once by contract (`docs/postgres-transactional-outbox.md`): a lease expiring after broker acknowledgement republishes the same event, so a consumer whose business effect is not naturally idempotent still repeats it. Pair the relay with a state guard or processed-event key, or escalate.
- Logging, metrics, tracing, and an in-memory event not yet dispatched are not domain effects. They become one when they persist, dispatch, or change a business obligation before acceptance.
- Escalate when no local ordering makes state and effect safe together, or when compensation, transaction, or reconciliation policy must change — `go-distributed` and `go-reliability` own those.

## Reject
```text
Payments should use a saga or outbox.
```
Failure: architecture before evidence. No violated precondition, no partial outcome, and nothing the author can verify.

```text
This consumer is not idempotent. Add dedupe.
```
Failure: no transition and no repeated business effect named, so the fix cannot be sized and the risk cannot be ranked.

## Validation Shape
The finding is complete when it names the effect, the path that reaches it without acceptance or reaches it twice, and one focused assertion — the rejected path performs zero effects, or the replayed path performs one.
