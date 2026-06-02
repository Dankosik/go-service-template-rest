# Data Model Design

Status: review-ready
Trigger: persisted microlease state, child lineage projection, checkpoint/close
evidence, cache contract, replay behavior, migration shape, retention, and
privacy constraints.

This file is design context only. Runtime schema authority remains
`env/migrations/*.sql`, SQLC query files, and generated SQLC output after
implementation planning.

## Current Baseline

The current money-core migration already has:

- `billing_accounts`;
- `account_balances` with USD atom settled/reserved/available invariants;
- durable `idempotency_records` and `operation_outcomes`;
- per-request `usage_operations`, `usage_holds`, `usage_terminal_outcomes`;
- `qualified_inference_evidence`;
- top-up, payment evidence, ledger, legacy import, reconciliation, and audit
  primitives.

Microleases extend this model. They do not replace account balances, ledger
immutability, idempotency, operation outcomes, reconciliation cases, or audit
events.

## Billing Tables To Add Or Extend

### `spending_microleases`

Owns one billing-issued parent grant.

Required fields:

- `microlease_id`;
- `account_id`, `account_scope_key`;
- `proxy_allocator_owner_id`;
- `microlease_generation` / `lease_fence`;
- `state`: `active`, `cutoff`, `closing`, `closed`, `expired`,
  `reconcile_required`, `manual_review`, `canceled`;
- `issued_cap_usd_atoms`;
- `available_child_cap_usd_atoms`;
- `allocated_child_cap_reported_usd_atoms`;
- `terminal_charged_usd_atoms`;
- `terminal_released_usd_atoms`;
- `write_off_usd_atoms`;
- `pricing_snapshot_id`, `pricing_snapshot_fingerprint`, `pricing_policy_version`;
- `fee_policy_version`, `microlease_policy_version`;
- `issued_at`, `debit_cutoff_at`, `expires_at`, `closed_at`;
- `last_checkpoint_sequence`, `last_checkpoint_fingerprint`;
- `idempotency_record_id`, `stored_outcome_id`;
- safe metadata only.

Constraints:

- currency is USD by design; persisted amount columns use USD atoms;
- `issued_cap_usd_atoms > 0`;
- `issued_cap_usd_atoms <= configured cap at issuance`;
- `available_child_cap_usd_atoms >= 0`;
- charged/released/write-off totals are non-negative;
- terminal totals plus unresolved exposure must not exceed issued cap;
- owner/fence uniqueness prevents two active rows with the same grant identity;
- expiry is after issuance and debit cutoff is before expiry.

### `microlease_child_debits`

Billing-side terminal/settlement projection for proxy durable child debit
identity. The proxy local row remains the before-execution allocation authority;
this table is the billing settlement and audit projection after events arrive.

Required fields:

- `microlease_child_debit_id`;
- `microlease_id`;
- `debit_authorization_id`;
- `usage_operation_id` or billing-generated child usage identity;
- `account_id`, `account_scope_key`;
- `proxy_allocator_owner_id`, `microlease_generation`;
- `child_sequence`;
- `child_cap_usd_atoms`;
- `charged_usd_atoms`, `released_usd_atoms`, `write_off_usd_atoms`;
- `request_basis_fingerprint`;
- `terminal_basis_fingerprint`;
- `pricing_snapshot_id`, `pricing_snapshot_fingerprint`;
- `terminal_kind`: `finalize`, `write_off`, `abort_release`,
  `reversal`, `compensation`;
- `state`: `terminal_pending`, `finalized`, `written_off`, `released`,
  `reversed`, `reconcile_required`, `conflict`;
- `qualified_inference_evidence_id`;
- `terminal_event_id`, `terminal_inbox_id`, `ledger_entry_id`,
  `settlement_effect_id`;
- `created_at`, `terminal_at`, `settled_at`.

Constraints:

- unique `(microlease_id, debit_authorization_id)`;
- same child ID plus changed fingerprint is conflict;
- `charged_usd_atoms <= child_cap_usd_atoms`;
- aggregate accepted child caps for a parent must not exceed the parent issued
  cap; over-debit opens reconciliation and caps customer charge;
- terminal kind is single-path for finalize/write-off/release unless explicit
  reversal/compensation follows.

### `microlease_checkpoints`

Stores checkpoint and close evidence from proxy durable allocator state.

Required fields:

