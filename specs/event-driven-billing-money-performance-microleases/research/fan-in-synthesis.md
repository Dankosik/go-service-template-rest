# Research Fan-In Synthesis

Status: research synthesis
Date: 2026-06-01
Scope: high-performance prepaid billing admission based on microleases.

This synthesis records evidence and recommended candidates for the next
specification phase. It is not `spec.md` and does not approve an architecture.

## Core Synthesis

Research consensus:

- The strongest known pattern match is escrow or bounded-counter rights
  allocation: a durable authority partitions a bounded quantity, and local
  actors operate only within their assigned rights.
- For GonkaGate, the durable authority is billing-service Postgres unless a
  future spec explicitly accepts a bounded platform write-off risk.
- A microlease can be production-sound only if its spendable tokens are backed
  by billing-reserved USD exposure or by an explicit bounded risk budget.
- In-memory and Redis token buckets are performance mechanisms, not customer
  money sources of truth.
- Kafka/Redpanda, Stripe Meter Events, Metronome, OpenMeter, ClickHouse, and
  outbox/inbox patterns support terminal usage ingestion, replay, aggregation,
  and projections. They do not authorize first paid execution for prepaid money.
- Durable proxy checkpoint/close batches are useful to prove and release unused
  lease capacity, but they cannot hide missing child lineage if every charge
  must be explainable.

## Subagent Fan-In

| Lane | Key contribution | Orchestrator reconciliation |
| --- | --- | --- |
| performance-agent | Escrow/microlease is the best performance/correctness match; Redis and local buckets are fast but non-authoritative; async metering proves ingestion, not admission. | Carry as support for a microlease candidate and performance proof obligations. |
| distributed-agent | Microleases map to escrow/bounded-counter rights; coordination-free spend is safe only after rights are allocated; leases need TTL plus fencing; Redpanda/Kafka EOS is scoped. | Carry as the core consistency model for specification. |
| domain-agent | Business controls must cap active exposure, crash write-off, low-balance strict mode, terminal lag, abuse mode, and visible balance behavior. | Carry into spec as invariants and proof obligations, not optional future hardening. |
| data-agent | Authoritative state belongs in Postgres; Redis/in-memory/Redpanda/ClickHouse/proxy rows are cache, limiter, proof, or projection unless explicitly risk-accepted. | Carry into source-of-truth and rejected-alternative sections. |
| reliability-agent | Closed before report after timeout. | No blocker. Reliability controls were covered locally and by other lanes; note this as an evidence limit. |

## Candidate Direction For Specification

Recommended primary candidate to specify:

- Billing-issued microleases as small escrowed USD rights.
- Billing reserves each microlease budget in Postgres before spend.
- Proxy consumes the microlease through one of two explicitly classified local
  mechanisms:
  1. durable child debit before external execution, with memory bucket as a
     cache; or
  2. memory/Redis pre-consumption with a tiny product-approved write-off cap,
     followed by durable terminal/checkpoint reconciliation.
- Async terminal usage events settle or release capacity through durable
  outbox/inbox and idempotent business identities.
- Strict or fail-closed mode applies to low-balance, high-risk, stale-pricing,
  stale-fence, backlog, Redis uncertainty, abuse, and manual-review cases.

The next specification must decide whether option 2 is acceptable. If not, the
current durable child-debit-per-request lease architecture remains the safest
performance design.

## Recommended Architecture Comparison Set

The specification should compare:

1. Strict durable reserve before every request.
2. Existing billing-issued lease with durable proxy child debit before every
   paid request.
3. Billing-issued microlease with in-memory per-instance token bucket over
   reserved capacity.
4. Billing-issued microlease with Redis shared token bucket over reserved
   capacity.
5. Durable proxy checkpoint batches as release/settlement optimization.
6. Pure async metering.
7. Redis/global counter as money gate.
8. Hybrid strict/fast modes.

## Provisional Classifications

Strong candidates:

- Escrow/bounded-counter microleases with billing Postgres reserve authority.
- Durable proxy child debit and terminal obligation before external execution.
- Durable proxy checkpoint/close batches for release and reconciliation proof.
- Async terminal facts with outbox/inbox and idempotent consumer semantics.

Conditional candidates:

- In-memory token bucket if it is only cache over durable authority, or if the
  spec accepts a tiny bounded platform write-off window.
- Redis shared bucket if it is only limiter/cache/projection over billing
  reserved capacity.
- Hybrid strict/fast mode if exact eligibility and default fail-closed behavior
  can be specified.

Rejected unless explicitly risk-accepted:

- Pure async metering as prepaid admission.
- Redis/global counter as customer-money authority.
- Process-local memory as unbounded spend authority.
- Silent release of expired capacity without proof.
- Charging above child or lease authority.

## Specification Questions To Decide

Must decide now:

- Is any memory-only or Redis-only spend window acceptable as platform exposure,
  or must every request have durable child debit lineage before external
  execution?
- What is the source of truth for spend authority: billing Postgres only, or
  billing Postgres plus an explicitly bounded local/Redis exposure budget?
- What are the semantics of "strict fallback": direct reserve, smaller durable
  microlease, no cache, or fail-closed?
- What exact conditions make a request/account/allocator ineligible for fast
  mode?
- What must be visible in available balance while microlease capacity is active?

Can defer to technical design after specification:

- Exact route names, event schemas, table names, package layout, and worker
  topology.
- Exact benchmark harness shape.
- Exact Redis client/library choice if Redis remains in scope.

Requires later proof before implementation readiness:

- Pricing-service USD-compatible immutable snapshot evidence.
- Benchmarks for active admission, cold refill, Redis path, terminal lag, and
  checkpoint cadence.
- Proxy durable allocator crash/restart proof.
- Privacy proof for logs/events/reconciliation rows.
- Product-approved write-off budget if memory/Redis leakage is accepted.

## Evidence Limits

- No live production traffic, DB rows, or deployment logs were used.
- No benchmark was run.
- No business-approved write-off amount is known.
- Reliability subagent did not return a standalone report before timeout; local
  research and other lanes still covered the main reliability surfaces.
- Existing `event-driven-billing-money-architecture` artifacts are an approved
  baseline, but this new track must not silently edit or supersede them without
  specification decisions.

## Handoff

Research is complete enough to start specification.

Next phase:

- Write `spec.md` in this task directory.
- Reconcile the research into explicit decisions, rejected options, invariants,
  proof obligations, and reopen conditions.
- Run or route formal clarification/challenge appropriate for full-orchestrated
  protected-money work.

Stop before technical design, planning, tasks, implementation, or validation.
