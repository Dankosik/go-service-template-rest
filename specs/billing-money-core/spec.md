# Billing Money Core Specification

Mode: full orchestrated
Status: approved for current sign-up-bonus and usage-only money scope; payment ingestion deferred

## Intent

Define the canonical decisions for the current billing money core so technical design can proceed without inventing source-of-truth, money representation, idempotency, reconciliation, legacy-compatibility policy, sign-up bonus grant semantics, or usage settlement behavior.

The original approved slice included future top-up/payment evidence concepts. A 2026-06-01 user clarification changes the current product boundary: customer payment/top-up is not implemented now, users cannot currently add balance themselves, and the only current balance-increase path is the `$10.00` sign-up bonus credited at registration/account admission. Usage can reserve/finalize/write off and therefore decreases or releases balance. Payment-provider top-ups, payments-service evidence handoff, and Redpanda payment-evidence ingestion remain future/conditional context, not current implementation scope. This spec still does not approve HTTP/OpenAPI contracts, runtime event schemas, service-code structure, migration files, runtime adapters, or implementation planning.

## Evidence Read

- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/workflow-plans/technical-design.md`
- `specs/billing-money-core/workflow-plans/technical-design-review.md`
- `specs/billing-money-core/research/money-math-and-settlement-context.md`
- `specs/billing-money-core/design/data-model.md`
- `specs/billing-money-core/design/event-ingestion-redpanda.md`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/prisma/schema.prisma`
- `/Users/daniil/Projects/GonkaGate/payments-service/docs/service-foundation/data-model-and-migration.md`
- `/Users/daniil/Projects/GonkaGate/payments-service/research/02-internal-money-flow/findings.md`
- User clarification on 2026-06-01: payment/top-up is not implemented now; users
  cannot currently add balance; the only current balance increase is a `$10.00`
  sign-up bonus at registration/account admission.

## Scope / Non-Goals

In scope:
- customer-account money source of truth;
- fixed-scale USD amount representation;
- ledger, balance read model, holds, usage operations, sign-up bonus grants,
  idempotency, reconciliation, audit readback, and legacy import compatibility;
- transactional/concurrency invariants that the schema must support;
- current funding-source boundary: registration/account admission can grant one
  `$10.00` sign-up bonus; customer-initiated payment/top-up is not a current
  balance-increase path;
- data-model test obligations.

Out of scope:
- HTTP route names, OpenAPI schemas, and public/internal API contract design;
- service package architecture and runtime adapter design;
- migration SQL files, backfill scripts, and generated access code;
- pricing-service, payments-service, identity-service, and proxy implementation details beyond storing their safe external references;
- customer payment/top-up product flow, payment provider sessions, payment
  presentation sync, normalized payment evidence application, payments-service
  writeback, and Redpanda payment-evidence ingestion for the current
  implementation slice;
- GNK treasury inventory schema except where the data model must prove it is not customer balance backing;
- `tasks.md` or implementation-readiness planning.

## Behavior / Contract Delta

ADDED:
- Billing owns a USD-only customer account ledger with append-only money effects and a transactionally maintained balance read model.
- Billing stores durable operation/idempotency records with fingerprints and replay-stable outcomes for every money-affecting command.
- Billing stores current sign-up bonus grant lineage, usage settlement lineage,
  reconciliation cases, and support-safe audit readback fields.

MODIFIED:
- `balanceNgonka` and `lockedRateUsd` are downgraded to legacy migration/display compatibility inputs only.
- `request_id` remains correlation-only and is never a settlement identity or uniqueness key for money effects.
- Payment evidence handoff from payments-service is deferred from current scope.
  If payment ingestion is reopened later, the prior version-scoped selector
  decision remains useful context but must be revalidated against the then-current
  product/payment contract before planning.

REMOVED FROM TARGET MONEY TRUTH:
- GNK/ngonka customer balance ownership.
- Floating-point money calculations.
- Redis or in-memory idempotency/balance state as a correctness source.
- Silent balance edits for correction or operator repair.
- Customer payment or top-up as a current source of balance.

## Decisions

### D1. Canonical Account Scope

Decision: billing owns a canonical account scope table with one stable internal account row per billable customer scope. The first admitted scope type is user-backed, but the schema is organization-ready from day one.

