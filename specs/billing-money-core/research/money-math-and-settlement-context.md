# Money Math And Settlement Context

Status: research complete  
Purpose: preserved evidence and synthesis for `billing-service` money-core specification

## Source Evidence

| Topic | Source |
| --- | --- |
| Current proxy money audit | `/Users/daniil/Projects/GonkaGate/gonka-proxy/docs/business/money-math-audit/current-money-math-audit.md` |
| Target treasury model | `/Users/daniil/Projects/GonkaGate/gonka-proxy/docs/business/money-math-audit/target-treasury-model.md` |
| Existing billing internal-money contract | `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts` |
| Settlement identity rule | `/Users/daniil/Projects/GonkaGate/gonka-proxy/docs/agent/gonka-domain-memory.md` |
| Current proxy ledger fields | `/Users/daniil/Projects/GonkaGate/gonka-proxy/prisma/schema.prisma` |
| Billing PRD | `docs/PRD.md` |
| Repository architecture | `docs/repo-architecture.md` |

## Current-State Findings

1. The proxy currently stores customer balance as `balanceNgonka` plus `lockedRateUsd`.
2. Visible USD balance is derived as `balanceNgonka * lockedRateUsd`.
3. Deposit math mints internal atomic GNK units from credited USD at deposit-time market rate.
4. Usage math charges USD from current market price and deducts internal atomic units through the user's effective locked rate.
5. Local reservation estimates are GNK/ngonka-shaped, not USD reserve ceilings.
6. This can keep the visible user USD projection internally consistent.
7. This does not prove treasury safety under GNK volatility.

## Target-State Findings

1. The authoritative customer ledger should be USD.
2. Target balance vocabulary should be `availableUsd`, `reservedUsd`, `settledUsd`, reserve lineage, and finalize lineage.
3. `balanceNgonka` and `lockedRateUsd` can survive only as compatibility or migration projections.
4. GNK inventory should be treated as service treasury asset, not customer balance backing.
5. Full pre-buy against all credited balances cannot guarantee non-negative economics under arbitrary adverse GNK movement with the intended finite fee model.
6. Target execution should use short-lived quotes, USD reserve ceilings, pricing snapshot lineage, explicit slippage, explicit execution fees, and pre-dispatch reject/requote rights.

## Existing Contract Signals

The existing `internal-money` billing contract already points toward the right boundary:

- reserve request has `clientUsageRequestId`, `accountId`, `maxCustomerExposureUsd`, `expectedReserveAmountUsd`, `expectedPricingSnapshotId`, and `requestBasisFingerprint`;
- reserve response has `usageOperationId`, `reserveAmountUsd`, optional `pricingSnapshot`, policy versions, and `usageState`;
- finalize uses `usageOperationId`, `meteredFactsFingerprint`, `expectedTotalCostUsd`, and `terminalBasisFingerprint`;
- finalize can include qualified inference evidence with `inferenceId`;
- finalize response returns `baseCostUsd`, `usageFeeRate`, `platformFeeUsd`, `totalCostUsd`, policy versions, stored outcome references, compensation references, and ledger reference;
- write-off is a first-class terminal command with evidence gap and timeout fingerprints;
- top-up evidence application returns `settlementEffectId` and conflict/review outcomes.

The Go service spec should reuse these concepts, not invent a second incompatible vocabulary.

## Critical Invariants To Carry Into Spec

- User-facing amounts are USD.
- Reserve in USD, not GNK.
- Finalize from authoritative metered and pricing evidence.
- Pricing snapshots must be pinned and freshness-checked.
- No stale-price fallback on money paths.
- Quote TTL should be short; target from source model is `<= 30 seconds`.
- Over-ceiling after external effect becomes write-off, compensation, or reconciliation, not retroactive overcharge.
- Ledger corrections are explicit compensating effects.
- Idempotency is durable and database-backed.
- Same key plus same fingerprint returns stored outcome.
- Same key plus changed fingerprint returns conflict.
- Redis and in-process maps are not correctness stores.
- `request_id` is trace/correlation only.
- `inferenceId` is settlement/verification identity.

## Performance Implications

The production money path should be designed around short, deterministic database transactions:

- reserve by `(account_scope, operation_key)`;
- finalize by `usageOperationId`;
- write off by `usageOperationId`;
- apply top-up evidence by `paymentEvidenceId` or equivalent stable evidence key;
- atomically append ledger effect and update balance read model;
- store operation outcome inside the same transaction;
- avoid outbound network calls while holding DB locks.

The likely fast path is fixed-scale integer USD atoms rather than floating point or ad hoc decimal math. The specification must decide the scale and storage type before schema design.

## Test Implications

The test strategy should be part of the spec, not only `tasks.md`.

Minimum proof classes:

- parser and formatter vectors for USD decimal strings;
- arithmetic vectors for reserve ceiling and final charge formulas;
- property tests for ledger conservation;
- duplicate and conflicting idempotency tests;
- concurrent replay tests against Postgres;
- stale quote and stale pricing fail-closed tests;
- top-up duplicate evidence tests;
- write-off and reconciliation tests for missing inference evidence;
- benchmark tests for reserve/finalize/write-off hot paths.

## Specification Handoff

The next `spec.md` should decide:

- exact amount representation and rounding rules;
- account scope model;
- ledger and balance source-of-truth shape;
- reserve/finalize/write-off states;
- idempotency namespace and conflict model;
- pricing snapshot and quote TTL contract;
- top-up evidence boundary with `payments-service`;
- reconciliation ownership;
- performance budgets;
- test gates before production cohort.
