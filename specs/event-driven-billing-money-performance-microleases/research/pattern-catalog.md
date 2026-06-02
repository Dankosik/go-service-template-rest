# Pattern Catalog

Status: research evidence
Date: 2026-06-01

This catalog names patterns the next specification should use or reject. It is
not an approved architecture.

## Escrow Transaction

Summary:

- A durable authority reserves part of an aggregate quantity and allows a
  transaction or actor to consume within that escrowed quantity.
- The invariant is preserved because the authority subtracts the escrowed
  amount from the globally available pool before local consumption begins.

Fit for GonkaGate:

- Strong fit.
- A billing-issued microlease is an escrowed USD spend right.
- Billing Postgres must reserve the lease budget before proxy can use it.

Limits:

- Unused capacity must be released only with proof or reconciliation.
- Larger escrow windows increase stale hold and write-off exposure.

## Bounded Counter / Rights Partitioning

Summary:

- A global numeric invariant is protected by partitioning rights across actors.
- Local increments/decrements are coordination-free only while local rights are
  sufficient.

Fit for GonkaGate:

- Strong conceptual fit for per-instance or per-shard microleases.
- Each proxy allocator gets a bounded right to spend a specific USD atom amount.

Limits:

- Rebalancing or replenishment still coordinates with the durable authority.
- Rights must be fenced and expire or close under controlled rules.

## Voucher / Spending Lease / Microlease

Summary:

- A short-lived, bounded, account-scoped spend grant.
- It can be consumed locally for fast admission and settled asynchronously.

Fit for GonkaGate:

- Strong candidate for next specification.
- Must include account scope, owner, generation/fence, amount cap, expiry,
  debit cutoff, pricing constraints, idempotency, stored outcome, child lineage,
  terminal deadlines, and reconciliation.

Limits:

- If local consumption is memory-only, crash exposure must be explicitly
  accepted and capped as platform write-off risk.
- If local consumption is durable per child debit, performance depends on the
  proxy local store rather than billing Postgres.

## Token Bucket

Summary:

- Tokens refill at a configured rate up to a burst cap; admission consumes
  tokens.
- Can be local in memory or shared through a datastore such as Redis.

Fit for GonkaGate:

- Good for smoothing request bursts and enforcing the consumption rate inside a
  bounded lease.
- Useful as an implementation model for spending a microlease.

Limits:

- A token bucket is a limiter algorithm, not money authority.
- The token supply must be backed by billing-reserved USD or by an explicitly
  accepted write-off budget.

## Local Rate Limiter

Summary:

- Each process or instance enforces its own local bucket without a shared
  decision on every request.

Fit for GonkaGate:

- Useful for low-latency abuse control and for a per-instance microlease if the
  lease is issued to that instance/allocator.

Limits:

- Local buckets multiply across replicas.
- They are unsafe for account-wide prepaid money unless each replica holds a
  separately reserved lease or shares durable allocator state.

## Global Rate Limiter / Redis Shared Bucket

Summary:

- A shared service or datastore enforces a global decision across instances.
- Redis/Lua can make the read-decide-update step atomic at low latency.

Fit for GonkaGate:

- Useful as an advisory shared bucket, abuse limiter, or backpressure surface
  over already-reserved capacity.

Limits:

- Redis persistence and replication do not provide the same correctness boundary
  as billing Postgres.
- Redis must not be the visible balance or prepaid money source of truth unless
  the future spec explicitly accepts and caps loss/failover write-off.

## Sharded Counter

Summary:

- Writes spread across shards to raise throughput; reads aggregate shards or use
  a rollup.

Fit for GonkaGate:

- Useful for usage analytics, dashboards, and coarse admission telemetry.
- Could model real reserved partitions only if each shard is a durable
  billing-minted right.

Limits:

- Generic sharded counters create stale reads and expensive exact totals.
- They are poor exact prepaid admission gates.

## Async Usage Metering

Summary:

- Usage events are durably ingested, deduped, aggregated, and billed later.
- Systems such as Stripe, Metronome, and OpenMeter demonstrate high-throughput
  ingestion and idempotency patterns.

Fit for GonkaGate:

- Strong fit for terminal usage settlement, analytics, and invoice/projection
  flows after prior authority.

Limits:

- Pure async usage events do not prevent prepaid overspend before external cost.
- Prepaid admission still needs a reserve, microlease, voucher, or explicit
  write-off model before execution.

## Outbox / Inbox / Idempotent Consumer

Summary:

- A service writes outbox rows in the same transaction as local state changes.
- Consumers record processed message/business IDs before committing effects or
  offsets.

Fit for GonkaGate:

- Required for terminal facts, lease checkpoint/close, billing facts, and
  reconciliation events.

Limits:

- Outbox/inbox gives replay safety, not first paid-admission authority.

## Strict Durable Reserve

Summary:

- Every paid request synchronously asks billing to reserve funds before
  execution.

Fit for GonkaGate:

- Strong correctness baseline and strict fallback candidate for risky cases if
  the future spec explicitly allows branchy strict/fast behavior.

Limits:

- Highest hot-path latency and account-row contention.
- Existing approved architecture rejected it as the uniform target for tiny
  paid requests.

## Pure Redis Or Global Counter Money Gate

Summary:

- Redis or another global counter answers whether money is available and
  decrements spend on the request path.

Fit for GonkaGate:

- Poor fit as authority.
- Useful only as limiter/cache/projection over another durable source.

Limits:

- Data loss, failover, split-brain, stale reads, and replay gaps become money
  errors unless separately bounded and reconciled.

## Hybrid Strict / Fast Modes

Summary:

- Fast path uses microleases for low-risk/high-volume requests.
- Strict path uses direct reserve or smaller/no microleases for low-balance,
  high-cost, stale, or abuse-risk cases.

Fit for GonkaGate:

- Worth comparing in specification because the prompt explicitly asks about
  strict fallback only for risky cases.

Limits:

- Branching increases product, implementation, observability, and rollout
  complexity.
- The future spec must avoid an unbounded "future hardening" split and must
  name exact eligibility, caps, and proof.