Required model:
- `account_id`: internal immutable surrogate key, preferred as `uuid`.
- `account_scope_key`: stable unique text key used by callers for O(1) lookup, formatted as `user:<identity_user_id>` for day-one user accounts and `org:<organization_id>` when organization ownership is explicitly contracted.
- `account_type`: enum-like constrained text: `user`, `organization`.
- `subject_authority`: constrained text naming the owning identity source, for example `identity-service` or future `organization-service`.
- `subject_id`: external subject identifier from the owning authority.
- `state`: `active`, `suspended`, `closed`, `manual_review`.
- `created_at`, `updated_at`, `closed_at`.

Uniqueness:
- `UNIQUE (account_scope_key)`.
- `UNIQUE (subject_authority, account_type, subject_id)`.

Rejected options:
- API-key scoped balances are rejected because API keys are spend-policy actors, not the customer-money owner.
- Email or mutable public identifiers are rejected as account keys because ownership and reconciliation need stable authority IDs.
- A user-only schema is rejected because it would force a source-of-truth migration for organizations even though account scope can be modeled now without exposing organization behavior.

### D2. USD Amount Representation

Decision: all customer money is represented as signed integer `usd_atoms` with scale `1 USD = 100,000,000 usd_atoms` (`1e-8 USD`). PostgreSQL storage uses `BIGINT` for atom amounts. Customer-visible input and output at service boundaries remains decimal strings, but data-model truth is integer atoms.

Range:
- `BIGINT` signed range permits about `+/- 92,233,720,368.54775807 USD`.
- Balance read-model columns use non-negative checks.
- Ledger delta columns are signed because corrections, charges, releases, and write-offs need direction.
- Command inputs that represent customer credit, reserve, or charge ceilings must be range-checked before persistence.

Parser rules:
- Accept only base-10 decimal strings with optional leading sign where the specific operation permits signed input.
- Reject exponent notation, binary/octal/hex notation, grouping separators, whitespace-padded values, currency symbols, `NaN`, and infinity.
- Reject more than eight fractional digits unless the operation explicitly names a rounding rule.
- Canonicalize `-0` to `0`.
- Store and compare atom integers, not decimal strings.

Formatter rules:
- Format from atoms exactly.
- Use canonical decimal output with no exponent and no currency symbol.
- Trim trailing fractional zeroes for display/readback except retain at least one integer digit and output `0` for zero.

Rounding rules:
- Default rule is reject excess precision rather than round.
- Reserve ceilings derived from rational pricing math round up toward higher customer authorization in atoms before checking available funds.
- Final customer charges round to nearest atom only when the pricing snapshot or fee policy gives an exact rational rule; ties use half-up unless the pricing policy explicitly names another rule.
- The sign-up bonus grant is an exact configured amount: `$10.00` equals
  `1,000,000,000 usd_atoms`; no runtime rounding is permitted.
- Write-off, reversal, compensation, and correction entries use exact atom amounts derived from prior ledger or grant/evidence rows.

Rejected options:
- Float and double are forbidden because they cannot provide exact equality, deterministic replay, or audit-grade conservation.
- PostgreSQL `money` is rejected because locale and arithmetic semantics do not match service invariants.
- PostgreSQL `numeric` as the primary hot-path amount type is rejected because integer atoms provide faster equality, simpler indexing/checks, and deterministic parser/formatter parity for the required 8-decimal USD precision.

### D3. Ledger Source Of Truth

Decision: `ledger_entries` is the append-only money-effect truth. Balance rows are a transactionally maintained read model and must be reconcilable from ledger plus active holds.

Required ledger concepts:
- one immutable row per money effect;
- scoped by `account_id` and `account_scope_key`;
- signed `amount_usd_atoms`;
- `currency = 'USD'` as a constrained value;
- effect type constrained to at least: `signup_bonus_credit`, `usage_charge`, `usage_hold`, `usage_hold_release`, `usage_write_off`, `usage_reversal`, `operator_adjustment`, `migration_import`, `reconciliation_correction`;
- stable `settlement_effect_id` for externally traceable settlement effects when applicable;
- links to operation/evidence lineage: `usage_operation_id`, `signup_bonus_grant_id`, `qualified_inference_evidence_id`, `reversal_of_ledger_entry_id`, `correction_of_ledger_entry_id`;
- payment/top-up lineage fields may remain as dormant future-ready schema context only when already introduced, but they are not current runtime settlement identities and must not drive current planning;
- effective and processed timestamps: `effective_at`, `created_at`.

