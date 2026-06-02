# Billing Money Core Data Model

Status: review-ready technical design after current sign-up-bonus repair
Owner: billing-service
Scope: production-ready current customer-money data model only
Consumes: `specs/billing-money-core/spec.md`

Repair scope: adds the current `$10.00` sign-up bonus grant model and demotes
customer payment/top-up evidence, payments-service writeback, and Redpanda
payment-evidence ingestion to future/conditional context after the approved
2026-06-01 funding-source correction. The historical TDR-F01/TDR-F02/TDR-F03
and REDPANDA-TDR-F01 repair context is preserved only where it remains useful
future context. This pass does not reopen API contracts, runtime adapters,
worker design, broader service architecture, migration SQL, generated code, or
implementation planning.

## Boundary

This artifact turns the approved data-model decisions into concrete PostgreSQL
tables, columns, constraints, indexes, transaction rules, readback shapes, import
compatibility surfaces, and data-model test obligations.

In scope:
- service-owned PostgreSQL truth for USD customer accounts, ledger entries,
  balances, holds, usage operations, sign-up bonus grants, durable idempotency,
  reconciliation cases, audit readback, and legacy imports;
- fixed-scale USD atom storage using `BIGINT`;
- concurrency rules for account-scoped money invariants;
- hot-path and support readback query shapes;
- historical/future-ready top-up and normalized payment evidence tables only as
  dormant context from the earlier data-model slice. They are not current
  balance-increase paths and must not drive current planning until payment/top-up
  scope is explicitly reopened.

Out of scope:
- migration SQL files;
- generated SQL access code;
- HTTP route names, OpenAPI payloads, or API response schemas;
- runtime adapters, workers, bootstrap wiring, or package architecture;
- customer payment/top-up product flow, payment provider sessions, normalized
  payment evidence application, payments-service writeback, and Redpanda
  payment-evidence ingestion for the current implementation slice;
- GNK treasury inventory schema.

## Modeling Conventions

### Storage And Type Defaults

- Primary datastore: billing-service-owned PostgreSQL OLTP tables.
- Internal surrogate IDs: `UUID`.
- Caller, provider, sibling-service, and legacy identifiers: `TEXT` with explicit
  authority/scope columns.
- Money amounts: signed `BIGINT` atom columns where
  `1 USD = 100,000,000 usd_atoms`.
- Currency: constrained `TEXT`; all customer-money rows require `currency = 'USD'`.
- States and kinds: constrained `TEXT` checks instead of PostgreSQL enum types so
  future compatible additions can use expand/validate/contract migration steps.
- Real instants: `TIMESTAMPTZ`.
- Mutable current-state rows carry `updated_at`; account balance also carries
  monotonic `version`.
- JSONB is allowed only for bounded support-safe adjunct snapshots, never for
  invariant-bearing IDs, money, states, or fingerprints.

### Identifier Separation

The schema deliberately separates identifiers that must not be reused:

| Identifier | Authority | Type | Use |
| --- | --- | --- | --- |
| `account_id` | billing | `UUID` | Internal joins and locking. |
| `account_scope_key` | billing contract | `TEXT` | O(1) caller lookup, formatted `user:<id>` or future `org:<id>`. |
| `subject_id` | `subject_authority` | `TEXT` | External identity subject reference. |
| `usage_operation_id` | billing | `UUID` | Usage settlement identity. |
| `client_usage_request_id` | caller | `TEXT` | Caller lineage/correlation for one usage attempt. |
| `request_id` | caller/transport | `TEXT` | Trace/correlation only; never unique settlement truth. |
| `inference_id` | execution/provider evidence | `TEXT` | Settlement evidence only when qualified by proof scope. |
| `signup_bonus_grant_id` | billing | `UUID` | Current balance-increase identity for the one-time sign-up bonus grant. |
| `admission_reference_id` | identity/account-admission authority | `TEXT` | Safe registration/account-admission lineage for the grant; exact runtime handoff contract is later. |
| `topup_operation_id` | billing | `UUID` | Dormant future product top-up lifecycle identity. Not current scope. |
| `payment_attempt_id` | payments-service/billing attempt lineage | `TEXT` | Dormant future attempt reference scoped to one top-up. |
| `payment_evidence_id` | normalized payment evidence authority | `TEXT` | Dormant future stable payments-owned evidence lineage. |
| `evidence_version` | normalized payment evidence authority | `BIGINT` | Dormant future monotonic version under `payment_evidence_id`; part of the historical/future billing evidence application selector. |
| `settlement_effect_id` | billing | `UUID` | Stable externally traceable money-effect reference. |
| `idempotency_key` | caller per operation | `TEXT` | Replay key scoped by account and operation kind. |

No cross-service foreign keys are allowed. External identifiers are relational
columns with source authority and uniqueness scope.

### Ledger Delta Semantics

`ledger_entries` is the append-only source for money-state effects. To make hold
and pending movement auditable without overloading a single signed amount, each
entry stores:

- `amount_usd_atoms`: signed canonical amount for the effect.
- `settled_delta_usd_atoms`: signed change to settled customer funds.
- `reserved_delta_usd_atoms`: signed change to active reserved funds.
- `pending_delta_usd_atoms`: signed change to non-spendable pending evidence.

Balance application rule:

```text
new_settled = old_settled + settled_delta_usd_atoms
new_reserved = old_reserved + reserved_delta_usd_atoms
new_pending = old_pending + pending_delta_usd_atoms
new_available = new_settled - new_reserved
```

`amount_usd_atoms` is:
- equal to `settled_delta_usd_atoms` for sign-up bonus credits, settled future
  credits, charges, reversals, corrections, imports, and payment reversals;
- equal to `reserved_delta_usd_atoms` for reservation-only effects;
- equal to `pending_delta_usd_atoms` for pending-evidence effects.

For finalization that charges and releases a hold in one effect,
`amount_usd_atoms` equals the settled charge delta, while
`reserved_delta_usd_atoms` releases the full consumed reservation. The after-state
columns on the ledger row are readback snapshots; the invariant source remains
the ledger row plus the transactional balance row.

## State Sets

These state values are constrained text. New states require an explicit schema
evolution decision and tests.

Closed constrained-text inventory:

| Table | Field | Constraint status |
| --- | --- | --- |
| `billing_accounts` | `account_type`, `state` | constrained in table checks. |
| `account_balances` | `currency` | constrained to `USD`. |
| `ledger_entries` | `currency`, `effect_type`, `created_by_kind` | constrained in table checks. |
| `idempotency_records` | `operation_kind`, `state`, `retention_class` | constrained in table checks. |
| `operation_outcomes` | `operation_kind`, `outcome_status`, `primary_resource_type` | constrained in table checks. |
| `usage_operations` | `state`, `operation_kind` | constrained in table checks. |
| `usage_holds` | `state` | constrained in table checks. |
| `usage_terminal_outcomes` | `terminal_kind` | constrained in table checks. |
| `signup_bonus_grants` | `state`, `currency` | constrained in table checks. |
| `topup_operations` | `state` | constrained in table checks; dormant future context, not current scope. |
| `payment_attempts` | `state` | constrained in table checks; dormant future context, not current scope. |
| `payment_evidence` | `state`, `evidence_kind`, `finality_class`, `rail_family` | constrained in table checks; dormant future context, not current scope. |
| `reconciliation_cases` | `reason`, `state`, `severity` | constrained in table checks. |
| `audit_events` | `actor_kind`, nullable `operation_kind` | constrained in table checks. |
| `legacy_import_batches` | `state` | constrained in table checks. |
| `legacy_balance_imports` | `parity_status` | constrained in table checks. |

`audit_events.before_state`, `audit_events.after_state`, reason/problem code
fields, source authority fields, and schema/version fields stay constrained by
their owning referenced row, producer contract, or support taxonomy rather than
one global `CHECK`, because they intentionally span multiple state sets or
externally versioned vocabularies.

### Account State

- `active`
- `suspended`
- `closed`
- `manual_review`

### Ledger Effect Type

Approved effect types from `spec.md`:
- `signup_bonus_credit`
- `usage_charge`
- `usage_hold`
- `usage_hold_release`
- `usage_write_off`
- `usage_reversal`
- `operator_adjustment`
- `migration_import`
- `reconciliation_correction`

Dormant future payment/top-up effect types retained only as historical/future
schema context from the earlier data-model slice:
- `topup_credit`
- `payment_reversal`
- `topup_pending`
- `topup_pending_release`

`topup_credit`, `payment_reversal`, `topup_pending`, and
`topup_pending_release` are not current-scope balance-increase or payment paths.
They must not be planned or implemented for current scope until payment/top-up
is explicitly reopened. `topup_pending` and `topup_pending_release` are not
spendable-money credits; they exist only to make future pending balance changes
durable and auditable.

### Sign-Up Bonus Grant State

- `pending`
- `credited`
- `conflict`
- `reconcile_required`
- `reversed`
- `manual_review`

### Usage Operation State

- `reserve_pending`
- `reserved`
- `finalize_pending`
- `finalized`
- `write_off_pending`
- `written_off`
- `reversed`
- `compensated`
- `reconcile_required`
- `manual_review`
- `expired`

### Usage Operation Kind

- `reserve`
- `finalize`
- `write_off`
- `reversal`
- `compensation`

### Hold State

- `active`
- `finalized`
- `released`
- `written_off`
- `expired`
- `reversed`
- `reconcile_required`
- `manual_review`

### Top-Up State

- `created`
- `payment_pending`
- `presentation_synced`
- `evidence_pending`
- `settlement_applied`
- `duplicate_evidence`
- `evidence_conflict`
- `late_evidence_review`
- `reversed`
- `expired`
- `manual_review`
- `reconcile_required`

### Idempotency State

- `started`
- `committed`
- `failed_stored`
- `conflict`
- `reconcile_required`

### Reconciliation Case State

- `open`
- `leased`
- `waiting_evidence`
- `manual_review`
- `resolved`
- `canceled`

## Tables

### `billing_accounts`

Canonical billable account scope. One row exists for each customer-money owner.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `account_id` | `UUID` | no | Primary key. |
| `account_scope_key` | `TEXT` | no | Stable key, `user:<identity_user_id>` now, `org:<organization_id>` later. |
| `account_type` | `TEXT` | no | `user` or `organization`. |
| `subject_authority` | `TEXT` | no | Owning authority, e.g. `identity-service`. |
| `subject_id` | `TEXT` | no | External subject identifier. |
| `state` | `TEXT` | no | Account state. |
| `version` | `BIGINT` | no | Monotonic current-state version for support/readback. |
| `created_at` | `TIMESTAMPTZ` | no | Creation instant. |
| `updated_at` | `TIMESTAMPTZ` | no | Last account-state update. |
| `closed_at` | `TIMESTAMPTZ` | yes | Required when `state = 'closed'`. |

