# Data Model Delta

Status: repaired review-ready technical design for billing-issued spending leases
Consumes: `../spec.md`, historical `../../billing-money-core/design/data-model.md`,
historical `../../billing-money-core/design/event-ingestion-redpanda.md`

## Boundary

This artifact describes persisted-state deltas required by the lease
architecture. It does not write migration SQL, generated SQL, repository code,
or tests.

In scope:

- billing-owned spending leases and reserved exposure;
- child debit authorization lineage observed from terminal events;
- lease checkpoint/close state and release proof;
- billing event inbox/outbox;
- lease issuance admission-control state;
- event-originated lineage on idempotency/outcome/reconciliation rows;
- replay, quarantine, retry, retention, and migration shape.

Out of scope:

- payment/top-up runtime state, payment evidence ingestion, payment
  reversals/refunds, and customer top-up flows;
- direct per-request reserve or per-request `usage_holds` as target admission;
- proxy durable allocator schema details beyond required semantic fields and
  invariants;
- migration SQL, generated SQL, runtime adapters, tests, and implementation
  code.

## Current Baseline To Preserve

Current repository evidence already contains useful money-core primitives:

- `billing_accounts` and `account_balances` with
  `available_usd_atoms = settled_usd_atoms - reserved_usd_atoms`.
- `idempotency_records` and `operation_outcomes` with durable replay/conflict
  state.
- `usage_operations`, `usage_holds`, `usage_terminal_outcomes`,
  `qualified_inference_evidence`, `ledger_entries`, and
  `reconciliation_cases`.
- SQLC query sources for account lookup/locking, idempotency, usage operation
  locking, ledger insertion, terminal outcome insertion, and reconciliation
  claiming.

This design preserves the account-first lock order, fixed USD atom
representation, append-only ledger model, stored outcomes, and explicit
reconciliation. The existing `usage_holds` shape is historical/per-request
reserve context; it is not the target admission record for migrated lease
cohorts.

Current migration/query files do not define spending lease rows, child debit
settlement lineage, lease checkpoints, `billing_event_inbox`,
`billing_event_outbox`, or `billing_admission_controls`. Those are required
deltas for this workflow.

## Required Deltas

### `spending_leases`

Purpose: billing-owned, account-scoped, bounded, expiring lease authority whose
full USD exposure is reserved before proxy can spend it.

Required fields:

| Field | Notes |
| --- | --- |
| `spending_lease_id` | Billing lease identity. |
| `account_id`, `account_scope_key` | Account scope. |
| `proxy_lease_owner_id` | Stable proxy allocator owner such as `proxy:<environment>:<allocatorShardId>`. |
| `spending_lease_generation` / `lease_fence` | Fencing token. |
| `state` | `issued`, `active`, `closing`, `closed`, `expired`, `revoked`, `reconcile_required`, `manual_review`. |
| `issued_usd_atoms` | Total capacity reserved by billing for this lease generation. |
| `remaining_reserved_usd_atoms` | Capacity still reserved in billing after debit settlements and releases. |
| `settled_usd_atoms`, `released_usd_atoms`, `written_off_usd_atoms` | Aggregate terminal accounting. |
| `pricing_snapshot_id`, `pricing_snapshot_fingerprint` | Issuance pricing basis or policy proof. |
| `pricing_policy_version`, `lease_policy_version`, `use_class` | Policy constraints. |
| `idempotency_record_id`, `stored_outcome_id`, `hold_ledger_entry_id` | Durable command and ledger lineage. |
| `issued_at`, `last_checkpoint_at`, `debit_cutoff_at`, `expires_at`, `closed_at`, `updated_at` | Lifecycle times. |

Required constraints and indexes:

- primary key on `spending_lease_id`;
- foreign key `(account_id, account_scope_key)` to `billing_accounts`;
- unique `(account_id, proxy_lease_owner_id, spending_lease_generation)` or
  equivalent fence uniqueness;
- positive `issued_usd_atoms`;
- non-negative aggregate amounts;
- aggregate `settled_usd_atoms + released_usd_atoms + written_off_usd_atoms <=
  issued_usd_atoms`;
- `remaining_reserved_usd_atoms = issued_usd_atoms - settled/released/write-off
  exposure` or equivalent transactional invariant;