Correction strategy:
- all corrections are explicit compensating ledger entries;
- no silent update may alter a posted ledger amount or balance total;
- metadata may be amended only through a narrow support/audit path that leaves an audit row and never changes money fields.

Rejected option:
- A mutable balance-only model is rejected because support and reconciliation must answer why money changed and must survive replay/debugging.

### D4. Balance Read Model

Decision: maintain one account balance row per account and update it in the same short PostgreSQL transaction as the ledger/hold/operation mutation.

Required columns:
- `settled_usd_atoms`: net posted customer funds after finalized charges, credits, reversals, and corrections.
- `reserved_usd_atoms`: active holds not yet finalized, released, written off, expired, or reconciled.
- `available_usd_atoms`: spendable funds, maintained as `settled_usd_atoms - reserved_usd_atoms`.
- `pending_usd_atoms`: non-settled incoming/outgoing evidence amount requiring finality, manual review, or reconciliation; not spendable. Current sign-up bonus and usage paths do not require payment pending balance.
- `version`: monotonic integer for optimistic readback/debugging.
- `last_ledger_entry_id`, `updated_at`.

Invariant:
- `settled_usd_atoms >= 0`.
- `reserved_usd_atoms >= 0`.
- `available_usd_atoms >= 0`.
- `pending_usd_atoms >= 0`.
- `available_usd_atoms = settled_usd_atoms - reserved_usd_atoms`.

Concurrency rule:
- sign-up bonus, reserve, finalize, and write-off application must lock the account balance row (`SELECT ... FOR UPDATE`) or equivalent account-scoped invariant row before changing balance-visible money.
- the lock order is account row first, then grant/operation/hold/evidence rows by stable key to avoid deadlock cycles.

Rejected option:
- Recomputing balances by scanning ledger on every hot-path command is rejected because reserve/finalize/write-off require short O(1) account lookups.

### D5. Hold / Reservation Model

Decision: usage reservations are durable hold rows, not cache entries. A hold reserves USD atoms against one account, one usage operation, one reserve command fingerprint, and one pricing snapshot lineage.

Required hold state:
- `active`, `finalized`, `released`, `written_off`, `expired`, `reversed`, `reconcile_required`, `manual_review`.

Required hold data:
- `hold_id`, `usage_operation_id`, `account_id`, `account_scope_key`;
- `reserved_usd_atoms`, `released_usd_atoms`, `charged_usd_atoms`, `write_off_usd_atoms`;
- `pricing_snapshot_id`, `pricing_snapshot_fingerprint`, `quote_expires_at`, `fee_policy_version`, `reserve_policy_version`;
- `client_usage_request_id` as caller correlation/lineage, not settlement truth;
- `request_basis_fingerprint`;
- timestamps for created, expiry, terminal transition, and update.

Invariant:
- exactly one active or terminal hold lineage exists for a `usage_operation_id`.
- the sum of active hold amounts for an account must match `balances.reserved_usd_atoms`.
- finalize cannot charge more than the authorized reserved ceiling; over-ceiling after possible external effect becomes explicit write-off, compensation, or reconciliation.

### D6. Usage Operation Model

Decision: usage operations are the lifecycle authority for reserve, finalize, write-off, reversal, and compensation around one paid usage attempt. Terminal operations are one-time transitions protected by constraints and idempotency.

States:
- `reserve_pending`, `reserved`, `finalize_pending`, `finalized`, `write_off_pending`, `written_off`, `reversed`, `compensated`, `reconcile_required`, `manual_review`, `expired`.

Operation kinds:
- `reserve`, `finalize`, `write_off`, `reversal`, `compensation`.

Required settlement identifiers:
- `usage_operation_id`: primary usage settlement identity.
- `qualified_inference_evidence_id`: local row for qualified `inferenceId` evidence when available.
- `inference_id`: stored only inside qualified evidence with `provider_family`, `verification_surface`, and uniqueness scope.
- `terminal_outcome_id`, `settlement_effect_id`, and any compensation effect IDs when applicable.
- `request_id`: optional correlation only.

