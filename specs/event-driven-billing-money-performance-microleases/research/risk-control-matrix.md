# Risk And Control Matrix

Status: research evidence
Date: 2026-06-01

This matrix captures controls the next specification should accept, reject, or
turn into proof obligations. It is not an implementation plan.

## Business And Money Controls

| Risk | Required control | Notes for specification |
| --- | --- | --- |
| Active exposure grows too large | `sum(active leases + allocated unsettled child caps + accepted memory/Redis leak window) <= account exposure cap` | For strict prepaid, all active lease exposure must already reduce billing available balance. |
| Account visible balance overstates funds | Available balance subtracts active lease/microlease exposure immediately | Terminal settlement may release unused capacity later, but visible balance cannot ignore outstanding spend rights. |
| Crash loses in-memory spend state | Either durable child debit exists before execution, or the memory-only window is capped as explicit platform write-off exposure | If accepted, spec must name owner, cap, metric, alert, and reconciliation behavior. |
| Low-balance account leaks over remaining funds | Low-balance strict mode disables memory-only or Redis-only spend; either use durable strict path or fail closed | Exact threshold belongs in spec. |
| Realized cost exceeds child cap | Customer charge capped at child and lease authority; excess is write-off, compensation, or reconciliation | Never retroactive overcharge. |
| Lease expires while terminal facts are missing | Stop new debits at cutoff, keep unresolved exposure reserved, reconcile before release | TTL alone cannot prove unused capacity. |
| Refill hides unsettled backlog | Refill only if terminal lag, stale child count, stale lease count, reconciliation backlog, and worker health are within gates | Backlog gates should be billing-owned and low-cardinality. |
| Abuse or manual-review signal | Stop or sharply reduce refill; shorten TTL; require strict mode or fail closed; keep existing exposure for settlement | Abuse controls must not silently mutate balances. |
| Pricing evidence stale or non-USD-compatible | Fail closed for new lease/replenishment and for child debit allocation requiring that evidence | Existing pricing snapshot uncertainty remains a spec/reopen point. |
| Multiple proxy allocators spend same capacity | Lease owner plus generation/fence; one durable allocator per fence or independently reserved leases per owner | Billing validates fence on terminal/checkpoint. |

## Redis And In-Memory Controls

| Surface | Allowed role | Required guardrail | Not allowed |
| --- | --- | --- | --- |
| In-memory bucket | Fast cache over reserved per-instance capacity; abuse limiter; optional bounded leak window if accepted | Per-owner cap, TTL, refill gates, restart behavior, durable grant, metrics, strict low-balance disable | Visible balance truth or unbounded spend authority |
| Redis shared bucket | Shared limiter/cache over reserved capacity; backpressure or abuse gate | Treat Redis state as rebuildable, reconcile against durable source, fail closed or degrade to strict mode on Redis uncertainty | Customer-money source of truth |
| Redis Lua | Atomic limiter operation | Single-key/hash-slot discipline, timeout budget, no raw money payloads, fallback policy | Durable idempotency substitute |
| Redis persistence/replication | Optional risk reducer | Explicit loss/failover assumption if Redis affects admission | Claiming CP/ledger authority |

## Event And Reconciliation Controls

| Risk | Required control | Notes |
| --- | --- | --- |
| Duplicate terminal event | Durable inbox or idempotency keyed by event/business identity and fingerprint | Same fingerprint replays; changed fingerprint conflicts. |
| Broker producer retry duplicate | Producer idempotence where available plus billing inbox dedupe | Broker settings do not replace business idempotency. |
| Offset committed before DB effect | Commit offset only after durable inbox/outcome, or use an inbox retry worker for accepted work | Money effect must be replay-safe. |
| DB committed but event not published | Transactional outbox or equivalent local outbox written with source state | Consumers remain idempotent. |
| Terminal event missing after external execution | Proxy durable terminal obligation plus retry/redrive; billing keeps lease exposure reserved until settlement or reconciliation | If no durable terminal obligation exists, spec must classify write-off exposure. |
| Poison event or invalid lineage | Quarantine/reject with safe error class and receipt identity, no money mutation | Do not store raw payload dumps. |
| Batch checkpoint hides per-child detail | Checkpoint includes high-water mark, child cap sum, unresolved child summary, terminal coverage, owner, fence, and fingerprint | Billing releases only provably unused capacity. |

## Strict/Fast Eligibility Controls

Fast mode may be eligible only when all are true:

- account has enough available balance for the next microlease plus safety
  floor;
- pricing snapshot is current, USD-compatible, and within expected cost variance;
- proxy allocator has current fence and healthy durable state or the spec has
  accepted a bounded memory-only write-off window;
- terminal backlog and reconciliation backlog are below gates;
- Redis or local limiter, if used, is healthy enough for its classified role;
- account is not in abuse, manual-review, suspended, or strict-budget state.

Strict mode or fail-closed is required when any are true:

- low balance or zero-overage contract;
- high-cost, high-variance, or unknown-cost operation;
- stale/missing pricing evidence;
- terminal lag warning/critical breach;
- reconciliation backlog breach;
- stale lease/debit age breach;
- proxy durable allocator unavailable;
- stale fence, duplicate child, changed fingerprint, or over-debit evidence;
- Redis loss/failover/split-brain when Redis affects admission;
- abuse/manual-review signal.

Specification must define whether strict mode means:

- direct per-request billing reserve;
- smaller billing-issued microlease with durable child debit only;
- no memory/Redis cache use;
- fail-closed with no alternate admission.

## Observability And Proof Controls

Metrics and logs should prove:

- active exposure by class, but without account IDs or high-cardinality labels;
- refill denied by reason;
- strict-mode/fail-closed reason;
- Redis limiter unavailable/degraded if used;
- terminal lag oldest age and counts;
- stale lease/debit age;
- reconciliation opened/resolved by safe reason;
- idempotency replay/conflict counts;
- write-off amount by safe class and bounded budget burn.

Privacy constraints:

- No raw prompts, completions, SSE chunks, API keys, bearer tokens, DSNs,
  payment secrets, raw provider payloads, dynamic proof URLs, raw event payloads,
  or sensitive request bodies in APIs, events, logs, traces, metrics, inbox,
  outbox, audit, reconciliation, or research artifacts.

Proof gaps for later phases:

- Live or synthetic workload distribution for request cost and account skew.
- Benchmark for active microlease admission, cold replenishment, Redis shared
  limiter, checkpoint/close, terminal ingestion, and first-token impact.
- Product-approved max write-off budget if memory-only or Redis-authoritative
  spend is considered.
- Security review for any dynamic proof locator or provider reference.
