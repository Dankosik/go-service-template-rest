# billing-service PRD

## Summary

`billing-service` is the GonkaGate source of truth for customer money. It owns account balances, top-up application, usage reservation, usage finalization, usage write-off, reversals, durable idempotency, and billing reconciliation.

The service should remove money-state ownership from `gonka-proxy` without changing the public OpenAI-compatible gateway contract. The proxy remains the request facade and execution coordinator; billing becomes the internal authority that decides whether customer funds can be committed and records the resulting USD ledger effects.

Detailed money-math, treasury, performance, and testing context lives in [critical-billing-context.md](critical-billing-context.md).

## Problem

GonkaGate currently mixes gateway execution concerns with billing concerns: request routing, devshard settlement, balance mutation, top-up evidence, usage accounting, and user-facing API behavior are too close together. That makes it hard to reason about money correctness, retry safety, reconciliation, and future payment/provider integrations.

Billing needs a dedicated service because money operations require different guarantees from request routing:

- durable exactly-once effect per business operation;
- account-level balance invariants;
- provider evidence reconciliation;
- replay-safe retries across service boundaries;
- auditable ledger history;
- privacy-safe operational telemetry;
- clear ownership of USD customer balances.

## Goals

- Own the authoritative customer ledger in USD.
- Own account balance read models derived from ledger effects.
- Own top-up lifecycle state after normalized payment evidence exists, including preview, create/readback, payment presentation sync, evidence application, credit, manual review, expiry, and reconciliation states.
- Own usage charge lifecycle: reserve, finalize, write off, reverse, and reconcile.
- Provide database-backed idempotent internal APIs for money-affecting operations.
- Provide operator/admin readbacks for support and reconciliation.
- Keep user-facing amount semantics consistently USD.
- Make retry, replay, and ambiguous-outcome handling explicit.
- Support a clean extraction path from `gonka-proxy` with no long-lived dual-writer balance state.

## Non-Goals

- Do not own public OpenAI-compatible `/v1*` routes.
- Do not own model routing, devshard selection, transfer-agent signing, or inference execution.
- Do not own pricing truth, model price catalogs, or quote math source data.
- Do not own payment-provider sessions, payment method collection, webhooks, or PSP-specific state machines.
- Do not own API-key lifecycle, identity, authentication, or public user/session management.
- Do not claim API-key spend-limit policy configuration without a separate contract moving that responsibility.
- Do not expose GNK, ngonka, credits, or internal treasury inventory as user-facing balance units.
- Do not store raw prompts, raw completions, SSE payloads, bearer tokens, API keys, DSNs, or payment secrets.
- Do not become a generic finance/accounting ERP. The first boundary is product billing and customer balance correctness.

## Actors And Clients

- `gonka-proxy`: requests reservations before paid usage, finalizes or writes off usage after execution, and reads account balance state for authenticated users.
- `payments-service`: sends normalized payment evidence and provider reference data; it remains the provider-integration owner.
- `pricing-service`: supplies price quote inputs and normalized charge amounts; it remains the pricing source of truth.
- `api-key-service` or identity authority: supplies caller/account attribution and policy context, but does not mutate customer money.
- Operators/support tools: read ledger, top-ups, reservations, finalization outcomes, and reconciliation status.
- Reconciliation workers: repair ambiguous or delayed provider/downstream outcomes using durable evidence, not raw request payloads.

## Core Responsibilities

### Customer Account Ledger

Billing owns customer account scopes, USD balances, holds, available balance, ledger entries, and immutable money effects. Every balance readback must be derived from durable ledger/hold state, not from a caller-provided value.

The service must define a canonical account scope key so all callers charge the same customer account consistently across API keys, identity sessions, referrals, bonuses, and future organizations.

### Top-Ups

Billing owns product-level top-up records and applies credit only after trusted normalized payment evidence is accepted. It should support preview/readback APIs so frontend and proxy surfaces can show expected credit amounts without becoming money writers.

Payment-provider details stay outside billing. Billing stores provider references and normalized evidence fingerprints needed for audit and reconciliation, not raw PSP secrets or webhook payloads.

### Usage Reservations

Before paid inference work starts, `gonka-proxy` should ask billing to reserve an amount for an account. Reservation must be idempotent by a stable usage operation key and must fail closed when funds, account state, or policy do not allow the charge.

A successful reservation creates a hold and returns enough data for the proxy to correlate later finalization without using public `request_id` as the settlement key.

### Usage Finalization

After execution and pricing are known, `gonka-proxy` finalizes the reserved usage with the canonical charge amount and execution references. Billing releases unused hold, commits the final charge, records the ledger effect, and returns a replay-stable outcome.