Constraints:
- `PRIMARY KEY (account_id)`.
- `UNIQUE (account_scope_key)`.
- `UNIQUE (subject_authority, account_type, subject_id)`.
- `UNIQUE (account_id, account_scope_key)` to support child consistency FKs.
- `CHECK (account_type IN ('user', 'organization'))`.
- `CHECK (state IN ('active', 'suspended', 'closed', 'manual_review'))`.
- `CHECK ((state = 'closed') = (closed_at IS NOT NULL))`.
- `CHECK (version > 0)`.
- `CHECK ((account_type = 'user' AND account_scope_key = 'user:' || subject_id)
  OR (account_type = 'organization' AND account_scope_key = 'org:' || subject_id))`
  for the first target state. If future account-scope formatting changes, that
  is a specification change, not an implementation detail.

Indexes:
- Unique constraints above cover account resolution by `account_scope_key` and
  subject lookup by authority.
- `idx_billing_accounts_state_updated` on `(state, updated_at, account_id)` for
  support lists and manual-review queues.

### `account_balances`

One transactionally maintained balance read model per account. This row is the
intentional contention point for account-level money writes.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `account_id` | `UUID` | no | Primary key and FK to `billing_accounts`. |
| `account_scope_key` | `TEXT` | no | Duplicated stable lookup key. |
| `currency` | `TEXT` | no | Must be `USD`. |
| `settled_usd_atoms` | `BIGINT` | no | Posted funds after finalized effects. |
| `reserved_usd_atoms` | `BIGINT` | no | Active holds. |
| `available_usd_atoms` | `BIGINT` | no | `settled_usd_atoms - reserved_usd_atoms`. |
| `pending_usd_atoms` | `BIGINT` | no | Non-spendable unresolved evidence. |
| `version` | `BIGINT` | no | Monotonic balance mutation version. |
| `last_ledger_entry_id` | `UUID` | yes | Last money-state ledger row applied. |
| `updated_at` | `TIMESTAMPTZ` | no | Last balance mutation. |

Constraints:
- `PRIMARY KEY (account_id)`.
- `FOREIGN KEY (account_id, account_scope_key)` references
  `billing_accounts(account_id, account_scope_key)`.
- `FOREIGN KEY (last_ledger_entry_id)` references
  `ledger_entries(ledger_entry_id)`, nullable for newly created zero balances.
- `CHECK (currency = 'USD')`.
- `CHECK (settled_usd_atoms >= 0)`.
- `CHECK (reserved_usd_atoms >= 0)`.
- `CHECK (available_usd_atoms >= 0)`.
- `CHECK (pending_usd_atoms >= 0)`.
- `CHECK (available_usd_atoms = settled_usd_atoms - reserved_usd_atoms)`.
- `CHECK (version > 0)`.

Indexes:
- Primary key supports
  `SELECT account_id FROM account_balances WHERE account_id = $1 FOR UPDATE`.
- `idx_account_balances_scope` on `(account_scope_key)` for balance readback by
  caller scope.
- `idx_account_balances_updated` on `(updated_at DESC, account_id)` for support
  diagnostics.

### `ledger_entries`

Append-only money-state effect truth. Posted rows are immutable for money
columns; correction uses compensating entries.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `ledger_entry_id` | `UUID` | no | Primary key. |
| `account_id` | `UUID` | no | FK account. |
| `account_scope_key` | `TEXT` | no | Stable account scope at posting time. |
| `currency` | `TEXT` | no | Must be `USD`. |
| `effect_type` | `TEXT` | no | Ledger effect type. |
| `amount_usd_atoms` | `BIGINT` | no | Signed canonical effect amount. |
| `settled_delta_usd_atoms` | `BIGINT` | no | Signed settled delta. |
| `reserved_delta_usd_atoms` | `BIGINT` | no | Signed reserved delta. |
| `pending_delta_usd_atoms` | `BIGINT` | no | Signed pending delta. |
| `settled_after_usd_atoms` | `BIGINT` | no | Support-safe after snapshot. |
| `reserved_after_usd_atoms` | `BIGINT` | no | Support-safe after snapshot. |
| `available_after_usd_atoms` | `BIGINT` | no | Support-safe after snapshot. |
| `pending_after_usd_atoms` | `BIGINT` | no | Support-safe after snapshot. |
| `balance_version_after` | `BIGINT` | no | Balance version after applying entry. |
| `settlement_effect_id` | `UUID` | yes | Externally traceable effect reference. |
| `idempotency_record_id` | `UUID` | yes | Command idempotency lineage. |
| `usage_operation_id` | `UUID` | yes | Usage lineage. |
| `signup_bonus_grant_id` | `UUID` | yes | Sign-up bonus grant lineage for current bonus credits. |
| `topup_operation_id` | `UUID` | yes | Dormant future top-up lineage. |
| `payment_attempt_id` | `TEXT` | yes | Dormant future payment attempt lineage. |
| `payment_evidence_id` | `TEXT` | yes | Dormant future normalized evidence lineage. |
| `payment_evidence_version` | `BIGINT` | yes | Dormant future versioned evidence claim when the ledger entry points to a specific normalized evidence version. |
| `qualified_inference_evidence_id` | `UUID` | yes | Qualified inference evidence lineage. |
| `reversal_of_ledger_entry_id` | `UUID` | yes | Original ledger row being reversed. |
| `correction_of_ledger_entry_id` | `UUID` | yes | Original ledger row being corrected. |
| `effective_at` | `TIMESTAMPTZ` | no | Business/effect instant. |
| `created_at` | `TIMESTAMPTZ` | no | Processing/posting instant. |
| `created_by_kind` | `TEXT` | no | `service`, `worker`, `operator`, `migration`. |
| `reason_code` | `TEXT` | no | Support-safe reason. |
| `safe_metadata` | `JSONB` | yes | Bounded adjunct data only; no raw prompts, tokens, secrets, or raw webhooks. |

Constraints:
- `PRIMARY KEY (ledger_entry_id)`.
- `FOREIGN KEY (account_id, account_scope_key)` references
  `billing_accounts(account_id, account_scope_key)`.
- FKs to local lineage tables when the referenced local table exists.
- `FOREIGN KEY (signup_bonus_grant_id)` references
  `signup_bonus_grants(signup_bonus_grant_id)` when present.
- `FOREIGN KEY (payment_evidence_id, payment_evidence_version)` references
  `payment_evidence(payment_evidence_id, evidence_version)` when both fields are
  present.
- `CHECK ((payment_evidence_id IS NULL) = (payment_evidence_version IS NULL))`.
- `CHECK (currency = 'USD')`.
- `CHECK (amount_usd_atoms <> 0)`.
- `CHECK (settled_after_usd_atoms >= 0)`.
- `CHECK (reserved_after_usd_atoms >= 0)`.
- `CHECK (available_after_usd_atoms >= 0)`.
- `CHECK (pending_after_usd_atoms >= 0)`.
- `CHECK (available_after_usd_atoms = settled_after_usd_atoms - reserved_after_usd_atoms)`.
- `CHECK (balance_version_after > 0)`.
- `CHECK (effect_type IN ('signup_bonus_credit', 'usage_charge', 'usage_hold', 'usage_hold_release', 'usage_write_off', 'usage_reversal', 'operator_adjustment', 'migration_import', 'reconciliation_correction', 'topup_credit', 'payment_reversal', 'topup_pending', 'topup_pending_release'))`.
- `CHECK (created_by_kind IN ('service', 'worker', 'operator', 'migration'))`.
- Delta-pattern checks:
  - `signup_bonus_credit`, `topup_credit`, `migration_import`: `amount_usd_atoms = settled_delta_usd_atoms`
    and `settled_delta_usd_atoms > 0` and
    `reserved_delta_usd_atoms = 0` and `pending_delta_usd_atoms = 0`.
  - `usage_hold`: `amount_usd_atoms = reserved_delta_usd_atoms` and
    `reserved_delta_usd_atoms > 0` and `settled_delta_usd_atoms = 0`
    and `pending_delta_usd_atoms = 0`.
  - `usage_hold_release`, `usage_write_off`: `amount_usd_atoms = reserved_delta_usd_atoms`
    and `reserved_delta_usd_atoms < 0` and `settled_delta_usd_atoms = 0`
    and `pending_delta_usd_atoms = 0`.
  - `usage_charge`: `amount_usd_atoms = settled_delta_usd_atoms` and
    `settled_delta_usd_atoms < 0` and `reserved_delta_usd_atoms <= 0`
    and `pending_delta_usd_atoms = 0`.
  - `topup_pending`: `amount_usd_atoms = pending_delta_usd_atoms` and
    `pending_delta_usd_atoms > 0` and `settled_delta_usd_atoms = 0`
    and `reserved_delta_usd_atoms = 0`.
  - `topup_pending_release`: `amount_usd_atoms = pending_delta_usd_atoms` and
    `pending_delta_usd_atoms < 0` and `settled_delta_usd_atoms = 0`
    and `reserved_delta_usd_atoms = 0`.
  - `usage_reversal`, `payment_reversal`, `operator_adjustment`,
    `reconciliation_correction`: `amount_usd_atoms = settled_delta_usd_atoms`
    and `reserved_delta_usd_atoms = 0` and `pending_delta_usd_atoms = 0`
    unless a future approved spec adds a narrower reserved/pending correction
    rule.
  - Every effect type must match exactly one approved pattern. Any balance
    component not named as mutable by that pattern is explicitly zero, so
    `usage_charge` cannot mutate `pending_usd_atoms`.

Indexes:
- `idx_ledger_account_recent` on
  `(account_id, created_at DESC, ledger_entry_id DESC)` for support readback and
  keyset pagination.
- `idx_ledger_scope_recent` on
  `(account_scope_key, created_at DESC, ledger_entry_id DESC)` for caller-scope
  readback after account resolution.
- `idx_ledger_usage` on `(usage_operation_id, created_at, ledger_entry_id)`
  where `usage_operation_id IS NOT NULL`.
- `idx_ledger_signup_bonus` on
  `(signup_bonus_grant_id, created_at, ledger_entry_id)` where
  `signup_bonus_grant_id IS NOT NULL`.
- `idx_ledger_topup` on `(topup_operation_id, created_at, ledger_entry_id)`
  where `topup_operation_id IS NOT NULL`.
- `idx_ledger_payment_evidence_version` on
  `(payment_evidence_id, payment_evidence_version, ledger_entry_id)` where
  `payment_evidence_id IS NOT NULL` and
  `payment_evidence_version IS NOT NULL`.
- `idx_ledger_payment_evidence_lineage` on
  `(payment_evidence_id, ledger_entry_id)` where `payment_evidence_id IS NOT NULL`
  for support readback across all versions in one evidence lineage.
- `UNIQUE (settlement_effect_id)` where `settlement_effect_id IS NOT NULL`.
- `idx_ledger_reversal_links` on
  `(reversal_of_ledger_entry_id, correction_of_ledger_entry_id)` for support
  traceability.

Immutability rule:
- after insert, money columns, effect type, deltas, after snapshots, account IDs,
  lineage IDs, and timestamps are not updated by normal runtime paths;