- `checkpoint_id`;
- `microlease_id`;
- `account_id`, `account_scope_key`;
- `proxy_allocator_owner_id`, `microlease_generation`;
- `checkpoint_sequence`;
- `checkpoint_kind`: `progress`, `cutoff`, `close`, `shutdown`, `repair`;
- `allocated_child_high_water`;
- `allocated_child_count`;
- `allocated_child_cap_sum_usd_atoms`;
- `terminal_submitted_count`;
- `terminal_published_count`;
- `terminal_accepted_count`;
- `unresolved_child_count`;
- `unresolved_child_cap_sum_usd_atoms`;
- `local_remaining_usd_atoms`;
- `checkpoint_fingerprint`;
- `inbox_id`, `created_at`, `applied_at`;

Release rule:

- billing may release only `issued_cap - allocated_child_cap_sum` after a valid
  close proof from the current owner/fence;
- unresolved child cap remains reserved until terminal or reconciliation;
- progress checkpoints can update lag/backpressure but cannot release on their
  own unless the close proof criteria are met.

### `billing_event_inbox`

Durable event consumer idempotency.

Required fields:

- topic, partition, offset, event ID, producer identity;
- business identity (`microlease_id`, `debit_authorization_id`,
  `checkpoint_sequence`, or close ID);
- event fingerprint;
- state: `received`, `applied`, `duplicate`, `conflict`, `quarantined`,
  `reconcile_required`;
- safe failure class and receipt metadata;
- received/applied timestamps.

Billing commits broker offsets only after the inbox state is durable and the
money effect or quarantine decision is committed.

### `billing_outbox`

Local DB outbox for billing facts produced from issuance, settlement, close,
reconciliation, and rejection outcomes.

Required fields:

- event ID and type;
- aggregate identity;
- event fingerprint;
- safe payload;
- state, attempt count, next attempt, published timestamp;
- trace correlation without raw payloads.

### `billing_admission_controls`

Billing-owned backpressure and operator controls used during microlease
issuance/replenishment.

Required fields:

- scope: global, account, allocator owner, use class;
- state: `open`, `throttle`, `strict`, `fail_closed`;
- reason code;
- expires/renewed timestamps;
- terminal lag bucket, stale age bucket, reconciliation backlog bucket;
- audited actor and safe metadata.

Missing, expired, malformed, or `fail_closed` controls deny new capacity.

## Ledger Effects

Planning must choose exact enum names, but the design requires distinct
microlease effects rather than hiding parent grant state inside per-request
`usage_holds`:

- microlease reserve: increases `reserved_usd_atoms` and decreases derived
  available;
- child final charge: decreases settled and releases reserved exposure up to the
  child cap;
- child release/write-off: releases reserved exposure without retroactive
  overcharge;
- close release: releases only proven unallocated capacity;
- reversal/compensation: explicit compensating ledger effects.

All effects keep append-only ledger behavior and non-negative available balance.

## Proxy Durable Data

Proxy must own durable local equivalents:

- microlease grant rows keyed by `microlease_id`, account, owner, generation,
  cap, remaining, cutoff, expiry, pricing basis, and stored billing outcome;
- child debit rows keyed by `debit_authorization_id`, parent microlease, child
  sequence, cap, request fingerprint, pricing fingerprint, state, and terminal
  obligation;
- terminal outbox rows with event identity, fingerprint, retry state, and safe
  references;
- checkpoint/close rows with allocator high-water and cap sums.

No proxy row is visible balance truth or billing ledger truth.

## Cache Contract

Process memory may cache:

- current grants;
- durable remaining capacity;
- cutoff/expiry;
- local backlog health;
- deny-only strict/fail-closed reason.

Cache loss or restart rebuilds from proxy durable state. A memory cache hit
does not authorize execution; durable child debit commit does.

Redis is absent from the first target. If introduced later, Redis must be
rebuildable from durable billing/proxy state and may only cache or deny. Redis
loss, timeout, failover, or split-brain must not create or lose spend authority.

## Migration Shape

Use `expand -> backfill/verify -> contract`:

1. Add billing microlease tables, enums/checks, indexes, inbox/outbox, and
   admission controls without enabling paid cohorts.
2. Add SQLC queries and repositories.
3. Add proxy durable grant/debit/terminal storage before any external execution
   uses microleases.
4. Run shadow/parity and no-dual-writer checks.
5. Enable microlease issuance only for gated cohorts.
6. Disable direct reserve fallback and old proxy-local money writes for migrated
   cohorts before declaring cutover.
7. Contract old compatibility paths only after migrated cohorts prove stable.

## Retention And Privacy

- Hot replay retention must cover terminal, checkpoint, close, idempotency, and
  inbox records long enough for retry storms and lag recovery.
- Audit/legal retention applies to ledger, reconciliation, and operation
  outcomes.
- Safe metadata must exclude raw prompts, completions, SSE chunks, bearer
  tokens, API keys, DSNs, payment secrets, raw provider payloads, raw event
  payloads, dynamic proof URLs, and sensitive request bodies.