Finalization must be idempotent. Replaying the same operation returns the original outcome; conflicting fingerprints are rejected.

### Usage Write-Off And Reversal

When execution fails after a reservation, or when an already-recorded effect must be compensated, billing owns write-off/reversal operations. These operations must be explicit ledger effects, not silent balance edits.

### Spend Limits And Usage Counters

Recommended target: billing owns authoritative spend aggregates when they are based on finalized or reserved money. API-key and identity services can own policy objects and key lifecycle, but billing should expose money-backed spend state so limit decisions can be derived from the same ledger authority as charges.

This is a guarded open decision for the next spec. Current ownership should be treated as API-key/policy-side unless a later contract explicitly moves evaluation into billing. The PRD requires one money authority; it does not require billing to own all policy configuration.

### Treasury Admission

Billing may participate in treasury admission or execution gating when customer intake depends on service-level GNK inventory, stale price protection, or solvency constraints. That does not make GNK inventory a customer balance.

The customer ledger remains USD. Internal GNK inventory is a treasury asset and must not be presented as per-customer backing. Treasury constraints should fail closed or route to explicit review; they must not become silent subsidy, retroactive overcharge, or hidden balance mutation.

### Reconciliation

Billing owns reconciliation state for money effects:

- duplicate top-up evidence;
- delayed payment evidence;
- reserved usage with missing finalization;
- ambiguous finalize/write-off outcomes;
- provider reference mismatch;
- operator adjustments;
- migration/import parity checks during cutover.

Reconciliation should operate from durable operation IDs, provider references, account IDs, and ledger entries. It must not depend on raw gateway request bodies.

## Interaction Model

Billing exposes synchronous internal HTTP for request-path money gates: balance read, top-up preview/create-readback, payment presentation sync, usage reserve, finalize, write-off, reversal, and operation readback. These calls should use the shared internal contract headers for contract version, deadline, request ID, caller principal, represented user context, and trace correlation.

Billing uses asynchronous workers for stale reservation repair, delayed provider evidence retry, reconciliation jobs, treasury/admin workflows, and support-safe backfills. Long-running recovery should not be hidden inside request handlers.

Paid usage and top-up creation fail closed when billing is unavailable. `gonka-proxy` may continue serving non-money public surfaces, but it must not mint a fresh money operation after a possible accepted timeout; it must retry with the same operation identity or reconcile first.

## Required Invariants

- User-facing amounts are USD.
- Ledger effects are append-only; corrections use explicit compensating entries.
- A money-affecting operation has one stable idempotency key and one stable fingerprint.
- Replay with the same key and fingerprint returns the first committed outcome.
- Replay with the same key and a different fingerprint is a conflict.
- Redis or in-process TTL caches may collapse traffic, but they are never the money correctness store.
- Public `request_id` and billing settlement/inference identifiers are different fields.
- Billing must be able to answer "why did this account balance change?" from durable records.
- Billing APIs fail closed on unknown account, unknown policy context, stale pricing evidence, unsupported currency, or missing idempotency key.
- Billing APIs fail closed on non-USD customer money input.
- Logs and traces never contain raw prompts, raw model outputs, SSE chunks, payment secrets, bearer tokens, API keys, DSNs, or full webhook bodies.

## Candidate Internal API Surface

The exact route names belong in the later API spec. The product contract should cover these capabilities:

- `POST /internal/billing/v1/accounts/resolve`
- `GET /internal/billing/v1/accounts/{accountScopeKey}/balance`
- `POST /internal/billing/v1/topups/preview`
- `POST /internal/billing/v1/topups`
- `POST /internal/billing/v1/topups/{topupOperationId}/presentation-sync`
- `POST /internal/billing/v1/topups/{topupOperationId}/evidence`
- `POST /internal/billing/v1/usage/reservations`
- `POST /internal/billing/v1/usage/finalizations`
- `POST /internal/billing/v1/usage/write-offs`
- `POST /internal/billing/v1/usage/reversals`
- `GET /internal/billing/v1/operations/{operationId}`
- `POST /internal/billing/v1/reconciliation/jobs`
- `GET /internal/billing/v1/admin/accounts/{accountScopeKey}/ledger`

All money-affecting POST operations must require an idempotency key, an operation fingerprint, caller identity, account scope, and trace/correlation metadata. Idempotency records must live in durable storage with stored outcomes and conflict fingerprints; they must not depend on gateway-local or Redis TTL idempotency.

## High-Level Flows

### Top-Up Application