- support metadata amendment, when later implemented, must write an audit row and
  cannot change money fields.

### `idempotency_records`

Durable replay and conflict guard for every money-affecting command.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `idempotency_record_id` | `UUID` | no | Primary key. |
| `account_id` | `UUID` | no | Account scope. |
| `operation_kind` | `TEXT` | no | Money-affecting semantic operation. |
| `idempotency_key` | `TEXT` | no | Caller-provided replay key. |
| `request_fingerprint` | `TEXT` | no | Canonical operation fingerprint. |
| `state` | `TEXT` | no | Idempotency state. |
| `stored_outcome_id` | `UUID` | yes | Replay-stable outcome. |
| `conflict_reason` | `TEXT` | yes | Safe conflict classifier. |
| `retention_class` | `TEXT` | no | `hot_replay`, `audit`, `legal_hold`, `expired_ready`. |
| `first_seen_at` | `TIMESTAMPTZ` | no | First request time. |
| `last_seen_at` | `TIMESTAMPTZ` | no | Last replay/conflict time. |
| `committed_at` | `TIMESTAMPTZ` | yes | Set for committed/stored-failure outcome. |
| `expires_at` | `TIMESTAMPTZ` | yes | Only set after replay safety window is complete. |

Constraints:
- `PRIMARY KEY (idempotency_record_id)`.
- `FOREIGN KEY (account_id)` references `billing_accounts(account_id)`.
- `UNIQUE (account_id, operation_kind, idempotency_key)`.
- `CHECK (state IN ('started', 'committed', 'failed_stored', 'conflict', 'reconcile_required'))`.
- `CHECK (operation_kind IN ('signup_bonus_grant', 'reserve', 'finalize', 'write_off', 'reversal', 'compensation', 'operator_adjustment', 'migration_import', 'reconciliation_correction', 'topup_create', 'topup_presentation_sync', 'topup_evidence', 'payment_reversal'))`.
- `CHECK (retention_class IN ('hot_replay', 'audit', 'legal_hold', 'expired_ready'))`.
- `CHECK ((state IN ('committed', 'failed_stored')) = (stored_outcome_id IS NOT NULL))`.
- `CHECK ((state = 'conflict') = (conflict_reason IS NOT NULL))`.
- `CHECK (expires_at IS NULL OR expires_at > committed_at)`.

Indexes:
- Unique key above is the hot-path lookup.
- `idx_idempotency_account_recent` on
  `(account_id, last_seen_at DESC, idempotency_record_id DESC)` for support
  history.
- `idx_idempotency_expiry` on `(retention_class, expires_at, idempotency_record_id)`
  where `expires_at IS NOT NULL`.

Replay rule:
- same `(account_id, operation_kind, idempotency_key)` plus same
  `request_fingerprint` returns `stored_outcome_id` when committed or stored
  failure;
- same key plus changed fingerprint transitions to or reads `conflict` and does
  not mutate money;
- for `operation_kind = 'signup_bonus_grant'`, the `idempotency_key` and
  `request_fingerprint` include `account_id`, `signup_bonus_grant_id`,
  `signup_bonus_policy_version`, `admission_authority`,
  `admission_reference_id`, and `grant_amount_usd_atoms`;
- for `operation_kind = 'topup_evidence'`, the `idempotency_key` and
  `request_fingerprint` include
  `(payment_evidence_id, evidence_version)` plus the versioned evidence
  fingerprint. This is dormant future context; bare `payment_evidence_id` is not
  a valid replay selector if payment/top-up scope is later reopened;
- ambiguous timeouts retry with the same operation identity or route to
  reconciliation.

### `operation_outcomes`

Replay-stable safe outcome records. These are not HTTP response contracts; they
store enough relational references and safe adjunct data to return the same
semantic outcome later.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `stored_outcome_id` | `UUID` | no | Primary key. |
| `idempotency_record_id` | `UUID` | no | Unique source idempotency row. |
| `account_id` | `UUID` | no | Account scope. |
| `operation_kind` | `TEXT` | no | Mirrors idempotency kind. |
| `outcome_status` | `TEXT` | no | `success`, `stored_failure`, `conflict`, `reconcile_required`. |
| `primary_resource_type` | `TEXT` | no | `signup_bonus_grant`, `usage_operation`, `topup_operation`, `payment_evidence`, `ledger_entry`, `reconciliation_case`. Top-up/payment resources are dormant future context. |
| `primary_resource_id` | `TEXT` | no | String form of the primary resource ID. For current sign-up bonus grants, this is `signup_bonus_grant_id`. For future `payment_evidence`, this is the versioned selector `payment_evidence_id:v<evidence_version>`. |
| `ledger_entry_id` | `UUID` | yes | Primary ledger effect, when one exists. |
| `settlement_effect_id` | `UUID` | yes | External effect reference, when one exists. |
| `failure_class` | `TEXT` | yes | Safe failure classifier. |
| `safe_problem_code` | `TEXT` | yes | Replay-safe problem code, not raw payload. |
| `safe_outcome` | `JSONB` | yes | Bounded, privacy-safe adjunct fields. |
| `created_at` | `TIMESTAMPTZ` | no | Outcome creation time. |

Constraints:
- `PRIMARY KEY (stored_outcome_id)`.
- `UNIQUE (idempotency_record_id)`.
- `FOREIGN KEY (idempotency_record_id)` references
  `idempotency_records(idempotency_record_id)`.
- `CHECK (operation_kind IN ('signup_bonus_grant', 'reserve', 'finalize', 'write_off', 'reversal', 'compensation', 'operator_adjustment', 'migration_import', 'reconciliation_correction', 'topup_create', 'topup_presentation_sync', 'topup_evidence', 'payment_reversal'))`.
- `CHECK (outcome_status IN ('success', 'stored_failure', 'conflict', 'reconcile_required'))`.
- `CHECK (primary_resource_type IN ('signup_bonus_grant', 'usage_operation', 'topup_operation', 'payment_evidence', 'ledger_entry', 'reconciliation_case'))`.
- `CHECK (safe_outcome IS NULL OR jsonb_typeof(safe_outcome) = 'object')`.

Indexes:
- `idx_operation_outcomes_account_recent` on
  `(account_id, created_at DESC, stored_outcome_id DESC)`.
- `idx_operation_outcomes_resource` on
  `(primary_resource_type, primary_resource_id, stored_outcome_id)`.

### `usage_operations`

Lifecycle authority for one paid usage attempt.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `usage_operation_id` | `UUID` | no | Primary key and settlement identity. |
| `account_id` | `UUID` | no | Account scope. |
| `account_scope_key` | `TEXT` | no | Stable account key. |
| `state` | `TEXT` | no | Usage operation state. |
| `operation_kind` | `TEXT` | no | Current or initiating operation kind. |
| `client_usage_request_id` | `TEXT` | no | Caller lineage; not settlement truth. |
| `request_id` | `TEXT` | yes | Correlation only. |
| `request_basis_fingerprint` | `TEXT` | no | Reserve input fingerprint. |
| `terminal_basis_fingerprint` | `TEXT` | yes | Finalize/write-off/reversal fingerprint. |
| `pricing_snapshot_id` | `TEXT` | no | Pricing lineage. |
| `pricing_snapshot_fingerprint` | `TEXT` | no | Pricing evidence fingerprint. |
| `quote_expires_at` | `TIMESTAMPTZ` | no | Reserve quote expiry. |
| `fee_policy_version` | `TEXT` | no | Fee policy lineage. |
| `reserve_policy_version` | `TEXT` | no | Reserve policy lineage. |
| `qualified_inference_evidence_id` | `UUID` | yes | Qualified inference evidence when available. |
| `terminal_outcome_id` | `UUID` | yes | Terminal outcome row. |
| `settlement_effect_id` | `UUID` | yes | Primary external settlement effect. |
| `created_at` | `TIMESTAMPTZ` | no | Created time. |
| `updated_at` | `TIMESTAMPTZ` | no | Last state update. |
| `reserved_at` | `TIMESTAMPTZ` | yes | Reserve success time. |
| `terminal_at` | `TIMESTAMPTZ` | yes | Terminal transition time. |

Constraints:
- `PRIMARY KEY (usage_operation_id)`.
- `FOREIGN KEY (account_id, account_scope_key)` references
  `billing_accounts(account_id, account_scope_key)`.
- `CHECK (state IN ('reserve_pending', 'reserved', 'finalize_pending', 'finalized', 'write_off_pending', 'written_off', 'reversed', 'compensated', 'reconcile_required', 'manual_review', 'expired'))`.
- `CHECK (operation_kind IN ('reserve', 'finalize', 'write_off', 'reversal', 'compensation'))`.
- `CHECK ((state IN ('finalized', 'written_off', 'reversed', 'compensated', 'expired')) = (terminal_at IS NOT NULL))`.
- `CHECK (quote_expires_at > created_at)`.

Indexes:
- Primary key supports finalize/write-off lookup by `usage_operation_id`.
- `idx_usage_operations_account_recent` on
  `(account_id, created_at DESC, usage_operation_id DESC)` for support lists.
- `idx_usage_operations_client_request` on
  `(account_id, client_usage_request_id, usage_operation_id)`.
- `idx_usage_operations_state_updated` on
  `(state, updated_at, usage_operation_id)` for reconciliation scans.
- `idx_usage_operations_request_id` on `(request_id, usage_operation_id)` where
  `request_id IS NOT NULL`; this is diagnostic only, never uniqueness.
- `UNIQUE (settlement_effect_id)` where `settlement_effect_id IS NOT NULL`.

### `usage_holds`

Durable reservation row. One hold lineage exists per usage operation.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `hold_id` | `UUID` | no | Primary key. |
| `usage_operation_id` | `UUID` | no | Unique usage lineage. |
| `account_id` | `UUID` | no | Account scope. |
| `account_scope_key` | `TEXT` | no | Stable account key. |
| `state` | `TEXT` | no | Hold state. |
| `reserved_usd_atoms` | `BIGINT` | no | Authorized reserved ceiling. |
| `released_usd_atoms` | `BIGINT` | no | Released reserve amount. |
| `charged_usd_atoms` | `BIGINT` | no | Customer-settled charge amount. |
| `write_off_usd_atoms` | `BIGINT` | no | Explicit non-customer-charged overrun or failure amount. |
| `pricing_snapshot_id` | `TEXT` | no | Pricing lineage. |
| `pricing_snapshot_fingerprint` | `TEXT` | no | Pricing evidence fingerprint. |
| `quote_expires_at` | `TIMESTAMPTZ` | no | Expiry instant. |
| `fee_policy_version` | `TEXT` | no | Fee policy lineage. |
| `reserve_policy_version` | `TEXT` | no | Reserve policy lineage. |
| `client_usage_request_id` | `TEXT` | no | Caller lineage. |
| `request_basis_fingerprint` | `TEXT` | no | Reserve input fingerprint. |
| `created_at` | `TIMESTAMPTZ` | no | Created time. |
| `updated_at` | `TIMESTAMPTZ` | no | Last state update. |
| `expires_at` | `TIMESTAMPTZ` | no | Hold expiry for stale reservation scans. |
| `terminal_at` | `TIMESTAMPTZ` | yes | Terminal transition time. |

