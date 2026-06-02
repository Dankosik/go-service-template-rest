# Event-Driven Billing Money Performance Microleases Specification

Mode: full orchestrated
Status: approved
Date: 2026-06-01
Owner: orchestrator

## Context

This specification decides the next prepaid paid-admission direction for
`billing-service` and `gonka-proxy` after the performance microlease research
track.

The read-only baseline is the approved
`specs/event-driven-billing-money-architecture` packet, which selected
billing-issued account-scoped spending leases, durable proxy child debit
allocation before external execution, asynchronous terminal settlement through
durable events, and billing-service PostgreSQL as the customer-money authority.

The performance research track asked whether that approved lease direction
should move toward escrow, bounded counters, microleases, token-bucket spend
inside bounded lease authority, async terminal usage events, bounded write-off
or reconciliation risk, and strict fallback only for risky cases.

Research under `research/` concluded:

- escrow and bounded-counter rights allocation are the closest correctness
  pattern for high-performance prepaid admission;
- billing-service PostgreSQL remains the durable customer-money authority unless
  a future specification explicitly accepts platform write-off exposure;
- in-memory and Redis token buckets are performance mechanisms, not customer
  money sources of truth;
- durable proxy checkpoint and close batches are useful proof and release
  inputs, but they cannot replace per-child lineage when every customer charge
  must be explainable;
- pure async metering is suitable for terminal usage ingestion and analytics,
  but not for first prepaid admission without prior authority;
- no product-approved write-off budget, benchmark result, production traffic
  distribution, or live deployment evidence exists for accepting unrecorded
  memory-only or Redis-only spend.

This specification therefore chooses a production-ready target-state direction:
billing-issued durable microleases are a tighter performance refinement of the
approved spending-lease model, not a move to cache-authorized customer money.

## Scope / Non-Goals

In scope:

- prepaid paid-admission semantics for billing-issued microleases;
- authority classification for billing PostgreSQL, proxy durable rows, process
  memory, Redis, Redpanda, ClickHouse, and checkpoint batches;
- accepted and rejected options from the research comparison set;
- strict or fail-closed behavior for low-balance, stale, high-risk, degraded,
  or abuse/manual-review cases;
- visible balance, active exposure, settlement, release, write-off, and
  reconciliation constraints for active microlease capacity;
- proof obligations and reopen conditions before technical design and planning.

Out of scope:

- editing `specs/event-driven-billing-money-architecture`;
- concrete route names, request/response schemas, event schemas, table names,
  package layout, migration SQL, generated code, adapters, worker topology, or
  implementation tasks;
- payment-provider sessions, payment webhooks, top-up runtime flows, payment
  evidence ingestion, and payment reversals/refunds;
- public OpenAI-compatible `/v1*` route behavior;
- pricing catalog ownership, model routing, devshard execution, transfer-agent
  signing, identity lifecycle, API-key lifecycle, or API-key policy
  configuration;
- accepting process-local memory, Redis, Redpanda, ClickHouse, proxy-local
  mutable balance state, or async metering as customer-money authority;
- a long-lived branch where migrated paid cohorts choose between direct
  per-request billing reserve and microlease admission by request risk.

## Constraints

- Customer money remains USD and billing-service PostgreSQL remains the only
  authoritative customer-money correctness boundary.
- Microlease spend authority is created only after billing has durably reserved
  the full USD microlease exposure in PostgreSQL.
- Active microlease exposure subtracts from visible available balance as soon
  as billing issues or replenishes the microlease.
- Proxy may admit paid external execution only when it can prove a valid
  billing-issued lease or microlease, current owner/fence, sufficient remaining
  durable local authority, and a durable child debit and terminal obligation.
- The accepted target has zero unbacked customer-money spend exposure. Any
  process-memory-only or Redis-only spend window has an approved budget of
  `0 USD` for this specification.
- Customer charge is capped by the child debit authorization and by aggregate
  parent lease or microlease authority. Excess after external execution becomes
  explicit write-off, compensation, or reconciliation, never retroactive
  overcharge.
- Redis may be used only as limiter, cache, projection, or backpressure surface
  over already-reserved authority. Redis state must be rebuildable from durable
  billing/proxy state and must not create customer-money capacity.
- Process memory may cache durable state or enforce local rate limits, but
  process memory cannot mint spend authority or become the only record of paid
  execution authorization.
