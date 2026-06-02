# Data Model Design

Status: review-ready
Date: 2026-06-02

## Existing Durable State To Reuse

The current migrations already provide the core customer-money state required
for this cutover:

- `billing_accounts`: canonical account scope and account state.
- `account_balances`: USD atom settled, reserved, available, pending, version,
  and last ledger link.
- `idempotency_records`: durable key/fingerprint/state/conflict records.
- `operation_outcomes`: replay-stable stored outcome envelope.
- `usage_operations`: usage operation lifecycle and pricing lineage.
- `usage_holds`: generic hold state. For migrated proxy cohorts, this table is
  not the direct fallback reserve authority.
- `usage_terminal_outcomes`: terminal finalize/write-off/reversal outcome
  lineage.
- `qualified_inference_evidence`: safe inference evidence identity and
  fingerprint.
- `ledger_entries`: immutable USD atom effects.
- `legacy_import_batches` and `legacy_balance_imports`: proxy balance import
  and parity evidence.
- `reconciliation_cases`: stale/ambiguous/conflict/manual-review cases.
- `audit_events`: support-safe account and operation audit trail.
- `billing_event_inbox`: durable event receipt, replay, conflict, quarantine,
  and applied state.
- `billing_outbox`: durable billing fact publication state.
- `billing_admission_controls`: global/account/use-class admission gates.
- `spending_microleases`: parent microlease reserve and active exposure.
- `microlease_child_debits`: per-request child debit terminal lineage.
- `microlease_checkpoints`: proxy allocator progress, close, and release proof.

## Account Resolve And Import Readiness

Account resolve reads:

- `billing_accounts` by `account_scope_key`;
- `account_balances` by account;
- latest `legacy_balance_imports` for imported proxy balance/parity state;
- open `reconciliation_cases` for import, usage, microlease, child debit,
  inbox, ledger, or manual-review reasons;
- current `billing_admission_controls` for account and use-class gates.

Required design rule:

- Paid migrated resolve passes only when account state is `active`, balance row
  exists, import/parity state is accepted, no blocking reconciliation/manual
  review state exists, and runtime admission state is not expired or
  fail-closed.

Schema expansion trigger:

- If planning cannot express "latest accepted import/parity state" from
  `legacy_import_batches` and `legacy_balance_imports` without expensive or
  ambiguous queries, add a narrow import-state projection keyed by account.
  That projection is derived from import batches and is not a second balance.

## Balance Read Model

Balance read is derived from:

- `account_balances` for settled, reserved, available, pending, version, and
  last ledger entry;
- active `usage_holds` for any generic hold state that exists;
- active `spending_microleases` for parent microlease exposure;
- unresolved `microlease_child_debits` and latest `microlease_checkpoints` for
  child exposure and terminal lag;
- open `reconciliation_cases` for stale, ambiguous, import, conflict, or
  manual-review flags;
- `billing_admission_controls` for runtime gate state.

Rules:

- Active microlease exposure is already part of `account_balances.reserved`
  because issue/replenish posts a reserved delta.
- Expired microlease exposure remains reserved until terminal, close, release,
  write-off, reversal, compensation, or reconciliation proof posts a release or
  charge.
- Balance read does not release or correct money.
- Any cached balance projection must be rebuildable from Postgres and must not
  be the authority for migrated paid admission.

## Migrated Usage Reserve Representation

For migrated proxy cohorts, a usage reserve represents child-debit lineage
under an already reserved parent microlease:

- `spending_microleases` owns parent capacity and reserved account exposure.
- `microlease_child_debits` owns child debit identity, cap, state, terminal
  kind, fingerprints, pricing lineage, safe metadata, and settlement effect.
- `usage_operations` may be linked through
  `microlease_child_debits.usage_operation_id` when a generic usage operation
  readback identity is needed.
- `idempotency_records` and `operation_outcomes` store replay/conflict outcomes
  for the usage command.

Rules:

- Migrated proxy reserve must not create a new `usage_holds` account-balance
  hold after microlease denial.
- If a `usage_holds` row is used for a non-migrated or future authority mode,
  that path must be disabled or rejected for migrated proxy callers unless a
  later spec reopens direct reserve authority.
- The child cap and parent microlease cap are the customer-charge ceiling.

## Finalize, Write-Off, Reversal, And Compensation

Terminal settlement writes:

- `qualified_inference_evidence` when safe terminal evidence is present;
- `microlease_child_debits` terminal fields and state;
- `usage_terminal_outcomes` when a linked generic usage operation exists;
- `ledger_entries` with one of the approved usage or microlease effect types;
- `account_balances` with exact USD atom deltas;
- `operation_outcomes` and `idempotency_records`;
- `reconciliation_cases` when child cap, parent cap, fingerprint, evidence, or
  lineage is unsafe;
- `billing_outbox` for safe billing facts.

Rules:

- Final charge cannot exceed child debit cap or parent microlease authority.
- Write-off releases reserved exposure explicitly and records write-off lineage.
- Reversal references the original ledger/effect and creates a compensating
  entry. It does not update an old ledger row.
- Operation conflict is stored, support-safe, and readbackable.

## Inbox, Outbox, And Reconciliation

Inbound Redpanda events:

- create or read `billing_event_inbox` by topic/event identity;
- verify producer identity and event fingerprint;
- lock business identity where needed;
- apply terminal/checkpoint/close state in the same local transaction that marks
  the inbox applied;
- quarantine or mark conflict with support-safe metadata on invalid events.

Outbox:

- app/repository creates `billing_outbox` records in the same transaction as
  the durable money/readback effect;
- worker claims pending/failed rows with skip-locked semantics;
- successful publish marks row `published`;
- failed publish schedules retry and does not roll back committed money state.

Reconciliation:

- stale reserve, stale microlease, stale child debit, close gap, event conflict,
  ambiguous terminal, missing inference evidence, and import mismatch are all
  represented as `reconciliation_cases`;
- each open case must have one durable lineage owner such as usage operation,
  child debit, microlease, checkpoint, inbox, ledger entry, or import row;
- reconciliation workers can open or update cases and then invoke the same
  terminal/write-off/reversal/compensation app commands used by normal paths.

## Retention And Privacy

- Idempotency records keep hot replay and audit retention classes as already
  modeled.
- Safe metadata JSON must stay bounded and privacy-safe.
- No raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs,
  payment secrets, raw provider payloads, raw event payloads, dynamic proof
  URLs, or sensitive request bodies are stored in these tables.
- Raw `request_id` remains trace correlation only and is not a settlement key.

## Data Review Questions To Falsify

Technical design review should check:

- whether migrated reserve can be implemented without creating a second
  account-balance hold;
- whether operation readback can locate by every promised safe identity without
  high-cardinality or ambiguous lookup behavior;
- whether import/parity state is queryable enough for account resolve and
  rollout gates;
- whether worker recovery can quarantine and retry without losing offset or
  applying duplicate money effects;
- whether every terminal path can store one replay-stable outcome.