Constraints:
- `PRIMARY KEY (hold_id)`.
- `UNIQUE (usage_operation_id)`.
- `FOREIGN KEY (usage_operation_id)` references
  `usage_operations(usage_operation_id)`.
- `FOREIGN KEY (account_id, account_scope_key)` references
  `billing_accounts(account_id, account_scope_key)`.
- `CHECK (state IN ('active', 'finalized', 'released', 'written_off', 'expired', 'reversed', 'reconcile_required', 'manual_review'))`.
- `CHECK (reserved_usd_atoms > 0)`.
- `CHECK (released_usd_atoms >= 0)`.
- `CHECK (charged_usd_atoms >= 0)`.
- `CHECK (write_off_usd_atoms >= 0)`.
- `CHECK (released_usd_atoms + charged_usd_atoms <= reserved_usd_atoms)`.
- `CHECK ((state IN ('finalized', 'released', 'written_off', 'expired', 'reversed')) = (terminal_at IS NOT NULL))`.
- `CHECK (expires_at = quote_expires_at)`.

Indexes:
- `idx_usage_holds_account_state` on
  `(account_id, state, updated_at DESC, hold_id DESC)` for support readback.
- `idx_usage_holds_stale` on `(state, expires_at, hold_id)` where
  `state = 'active'` for stale reservation discovery.
- `idx_usage_holds_operation` on `(usage_operation_id, hold_id)`.

Invariant:
- `account_balances.reserved_usd_atoms` equals the sum of
  `reserved_usd_atoms - released_usd_atoms - charged_usd_atoms` for active holds
  plus any approved reserved correction effects.

### `usage_terminal_outcomes`

Terminal outcome record for finalize, write-off, reversal, or compensation.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `usage_terminal_outcome_id` | `UUID` | no | Primary key. |
| `usage_operation_id` | `UUID` | no | Usage lineage. |
| `terminal_kind` | `TEXT` | no | `finalize`, `write_off`, `reversal`, `compensation`. |
| `idempotency_record_id` | `UUID` | no | Replay lineage. |
| `stored_outcome_id` | `UUID` | no | Stored outcome. |
| `ledger_entry_id` | `UUID` | yes | Primary terminal ledger effect. |
| `settlement_effect_id` | `UUID` | yes | External traceable effect ID. |
| `charged_usd_atoms` | `BIGINT` | no | Customer charge, if any. |
| `released_usd_atoms` | `BIGINT` | no | Released reserve, if any. |
| `write_off_usd_atoms` | `BIGINT` | no | Explicit write-off/compensation amount. |
| `created_at` | `TIMESTAMPTZ` | no | Terminal outcome time. |

Constraints:
- `PRIMARY KEY (usage_terminal_outcome_id)`.
- `FOREIGN KEY (usage_operation_id)` references
  `usage_operations(usage_operation_id)`.
- `UNIQUE (idempotency_record_id)`.
- `UNIQUE (stored_outcome_id)`.
- `UNIQUE (settlement_effect_id)` where `settlement_effect_id IS NOT NULL`.
- `CHECK (terminal_kind IN ('finalize', 'write_off', 'reversal', 'compensation'))`.
- `CHECK (charged_usd_atoms >= 0)`.
- `CHECK (released_usd_atoms >= 0)`.
- `CHECK (write_off_usd_atoms >= 0)`.
- partial `UNIQUE (usage_operation_id)` where
  `terminal_kind IN ('finalize', 'write_off')` to prevent both terminal paths.
- partial `UNIQUE (usage_operation_id, terminal_kind)` where
  `terminal_kind IN ('finalize', 'write_off')`.

Indexes:
- `idx_usage_terminal_operation_recent` on
  `(usage_operation_id, created_at DESC, usage_terminal_outcome_id DESC)`.

### `qualified_inference_evidence`

Local proof row for `inferenceId` only when it is qualified by provider family,
verification surface, and declared proof scope.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `qualified_inference_evidence_id` | `UUID` | no | Primary key. |
| `usage_operation_id` | `UUID` | no | Usage lineage. |
| `provider_family` | `TEXT` | no | Provider family/source. |
| `verification_surface` | `TEXT` | no | Evidence surface used for qualification. |
| `proof_scope` | `TEXT` | no | Declared uniqueness scope. |
| `inference_id` | `TEXT` | no | Qualified inference identifier. |
| `evidence_fingerprint` | `TEXT` | no | Safe canonical evidence fingerprint. |
| `observed_at` | `TIMESTAMPTZ` | no | Evidence event instant if known. |
| `created_at` | `TIMESTAMPTZ` | no | Processing instant. |

Constraints:
- `PRIMARY KEY (qualified_inference_evidence_id)`.
- `FOREIGN KEY (usage_operation_id)` references
  `usage_operations(usage_operation_id)`.
- `UNIQUE (provider_family, verification_surface, proof_scope, inference_id)`.
- `UNIQUE (evidence_fingerprint)`.

Indexes:
- `idx_inference_evidence_usage` on `(usage_operation_id, created_at DESC)`.

### `signup_bonus_grants`

Current one-time balance-increase path. A row represents the service-granted
`$10.00` sign-up bonus for one admitted billing account and one policy version.
It is not a top-up, payment, operator adjustment, or migration import.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `signup_bonus_grant_id` | `UUID` | no | Billing-owned stable grant identity. |
| `account_id` | `UUID` | no | Account scope. |
| `account_scope_key` | `TEXT` | no | Stable account key. |
| `subject_authority` | `TEXT` | no | Owning subject authority, currently `identity-service`. |
| `subject_id` | `TEXT` | no | External subject identifier. |
| `admission_authority` | `TEXT` | no | Authority that admitted the account or registration. |
| `admission_reference_id` | `TEXT` | no | Safe registration/account-admission lineage reference. Exact runtime contract is later. |
| `signup_bonus_policy_version` | `TEXT` | no | Immutable policy lineage for the grant. |
| `currency` | `TEXT` | no | Must be `USD`. |
| `grant_amount_usd_atoms` | `BIGINT` | no | Exact current grant amount, `1,000,000,000` atoms for `$10.00`. |
| `grant_fingerprint` | `TEXT` | no | Canonical fingerprint over account, subject, admission reference, policy version, currency, and amount. |
| `state` | `TEXT` | no | Grant state. |
| `idempotency_record_id` | `UUID` | yes | Replay lineage. |
| `stored_outcome_id` | `UUID` | yes | Stored outcome for replay. |
| `ledger_entry_id` | `UUID` | yes | Posted `signup_bonus_credit` ledger row. |
| `settlement_effect_id` | `UUID` | yes | Externally traceable grant credit effect. |
| `created_at` | `TIMESTAMPTZ` | no | Grant row creation time. |
| `updated_at` | `TIMESTAMPTZ` | no | Last state update. |
| `credited_at` | `TIMESTAMPTZ` | yes | Required when state is `credited`. |
| `reversed_at` | `TIMESTAMPTZ` | yes | Set when the grant has been reversed. |

Constraints:
- `PRIMARY KEY (signup_bonus_grant_id)`.
- `FOREIGN KEY (account_id, account_scope_key)` references
  `billing_accounts(account_id, account_scope_key)`.
- `FOREIGN KEY (idempotency_record_id)` references
  `idempotency_records(idempotency_record_id)` when present.
- `FOREIGN KEY (stored_outcome_id)` references
  `operation_outcomes(stored_outcome_id)` when present.
- `FOREIGN KEY (ledger_entry_id)` references `ledger_entries(ledger_entry_id)`
  when present.
- `UNIQUE (account_id, signup_bonus_policy_version)`.
- `UNIQUE (subject_authority, subject_id, signup_bonus_policy_version)`.
- `UNIQUE (admission_authority, admission_reference_id, signup_bonus_policy_version)`.
- `UNIQUE (grant_fingerprint)`.
- `UNIQUE (idempotency_record_id)` where `idempotency_record_id IS NOT NULL`.
- `UNIQUE (stored_outcome_id)` where `stored_outcome_id IS NOT NULL`.
- `UNIQUE (ledger_entry_id)` where `ledger_entry_id IS NOT NULL`.
- `UNIQUE (settlement_effect_id)` where `settlement_effect_id IS NOT NULL`.
- `CHECK (currency = 'USD')`.
- `CHECK (grant_amount_usd_atoms = 1000000000)` for the current approved
  `$10.00` sign-up bonus policy. A later product policy amount requires a
  specification reopen and compatible schema evolution.
- `CHECK (state IN ('pending', 'credited', 'conflict', 'reconcile_required', 'reversed', 'manual_review'))`.
- `CHECK ((state IN ('credited', 'reversed')) = (credited_at IS NOT NULL))`.
- `CHECK ((state = 'reversed') = (reversed_at IS NOT NULL))`.
- `CHECK (state NOT IN ('credited', 'reversed') OR ledger_entry_id IS NOT NULL)`.

Indexes:
- `idx_signup_bonus_account_recent` on
  `(account_id, created_at DESC, signup_bonus_grant_id DESC)`.
- `idx_signup_bonus_admission` on
  `(admission_authority, admission_reference_id, signup_bonus_grant_id)`.
- `idx_signup_bonus_state_updated` on
  `(state, updated_at, signup_bonus_grant_id)` where
  `state IN ('conflict', 'reconcile_required', 'manual_review')`.

Application invariant:
- the same account cannot receive the same `signup_bonus_policy_version` twice;
- duplicate delivery with the same grant fingerprint returns the stored outcome;
- duplicate delivery for the same account or admission reference with a changed
  fingerprint creates or reads a `signup_bonus_conflict` reconciliation case and
  cannot post another credit;
- the linked `signup_bonus_credit` ledger entry amount and settled delta must
  equal `grant_amount_usd_atoms`;
- a posted sign-up bonus credit is corrected only by an explicit reversal or
  reconciliation correction ledger effect linked to the original grant and
  ledger row.

### `topup_operations`