Uniqueness:
- `UNIQUE (usage_operation_id)`.
- terminal finalize and terminal write-off each have a unique terminal outcome per usage operation.
- qualified inference evidence uniqueness is scoped by its declared proof scope, not by raw `request_id`.

Rejected option:
- Treating `request_id` as the settlement key is rejected because stream aborts and retry paths can lack or change settlement evidence.

### D7. Sign-Up Bonus And Future Payment Boundary

Decision: the current approved balance-increase path is a service-granted
`$10.00` sign-up bonus credited once per admitted user account. Customer
payment/top-up, payment presentation, normalized payment evidence application,
and payments-service writeback are future/conditional behavior and must not be
planned or implemented from this current slice.

Sign-up bonus grant:
- `signup_bonus_grant_id` is the stable billing-owned grant identity.
- The day-one grant amount is exactly `10.00 USD`, stored as
  `1,000,000,000 usd_atoms`.
- The grant is scoped to one `account_id` / `account_scope_key` and one
  registration/account-admission lineage from the owning identity source.
- The grant carries a `signup_bonus_policy_version` so a later product decision
  can change bonus policy without rewriting historical grants.
- The same account cannot receive the same sign-up bonus policy grant twice.

Application invariant:
- sign-up bonus credit is an explicit ledger effect, not a synthetic top-up and
  not an operator adjustment;
- duplicate delivery of the same registration/account-admission grant returns
  the stored outcome;
- changed grant fingerprint for the same account and policy version is an
  idempotency conflict or reconciliation case and cannot credit again;
- correction or removal of a posted sign-up bonus uses an explicit reversal or
  reconciliation correction ledger effect; the original credit is never edited.

Future payment/top-up boundary:
- Existing top-up/payment-evidence design context is not current runtime scope.
  If retained in schema or docs, it is dormant future-ready context only.
- If customer payments are reopened later, the specification must explicitly
  reopen payment/top-up scope, validate the live payments-service contract, and
  decide whether the prior `(paymentEvidenceId, evidenceVersion)` selector still
  applies before technical design or planning starts.
- No current Redpanda payment-evidence ingestion, payment provider evidence
  application, presentation sync, payment reversal/refund, or customer-initiated
  top-up task may be planned from this spec.

### D8. Durable Idempotency

Decision: every money-affecting command uses a durable idempotency record written and finalized in the same local transaction as the operation state and ledger/balance effects.

Required idempotency model:
- scope fields: `account_id`, `operation_kind`, `idempotency_key`;
- `request_fingerprint`: immutable hash over canonical operation input, policy/evidence references, and intended semantic operation;
- `state`: `started`, `committed`, `failed_stored`, `conflict`, `reconcile_required`;
- stored outcome reference: `stored_outcome_id`;
- conflict reason and first/last seen timestamps;
- retention class and `expires_at` only after the replay safety window is explicitly complete.

Replay rules:
- same idempotency key plus same fingerprint returns the stored outcome for committed or stored-failure records.
- same idempotency key plus changed fingerprint returns conflict and does not mutate money.
- ambiguous timeouts must retry with the same operation identity or open reconciliation; callers must not mint a new money operation to guess completion.
- for sign-up bonus grants, the idempotency key and request fingerprint include
  `account_id`, `signup_bonus_grant_id`, `signup_bonus_policy_version`, and the
  fixed grant amount;
- payment evidence idempotency rules are future/conditional and not part of the
  current implementation scope.

Uniqueness:
- `UNIQUE (account_id, operation_kind, idempotency_key)`.
- a partial or state-aware uniqueness rule prevents two committed outcome rows for the same idempotency record.

### D9. Reconciliation Case Model

Decision: reconciliation cases are durable work records linked to grant, operation,
evidence, and ledger rows. They classify money ambiguity without becoming an
alternate money source of truth.

Case reasons:
- `stale_reservation`;
- `ambiguous_terminal_state`;
- `missing_inference_evidence`;
- `signup_bonus_conflict`;
- `legacy_import_mismatch`;
- `operator_adjustment_required`.

Future payment/top-up reasons such as `duplicate_payment_evidence`,
`evidence_conflict`, `late_payment_evidence`, and
`provider_reference_mismatch` are future/conditional and not current planning
requirements.

Case states:
- `open`, `leased`, `waiting_evidence`, `manual_review`, `resolved`, `canceled`.