- Redpanda, Kafka-style transactions, ClickHouse, analytics stores, and async
  metering pipelines can transport, dedupe, aggregate, or project terminal
  facts. They do not authorize first prepaid paid execution.
- No raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs,
  payment secrets, raw provider payloads, raw event payloads, dynamic proof
  URLs, or sensitive request bodies may appear in APIs, events, logs, traces,
  metrics, inbox/outbox rows, proxy local rows, audit rows, reconciliation
  records, research notes, or later workflow artifacts.

## Decisions

### D1. Target Direction

Decision: specify billing-issued durable microleases as the next prepaid
admission direction.

A durable microlease is a small, account-scoped, owner-fenced, short-lived USD
spend right minted by billing-service. It is an escrowed subset of customer
available balance, reserved in billing PostgreSQL before proxy can spend it.

The target path is:

1. Billing-service reserves a bounded microlease amount in PostgreSQL and
   returns a replay-stable stored outcome to the authorized proxy allocator.
2. Proxy persists the microlease grant and spends it only through durable child
   debit authorization before external execution.
3. Optional in-memory or Redis token buckets can pre-check, smooth, or throttle
   admission, but a token does not authorize external execution unless the
   durable child debit is already committed or committed in the same local
   admission step.
4. Proxy records terminal obligations and publishes terminal facts
   asynchronously through durable outbox/event mechanics.
5. Billing consumes terminal, checkpoint, and close evidence through durable
   inbox/idempotency, settles or releases capacity in PostgreSQL, emits derived
   facts through outbox, and reconciles stale or ambiguous lineage.

Rationale:

- This preserves the approved escrow-like lease invariant while making the
  intended unit smaller, shorter lived, and more explicitly gate-controlled.
- It leaves the high-frequency active path at the proxy durable allocator
  instead of billing PostgreSQL, but it does not accept unrecorded spend.
- It gives technical design room to optimize with cache, limiter, refill, and
  checkpoint mechanics without reopening money authority.

### D2. Zero Unbacked Spend Exposure

Decision: memory-only or Redis-only spend before durable child debit is rejected
for the approved target.

The specification does not accept a platform write-off budget for unrecorded
local spend because no product-approved amount, owner, alert threshold, or
release rollback policy exists. The accepted unbacked exposure budget is
therefore `0 USD`.

Consequences:

- Process restart, Redis loss, Redis failover, or limiter data loss must not
  lose the only record that a paid request was authorized.
- Proxy durable child debit lineage remains required before external execution.
- Redis or memory can deny, slow, shape, or pre-screen paid admission, but a
  Redis or memory token cannot by itself authorize customer-money spend.
- If a later product/platform decision wants to accept memory-only or
  Redis-only leakage, specification must reopen with explicit owner, cap,
  metric, alert, budget burn behavior, reconciliation treatment, and rollback
  rules.

### D3. Authority Classification

Decision: every admission surface has an explicit authority class.

| Surface | Accepted role | Rejected role |
| --- | --- | --- |
| Billing PostgreSQL | Customer-money source of truth; balance, reserved exposure, ledger, idempotency, stored outcomes, reconciliation, inbox/outbox authority. | None for customer money. |
| Proxy durable lease/debit rows | Non-authoritative proof and recovery state for billing-issued microlease grants, local child debits, terminal obligations, publish state, checkpoints, and close evidence. | Visible balance truth or capacity minted without billing reserve. |
| Process memory | Cache of durable proxy/billing state; local limiter; performance precheck. | Sole spend ledger, sole child-debit record, or unbounded capacity. |
| Redis | Shared limiter/cache/projection/backpressure over reserved capacity. | Customer-money source of truth, durable idempotency substitute, visible balance, or authoritative prepaid gate. |
| Redpanda/Kafka | Terminal/checkpoint/close/billing-fact transport and replay substrate. | Reserve authority, money mutation authority, or substitute for billing inbox/idempotency. |
| ClickHouse/analytics stores | Metering projection, reporting, aggregates, and dashboards. | Exact prepaid admission gate or ledger authority. |
| Durable checkpoint batches | Release/reconciliation proof and compression of already durable child lineage. | Replacement for per-child authorization identity when charges must be explainable. |

Technical design may introduce more concrete surfaces only if they map back to
one of these classes or reopen specification.

### D4. Strict And Fast Mode Semantics

Decision: the target supports one authority model with fast and strict
eligibility gates, not two customer-money authorities.

Fast mode means:

- account, pricing, proxy allocator, terminal backlog, reconciliation backlog,
  Redis/cache health if used, and abuse/manual-review status are all inside
  configured gates;