Dormant future product-level top-up lifecycle state from the earlier data-model
slice. Billing does not currently apply customer top-up credit; this section is
future/conditional context only until payment/top-up scope is explicitly
reopened.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `topup_operation_id` | `UUID` | no | Primary key. |
| `account_id` | `UUID` | no | Account scope. |
| `account_scope_key` | `TEXT` | no | Stable account key. |
| `state` | `TEXT` | no | Top-up state. |
| `accepted_quote_id` | `TEXT` | no | Accepted quote lineage. |
| `credited_usd_atoms` | `BIGINT` | no | Amount that may become settled credit. |
| `deposit_fee_usd_atoms` | `BIGINT` | no | Billing deposit fee. |
| `payin_amount_value` | `TEXT` | yes | Safe normalized pay-in display/evidence value. |
| `payin_currency` | `TEXT` | yes | Evidence currency, not customer ledger currency. |
| `pricing_snapshot_id` | `TEXT` | no | Pricing lineage. |
| `pricing_snapshot_fingerprint` | `TEXT` | no | Pricing/evidence fingerprint. |
| `settlement_policy_version` | `TEXT` | no | Settlement policy lineage. |
| `billing_fee_policy_version` | `TEXT` | no | Billing fee policy lineage. |
| `current_payment_attempt_id` | `TEXT` | yes | Current attempt reference. |
| `attempt_generation` | `INTEGER` | no | Current attempt generation. |
| `presentation_version` | `INTEGER` | no | Current presentation version. |
| `presentation_fingerprint` | `TEXT` | yes | Safe presentation fingerprint. |
| `settlement_effect_id` | `UUID` | yes | Applied credit/reversal effect. |
| `created_at` | `TIMESTAMPTZ` | no | Created time. |
| `updated_at` | `TIMESTAMPTZ` | no | Last state update. |
| `expires_at` | `TIMESTAMPTZ` | yes | Top-up expiry. |
| `settlement_applied_at` | `TIMESTAMPTZ` | yes | Credit application time. |

Constraints:
- `PRIMARY KEY (topup_operation_id)`.
- `FOREIGN KEY (account_id, account_scope_key)` references
  `billing_accounts(account_id, account_scope_key)`.
- `CHECK (state IN ('created', 'payment_pending', 'presentation_synced', 'evidence_pending', 'settlement_applied', 'duplicate_evidence', 'evidence_conflict', 'late_evidence_review', 'reversed', 'expired', 'manual_review', 'reconcile_required'))`.
- `CHECK (credited_usd_atoms > 0)`.
- `CHECK (deposit_fee_usd_atoms >= 0)`.
- `CHECK (attempt_generation >= 0)`.
- `CHECK (presentation_version >= 0)`.
- `CHECK (state <> 'settlement_applied' OR settlement_applied_at IS NOT NULL)`.
- `CHECK (settlement_applied_at IS NULL OR state IN ('settlement_applied', 'duplicate_evidence', 'reversed', 'manual_review', 'reconcile_required'))`.
- `UNIQUE (settlement_effect_id)` where `settlement_effect_id IS NOT NULL`.

Indexes:
- `idx_topup_account_recent` on
  `(account_id, created_at DESC, topup_operation_id DESC)`.
- `idx_topup_state_updated` on `(state, updated_at, topup_operation_id)`.
- `idx_topup_current_attempt` on
  `(topup_operation_id, current_payment_attempt_id)` where
  `current_payment_attempt_id IS NOT NULL`.

### `payment_attempts`

Dormant future attempt lineage under a top-up. Provider integration truth
remains in payments-service, and no current planning may consume this table as a
balance-increase path.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `payment_attempt_row_id` | `UUID` | no | Primary key. |
| `topup_operation_id` | `UUID` | no | Top-up lineage. |
| `payment_attempt_id` | `TEXT` | no | Attempt reference. |
| `attempt_generation` | `INTEGER` | no | Generation under top-up. |
| `state` | `TEXT` | no | `created`, `presentation_synced`, `evidence_pending`, `evidence_accepted`, `expired`, `conflict`, `manual_review`. |
| `presentation_version` | `INTEGER` | no | Version of current presentation. |
| `presentation_fingerprint` | `TEXT` | yes | Safe presentation fingerprint. |
| `created_at` | `TIMESTAMPTZ` | no | Created time. |
| `updated_at` | `TIMESTAMPTZ` | no | Last update time. |
| `expires_at` | `TIMESTAMPTZ` | yes | Attempt expiry. |

Constraints:
- `PRIMARY KEY (payment_attempt_row_id)`.
- `FOREIGN KEY (topup_operation_id)` references
  `topup_operations(topup_operation_id)`.
- `UNIQUE (topup_operation_id, payment_attempt_id)`.
- `UNIQUE (topup_operation_id, attempt_generation)`.
- `CHECK (state IN ('created', 'presentation_synced', 'evidence_pending', 'evidence_accepted', 'expired', 'conflict', 'manual_review'))`.
- `CHECK (attempt_generation >= 0)`.
- `CHECK (presentation_version >= 0)`.

Indexes:
- `idx_payment_attempts_topup_recent` on
  `(topup_operation_id, attempt_generation DESC)`.
- `idx_payment_attempts_id` on `(payment_attempt_id, payment_attempt_row_id)`.

### `payment_evidence_lineages`

Dormant future stable payments-owned normalized evidence lineage as accepted by
billing. This row prevents one `payment_evidence_id` from rebinding to another
top-up, attempt, or account scope while allowing append-only evidence versions
under the same lineage when payment/top-up scope is explicitly reopened.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `payment_evidence_id` | `TEXT` | no | Stable payments-owned normalized evidence lineage. |
| `topup_operation_id` | `UUID` | no | Top-up lineage permanently bound to this evidence lineage. |
| `payment_attempt_id` | `TEXT` | no | Attempt reference permanently bound to this evidence lineage. |
| `account_id` | `UUID` | no | Account scope. |
| `account_scope_key` | `TEXT` | no | Stable account key. |
| `latest_evidence_version` | `BIGINT` | no | Highest accepted or recorded version for support readback; not the replay selector by itself. |
| `created_at` | `TIMESTAMPTZ` | no | First billing receipt time for this lineage. |
| `updated_at` | `TIMESTAMPTZ` | no | Last version or state update. |

Constraints:
- `PRIMARY KEY (payment_evidence_id)`.
- `FOREIGN KEY (topup_operation_id)` references
  `topup_operations(topup_operation_id)`.
- `FOREIGN KEY (topup_operation_id, payment_attempt_id)` references
  `payment_attempts(topup_operation_id, payment_attempt_id)`.
- `FOREIGN KEY (account_id, account_scope_key)` references
  `billing_accounts(account_id, account_scope_key)`.
- `CHECK (latest_evidence_version > 0)`.

Indexes:
- `idx_payment_evidence_lineage_topup` on
  `(topup_operation_id, payment_evidence_id)`.
- `idx_payment_evidence_lineage_account_recent` on
  `(account_id, updated_at DESC, payment_evidence_id DESC)`.

Application invariant:
- the same `payment_evidence_id` must never rebind to another
  `topup_operation_id`, `payment_attempt_id`, `account_id`, or
  `account_scope_key`;
- `latest_evidence_version` is readback convenience only. Replay and money
  application always use `(payment_evidence_id, evidence_version)`.

### `payment_evidence`

Dormant future normalized payment evidence as accepted by billing. No raw PSP
webhooks, payment secrets, or full provider payloads are stored here. This table
is not a current planning input for the sign-up-bonus and usage-only scope.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `payment_evidence_row_id` | `UUID` | no | Internal primary key for one versioned evidence claim. |
| `payment_evidence_id` | `TEXT` | no | Stable normalized evidence lineage. |
| `evidence_version` | `BIGINT` | no | Required monotonic version under `payment_evidence_id`. |
| `topup_operation_id` | `UUID` | no | Top-up lineage. |
| `payment_attempt_id` | `TEXT` | no | Attempt reference. |
| `account_id` | `UUID` | no | Account scope. |
| `account_scope_key` | `TEXT` | no | Stable account key. |
| `state` | `TEXT` | no | `accepted`, `duplicate`, `conflict`, `late_review`, `reversed`, `reconcile_required`, `manual_review`. |
| `evidence_payload_fingerprint` | `TEXT` | no | Canonical normalized evidence fingerprint. |
| `evidence_kind` | `TEXT` | no | Normalized evidence kind. |
| `schema_version` | `TEXT` | no | Evidence schema version. |
| `finality_class` | `TEXT` | no | Evidence finality class. |
| `rail_family` | `TEXT` | no | Payment rail family. |
| `settlement_amount_usd_atoms` | `BIGINT` | no | Trusted normalized USD settlement amount. |
| `settlement_effect_id` | `UUID` | yes | Ledger credit/reversal effect. |
| `ledger_entry_id` | `UUID` | yes | Applied ledger row. |
| `prior_payment_evidence_id` | `TEXT` | yes | Reversal/adjustment lineage. |
| `prior_evidence_version` | `BIGINT` | yes | Prior versioned claim when reversal/adjustment points to a specific evidence version. |
| `received_at` | `TIMESTAMPTZ` | no | Evidence receipt time. |
| `provider_event_at` | `TIMESTAMPTZ` | yes | Provider event time if supplied. |
| `created_at` | `TIMESTAMPTZ` | no | Processing time. |

Constraints:
- `PRIMARY KEY (payment_evidence_row_id)`.
- `UNIQUE (payment_evidence_id, evidence_version)`.
- `FOREIGN KEY (payment_evidence_id)` references
  `payment_evidence_lineages(payment_evidence_id)`.
- `FOREIGN KEY (topup_operation_id)` references
  `topup_operations(topup_operation_id)`.
- `FOREIGN KEY (topup_operation_id, payment_attempt_id)` references
  `payment_attempts(topup_operation_id, payment_attempt_id)`.
- `FOREIGN KEY (account_id, account_scope_key)` references
  `billing_accounts(account_id, account_scope_key)`.
- `UNIQUE (evidence_payload_fingerprint)`.
- `UNIQUE (settlement_effect_id)` where `settlement_effect_id IS NOT NULL`.
- `FOREIGN KEY (prior_payment_evidence_id, prior_evidence_version)` references
  `payment_evidence(payment_evidence_id, evidence_version)` when both fields are
  present.
- `CHECK (evidence_version > 0)`.
- `CHECK ((prior_payment_evidence_id IS NULL) = (prior_evidence_version IS NULL))`.
- `CHECK (settlement_amount_usd_atoms > 0)`.
- `CHECK (state IN ('accepted', 'duplicate', 'conflict', 'late_review', 'reversed', 'reconcile_required', 'manual_review'))`.
- `CHECK (evidence_kind IN ('settlement', 'duplicate_notice', 'reversal', 'refund', 'adjustment'))`.
- `CHECK (finality_class IN ('final', 'provisional', 'reversed', 'disputed'))`.
- `CHECK (rail_family IN ('card', 'bank_transfer', 'crypto', 'internal', 'manual'))`.

Indexes:
- `idx_payment_evidence_selector` on
  `(payment_evidence_id, evidence_version)`.
- `idx_payment_evidence_lineage_versions` on
  `(payment_evidence_id, evidence_version DESC, payment_evidence_row_id DESC)`.
- `idx_payment_evidence_topup_recent` on
  `(topup_operation_id, created_at DESC, payment_evidence_id DESC, evidence_version DESC)`.
- `idx_payment_evidence_account_recent` on
  `(account_id, created_at DESC, payment_evidence_id DESC, evidence_version DESC)`.
- `idx_payment_evidence_state` on
  `(state, created_at, payment_evidence_id, evidence_version)`.

Application invariant:
- same `(payment_evidence_id, evidence_version)` and same fingerprint returns
  the stored outcome;
