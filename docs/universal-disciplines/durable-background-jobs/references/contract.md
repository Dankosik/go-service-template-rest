# Durable Job Contract

Load when durable acceptance, identity, lease, transition, effect, or recovery
can change the decision.

Keep work synchronous when the caller needs the result and it reliably fits the
request budget. Otherwise name the caller-visible acceptance point: completed,
durably accepted with a job ID, or rejected before acceptance.

Keep distinct identities for logical job, producer deduplication, attempt plus
lease generation, schedule occurrence, and business effect. Commit business
state with the job row, or with an outbox intent when a broker crosses the
boundary. An ambiguous enqueue retries only with the same producer identity and
durable readback.

A lease defines owner, attempt/generation, acquisition, expiry, renewal, time
source, maximum runtime, and conditional completion/failure. Stale owners cannot
mutate current state. Commit the durable effect and result before completion or
acknowledgement; `effect committed, completion unknown` reconciles before
replay. Prefer one atomic effect/idempotency transaction, then unique effect
ledger/conditional write, provider idempotency, and finally reconciliation.

Classify transient, permanent, poison, and operator-actionable failure into
bounded retry, terminal, quarantine, or authorized recovery. Crash proof covers
acceptance, claim, effect-before-complete, lost completion response, expiry, and
stale attempt at the real engine seam.
