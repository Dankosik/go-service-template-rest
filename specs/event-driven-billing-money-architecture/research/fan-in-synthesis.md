# Research Fan-In Synthesis

Status: research synthesis
Date: 2026-06-01
Scope: R1-R9 lane reconciliation for `specs/event-driven-billing-money-architecture`

This file records research conclusions and unresolved decisions for the next specification phase. It is not `spec.md` and does not approve an architecture.

## Core Synthesis

Research consensus:
- Billing-service Postgres must remain the customer-money authority for USD ledger effects, visible balances, reserves, terminal settlement, stored outcomes, and reconciliation.
- Redpanda can be transport, replay, or notification infrastructure. It cannot enforce the no-negative invariant by itself.
- Normal paid usage needs a billing-owned pre-execution gate: direct synchronous reserve, command-event reserve where proxy waits for a durable outcome, or billing-issued spend rights whose exposure is already reserved.
- Fully async fire-and-forget usage events are not viable as the normal paid-usage architecture unless another prior billing-owned reservation, lease, or spend token already bounded exposure.
- Current proxy-local balance state, in-process reservations, and local balance mutation cannot remain money correctness authority in the target architecture.
- Current provider/consumer drift must be resolved before design: billing-service OpenAPI has no business money surface, proxy has both TypeBox internal-money contracts and older bridge paths, and pricing selector evidence differs between proxy and pricing-service.

## Candidate Option Evaluation

| Option | Research Classification | Key Reason |
| --- | --- | --- |
| 1. Fully synchronous billing reserve/finalize/write-off | Viable, simplest correctness model | Proxy waits for billing. Postgres transaction, idempotency, holds, ledger, and stored outcomes enforce money safety. Main tradeoff is request-path latency and same-account lock contention. |
| 2. Fully async fire-and-forget usage events | Rejected for normal paid usage unless behind prior preauthorization | Proxy could execute before billing reserves funds, so broker lag/outage/replay cannot prevent overspend before external cost exists. |
| 3. Event-driven reserve request plus outcome before execution | Conditionally viable, likely async RPC | Correct only if proxy waits for durable billing reserve outcome and fails closed on missing/ambiguous outcome. Adds broker, consumer, outbox, correlation, and lag failure modes versus HTTP. |
| 4. Proxy-local fast admission backed by spend tokens | Conditionally viable | Only safe if billing mints bounded, non-reusable, account-scoped, expiring spend rights and reserves the full exposure in Postgres. Proxy cache is derived, not authority. |
| 5. Preauthorized spending windows/account leases | Conditionally viable but high risk | Full window must be reserved in billing up front and visible balance must subtract it. Larger windows reduce latency but increase stale holds, reconciliation exposure, and fencing complexity. |
| 6. Hybrid reserve-before-execution plus async finalization/write-off | Strongest current candidate for specification | Synchronous reserve protects no-negative balance before external execution; terminal success/failure can be durable async with inbox/outbox, idempotency, and stale-hold reconciliation. |
| 7. Stronger production pattern discovered | Carry forward as refinement of option 6 | Sync reserve, durable terminal submission, sync terminal fallback when possible, durable proxy outbox if Redpanda is unavailable after execution, and Postgres-owned reconciliation. Optional spend rights only if performance proof requires them. |

## Specification Decisions Required

Architecture:
- Select the target pattern, likely option 6 or option 7 unless the specification records stronger evidence for option 1, 3, 4, or 5.
- Formally reject option 2 for normal paid usage unless the spec defines a prior billing-owned preauthorization mechanism.
- Decide what the proxy waits for before execution, after successful execution, after failed/interrupted execution, during billing outage, and during broker lag/outage.

Contracts:
- Choose the target HTTP/internal contract surface: current proxy TypeBox internal-money contract, current proxy shared-balance bridge, a new billing OpenAPI shape, or an explicit compatibility bridge with exit criteria.
- Define reserve/finalize/write-off/readback idempotency keys, fingerprints, operation IDs, stored outcome readback, conflict semantics, and HTTP status versus business-result envelope policy.
- Decide account-scope resolution, represented user context, caller principal/scope, route scopes, and whether billing or proxy owns pricing reads for reserve/finalize.
- Resolve proxy/pricing selector drift before money decisions use pricing snapshots.
- Keep payments/top-up evidence future-compatible, but do not let missing payments-service business OpenAPI block the current usage architecture unless top-ups are included in the accepted spec scope.