- same `(payment_evidence_id, evidence_version)` with changed fingerprint is an
  evidence conflict;
- a different `evidence_version` under the same lineage is a new immutable
  evidence claim for evaluation, not a rewrite of a prior accepted version;
- same fingerprint with another evidence ID is duplicate evidence and cannot
  credit twice;
- reversal/refund creates a new ledger effect linked to the original evidence and
  ledger row through the versioned prior-evidence selector when applicable.

### `reconciliation_cases`

Durable repair work records. Cases classify ambiguity; they are not an alternate
money source of truth.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `reconciliation_case_id` | `UUID` | no | Primary key. |
| `account_id` | `UUID` | no | Account scope. |
| `reason` | `TEXT` | no | Case reason. |
| `state` | `TEXT` | no | Case state. |
| `severity` | `TEXT` | no | `low`, `medium`, `high`, `critical`. |
| `signup_bonus_grant_id` | `UUID` | yes | Sign-up bonus grant lineage. |
| `usage_operation_id` | `UUID` | yes | Usage lineage. |
| `topup_operation_id` | `UUID` | yes | Dormant future top-up lineage. |
| `payment_attempt_id` | `TEXT` | yes | Dormant future payment attempt lineage. |
| `payment_evidence_id` | `TEXT` | yes | Dormant future evidence lineage. |
| `payment_evidence_version` | `BIGINT` | yes | Dormant future evidence version when the case is tied to one normalized evidence claim. |
| `settlement_effect_id` | `UUID` | yes | Settlement effect lineage. |
| `qualified_inference_evidence_id` | `UUID` | yes | Inference evidence lineage. |
| `ledger_entry_id` | `UUID` | yes | Primary ledger row. |
| `legacy_balance_import_id` | `UUID` | yes | Legacy import row lineage for import mismatch cases. |
| `resolution_ledger_entry_id` | `UUID` | yes | Money-changing resolution row. |
| `resolution_settlement_effect_id` | `UUID` | yes | Resolution effect ID. |
| `lease_owner` | `TEXT` | yes | Worker/operator lease owner. |
| `lease_deadline_at` | `TIMESTAMPTZ` | yes | Lease deadline. |
| `attempt_count` | `INTEGER` | no | Claim/repair attempts. |
| `next_attempt_at` | `TIMESTAMPTZ` | no | Claim eligibility. |
| `support_safe_notes` | `TEXT` | yes | No raw sensitive payloads. |
| `created_at` | `TIMESTAMPTZ` | no | Created time. |
| `updated_at` | `TIMESTAMPTZ` | no | Last update time. |
| `resolved_at` | `TIMESTAMPTZ` | yes | Resolution time. |

Constraints:
- `PRIMARY KEY (reconciliation_case_id)`.
- `FOREIGN KEY (account_id)` references `billing_accounts(account_id)`.
- local FKs for non-null local lineage references.
- `FOREIGN KEY (signup_bonus_grant_id)` references
  `signup_bonus_grants(signup_bonus_grant_id)` when present.
- `FOREIGN KEY (payment_evidence_id, payment_evidence_version)` references
  `payment_evidence(payment_evidence_id, evidence_version)` when both fields are
  present.
- `CHECK ((payment_evidence_id IS NULL) = (payment_evidence_version IS NULL))`
  for reasons that require a specific evidence claim.
- `CHECK (reason IN ('stale_reservation', 'ambiguous_terminal_state', 'missing_inference_evidence', 'signup_bonus_conflict', 'legacy_import_mismatch', 'operator_adjustment_required', 'duplicate_payment_evidence', 'evidence_conflict', 'late_payment_evidence', 'provider_reference_mismatch'))`.
- `CHECK (state IN ('open', 'leased', 'waiting_evidence', 'manual_review', 'resolved', 'canceled'))`.
- `CHECK (severity IN ('low', 'medium', 'high', 'critical'))`.
- `CHECK (attempt_count >= 0)`.
- `CHECK ((state = 'leased') = (lease_owner IS NOT NULL AND lease_deadline_at IS NOT NULL))`.
- `CHECK ((state = 'resolved') = (resolved_at IS NOT NULL))`.
- Required lineage checks by reason:
  - `stale_reservation`, `ambiguous_terminal_state`, and
    `missing_inference_evidence` require `usage_operation_id`.
  - `signup_bonus_conflict` requires `signup_bonus_grant_id`.
  - `duplicate_payment_evidence`, `evidence_conflict`, and
    `late_payment_evidence` require
    `(payment_evidence_id, payment_evidence_version)` when tied to a specific
    normalized evidence claim.
  - `provider_reference_mismatch` requires
    `(topup_operation_id, payment_attempt_id)` or `settlement_effect_id`.
  - `legacy_import_mismatch` requires `legacy_balance_import_id`.
  - `operator_adjustment_required` requires `ledger_entry_id` or
    `settlement_effect_id` when the case is tied to an existing money effect;
    account-level operator queues are allowed to omit those keys because
    multiple independent operator adjustments can be open for one account.

Indexes:
- `idx_reconciliation_claim` on
  `(state, next_attempt_at, reconciliation_case_id)` where
  `state IN ('open', 'waiting_evidence')`.
- `idx_reconciliation_leases` on
  `(state, lease_deadline_at, reconciliation_case_id)` where `state = 'leased'`.
- `idx_reconciliation_account_recent` on
  `(account_id, created_at DESC, reconciliation_case_id DESC)`.
- partial uniqueness to prevent duplicate open cases by reason and lineage:
  - `(reason, usage_operation_id)` where `reason IN ('stale_reservation', 'ambiguous_terminal_state', 'missing_inference_evidence')`
    and `usage_operation_id IS NOT NULL` and `state NOT IN ('resolved', 'canceled')`;
  - `(reason, signup_bonus_grant_id)` where
    `reason = 'signup_bonus_conflict'` and `signup_bonus_grant_id IS NOT NULL`
    and `state NOT IN ('resolved', 'canceled')`;
  - `(reason, topup_operation_id, payment_attempt_id)` where
    `reason = 'provider_reference_mismatch'` and `topup_operation_id IS NOT NULL`
    and `payment_attempt_id IS NOT NULL` and `state NOT IN ('resolved', 'canceled')`;
  - `(reason, payment_evidence_id, payment_evidence_version)` where
    `reason IN ('duplicate_payment_evidence', 'evidence_conflict', 'late_payment_evidence')`
    and `payment_evidence_id IS NOT NULL` and `payment_evidence_version IS NOT NULL`
    and `state NOT IN ('resolved', 'canceled')`;
  - `(reason, settlement_effect_id)` where
    `reason IN ('ambiguous_terminal_state', 'provider_reference_mismatch', 'operator_adjustment_required')`
    and `settlement_effect_id IS NOT NULL` and `state NOT IN ('resolved', 'canceled')`;
  - `(reason, qualified_inference_evidence_id)` where
    `qualified_inference_evidence_id IS NOT NULL` and
    `state NOT IN ('resolved', 'canceled')`;
  - `(reason, ledger_entry_id)` where
    `reason IN ('operator_adjustment_required', 'legacy_import_mismatch', 'ambiguous_terminal_state')`
    and `ledger_entry_id IS NOT NULL` and `state NOT IN ('resolved', 'canceled')`;
  - `(reason, legacy_balance_import_id)` where
    `reason = 'legacy_import_mismatch'` and `legacy_balance_import_id IS NOT NULL` and
    `state NOT IN ('resolved', 'canceled')`.

Duplicate-open-case mapping:
- `stale_reservation`: dedupe by `usage_operation_id`.
- `ambiguous_terminal_state`: dedupe by `usage_operation_id`; if the ambiguity
  is attached only to an emitted effect, also dedupe by `settlement_effect_id`.
- `signup_bonus_conflict`: dedupe by `signup_bonus_grant_id`. The grant row is
  the durable conflict anchor even when the conflict is discovered from a
  duplicate account-admission delivery before any extra credit is posted.
- `duplicate_payment_evidence`, `evidence_conflict`, and
  `late_payment_evidence`: dedupe by the versioned selector
  `(payment_evidence_id, payment_evidence_version)` when the case concerns one
  normalized evidence claim. These are dormant future payment/top-up reasons and
  are not current planning requirements. Top-up and payment attempt lineage
  remain readback context, and lineage-wide support views can still group by
  `payment_evidence_id`.
- `provider_reference_mismatch`: dedupe by
  `(topup_operation_id, payment_attempt_id)` for attempt-scoped provider
  mismatch, or by `settlement_effect_id` for effect-scoped mismatch.
- Top-up-only lineage is intentionally not a duplicate-open-case key for the
  approved reasons. Create paths must use the stronger payment-attempt,
  payment-evidence, settlement-effect, or legacy-import key so unrelated
  top-up issues do not collapse into one case.
- `missing_inference_evidence`: dedupe by `usage_operation_id`; once qualified
  evidence exists, any conflicting evidence-specific case dedupes by
  `qualified_inference_evidence_id`.
- `legacy_import_mismatch`: dedupe by `legacy_balance_import_id`.
- `operator_adjustment_required`: dedupe by `ledger_entry_id` or
  `settlement_effect_id` when the adjustment targets a concrete effect.
  Account-level operator adjustment queues intentionally do not use a database
  uniqueness rule beyond account/readback indexes because multiple unrelated
  operator issues may be open for one account.

Resolution invariant:
- any resolution that changes customer money creates an explicit
  `ledger_entries` row and links it through `resolution_ledger_entry_id`.

### `audit_events`

Support-safe audit trail for manual action, metadata amendment, reconciliation,
state repair, and migration actions. Audit rows are explanatory, not money truth.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `audit_event_id` | `UUID` | no | Primary key. |
| `account_id` | `UUID` | yes | Account context when applicable. |
| `actor_kind` | `TEXT` | no | `service`, `worker`, `operator`, `migration`. |
| `actor_id` | `TEXT` | no | Service principal/operator ID. |
| `reason_code` | `TEXT` | no | Safe reason. |
| `operation_kind` | `TEXT` | yes | Related operation kind. |
| `signup_bonus_grant_id` | `UUID` | yes | Sign-up bonus grant reference. |
| `usage_operation_id` | `UUID` | yes | Usage reference. |
| `topup_operation_id` | `UUID` | yes | Dormant future top-up reference. |
| `payment_evidence_id` | `TEXT` | yes | Dormant future evidence reference. |
| `payment_evidence_version` | `BIGINT` | yes | Dormant future evidence version when the audit row references one normalized evidence claim. |
| `ledger_entry_id` | `UUID` | yes | Ledger reference. |
| `reconciliation_case_id` | `UUID` | yes | Reconciliation reference. |
| `before_state` | `TEXT` | yes | Safe prior state name. |
| `after_state` | `TEXT` | yes | Safe new state name. |
| `amount_usd_atoms` | `BIGINT` | yes | Duplicated readback amount only. |
| `request_id` | `TEXT` | yes | Correlation only. |
| `trace_id` | `TEXT` | yes | Trace correlation. |
| `safe_metadata` | `JSONB` | yes | Bounded adjunct data only. |
| `created_at` | `TIMESTAMPTZ` | no | Event time. |

