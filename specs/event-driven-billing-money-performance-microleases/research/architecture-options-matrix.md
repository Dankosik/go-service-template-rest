# Architecture Options Matrix

Status: research evidence
Date: 2026-06-01

This matrix compares candidates for the next specification phase. It does not
approve an architecture.

## Summary Matrix

| Option | Admission authority | Hot-path cost | No-overspend posture | Main risk | Research classification |
| --- | --- | --- | --- | --- | --- |
| A. Strict durable reserve before every request | Billing Postgres reserve per request | Billing HTTP plus billing transaction per request | Strongest simple invariant | Latency and same-account lock contention | Viable baseline, likely too slow for target tiny paid requests |
| B. Existing billing-issued lease with durable child debit per request | Billing Postgres reserves lease; proxy durable store allocates child debit | Proxy local durable write per request; billing only for replenishment/settlement | Strong if durable child debit precedes execution | Proxy local store latency and durability complexity | Current approved baseline; strong candidate |
| C. Billing-issued microlease with in-memory per-instance token bucket | Billing reserves microlease; memory consumes local tokens | Fastest active-lease path | Strong only if memory tokens mirror durable child state or write-off risk is explicitly bounded | Crash/memory loss can leak capacity or terminal lineage | Candidate only with explicit authority classification and cap |
| D. Billing-issued microlease with Redis shared token bucket | Billing reserves microlease; Redis consumes shared tokens | Low network hop to Redis per request | Strong only if Redis is cache over Postgres/proxy durable state, not authority | Redis loss/failover/split brain can over/under admit | Candidate as limiter/cache, not money authority |
| E. Durable proxy checkpoint batches | Billing reserves lease; proxy records allocations/terminals and batches checkpoints/close | Can reduce billing release/settlement chatter; may or may not reduce per-request proxy writes | Strong only if each request still has durable lineage before execution | Batch totals can hide missing child lineage | Good supporting pattern, not first authority |
| F. Pure async metering | Terminal events/usage stream after execution | Fastest request path | Weak for prepaid unless prior authority exists | Overspend before billing sees usage | Reject for normal prepaid admission |
| G. Redis/global counter as money gate | Redis/counter answers spend availability | Low latency but external dependency per request | Weak unless backed by durable preallocation and loss budget | Redis becomes hidden money authority | Reject as authority; maybe advisory limiter |
| H. Hybrid strict/fast modes | Fast microlease for eligible cases; strict reserve or fail-closed for risky cases | Mixed | Can be strong if eligibility is explicit and strict path is durable | Branch complexity and inconsistent user behavior | Worth specification comparison |

## Option A. Strict Durable Reserve Before Every Request

Research finding:

- This is the cleanest correctness model: billing sees every request before
  external execution and can lock/update account state transactionally.
- It matches existing PRD language for reserve-before-usage.

Why it may not fit the new objective:

- It puts billing HTTP and a billing transaction on every low-value request.
- Same-account hot keys can become throughput bottlenecks.

Specification implication:

- Keep as baseline and possible strict fallback candidate.
- If the next spec uses it only for risky cases, it must define exact
  eligibility and not leave a branchy path undocumented.

## Option B. Billing-Issued Lease With Durable Child Debit Per Request

Research finding:

- This is escrow-like: billing reserves a lease budget; proxy locally allocates
  child debits under that lease.
- Existing approved architecture already selected this shape.

Why it fits:

- It amortizes billing reserve cost across many requests.
- It preserves durable per-request lineage and terminal obligation before
  external execution.

Remaining performance concern:

- The active path still has a proxy durable write per request.
- The next spec should decide whether that durable child write is required for
  every request or whether smaller microleases can accept bounded memory/Redis
  leakage.

Specification implication:

- Strong default candidate if the product refuses any unrecorded write-off
  window.

## Option C. Microlease With In-Memory Per-Instance Token Bucket

Research finding:

- Local token buckets are extremely fast and proven for rate limiting.
- They are safe for money only if the tokens are a cache of pre-reserved,
  instance-owned lease capacity.

Possible authority classifications:

1. Memory cache over durable per-child allocation.
   - Money-safe, but does not remove the per-request durable write.
2. Memory pre-consumption with async durable checkpoint.
   - Removes per-request durable write but creates crash/write-off exposure.