Distributed flow:
- If Redpanda is used, define event identity, schema version, fingerprint, producer authority, partition key, dedupe identity, event retention, inbox state machine, outbox publish state, DLQ/quarantine behavior, retry ownership, redrive authorization, and cleanup windows.
- Reserve commands over Redpanda are only money-safe when proxy waits for the durable reserve outcome before execution.
- Terminal events after reserve must be replay-safe and may arrive duplicated, delayed, or out of order; billing must consume them through durable inbox and business idempotency.
- Billing-emitted facts should be published from a transactional outbox after the Postgres money transaction commits; downstream consumers dedupe by stable ledger/outcome/effect identity.

Data:
- Keep balance/ledger/hold/idempotency correctness in Postgres.
- Add event inbox/outbox tables only if the selected architecture uses Redpanda for commands, terminal events, or billing-emitted facts.
- Add spend-token, allowance-window, or account-lease state only if selected for latency and backed by explicit full-reservation, expiry, fencing, consumption, release, and reconciliation semantics.
- Do not rely on Redis, in-process maps, proxy-local DB balances, request IDs, event IDs, or broker partitions as settlement identity.

Reliability and performance:
- Paid admission fails closed when billing reserve authority is unavailable, when pricing/account evidence is stale or mismatched, or when terminal lag threatens stale-hold budgets.
- Set budgets for reserve-before-upstream latency, terminal settlement latency, same-account lock contention, first-token impact, Redpanda lag, stale-reservation SLA, timeout/backoff, Postgres lock timeout, and connection pool limits.
- If terminal settlement is async, define durable proxy terminal-submission behavior for proxy restart or Redpanda outage after external execution.
- Benchmarks must cover uncontended, same-account contention, duplicate replay, timeout/readback, broker lag, and stale-hold recovery.

Security and privacy:
- Define real service authentication, route scopes, producer authenticity, event envelope signing or broker ACL policy, and authorization for redrive/replay.
- Bind every money command/event to billing-resolved account scope and represented subject; API keys are policy actors, not balance owners.
- Avoid raw prompts, completions, SSE chunks, credentials, DSNs, raw payment/webhook bodies, or unbounded provider payloads in APIs, logs, traces, metrics, inbox/outbox, audit rows, and research notes.
- Do not put account/user/operation identifiers into path shapes unless logging uses route templates or redaction.
- Treat `verifyUrl` evidence locators as SSRF-sensitive unless restricted to fixed provider allowlists or replaced with provider proof references.

Validation:
- Later `test-plan.md` must cover unit/property money math, Postgres invariants, idempotency replay/conflict, account row concurrency, API contract behavior, event replay/idempotency, crash-after-DB-before-offset, outbox duplicate publish, lag/stale hold recovery, privacy/redaction, unauthorized/account-mismatch failures, and proxy performance.
- Existing data-model tests are useful precedent but do not prove the new proxy/billing/event architecture.

## Material Open Questions

- Should the spec choose target internal-money v1 contracts directly, implement a compatibility bridge first, or define a new OpenAPI source of truth and adapt proxy to it?
- Where is canonical account scope resolved, and how does API-key `spend_limit_check_required` feed final money checks?
- Should billing call pricing itself, or should proxy pass immutable pricing snapshot evidence from pricing-service?
- What exact timeout/readback behavior applies after a reserve or terminal command may have been accepted but the caller timed out?
- Which Redpanda topics are in scope for current usage architecture: terminal events, reserve command/outcome, billing facts, payment evidence, or none?
- What are the numeric budgets for latency, throughput, stale holds, lag, retries, retention, and spend-window exposure?
- How will web-search holds/settlement/refunds map into usage reserve/finalize/write-off/reversal semantics?

## Specification Readiness

Research is complete enough to start specification. There is no research blocker.

The next phase should draft `spec.md` from this synthesis and current source evidence, then run the required formal spec clarification challenge before approval. The specification phase must not treat any lane output as final authority without recording the orchestrator's decision in `spec.md`.