Constraints:
- `PRIMARY KEY (audit_event_id)`.
- local FKs for non-null local references.
- `CHECK (actor_kind IN ('service', 'worker', 'operator', 'migration'))`.
- `CHECK (operation_kind IS NULL OR operation_kind IN ('signup_bonus_grant', 'reserve', 'finalize', 'write_off', 'reversal', 'compensation', 'operator_adjustment', 'migration_import', 'reconciliation_correction', 'topup_create', 'topup_presentation_sync', 'topup_evidence', 'payment_reversal'))`.
- `CHECK (safe_metadata IS NULL OR jsonb_typeof(safe_metadata) = 'object')`.

Indexes:
- `idx_audit_account_recent` on
  `(account_id, created_at DESC, audit_event_id DESC)` where
  `account_id IS NOT NULL`.
- `idx_audit_signup_bonus` on
  `(signup_bonus_grant_id, created_at, audit_event_id)` where
  `signup_bonus_grant_id IS NOT NULL`.
- `idx_audit_ledger` on `(ledger_entry_id, audit_event_id)` where
  `ledger_entry_id IS NOT NULL`.
- `idx_audit_reconciliation` on
  `(reconciliation_case_id, created_at, audit_event_id)` where
  `reconciliation_case_id IS NOT NULL`.

### `legacy_import_batches`

Batch-level legacy import evidence and parity status.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `legacy_import_batch_id` | `UUID` | no | Primary key. |
| `source_system` | `TEXT` | no | Legacy source, initially `gonka-proxy`. |
| `source_snapshot_fingerprint` | `TEXT` | no | Snapshot fingerprint. |
| `state` | `TEXT` | no | `loaded`, `parity_checked`, `applied`, `failed`, `superseded`. |
| `account_count` | `BIGINT` | no | Batch count. |
| `derived_total_usd_atoms` | `BIGINT` | no | Batch derived USD atom total. |
| `created_at` | `TIMESTAMPTZ` | no | Created time. |
| `updated_at` | `TIMESTAMPTZ` | no | Last update time. |
| `applied_at` | `TIMESTAMPTZ` | yes | Applied time. |

Constraints:
- `PRIMARY KEY (legacy_import_batch_id)`.
- `UNIQUE (source_system, source_snapshot_fingerprint)`.
- `CHECK (state IN ('loaded', 'parity_checked', 'applied', 'failed', 'superseded'))`.
- `CHECK (account_count >= 0)`.
- `CHECK (derived_total_usd_atoms >= 0)`.

Indexes:
- `idx_legacy_import_batches_state` on
  `(state, created_at DESC, legacy_import_batch_id DESC)`.

### `legacy_balance_imports`

Per-account legacy balance evidence. These rows never participate in live
balance calculation after cutover.

| Column | Type | Null | Notes |
| --- | --- | --- | --- |
| `legacy_balance_import_id` | `UUID` | no | Primary key. |
| `legacy_import_batch_id` | `UUID` | no | Batch. |
| `account_id` | `UUID` | no | Target billing account. |
| `account_scope_key` | `TEXT` | no | Target account scope. |
| `legacy_source_system` | `TEXT` | no | Initially `gonka-proxy`. |
| `legacy_subject_id` | `TEXT` | no | Legacy user/account identifier. |
| `legacy_balance_ngonka_text` | `TEXT` | no | Exact source display/value text. |
| `legacy_locked_rate_usd_text` | `TEXT` | no | Exact source rate text. |
| `derived_usd_atoms` | `BIGINT` | no | Parsed/imported USD atom amount. |
| `import_fingerprint` | `TEXT` | no | Row-level import fingerprint. |
| `parity_status` | `TEXT` | no | `pending`, `matched`, `mismatch`, `corrected`, `ignored`. |
| `migration_ledger_entry_id` | `UUID` | yes | Linked `migration_import` ledger entry. |
| `correction_ledger_entry_id` | `UUID` | yes | Linked correction entry if needed. |
| `created_at` | `TIMESTAMPTZ` | no | Created time. |
| `updated_at` | `TIMESTAMPTZ` | no | Last parity update. |

Constraints:
- `PRIMARY KEY (legacy_balance_import_id)`.
- `FOREIGN KEY (legacy_import_batch_id)` references
  `legacy_import_batches(legacy_import_batch_id)`.
- `FOREIGN KEY (account_id, account_scope_key)` references
  `billing_accounts(account_id, account_scope_key)`.
- `UNIQUE (legacy_import_batch_id, legacy_source_system, legacy_subject_id)`.
- `UNIQUE (legacy_import_batch_id, account_id)`.
- `UNIQUE (import_fingerprint)`.
- `CHECK (derived_usd_atoms >= 0)`.
- `CHECK (parity_status IN ('pending', 'matched', 'mismatch', 'corrected', 'ignored'))`.

Indexes:
- `idx_legacy_imports_account` on
  `(account_id, created_at DESC, legacy_balance_import_id DESC)`.
- `idx_legacy_imports_parity` on
  `(legacy_import_batch_id, parity_status, legacy_balance_import_id)`.

Compatibility rule:
- `balanceNgonka` and `lockedRateUsd` stay import evidence only;
- live balance readback uses `account_balances` and `ledger_entries`;
- proxy-local writes must be disabled for migrated scopes before billing is
  declared writer.
- `legacy_import_mismatch` reconciliation cases link the specific
  `legacy_balance_import_id` so duplicate unresolved import-mismatch cases are
  deduped by the import row instead of by account or support note text.

## Transaction And Concurrency Rules

### Isolation Level

Default money-command transactions use PostgreSQL `READ COMMITTED` plus explicit
row locks and unique constraints. The non-negative account-balance invariant is
localized to one `account_balances` row. Stronger isolation is reserved for a
future named anomaly with retry tests; it is not the default.

### Lock Order

All money mutations follow this order:

1. Resolve or read the target account/operation/evidence outside the invariant
   lock only to discover `account_id`.
2. Start one short transaction.
3. Lock the account balance row:
   `SELECT account_id FROM account_balances WHERE account_id = $1 FOR UPDATE`.
4. Lock or insert the `idempotency_records` row for the semantic command.
5. Lock operation/hold/grant/evidence rows by stable key and primary key:
   `signup_bonus_grants`, `usage_operations`, `usage_holds`,
   `reconciliation_cases`, and, for dormant future payment scope only,
   `topup_operations` or `payment_evidence`.
6. Insert ledger entries, terminal outcomes, stored outcomes, audit rows, and
   update current-state rows.
7. Commit.

Read-only idempotency probes may happen before the transaction for fast replay,
but any mutation path must lock the account balance before locking mutable
operation, grant, or evidence rows. This preserves the approved account-first
lock order.

### Timeout And Retry Classification

- Account row lock timeout: `account_contention_timeout`.
- Idempotency uniqueness conflict with same fingerprint: replay.
- Idempotency uniqueness conflict with changed fingerprint: idempotency conflict.
- Deadlock or serialization error: retryable transaction failure with bounded
  retry policy in a later implementation plan.
- Constraint violation on money invariant: correctness failure or stale command,
  not a generic 500.

No outbound HTTP, pricing lookup, payment lookup, or provider call may run while
holding a database transaction.

### Reconciliation Claiming

Worker-style claiming uses lease semantics only for queue-like case ownership:

```text
SELECT reconciliation_case_id
FROM reconciliation_cases
WHERE state IN ('open', 'waiting_evidence')
  AND next_attempt_at <= now()
ORDER BY next_attempt_at, reconciliation_case_id
FOR UPDATE SKIP LOCKED
LIMIT $n
```

Claiming updates `state = 'leased'`, `lease_owner`, `lease_deadline_at`, and
`attempt_count` in the same short transaction. Money-changing resolution is a
separate transaction that reacquires locks in the account-first order above.

## Hot-Path Query Shapes

### Sign-Up Bonus Grant

Access path:
- resolve or create the admitted `billing_accounts` row and `account_balances`
  row by `account_scope_key` before money mutation;
- optional committed idempotency probe by
  `(account_id, 'signup_bonus_grant', idempotency_key)`;
- transaction locks `account_balances` by `account_id`;
- insert or lock the idempotency row;
- insert or lock `signup_bonus_grants` by
  `(account_id, signup_bonus_policy_version)` and by
  `(admission_authority, admission_reference_id, signup_bonus_policy_version)`;
- if an existing grant has the same `grant_fingerprint`, return the stored
  outcome;
- if an existing grant or admission reference has a changed fingerprint, store
  or read conflict outcome and open or read `signup_bonus_conflict`;
- insert `ledger_entries(effect_type = 'signup_bonus_credit')`;
- update `account_balances.settled_usd_atoms` and `available_usd_atoms`;
- insert `operation_outcomes` with `primary_resource_type =
  'signup_bonus_grant'`;
- mark idempotency and grant `committed`/`credited`.

Required checks:
- account state permits admission credit;
- `grant_amount_usd_atoms = 1000000000` for the current approved `$10.00`
  policy;
- one grant per `(account_id, signup_bonus_policy_version)`;
- one grant per `(subject_authority, subject_id, signup_bonus_policy_version)`;
- the same admission reference cannot bind to another account or amount;
- duplicate delivery with the same fingerprint cannot create another ledger
  credit.

### Reserve

Access path:
- resolve `billing_accounts` by `account_scope_key`;
- optional committed idempotency probe by
  `(account_id, 'reserve', idempotency_key)`;
- transaction locks `account_balances` by `account_id`;
- insert or lock idempotency row;
- insert `usage_operations`;
- insert `usage_holds`;
- insert `ledger_entries(effect_type = 'usage_hold')`;
- update `account_balances`;
- insert `operation_outcomes`;
- mark idempotency `committed`.

Required checks:
- account state permits spend;
- `available_usd_atoms >= reserve_usd_atoms`;
- quote has not expired before reservation;
- duplicate usage operation or idempotency replay cannot double-reserve.

### Finalize

Access path:
- lookup `usage_operations` by `usage_operation_id` to discover `account_id`;
- optional committed idempotency probe;
- transaction locks `account_balances`;
- lock idempotency row;
- lock `usage_operations` and `usage_holds`;
- insert terminal outcome if none exists;
- insert `ledger_entries(effect_type = 'usage_charge')` or a release-only
  entry when final charge is zero;
- update hold charged/released fields and terminal state;
- update balance and stored outcome.

Required checks:
- terminal finalize is unique for the usage operation;
- charge does not exceed authorized reserved ceiling;
- excess after possible external effect is represented by write-off,
  compensation, or reconciliation, not overcharge.

### Write-Off

Access path:
- lookup usage operation by `usage_operation_id`;
- lock account balance, idempotency, usage operation, and hold;
- insert terminal write-off outcome;
- insert `ledger_entries(effect_type = 'usage_write_off')` for reservation
  release and customer-money readback;