- proxy can allocate a durable child debit from a valid billing-issued
  microlease before execution;
- optional memory/Redis token buckets may be used as performance prechecks or
  load-shaping over the same reserved authority.

Strict mode means:

- memory and Redis acceleration are bypassed or treated as deny-only;
- proxy must rely on durable billing/proxy authority only;
- technical design may reduce microlease size, require fresh replenishment, use
  durable child-debit-only admission, or fail closed;
- strict mode does not mean direct per-request billing reserve fallback unless
  specification is reopened and approved.

Fail closed is required when the system cannot prove fast or strict eligibility.

Strict or fail-closed is required for:

- low balance, zero-overage, or insufficient safety floor;
- high-cost, high-variance, unknown-cost, stale-pricing, or non-USD-compatible
  work;
- stale fence, duplicate child, changed fingerprint, over-debit evidence, or
  proxy allocator uncertainty;
- terminal lag, reconciliation backlog, stale lease/debit age, or worker health
  breach;
- Redis loss, failover, split-brain, or timeout when Redis affects admission;
- abuse, manual review, suspended account, or operator fail-closed control.

### D5. Visible Balance And Exposure

Decision: active microlease exposure is visible reserved customer-money exposure
from billing's point of view.

Rules:

- Available balance must subtract the full issued microlease amount until
  settlement, valid close proof, expiry reconciliation, or explicit repair
  releases unused capacity.
- The sum of active leases, microleases, allocated unsettled child caps, and
  any future explicitly accepted leakage budget must stay within the account's
  exposure cap. For this specification, future leakage budget is zero.
- Lease or microlease expiry stops new child debits, but expiry alone does not
  prove unused capacity. Billing releases only from durable terminal,
  checkpoint/close, or reconciliation proof.
- Refill must be denied or reduced when terminal lag, stale child count, stale
  lease count, reconciliation backlog, worker health, pricing freshness, or
  account policy health is outside gates.

### D6. Accepted Option Set

Accepted:

- escrow/bounded-counter microleases as the conceptual model;
- billing-issued microleases reserved in billing PostgreSQL before spend;
- durable proxy child debit authorization and terminal obligation before
  external execution;
- in-memory token bucket as cache/precheck/limiter over durable authority;
- Redis shared bucket as advisory limiter/cache/backpressure over reserved
  authority;
- durable proxy checkpoint/close batches for proving terminal coverage and
  releasing unused capacity;
- async terminal facts with outbox/inbox and idempotent consumer semantics;
- strict/fail-closed eligibility gates that preserve one money authority.

Rejected:

- strict durable reserve before every request as the uniform target path;
- direct per-request reserve fallback as a hidden or routine alternate path for
  risky migrated paid requests;
- pure async metering as prepaid admission without prior reserved authority;
- Redis/global counter as customer-money authority;
- process-local memory as authoritative or unbounded spend authority;
- durable checkpoint totals without per-child debit identity;
- silent release of expired capacity without proof;
- charging above child or parent microlease authority;
- moving account, pricing, API-key policy, payment/top-up, or public gateway
  authority into this performance microlease scope.

### D7. Relationship To The Existing Lease Packet

Decision: this specification refines the approved lease architecture direction
but does not edit the existing `event-driven-billing-money-architecture`
artifacts.

The next technical-design session for this task must decide how to derive a
microlease design packet from the approved lease packet:

- preserve billing PostgreSQL money authority and durable proxy child debit
  lineage;
- narrow lease size, TTL, refill, cutoff, backlog, and checkpoint policy toward
  microlease-scale operation;
- classify any Redis or memory participation as cache/limiter/projection only;
- keep direct per-request reserve fallback rejected unless specification is
  reopened.

If technical design finds that the existing approved lease packet cannot be
adapted to microlease scale without accepting memory-only/Redis-only authority,
it must reopen specification.

## Formal Clarification Gate Reconciliation

Formal clarification was required because this is full-orchestrated
protected-money work touching customer money, persisted state, distributed
admission, fallback behavior, reliability, rollout, and validation.

Challenge execution:

- Method: local read-only formal clarification using the
  `spec-clarification-challenge` rubric.
- Scoped-down rationale: the available multi-agent tool is restricted to
  explicit user authorization for spawning subagents, and the user requested a
  specification-only phase without delegation. To honor both tool policy and the
  repository gate, the orchestrator ran the formal challenge locally across the
  default five lenses and recorded reconciliation here and in
  `workflow-plans/specification.md`.
