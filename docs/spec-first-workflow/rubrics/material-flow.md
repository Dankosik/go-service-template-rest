# Material Flow

Load when System / Integration Design must close a runtime or durable flow.

Trace each actor or trigger to caller-visible completion or durable finality.
At every material crossing record only what Implementation must not invent:

- initiating and receiving owners, transferred responsibility, contract, and
  produced state/effect;
- canonical representation, transformation, source of truth, identifiers,
  units, absence, consistency, visibility, freshness, ordering, and finality;
- failure point, cancellation/timeout, possibly committed effect, retry and
  exhaustion owner, idempotency/replay/reconciliation, degradation, and
  restoration boundary;
- when scale-sensitive, workload envelope, critical path, amplification,
  serialization/parallelism/queueing, structural ceiling, and proving boundary.

Show current and target flows only when their difference changes a decision.
Use a diagram only when compact text cannot make ordering, fan-out,
transformation, or recovery reviewable; normative text and canonical contracts
remain authoritative.