- update hold and operation terminal state;
- update balance and stored outcome.

Required checks:
- write-off terminal outcome is unique;
- released reservation cannot exceed active reserved amount;
- external loss/overrun stays explicit in `write_off_usd_atoms`.

### Dormant Future Top-Up Evidence Application

This path is historical/future design context only. It is not a current
balance-increase path and must not be planned until customer payment/top-up scope
is explicitly reopened and the live payments-service contract is revalidated.

Access path:
- lookup or attempt insert `payment_evidence_lineages` by
  `payment_evidence_id`;
- lookup or attempt insert the versioned `payment_evidence` row by
  `(payment_evidence_id, evidence_version)`;
- use `topup_operation_id` to discover account;
- lock account balance;
- lock idempotency row for `topup_evidence`;
- lock top-up and payment attempt;
- enforce evidence lineage binding, versioned selector uniqueness, and
  fingerprint uniqueness;
- for final accepted evidence, insert pending release if pending exists and
  `ledger_entries(effect_type = 'topup_credit')`;
- update top-up/evidence state, balance, and stored outcome.

Required checks:
- same `(payment_evidence_id, evidence_version)` plus changed fingerprint is
  conflict;
- a new `evidence_version` under the same lineage must evaluate against prior
  evidence lineage and cannot silently replace an already-posted ledger effect;
- duplicate fingerprint cannot credit twice;
- trusted normalized evidence amount equals the top-up credited amount or routes
  to conflict/reconciliation.

### Balance Readback

Strict read-after-write balance reads use primary PostgreSQL:

```text
SELECT account + account_balances
WHERE account_scope_key = $1
```

Replica or cache reads are not correctness paths unless a later design records a
staleness budget. The initial money core treats balance readback as primary-read
truth.

## Support And Reconciliation Readback Surfaces

These are data access shapes, not HTTP contract designs.

### Account Support Readback

By `account_scope_key`:
- `billing_accounts`;
- `account_balances`;
- latest `ledger_entries` ordered by
  `(created_at DESC, ledger_entry_id DESC)`;
- `signup_bonus_grants` and linked stored outcome/ledger credit;
- active and terminal `usage_holds`;
- `usage_operations` and `usage_terminal_outcomes`;
- dormant future `topup_operations`, `payment_attempts`, and `payment_evidence`
  only when payment/top-up scope is later reopened or historical rows exist;
- `idempotency_records` and `operation_outcomes`;
- `reconciliation_cases`;
- `audit_events`.

Pagination:
- ledger, audit, top-up, and usage lists use keyset pagination with the timestamp
  plus UUID/text tie-breaker from the listed indexes;
- no offset pagination on high-churn operational lists.

### Reconciliation Readback

Case detail reads include:
- case row and lease state;
- linked grant, operation, evidence, and ledger rows;
- linked stored outcome and idempotency rows;
- all resolution ledger entries and audit rows.

Stale reservation discovery reads:
- `usage_holds` using `idx_usage_holds_stale`;
- existing unresolved `reconciliation_cases` using partial uniqueness before
  creating a new stale-reservation case.

Sign-up bonus conflict reads:
- `signup_bonus_grants` by `signup_bonus_grant_id`;
- `signup_bonus_grants` by
  `(account_id, signup_bonus_policy_version)`;
- `signup_bonus_grants` by
  `(admission_authority, admission_reference_id, signup_bonus_policy_version)`;
- linked ledger, idempotency, outcome, audit, and reconciliation rows.

Dormant future duplicate/conflicting payment evidence reads:
- `payment_evidence` by `(payment_evidence_id, evidence_version)`;
- `payment_evidence_lineages` and version history by `payment_evidence_id`;
- `payment_evidence` by `evidence_payload_fingerprint`;
- linked top-up, attempt, ledger, idempotency, and outcome rows.

## Retention And Deletion Posture

- Ledger entries, account balance history implied by ledger, sign-up bonus grant
  rows, usage operations, reconciliation cases, legacy import evidence, and any
  dormant future payment/top-up evidence rows are retained as audit-grade
  financial records. They are not hard-deleted by normal product flows.
- Idempotency records may receive `expires_at` only after the replay safety
  window is complete. Expiry cannot remove ledger entries, operation rows, or
  evidence needed for support/reconciliation.
- Audit and support metadata must be privacy-safe at write time. The schema does
  not store raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs,
  payment secrets, or raw PSP webhook bodies.
- Account closure sets `billing_accounts.state = 'closed'` and `closed_at`; it
  does not delete money history.

## Legacy Import Compatibility

Legacy import uses `legacy_import_batches`, `legacy_balance_imports`, and one
`ledger_entries(effect_type = 'migration_import')` per migrated account.

Import readback must prove:
- source snapshot fingerprint;
- legacy subject identifier;
- exact legacy `balanceNgonka` text;
- exact legacy `lockedRateUsd` text;
- derived USD atom value;
- import row fingerprint;
- linked migration ledger entry;
- parity status.

Live balance calculation must never read legacy balance/rate fields. If parity
fails after import, repair uses explicit `reconciliation_correction` or
`migration_import` correction entries linked from the import evidence row.

## Data-Model Test Obligations

Later planning must include tests that prove the data model, not only app logic.

### Money Representation

- decimal parser vectors for valid, invalid, signed, zero, `-0`, max range, and
  excess precision inputs;
- formatter vectors for atom-to-decimal output with no exponent and trimmed
  trailing fractional zeroes;
- rounding vectors for reject-by-default, reserve-ceiling round-up,
  final-charge half-up policy rounding, exact `$10.00` sign-up bonus atoms,
  exact correction/reversal amounts, and dormant future top-up evidence precision
  rejection only when payment/top-up scope is reopened.

### Constraints And Invariants

- account uniqueness by `account_scope_key`;
- subject uniqueness by `(subject_authority, account_type, subject_id)`;
- non-negative balance checks and `available = settled - reserved`;
- ledger delta-pattern checks by `effect_type`, including negative cases that
  prove disallowed balance components are zero (`usage_charge` cannot mutate
  `pending_delta_usd_atoms`);
- constrained-text checks for every closed state/kind/class/resource-type field
  listed in the constrained-text inventory;
- unique `settlement_effect_id`, sign-up bonus grant, usage operation, terminal
  outcome, and idempotency key constraints;
- unique sign-up bonus constraints for `(account_id,
  signup_bonus_policy_version)`, `(subject_authority, subject_id,
  signup_bonus_policy_version)`, and `(admission_authority,
  admission_reference_id, signup_bonus_policy_version)`;
- dormant future unique `(payment_evidence_id, evidence_version)` for the
  billing evidence application selector plus lineage binding that prevents
  evidence ID rebinds when payment/top-up scope is reopened;
- partial uniqueness preventing duplicate unresolved reconciliation cases for
  usage, sign-up bonus, settlement-effect, qualified-inference-evidence,
  ledger-entry, legacy-import, and dormant future top-up/payment lineages.

### Ledger Conservation

- property tests over sign-up bonus credits, holds, releases, charges,
  write-offs, reversals, operator adjustments, migration imports, and
  reconciliation corrections. Payment reversals are dormant future proof only
  when payment/top-up scope is reopened;
- recomputation checks that `account_balances` equals ledger deltas plus active
  hold state for sampled accounts;
- no posted ledger money-field mutation path.

### Idempotency

- replay same key and same fingerprint for reserve, finalize, write-off,
  sign-up bonus grant, reversal/compensation, migration import, and
  reconciliation correction;
- changed fingerprint conflict for each money-affecting operation kind;
- sign-up bonus grant idempotency keys and fingerprints include
  `signup_bonus_grant_id`, `signup_bonus_policy_version`,
  `admission_reference_id`, and the exact grant amount;
- dormant future top-up evidence idempotency keys and fingerprints include
  `(payment_evidence_id, evidence_version)`, not bare `payment_evidence_id`,
  only when payment/top-up scope is reopened;
- replay after stored failure;
- expiry behavior that never deletes money truth.

### Concurrency

- concurrent reserve race against one account balance row cannot make
  `available_usd_atoms` negative;
- concurrent duplicate sign-up bonus grant delivery creates one grant row, one
  `signup_bonus_credit` ledger effect, and replay-stable outcome;
- linked sign-up bonus ledger amount equals the immutable grant amount;
- concurrent changed-fingerprint sign-up bonus delivery cannot credit twice and
  opens or reads `signup_bonus_conflict`;
- concurrent finalize replay creates exactly one terminal outcome and one
  customer charge effect;
- concurrent write-off/finalize race allows only one terminal path;
- dormant future duplicate top-up evidence delivery credits once only when
  payment/top-up scope is reopened;
- dormant future concurrent distinct `evidence_version` deliveries under one
  `payment_evidence_id` cannot rebind lineage or rewrite a prior ledger effect;
- row-lock timeout, deadlock, and retry classification tests.

### Reconciliation And Import

- stale reservation case creation is deduped;
- duplicate open reconciliation cases are rejected for each mapped reason and
  lineage key, including sign-up bonus conflict by `signup_bonus_grant_id` and
  legacy import mismatch by `legacy_balance_import_id`;
- leased reconciliation cases are not claimed by two workers;
- resolution that changes money creates a linked ledger entry;
- sign-up bonus conflict routes to stored outcome, conflict, or reconciliation
  without duplicate credit;
- dormant future duplicate payment evidence and changed-fingerprint evidence
  conflicts route to stored outcome, conflict, or reconciliation without double
  credit, using `(payment_evidence_id, evidence_version)` as the specific claim
  selector and `payment_evidence_id` for lineage-wide support grouping when
  payment/top-up scope is reopened;
- legacy import parity mismatch creates a case or correction path without using
  legacy fields for live balance.

### Performance Evidence

- sign-up bonus grant, reserve, finalize, and write-off paths prove O(1) lookup
  by account/grant/operation/idempotency keys;
- dormant future top-up evidence paths prove O(1) lookup by
  account/operation/evidence/idempotency keys only when payment/top-up scope is
  reopened;
- benchmark captures account-row contention for high-concurrency same-account
  reserve/finalize/write-off workloads;
- support readback uses keyset pagination and the declared account/recent
  indexes.

## Review Packet Summary

Technical design review should inspect this artifact against:
- approved `specs/billing-money-core/spec.md`;
- source-of-truth separation between ledger, balance read model, holds, sign-up
  bonus grants, audit rows, legacy import evidence, and dormant future
  payment/top-up evidence;
- account-first lock order and one short transaction per money command;
- database-enforced uniqueness and non-negative balance invariants;
- support-safe readback and privacy exclusions;
- completeness of data-model test obligations.

No implementation, migration, API contract, runtime adapter, runtime event
schema, or task ledger is approved by this data-model design. Customer
payment/top-up, payments-service evidence writeback, and Redpanda
payment-evidence ingestion remain future/conditional until a later
specification reopen approves them.