- Lenses: scope/spec coherence; domain invariants and edge cases; architecture
  ownership and dependency boundaries; API/data/compatibility/source-of-truth;
  security/reliability/delivery/validation proof.

Resolution:

| Lens | Strongest question | Resolution |
| --- | --- | --- |
| Scope/spec coherence | Does the spec decide the prepaid admission direction without writing design? | Yes. It selects durable microleases with zero unbacked spend exposure and leaves schemas, route shape, package layout, and numeric tuning to technical design. |
| Domain invariants and edge cases | Can memory/Redis spend leak customer funds on crash, failover, or low-balance accounts? | Memory-only and Redis-only spend are rejected. Active exposure is billing-reserved and visible; strict/fail-closed gates cover low-balance, stale, degraded, and abuse states. |
| Architecture ownership and dependencies | Does any non-billing surface become hidden money authority? | No. Billing PostgreSQL is authority; proxy durable rows are proof/recovery; memory/Redis are cache or limiter only; Redpanda/ClickHouse are transport/projection. |
| API/data/compatibility/source-of-truth | Does this conflict with the existing approved lease packet or require contract details now? | No. The existing packet remains read-only context. This spec refines it for microlease scale and routes concrete OpenAPI/proto/data design to the next phase. |
| Security/reliability/delivery/validation proof | Are privacy, outage, rollout, and proof obligations explicit enough for technical design? | Yes. The spec carries fail-closed behavior, privacy exclusions, authority classes, benchmark/proof obligations, and reopen conditions. |

Gate status: complete.

No unresolved clarification blocker remains for specification approval. A fresh
clarification gate is required if technical design introduces memory-only or
Redis-only spend, direct per-request reserve fallback, weaker billing PostgreSQL
authority, weaker proxy durable lineage, broader payment/top-up/account/pricing
scope, or weaker privacy/outage policy.

## Open Questions / Assumptions

- [defer_to_design] Exact microlease cap, TTL, debit cutoff, refill threshold,
  safety floor, terminal lag gate, stale lease/debit age, reconciliation SLA,
  and benchmark workload belong to technical design. They must preserve zero
  unbacked spend exposure and active exposure subtraction from visible
  available balance.
- [defer_to_design] Technical design must choose whether Redis is present at
  all. If present, Redis must be a limiter/cache/backpressure surface over
  durable authority and must fail closed or degrade to strict durable admission
  on uncertainty.
- [defer_to_design] Technical design must choose how memory token buckets map
  to durable child debit allocation. Any design where memory token consumption
  precedes durable child debit and can be lost on crash reopens specification.
- [defer_to_design] Technical design must choose checkpoint/close evidence and
  release semantics, including high-water marks, child count, child cap sum,
  unresolved child summary, terminal coverage, owner/fence, and fingerprint.
- [defer_to_design] Technical design must decide whether this microlease track
  repairs or supersedes the existing `event-driven-billing-money-architecture`
  design bundle for a future planning pass. This session does not modify the
  existing packet.
- [assumption] No live traffic distribution, production DB rows, deployment
  evidence, or benchmark evidence was needed to approve the spec-level authority
  decision. Later technical design, planning, readiness, or rollout may require
  that evidence before implementation or release claims.
- [assumption] Proxy durable storage can be designed to support child debit
  allocation fast enough for the target paid path. If proof later shows it
  cannot, the workflow reopens specification rather than moving authority to
  memory or Redis silently.
- [reopen_spec_if_false] Reopen specification if pricing-service cannot provide
  or attest USD-compatible immutable pricing snapshot evidence for microlease
  issuance, child debit caps, and terminal settlement.
- [reopen_spec_if_false] Reopen specification if a web-search-like paid path
  cannot map into microlease, child debit, terminal settlement, write-off,
  reversal, and reconciliation semantics.
- [reopen_spec_if_false] Reopen specification if product/platform explicitly
  wants a nonzero memory-only or Redis-only write-off exposure budget.
- [reopen_spec_if_false] Reopen specification if technical design needs direct
  per-request billing reserve fallback for migrated paid cohorts.
- [reopen_spec_if_false] Reopen specification if technical design cannot meet
  the target performance envelope with billing-issued reserved microleases and
  durable local child debit allocation without weakening money authority.

## Task Breakdown / Handoff Link

Next phase: technical design.

Expected technical-design output:

