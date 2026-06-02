# Critical Billing Context

Status: transferred from `gonka-proxy` research  
Scope: product and engineering constraints for building `billing-service` as the authoritative money service

## Why This Document Exists

Future `billing-service` work should not need to rediscover the core money model from `gonka-proxy`. This document carries the high-value facts from the current proxy implementation, the money-math audit, the target treasury model, and the internal-money contract into this repository.

Use this document before writing specs, API contracts, migrations, performance plans, or tests for customer-money behavior.

## Target Boundary

`billing-service` owns:

- authoritative USD customer ledger;
- account scope and balance read model;
- available, reserved, and settled USD amounts;
- top-up readback, evidence application, credit, expiry, manual review, and reconciliation;
- usage reserve, finalize, write-off, reversal, and compensation effects;
- durable idempotency records, operation fingerprints, and stored outcomes;
- support-safe ledger and reconciliation readbacks.

`billing-service` does not own:

- public OpenAI-compatible `/v1*` routes;
- gateway/devshard execution, model routing, or transfer-agent signing;
- pricing truth or market-rate publication;
- PSP sessions, webhooks, raw provider lifecycle, or payment secrets;
- API-key lifecycle or policy configuration unless a later contract explicitly moves it;
- raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs, or raw webhook payloads.

## Current Proxy Money Problem

The current proxy keeps user balance as `balanceNgonka` plus `lockedRateUsd`. The visible balance is:

```text
balanceUsd = balanceNgonka * lockedRateUsd
```

That can keep the user-facing USD projection internally consistent, but it does not prove treasury safety. `lockedRateUsd` is not a hedge. Request-local market-rate caching is not a hedge. Conservative rounding is not a hedge.

The target service must not carry forward `balanceNgonka` as the customer ledger. It may only appear as migration compatibility or legacy display projection during cutover.

## Target Money Model

Customer money is USD.

Use target balance concepts:

- `availableUsd`
- `reservedUsd`
- `settledUsd`
- reserve lineage
- finalize lineage
- append-only ledger effects

GNK inventory is a service treasury asset, not customer balance backing. The system should not pre-buy GNK for the full credited customer balance and then claim non-negative economics under arbitrary GNK price movement. That guarantee is mathematically unsupported without hedging or a large explicit worst-case risk premium.

Default treasury target:

```text
no full pre-buy inventory against all customer balances
```

Preferred execution model:

- JIT treasury or tightly capped rolling GNK buffer;
- short-lived quote before upstream dispatch;
- explicit reserve ceiling in USD;
- fail-closed stale pricing and liquidity checks;
- phase-aware requote or reject before irreversible external cost;
- write-off or reconciliation for over-ceiling effects discovered after external execution may exist.

## Reserve And Finalize Math

Reserve in USD, not GNK.

At reserve time, bind:

- account scope;
- stable client usage request ID;
- immutable request fingerprint;
- pricing snapshot identity and fingerprint;
- quote timestamp and expiry;
- usage fee policy version;
- max customer exposure in USD;
- slippage budget;
- execution-fee basis;
- liquidity or buffer-capacity proof.

Target reserve ceiling:

```text
quoteBaseUsd = q_quote * M_quote
maxBaseExecutionUsd = quoteBaseUsd * (1 + slippageBudget)
usageMarginCeilingUsd = maxBaseExecutionUsd * usageFeeRate
executionFeeCeilingUsd = maxBaseExecutionUsd * executionFeeRate
reserveCeilingUsd = maxBaseExecutionUsd + usageMarginCeilingUsd + executionFeeCeilingUsd
```

Finalize from realized evidence:

```text
baseExecutionUsd = q_fill * M_exec
executionFeesUsd = baseExecutionUsd * realizedExecutionFeeRate
finalChargeUsd = baseExecutionUsd * (1 + usageFeeRate) + executionFeesUsd
```

Finalize is inside the authorization only when:

```text
finalChargeUsd <= reserveCeilingUsd
```

Before upstream dispatch, an expected over-ceiling condition must requote or reject. After an external effect may exist, the customer charge stays capped by the authorized reserve policy and the excess becomes explicit write-off, compensation, or reconciliation work. It must not become a silent subsidy or retroactive overcharge.

## Amount Representation

Do not use floating point for money.

Recommended design direction for the first technical spec:

- API amounts stay decimal strings.
- Internal hot-path arithmetic uses a fixed USD atom scale that preserves the existing 8-decimal USD precision, for example `1 USD = 100,000,000 usd_atoms`.
- Database ledger amount columns use signed integer atom fields for speed, exact equality, and simple constraints.
- Any conversion from decimal string to atom must be canonical, range-checked, and reject excess precision unless the spec defines an explicit rounding rule.
- Rounding rules must be named per operation, tested with vectors, and biased intentionally only where the product contract says so.

If the spec chooses PostgreSQL `numeric` instead, it must justify the performance and equality tradeoffs and still prohibit float-based calculations.

## Idempotency And Stored Outcomes

Billing correctness cannot depend on gateway-local or Redis TTL idempotency.

Every money-affecting command needs a durable database-backed operation record:

- operation kind;
- idempotency key;
- immutable request fingerprint;
- account scope;
- operation state;
- stored success or failure outcome;
- conflict reason when the same key arrives with a changed fingerprint;
- correlation IDs;
- safe provider or execution references.

Replay rules:

- same key plus same fingerprint returns the stored outcome;
- same key plus changed fingerprint returns conflict;
- possible external acceptance after timeout must retry with the same operation identity or reconcile first;
- do not mint a fresh money operation after an ambiguous timeout.

Redis may be used for load shedding, short-lived collapse, or cache. It is not the money correctness store.

## Settlement Identity

`request_id` and `inferenceId` are different fields.

- `request_id`: HTTP trace and correlation identifier.
- `inferenceId`: settlement and inference verification identifier.

Never use `request_id` as the settlement lookup key. If stream mode aborts before `inferenceId` exists, keep `request_id` for traceability and treat settlement as unresolved until proper evidence exists.

Billing settlement should prefer:

- `usageOperationId`
- `topupOperationId`
- `paymentAttemptId`
- `paymentEvidenceId`
- `settlementEffectId`
- qualified `inferenceId`

## Performance Posture

The money path must be fast and exact.

Hot-path target:

- O(1) reserve/finalize/write-off operations by operation ID and account scope;
- one short Postgres transaction per money command;
- account balance row lock or equivalent single-writer invariant inside that transaction;
- no cross-service HTTP calls while holding a DB transaction;
- unique indexes for idempotency keys, ledger effect IDs, provider evidence IDs, and terminal usage outcomes;
- precomputed balance read model updated transactionally with append-only ledger effects;
- low-cardinality metrics only.

Do not trade correctness for cache speed. Caches can accelerate readbacks, but ledger, holds, idempotency, and stored outcomes must remain correct after process restart, Redis loss, retry storms, and concurrent replays.

## Test Strategy

This service should be test-heavy by default. The implementation is not ready without proof for both math and concurrency.

Required proof layers:

- money parser and formatter vectors;
- fixed-scale arithmetic vectors for every rounding rule;
- property tests for ledger conservation and non-negative available balance;
- idempotency replay and conflict tests for every money command;
- concurrency tests for duplicate reserve/finalize/write-off races;
- Postgres integration tests for uniqueness, row locking, transaction rollback, and migration constraints;
- top-up evidence duplicate/conflict tests;
- stale pricing, expired quote, over-ceiling, and missing inference evidence tests;
- reconciliation tests for stale reservations and ambiguous terminal outcomes;
- performance benchmarks for reserve/finalize/write-off hot paths;
- privacy tests or assertions that logs never contain raw prompts, completions, SSE chunks, tokens, DSNs, API keys, payment secrets, or full webhook bodies.

The first implementation ledger should treat tests as primary work, not as a final cleanup step.

## Cutover Rules

Before enabling real shared-balance cohorts:

- import or map existing proxy balances into an explicit USD ledger model;
- run shadow readback and parity checks;
- wire `gonka-proxy` reserve/finalize/write-off calls through billing;
- disable proxy-local money writes for the migrated scope;
- keep current proxy adapter paths as transitional only unless the later API spec intentionally keeps them;
- add reconciliation before customer-money writes are considered production-ready.

## Source Map

- Current audit: `/Users/daniil/Projects/GonkaGate/gonka-proxy/docs/business/money-math-audit/current-money-math-audit.md`
- Target treasury model: `/Users/daniil/Projects/GonkaGate/gonka-proxy/docs/business/money-math-audit/target-treasury-model.md`
- Billing PRD in this repository: `docs/PRD.md`
- Existing internal-money billing contract: `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`
- Settlement identifier rule: `/Users/daniil/Projects/GonkaGate/gonka-proxy/docs/agent/gonka-domain-memory.md`
- Current proxy balance fields: `/Users/daniil/Projects/GonkaGate/gonka-proxy/prisma/schema.prisma`