Required fields:
- `reconciliation_case_id`, `account_id`, reason, state, severity;
- safe links to `signup_bonus_grant_id`, `usage_operation_id`,
  `settlement_effect_id`, `qualified_inference_evidence_id`, and ledger entries;
- dormant future payment/top-up references may exist only when kept as future-ready
  schema context and must not be current-scope reconciliation keys;
- lease owner/deadline, attempt count, next attempt time, resolution ledger entry/effect IDs, and support-safe notes.

Invariant:
- resolution that changes money must create explicit ledger effects.
- reconciliation workers may claim cases with lease semantics, but correctness remains in ledger/operation/idempotency rows.

### D10. Audit And Support Readback

Decision: support readback is built from authoritative tables plus support-safe audit rows. It must never require raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs, payment secrets, or raw PSP webhook bodies.

Required readback by account:
- account scope and account state;
- current balance row;
- ledger entries ordered by `(created_at DESC, ledger_entry_id DESC)`;
- sign-up bonus grant and stored outcome;
- holds/reservations and terminal outcomes;
- dormant future top-up/payment evidence rows only when present in the data model;
- idempotency replay/conflict history;
- reconciliation cases and resolution effects.

Audit rows:
- actor kind, actor ID or service principal, reason code, operation/evidence references, before/after state names when safe, and safe correlation IDs.
- Money amounts in audit rows may duplicate atom values for readback but are not the source of truth.

### D11. Legacy Migration Compatibility

Decision: legacy `balanceNgonka` and `lockedRateUsd` are import evidence only. The target billing source of truth is an explicit USD ledger import effect.

Required compatibility model:
- store legacy source identifiers, legacy `balanceNgonka`, legacy `lockedRateUsd`, derived USD atom value, import batch ID, import fingerprint, and parity status in an import/evidence table;
- create one `migration_import` ledger entry per migrated account or a clearly linked import correction entry;
- preserve enough legacy evidence for parity/debugging, but never use legacy fields for live customer balance after cutover;
- proxy-local balance writes must be disabled for migrated scopes before billing is declared the writer.

Rejected option:
- Maintaining `balanceNgonka * lockedRateUsd` as a parallel live balance source is rejected because it creates split truth and does not solve treasury safety.

### D12. Hot-Path Transactions And Access Patterns

Decision: the money core optimizes for short O(1) PostgreSQL transactions and explicit uniqueness/row-locking constraints.

Expected hot-path query shapes:
- sign-up bonus grant: lookup account by `account_scope_key`, lock account
  balance, lock idempotency by grant identity, insert a `signup_bonus_credit`
  ledger entry once, update balance, store outcome;
- reserve: lookup idempotency by `(account_id, operation_kind, idempotency_key)`, lock account balance by `account_scope_key`, insert usage operation/hold/ledger hold entry, update balance, store outcome.
- finalize: lookup idempotency, lookup usage operation by `usage_operation_id`, lock account balance, lock hold, append charge/release/write-off effects as needed, update balance, store outcome.
- write-off: lookup idempotency, lookup usage operation by `usage_operation_id`, lock account balance, lock hold, release or write off active reserved atoms, append effect, update balance, store outcome.

Required indexes/constraints must support:
- account lookup by `account_scope_key`;
- balance lock by `account_id`;
- sign-up bonus grant uniqueness by `(account_id, signup_bonus_policy_version)`
  or an equivalent grant identity that proves one bonus per admitted account per
  policy;
- usage operation lookup by `usage_operation_id`;
- idempotency lookup by `(account_id, operation_kind, idempotency_key)`;
- ledger/support readback by `(account_id, created_at DESC, ledger_entry_id DESC)`;
- reconciliation worker claims by `(state, next_attempt_at, reconciliation_case_id)`;
- stale reservations by `(state, expires_at, hold_id)`.

Transaction rules:
- no cross-service calls inside DB transactions;
- one short transaction per money command where possible;
- use account balance row locking for account-level money invariants;
- use bounded lock timeout and classify lock timeout separately from idempotency conflict and validation rejection;
- Redis/in-memory caches may only collapse reads or reduce load; losing them must not change money correctness.

Expected bottleneck:
- the account balance row is the intentional contention point for high-concurrency commands against one account. This is accepted because it localizes the non-negative available-balance invariant. The design phase must size indexes and benchmark this path rather than weakening the invariant.