- task-local design bundle for durable billing-issued microleases;
- explicit reconciliation with the approved `event-driven-billing-money-architecture`
  packet as read-only context;
- authority and component map for billing PostgreSQL, proxy durable allocator,
  memory, Redis if used, Redpanda, ClickHouse/analytics if referenced, and
  operator/reconciliation surfaces;
- sequence/failure design for microlease issue/replenish, durable child debit,
  token-bucket precheck, strict/fail-closed gates, terminal publication,
  checkpoint/close, expiry, release, write-off, and reconciliation;
- data/contract design for microlease state, child lineage, checkpoint/close
  evidence, inbox/outbox, protected HTTP, and event contracts, without editing
  runtime contracts in the design phase unless that phase explicitly owns them;
- validation and rollout obligations for benchmark, privacy, Redis degradation
  if used, proxy crash/restart, terminal lag, stale lease/debit recovery, and
  no direct reserve fallback.

Technical design must not create `tasks.md`, migrations, schemas, generated
artifacts, adapters, tests, or implementation. Planning is blocked until
technical design and mandatory technical design review complete with `PASS` or
eligible `CONCERNS`.

## Validation

Forward-looking proof obligations:

- money-invariant tests for microlease issuance, replenishment, reserved
  exposure, visible balance, child cap, parent cap, release, write-off,
  reversal, compensation, idempotency replay/conflict, and non-negative
  available balance;
- proxy durable allocator proof for child debit allocation before execution,
  single-writer fencing, crash/restart, duplicate child ID, changed fingerprint,
  durable terminal obligation, local store outage, and no memory-only spend;
- Redis proof if Redis is included: limiter/cache-only classification,
  rebuildability from durable state, timeout/failover/split-brain behavior,
  strict degradation, and no customer-money authority;
- event and inbox/outbox proof for duplicate terminal events, changed
  fingerprints, out-of-order terminal facts, broker outage, committed DB effect
  before offset commit, outbox retry, checkpoint/close replay, quarantine, and
  redrive;
- reconciliation proof for stale microleases, stale child debits, terminal lag,
  missing terminal evidence, invalid fence, proxy over-debit, expired capacity,
  unresolved child summaries, and explicit write-off/compensation;
- security/privacy proof that APIs, events, logs, traces, metrics, inbox/outbox
  rows, audit rows, proxy local rows, Redis keys/values if used, and
  reconciliation records exclude raw prompts, completions, SSE chunks, bearer
  tokens, API keys, DSNs, payment secrets, raw provider payloads, raw event
  payloads, dynamic proof URLs, and sensitive request bodies;
- benchmark proof for active microlease admission, durable child debit
  allocation, optional memory precheck, optional Redis shared limiter, cold
  replenishment, checkpoint/close, terminal ingestion, reconciliation, and
  proxy first-token impact;
- rollout proof for default-closed controls, cohort gates, shadow/parity,
  old proxy writer disablement, direct reserve fallback disablement, Redis
  disable/degrade behavior if used, rollback/failback, and no dual writer for
  migrated cohorts.

Implementation validation completed on 2026-06-01:

- billing-service `rtk make check`, `rtk make openapi-check`, `rtk make
  proto-check`, `rtk make sqlc-check`, `rtk make migration-validate`, `rtk
  make go-security`, `rtk make secret-scan`, targeted integration tests,
  worker/event tests, and `rtk make check-full` passed;
- proxy targeted tests for durable child debit allocation, migrated-cohort
  fallback policy, pricing/API-key lineage, and allocator performance passed;
- performance proof met the approved durable-authority budgets without Redis or
  unbacked memory spend;
- privacy/security proof found no raw prompt, completion, SSE, provider
  payload, bearer token, API key, DSN, payment secret, dynamic proof URL, or
  sensitive request-body leakage in the implemented billing surfaces.

## Outcome

Specification approved on 2026-06-01.

The accepted direction is durable billing-issued microleases with zero unbacked
spend exposure. Billing PostgreSQL remains customer-money authority; proxy
durable child debit lineage remains required before external paid execution;
memory and Redis are limited to cache, limiter, projection, or backpressure
roles over already-reserved authority.

Implementation ledger T001 through T018 was executed through final validation
and closeout. The resulting implementation keeps billing PostgreSQL as the
customer-money authority, requires durable proxy child debit plus terminal
obligation before paid execution, treats memory as cache/precheck only, and
does not introduce Redis or direct per-request reserve fallback for migrated
paid cohorts.