1. Client starts a top-up through the public product surface.
2. `payments-service` creates provider state and returns presentation data.
3. Provider sends payment result to `payments-service`.
4. `payments-service` normalizes and authenticates the evidence.
5. `billing-service` accepts the evidence idempotently, credits the account ledger, and marks the top-up applied.
6. Product surfaces read the updated balance from billing.
7. Duplicate or conflicting evidence returns the stored readback, `evidence_conflict`, `manual_review`, or `reconcile_required` rather than crediting twice.

### Paid Usage

1. `gonka-proxy` authenticates the caller and resolves account attribution.
2. `gonka-proxy` asks pricing for a bounded quote or maximum reservation amount.
3. `gonka-proxy` asks billing to reserve funds using a stable client usage request ID, immutable request fingerprint, pricing snapshot identity, and account scope.
4. `gonka-proxy` executes the request through the gateway/devshard path.
5. `gonka-proxy` finalizes the usage by billing `usageOperationId` with canonical cost, terminal fingerprint, and execution identifiers.
6. Billing commits the final ledger effect and releases any unused hold.
7. If execution fails after reservation, `gonka-proxy` writes off/releases the reservation through billing.

### Ambiguous Outcome Repair

1. Caller retries with the same idempotency key.
2. Billing returns the prior committed result when fingerprint matches.
3. Conflicting replay is rejected and surfaced as an operational issue.
4. Reconciliation jobs inspect durable records and apply explicit compensating entries where needed.

## Data Ownership

Billing should own these durable concepts:

- account scope;
- balance snapshot/read model;
- ledger entry;
- hold/reservation;
- billing operation;
- idempotency record;
- stored operation outcome;
- operation fingerprint;
- top-up product record;
- normalized payment evidence reference;
- usage charge;
- write-off/reversal;
- reconciliation case;
- operator adjustment;
- audit trail.

Billing may consume but should not own:

- identity user records;
- API key records;
- model price catalogs;
- raw gateway request/response payloads;
- PSP webhook raw payloads;
- devshard execution internals;
- transfer-agent control-plane state;
- service-level GNK treasury inventory.

## Observability

Billing telemetry must be optimized for money debugging without leaking sensitive payloads.

Required signals:

- operation outcome counters by operation type and failure class;
- idempotency replay/conflict counters;
- reservation, finalization, write-off, and reversal latency;
- stale reservation age and count;
- reconciliation case count by reason;
- balance mutation count by ledger effect type;
- provider evidence duplicate/conflict counters;
- dependency timeout/error counters for pricing, payments, identity, and storage.

Every money operation should include trace correlation, account scope, operation ID, and safe provider/execution references where available. Avoid high-cardinality raw request identifiers in metrics.

## Migration And Cutover Requirements

- Inventory current balance, top-up, and usage mutation paths in `gonka-proxy`.
- Define a migration ledger/import process for existing balances.
- Run shadow readback/parity before making billing the writer.
- Wire `gonka-proxy` reserve/finalize/write-off calls through billing before enabling shared-balance cohorts.
- Treat current proxy adapter paths as transitional compatibility routes unless the later API spec intentionally keeps them.
- Disable old balance writes in `gonka-proxy` before declaring cutover complete.
- Keep a rollback plan that prevents dual writes from silently diverging.
- Add reconciliation jobs before enabling real customer-money writes.

## Acceptance Criteria

The service is product-ready for first internal adoption when:

- all money-affecting APIs are idempotent and replay-tested;
- ledger and balance invariants are enforced by storage constraints or transactional code;
- top-up evidence cannot credit the same provider event twice;
- top-up replay semantics distinguish stored readback, payload conflict, evidence conflict, manual review, and reconciliation-required outcomes;
- reservation/finalization/write-off paths survive caller retry and process restart;
- reconciliation can list and repair stale reservations and ambiguous payment evidence;
- `gonka-proxy` no longer writes the same balance state for the migrated scope;
- privacy-safe logs/traces/metrics prove each operation outcome without exposing sensitive payloads;
- runbooks explain support readback, replay conflicts, and reconciliation procedures.

## Open Decisions For The Next Spec

- Canonical account-scope model: user-only, organization-ready, or both from day one.
- Exact ownership split for spend-limit policy configuration vs spend aggregate evaluation.
- Exact treasury admission contract between billing, pricing, and execution services.
- Whether reservation amount is always quote-derived from `pricing-service` or sometimes caller-provided with billing-side validation.
- Which existing `gonka-proxy` IDs become billing operation IDs and which must remain correlation-only.
- Synchronous vs queued reconciliation worker shape.
- Initial database schema and transaction isolation strategy.
- Required internal auth model between `gonka-proxy`, `payments-service`, `pricing-service`, and billing.