## Data-Model Test Obligations

The data-model design and later implementation ledger must carry these test classes:

- money amount parser/formatter vector tests;
- rounding-rule vector tests for reject, reserve-ceiling round-up, final-charge
  policy rounding, exact `$10.00` sign-up bonus atoms, and correction exactness;
- ledger conservation property tests over credits, reserves, releases, charges, write-offs, reversals, and corrections;
- non-negative available balance property and integration tests;
- idempotency replay/conflict tests for sign-up bonus grant, reserve, finalize,
  write-off, reversal/compensation, and reconciliation correction;
- concurrent reserve/finalize/write-off race tests against the account balance row and operation rows;
- PostgreSQL uniqueness and row-locking integration tests for idempotency keys,
  operation IDs, sign-up bonus grant IDs, qualified evidence IDs, terminal
  outcomes, active holds, and duplicate ledger effects;
- stale reservation and reconciliation tests;
- sign-up bonus uniqueness tests proving one grant per admitted account/policy and
  changed-fingerprint conflict without duplicate credit;
- payment evidence/versioning tests are not current-scope proof obligations unless
  a later specification reopen reintroduces customer payment/top-up behavior;
- benchmark targets for reserve/finalize/write-off hot paths, with per-command O(1) lookup proof and account-row contention measurement.

## Handoff To Technical Design

Historical data-model handoff: complete before the 2026-06-01 current
funding-source correction. `specs/billing-money-core/design/data-model.md` and
`specs/billing-money-core/design/event-ingestion-redpanda.md` are now stale for
current planning where they treat payment/top-up evidence ingestion as active
scope.

Next phase: technical design repair for current sign-up-bonus and usage-only
money scope.

The repair must update the affected technical design context to:
- add or repair sign-up bonus grant identity, ledger effect, idempotency,
  constraints, support readback, and tests;
- preserve reserve/finalize/write-off usage settlement behavior;
- demote customer payment/top-up evidence, payments-service writeback, and
  Redpanda payment-evidence ingestion to future/conditional context;
- mark the prior Redpanda event-ingestion addendum and follow-up technical design
  review as superseded for current planning until payment/top-up scope is
  explicitly reopened.

The repair must not create migrations, generated SQL, code, runtime adapters,
runtime event contract schemas, or `tasks.md`. A follow-up technical design
review is mandatory before planning.

## Outcome

Data-model implementation was completed for the earlier approved
billing-money-core slice as of 2026-06-01, before the current funding-source
correction.

Implemented surfaces:
- deterministic PostgreSQL up/down migration for the then-approved billing
  accounts, balances, ledger, idempotency, outcome, usage, top-up, payment
  evidence, reconciliation, audit, and legacy import tables;
- approved schema constraints, partial uniqueness, account-first lock support,
  support readback indexes, reconciliation claim indexes, legacy import linkage,
  and ledger immutability for posted money fields;
- SQLC query sources and generated access code for data-model access shapes only;
- fixed-scale USD atom helper tests, integration tests for constraints,
  idempotency, reconciliation, legacy import, concurrency, and benchmark
  evidence for the named hot paths.

Validated proof:
- `rtk make sqlc-check`;
- `rtk make migration-validate`;
- `rtk go test ./...`;
- `rtk make test-integration` plus fresh supplemental
  `rtk go test -tags=integration -count=1 ./test/...`;
- `rtk make test-race` plus fresh supplemental
  `rtk go test -race -count=1 ./...`;
- `rtk go test -tags=integration -run '^$' -bench 'BenchmarkBillingMoneyCore' ./test/...`;
- `rtk git diff --check`.

This outcome does not approve or implement HTTP/OpenAPI contracts, runtime
adapters, workers, bootstrap wiring, GNK inventory ownership, or any live
cross-service provider behavior outside the approved data-model slice.

Current funding-source correction approved at specification level as of
2026-06-01. It supersedes the Redpanda payment-evidence addendum for current
planning: customer payment/top-up is not implemented now, users cannot currently
add balance, and the only current balance-increase path is the `$10.00` sign-up
bonus. No payment-provider top-up, payments-service evidence handoff, Redpanda
payment-evidence ingestion, runtime event schema, migration, generated SQL,
adapter, worker, or task-ledger work is approved by the superseded payment
addendum for the current slice.