- account/state/expiry indexes for readback and reconciliation;
- owner/fence index for terminal event validation.

Issuance/replenishment transaction:

- lock account balance first;
- lock/create idempotency record;
- create or update lease state;
- insert ledger effect `spending_lease_hold` or `spending_lease_replenishment`
  that increases reserved USD and reduces available USD;
- write stored outcome and outbox fact;
- commit.

### `spending_lease_debit_settlements`

Purpose: billing's durable record of child debit lineage and terminal settlement
observed from proxy events. It validates proxy child evidence and prevents a
second money mutation. It is not the proxy hot-path allocator.

Required fields:

| Field | Notes |
| --- | --- |
| `debit_settlement_id` | Internal primary key. |
| `spending_lease_id`, `spending_lease_generation` / `lease_fence` | Parent lease authority. |
| `debit_authorization_id` | Proxy child debit identity. |
| `usage_operation_id` | Settlement identity for the paid attempt. |
| `account_id`, `account_scope_key` | Account scope. |
| `child_cap_usd_atoms` | Maximum customer charge for this child. |
| `charged_usd_atoms`, `released_usd_atoms`, `write_off_usd_atoms` | Terminal accounting. |
| `terminal_kind` | `finalize`, `write_off`, `reversal`, `compensation`, or conflict/reconciliation classification. |
| `operation_fingerprint`, `terminal_fingerprint` | Replay/conflict basis. |
| `pricing_snapshot_id`, `pricing_snapshot_fingerprint` | Original child pricing evidence. |
| `source_event_inbox_id`, `stored_outcome_id`, `ledger_entry_id`, `reconciliation_case_id` | Lineage. |
| `allocated_at`, `terminal_observed_at`, `settled_at`, `created_at`, `updated_at` | Times. |

Required constraints and indexes:

- unique `(spending_lease_id, debit_authorization_id)`;
- unique `usage_operation_id` for chargeable terminal paths;
- non-negative charge/release/write-off amounts;
- `charged_usd_atoms <= child_cap_usd_atoms`;
- child settlement sums cannot make aggregate valid lease charge exceed the
  billing-issued lease budget;
- changed fingerprint for same child or usage identity becomes conflict or
  reconciliation without a second money mutation;
- indexes by lease, usage operation, account recent, and reconciliation state.

### `spending_lease_checkpoints`

Purpose: durable billing receipt of proxy lease checkpoint/close evidence and
release decisions.

Required fields:

| Field | Notes |
| --- | --- |
| `checkpoint_id` | Internal primary key. |
| `spending_lease_id`, `spending_lease_generation` / `lease_fence` | Parent lease. |
| `proxy_lease_owner_id` | Authorized allocator. |
| `checkpoint_sequence` | Monotonic proxy sequence. |
| `allocated_child_cap_sum_usd_atoms` | Proxy-reported aggregate allocated cap. |
| `terminal_submitted_child_cap_sum_usd_atoms` | Proxy-reported terminal coverage. |
| `local_remaining_usd_atoms` | Proxy-reported local unused capacity. |
| `checkpoint_kind` | `progress`, `close_requested`, `cancel_requested`, `expired_scan`, `operator_repair`. |
| `checkpoint_fingerprint` | Replay basis. |
| `source_event_inbox_id`, `idempotency_record_id`, `stored_outcome_id`, `ledger_entry_id`, `reconciliation_case_id` | Lineage. |
| `received_at`, `processed_at`, `created_at` | Times. |

Rules:

- same lease/fence/sequence/fingerprint replays the stored outcome;
- changed fingerprint for the same sequence conflicts;
- close can release only capacity billing can prove is unallocated or
  terminally settled/released;
- incomplete proof keeps capacity reserved and opens reconciliation.

### `billing_event_inbox`

Purpose: durable receipt, replay, conflict, quarantine, retry, and stored
outcome for consumed Redpanda events before offset commit.

Required fields:

| Field | Notes |
| --- | --- |
| `event_inbox_id` | Internal primary key. |
| `event_id` | Producer event identity. Nullable only for poison receipts. |
| `event_receipt_identity` | `event:<topic>:<eventId>` or `offset:<topic>:<partition>:<offset>`. |
| `event_identity_basis` | `producer_event_id` or `broker_offset_receipt`. |
| `producer_authority` | Expected producer for topic. |
| `topic`, `partition_id`, `offset`, `consumer_group` | Broker receipt coordinates. |
| `event_schema_version`, `event_fingerprint` | Contract and replay basis. |
| `operation_kind` | `usage_finalize`, `usage_write_off`, `lease_checkpoint`, `lease_close`, `reconciliation_command`, `ignored`. |
| `operation_identity_type`, `operation_identity` | Business identity when valid. |
| `account_id`, `account_scope_key`, `spending_lease_id`, `debit_authorization_id`, `usage_operation_id` | Lineage when available. |
| `processing_state` | `received`, `processing`, `committed`, `duplicate_replay`, `conflict`, `waiting_dependency`, `retry_scheduled`, `quarantined`, `reconcile_required`, `ignored`. |
| `stored_outcome_id`, `reconciliation_case_id` | Outcome links. |
| `retry_count`, `last_error_class`, `last_error_safe_code`, `next_attempt_at` | Retry metadata; safe classes only. |
| `claim_owner`, `claim_generation`, `claim_deadline_at`, `claimed_at` | Inbox retry ownership. |
| `received_at`, `processed_at`, `created_at`, `updated_at` | Timestamps. |

Required constraints and indexes:

- unique `event_receipt_identity`;
- partial unique `(topic, event_id)` where `event_id IS NOT NULL`;
- unique `(topic, partition_id, offset)`;
- constrained topic, operation kind, identity type, and processing state sets;
- poison broker-offset receipts cannot mutate money;
- claim and retry indexes for worker recovery.

### `billing_event_outbox`

Purpose: transactional source for billing-emitted facts. Rows are written in the
same transaction as source lease, debit, ledger, outcome, reconciliation, or
inbox state.

Required fields:

| Field | Notes |
| --- | --- |
| `outbox_event_id` | Internal primary key. |
| `event_id` | Billing-produced stable event identity. |
| `topic`, `event_key`, `event_schema_version`, `event_fingerprint` | Broker contract fields. |
| `source_table`, `source_id` | Source row identity. |
| `account_id`, `account_scope_key`, `spending_lease_id`, `debit_authorization_id`, `usage_operation_id` | Optional lineage. |
| `state` | `pending`, `publishing`, `published`, `retry_scheduled`, `failed`. |
| `payload` | Bounded privacy-safe JSON payload. |
| `attempt_count`, `last_error_class`, `next_attempt_at` | Retry metadata. |
| `created_at`, `published_at`, `updated_at` | Timestamps. |

Payloads must not include raw prompts, completions, SSE chunks, tokens,
secrets, DSNs, payment payloads, provider bodies, or raw event dumps.

### `billing_admission_controls`

Purpose: Postgres-visible control state for new lease issuance/replenishment and
cohort paid-admission gates. This table is not money truth and does not
release, charge, or mutate existing leases.

Required fields:

| Field | Notes |
| --- | --- |
| `admission_control_id` | Internal primary key. |
| `control_key` | Current required value: `paid_usage_admission`. |
| `scope_type` | `global`, `account`, or `cohort` when rollout needs cohort controls. |
| `scope_key` | `global`, account scope, or rollout cohort key. |
| `state` | `open`, `throttle`, or `fail_closed`. Missing, invalid, or expired state is treated as `fail_closed`. |
| `reason` | Safe reason such as `healthy`, `terminal_lag_warning`, `terminal_lag_critical`, `stale_lease_budget_breach`, `stale_debit_budget_breach`, `worker_not_ready`, `operator_override`, or `recovery_validation`. |
| `source` | `worker_auto` or `operator_admin`. |
| `generation` | Monotonic writer generation. |
| `observed_terminal_lag_ms`, `observed_stale_lease_count`, `observed_stale_debit_count`, `observed_reconciliation_backlog_count` | Nullable safe observations. |
| `last_evaluated_at`, `valid_until` | Short lease for control health. |
| `details_fingerprint` | Optional fingerprint of safe diagnostics. |
| `created_at`, `updated_at` | Timestamps. |

Read path:

- `cmd/service` reads global plus applicable cohort/account rows inside lease
  issuance/replenishment before creating new reserved capacity.
- Missing, expired, malformed, `throttle`, or `fail_closed` rows reject without
  money mutation.
