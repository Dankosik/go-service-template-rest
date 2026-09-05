# Messaging Failure Proof

Load when a multi-boundary guarantee needs executable failure evidence.

For each publish or consume boundary name the semantic guarantee, stable
identity/scope, first durable commit, ambiguous response, recovery owner,
business observable, and falsifier. Evidence from one endpoint, queue, or
consumer never supplies a sibling guarantee.

Inject failure on both sides of every in-scope durable commit:

- business state before/after outbox intent;
- broker acceptance before/after publisher completion or lost response;
- business effect before/after inbox/idempotency commit;
- effect commit before/after acknowledgement/delete/offset;
- claim/visibility/lease expiry and ownership transfer;
- retry exhaustion, poison quarantine, and bounded redrive when present;
- mixed versions, auth denial, tenant crossing, and replay when those risks are
  part of the claim.

Pin broker/client versions, topology/durability, workload, concurrency, and
fault mechanism. Assert authority state, logical identity, broker progress,
attempts, quarantine, and recovery signals. The falsifier must fail when the
claimed commit, identity, or fence is removed.

A lost response has one durable disposition; strict order does not promise
progress past a poison predecessor; replay/redrive is not ordinary redelivery;
and a quarantine design is incomplete until a canary preserves identity,
suppresses already-applied effects, applies only intended repaired work, and
stops on the declared invariant.