3. Memory abuse limiter only.
   - Not money authority.

Required controls if accepted:

- Tiny per-instance cap.
- Short TTL and earlier debit cutoff.
- Refill only under healthy terminal lag and reconciliation state.
- Durable lease grant before spend.
- Crash write-off budget owned by product/platform.
- Strict low-balance mode that disables memory-only spend.

Specification implication:

- Candidate only if the spec explicitly accepts bounded platform exposure or
  keeps memory as cache/limiter.

## Option D. Microlease With Redis Shared Token Bucket

Research finding:

- Redis/Lua supports atomic read-decide-update and low-latency shared buckets.
- Redis persistence/replication does not make it a strong customer-money store.

Possible roles:

1. Advisory global limiter over billing-reserved capacity.
2. Shared bucket over proxy durable allocator state.
3. Abuse/backpressure control.
4. Money authority.

Classification:

- Roles 1-3 are viable candidates.
- Role 4 should be rejected unless the future spec explicitly accepts bounded
  write-off risk and defines rebuild/reconciliation semantics.

Specification implication:

- If Redis is included, name it as limiter/cache/projection unless a deliberate
  risk acceptance says otherwise.

## Option E. Durable Proxy Checkpoint Batches

Research finding:

- Checkpoint/close batches are useful for releasing unused lease capacity,
  summarizing local allocator progress, and reducing billing round trips.

What they cannot do:

- They cannot replace durable child identity if billing must explain every
  customer charge and replay/conflict outcome.
- Batch totals alone are insufficient for money audit.

Specification implication:

- Good supporting pattern with required fields: high-water mark, child count,
  child cap sum, terminal submitted/published count, remaining capacity,
  checkpoint fingerprint, lease owner, generation/fence, and unresolved-child
  summary.

## Option F. Pure Async Metering

Research finding:

- Stripe, Metronome, and OpenMeter show high-throughput async metering,
  idempotent ingestion, queue/retry, and aggregation patterns.

Why it fails prepaid admission:

- It observes usage after the external effect.
- It cannot stop an account from overspending before billing processes the
  event.

Specification implication:

- Reject for normal prepaid paid admission unless preceded by a durable reserve,
  voucher, microlease, or explicitly accepted postpaid/write-off model.

## Option G. Redis Or Global Counter As Money Gate

Research finding:

- Counters are attractive for speed, but official docs warn about idempotency,
  read aggregation cost, persistence, and replication limits.

Why it fails authority:

- It creates a hidden money source outside billing Postgres.
- Loss, failover, stale reads, or duplicate updates can become customer-money
  errors.

Specification implication:

- Reject as customer-money authority.
- Consider only as limiter/cache/projection with durable source-of-truth
  reconciliation.

## Option H. Hybrid Strict / Fast Modes

Research finding:

- A hybrid may match the prompt's "strict fallback only for risky cases."
- Risky cases include low balance, stale pricing, high cost variance, terminal
  lag, abuse/manual review, allocator uncertainty, Redis degradation, and
  missing durable lineage.

Design pressure:

- Branching can create hard-to-explain behavior and hidden architecture
  divergence.

Specification implication:

- Worth comparing, but the spec must define:
  - exact eligibility;
  - default behavior when eligibility cannot be evaluated;
  - whether strict mode means direct reserve, smaller microlease, no memory-only
    spend, or fail-closed;
  - observability and rollout proof.

## Recommended Candidates For Specification Comparison

Recommended candidate set:

1. Preserve current durable lease plus child-debit architecture as the
   correctness baseline.
2. Evaluate smaller billing-issued microleases with in-memory per-instance token
   bucket as a performance variant, but only with explicit write-off cap or
   durable-child-cache classification.
3. Evaluate Redis shared token bucket only as a limiter/cache over billing
   reserved capacity, not as authority.
4. Evaluate durable proxy checkpoint batches as release/proof optimization.
5. Evaluate hybrid strict/fast mode only if exact risky-case eligibility can be
   specified without creating an unbounded alternate architecture.

Rejected for normal prepaid admission:

- Pure async metering without prior authority.
- Redis/global counter as money authority.
- Process-local memory spend without bounded exposure and owner-approved
  write-off policy.