- Controls do not silently release existing leases and do not authorize proxy
  local balance writes.

Writer path:

- `cmd/billing-worker` renews `open` only when terminal lag, inbox/outbox
  backlog, stale lease/debit scans, and reconciliation eligibility are within
  accepted budgets.
- Operator/admin writes are protected, audited, and use the same table.
- Config supplies thresholds and startup defaults only.

### Lineage Columns And Existing Table Repairs

Planning should add lineage only where it is needed for support, replay, and
reconciliation:

- nullable `source_event_inbox_id` on `idempotency_records`;
- nullable `source_event_inbox_id` on `operation_outcomes`;
- nullable `source_event_inbox_id` on `usage_terminal_outcomes` if reused;
- nullable `event_inbox_id`, `spending_lease_id`, and `debit_authorization_id`
  on `reconciliation_cases`;
- optional lease/debit lineage on `ledger_entries` through columns or
  support-safe metadata, with a preference for explicit foreign keys for
  money-critical lineage.

Existing ledger enums must be extended for lease semantics, for example:

- `spending_lease_hold`;
- `spending_lease_replenishment`;
- `spending_lease_debit_charge`;
- `spending_lease_capacity_release`;
- `spending_lease_write_off`;
- `spending_lease_reversal`;
- `spending_lease_compensation`;
- `spending_lease_reconciliation_correction`.

Planning must verify exact enum names against migration style. The design
requirement is explicit effects for hold, charge, release, write-off, reversal,
compensation, and correction, not these spellings specifically.

## Event Processing Transaction Shape

Each terminal or checkpoint event gets one short transaction:

1. Insert or lock `billing_event_inbox` by event identity or poison receipt
   identity.
2. If duplicate same fingerprint, read stored inbox/business outcome and
   commit.
3. If changed fingerprint, store conflict/quarantine, optionally create
   reconciliation and outbox row, then commit.
4. Resolve account and lock `account_balances` before mutable lease/debit rows.
5. Lock parent `spending_leases` by lease ID and fence.
6. Lock or create event-originated idempotency record.
7. Insert or replay child debit settlement, checkpoint/close outcome, ledger
   entries, operation outcomes, audit rows when available, and outbox rows as
   needed.
8. Update balance, lease, idempotency, and inbox state.
9. Commit Postgres.
10. Commit Redpanda offset after database commit.

No outbound HTTP calls run inside this transaction.

## Migration Shape

Planning should use expand/verify/contract sequencing:

1. Expand: add spending lease, debit settlement, checkpoint, inbox/outbox,
   admission-control, lineage, indexes, constraints, and generated query inputs
   without enabling producers or consumers.
2. Expand: seed `billing_admission_controls` default global
   `paid_usage_admission = fail_closed` before any lease issuance traffic uses
   billing.
3. Verify: run migration validation, SQLC generation checks,
   constraint/invariant tests, lease issue/replenish replay/conflict tests,
   child debit cap tests, stale/fail-closed admission tests, and privacy-safe
   fixture checks.
4. Enable dark: deploy worker disabled or in validation-only mode if rollout
   chooses shadow mode; do not open lease issuance until worker or authorized
   operator renews a healthy admission-control lease.
5. Enable controlled cohorts only after proxy durable allocator, terminal
   submission, event contracts, and admission-control recovery checks are in
   place.
6. Contract: disable old proxy-local money writes and any compatibility bridge
   paths per `rollout.md`.

No data backfill is required for new inbox/outbox tables before enablement.
Existing proxy balances still require import/parity before migrated cohorts use
billing as writer.

## Retention And Privacy

- `spending_leases`, child debit settlements, ledger entries, outcomes,
  reconciliation, and audit rows remain money-audit records and must outlive
  hot event replay windows.
- `billing_event_inbox` and `billing_event_outbox` are retained at least
  through the hot replay windows in `contracts/redpanda-events.md`.
- Retention expiry must not remove records required to explain customer balance
  changes.
- Inbox/outbox payloads store bounded safe fields and fingerprints only.
- Quarantine rows store safe failure class, receipt identity, topic/partition/
  offset, and fingerprints, not raw payload.
- Metrics must not label by account ID, API key, request ID, event ID,
  inference ID, lease ID, debit authorization ID, payment evidence ID, or
  provider identifiers.
