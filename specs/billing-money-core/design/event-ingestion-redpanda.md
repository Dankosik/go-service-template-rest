# Redpanda Event Ingestion Design

Status: historical/future-conditional after current sign-up-bonus scope repair
Owner: billing-service
Scope: superseded Redpanda/Kafka-compatible async event-intake context; not current planning input
Consumes: `specs/billing-money-core/spec.md`,
`specs/billing-money-core/design/data-model.md`,
`docs/PRD.md`, `docs/critical-billing-context.md`,
`docs/repo-architecture.md`

## Boundary

This artifact records the historical Redpanda-backed Kafka-compatible
event-ingestion design without moving money truth out of Postgres. After the
2026-06-01 funding-source correction, this artifact is future/conditional
context only for current planning: customer payment/top-up is not implemented,
users cannot add balance, and Redpanda payment-evidence ingestion is not an
active balance-increase path. Current planning may consume the repaired
`design/data-model.md` sign-up bonus and usage money semantics, but must not
plan Redpanda payment-evidence ingestion or runtime event schemas from this
addendum.

In scope:
- historical Redpanda as event transport for terminal usage, future payment
  evidence, committed billing effects, operation outcomes, and reconciliation
  signals.
- Consumer groups, topic naming, partition keys, ordering assumptions, lag
  behavior, and quarantine/redrive policy.
- Durable inbox/idempotent event processing mapped onto the existing billing
  money data model.
- Transaction order for applying one consumed event.
- Future data-model additions required by event ingestion and emitted events.
- Failure handling, reconciliation hooks, performance posture, observability,
  and tests.

Out of scope:
- Code, migrations, generated SQL, runtime wiring, and `tasks.md`.
- Public HTTP/OpenAPI contract design.
- Replacing synchronous reserve before execution.
- Redesigning the approved core money model.
- Sign-up bonus grant application; that current money path is designed in
  `design/data-model.md` and does not require Redpanda payment evidence.
- Current customer payment/top-up product flow, payments-service writeback, and
  Redpanda payment-evidence ingestion.
- Storing raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs,
  payment secrets, or raw provider webhook payloads.

## Decision Summary

- Current-scope supersession: this addendum is not an implementation or planning
  handoff for the sign-up-bonus and usage-only money scope. It remains
  future/conditional context until async event ingestion or payment/top-up scope
  is explicitly reopened.
- Redpanda is the event backbone and replay transport. It is not the billing
  source of truth.
- Billing Postgres remains the only authority for ledger effects, balances,
  holds, idempotency, operation outcomes, reconciliation cases, and event inbox
  outcomes.
- Broker-level exactly-once semantics are not a billing correctness guarantee.
  Billing correctness comes from local database transactions, durable inbox
  records, business-operation idempotency, unique constraints, account-row
  locking, and replay-safe stored outcomes.
- Consumer offsets are committed only after the database transaction has stored
  a durable processing outcome for the event.
- Future payments-service evidence ingestion, if reopened, previously used
  `(paymentEvidenceId, evidenceVersion)` as the payment evidence application
  selector for inbox lineage, business idempotency, replay, conflict handling,
  reconciliation, and support readback. That selector must be revalidated against
  the then-current payments-service contract before planning.
- Poison or quarantined events whose producer `eventId` or semantic identity is
  malformed still get a durable inbox receipt keyed by broker coordinates
  `(topic, partition, offset)` and a safe receipt identity; they do not block the
  partition and do not require storing raw payload.
- Committed-offset `retry_scheduled` rows are owned by a billing event-ingestion
  inbox retry worker that claims rows from Postgres and re-enters the same
  idempotent business processor; Redpanda redelivery is not the owner after the
  offset has been committed.
- Billing emits events through a Postgres outbox written in the same transaction
  as the source ledger/outcome/reconciliation state change.
- Analytics consumers use independent consumer groups and must never share
  billing correctness offsets, locks, or inbox rows.

## Event Backbone Choice

Redpanda is the Kafka-compatible event log used for delivery, fan-out, and
operational replay. It is intentionally weaker than the money source of truth:

- Redpanda may deliver an event more than once.
- Consumers may crash after committing billing state and before committing the
  Redpanda offset.
- Events may be delayed, replayed, or redriven.
- Ordering is only per partition and only as strong as the producer partition
  key policy.
- Topic retention is an operational replay window, not audit retention.

The billing guarantee is therefore:

```text
at-least-once event delivery + durable inbox + business idempotency
  => at-most-once money effect per semantic billing operation
```

Every money effect still lands first in `ledger_entries` and the transactional
`account_balances` read model. Redpanda messages can carry evidence and publish
facts, but they do not authorize a second balance truth.

## Event Categories

### Consumed First

| Category | Producer | Purpose | Money path |
| --- | --- | --- | --- |
| Usage terminal completion | `gonka-proxy` | Finalize a previously reserved usage operation after execution and metering evidence exists. | Critical. May commit `usage_charge`, release reserved atoms, and store terminal outcome. |
| Usage failure, timeout, or write-off | `gonka-proxy` | Release or write off a reserved usage operation when execution did not produce a chargeable completion or evidence is missing after possible external effects. | Critical. May commit `usage_write_off` or create reconciliation. |
| Normalized payment evidence | `payments-service` | Future/conditional only: apply trusted normalized payment evidence after provider-specific handling is complete outside billing if customer payment/top-up scope is reopened. | Not current scope. Must not commit `topup_credit` or payment reversal in current planning. |
| Reconciliation/admin repair signals | billing admin/reconciliation tooling | Request safe replay, redrive, or repair of ambiguous existing state. | Optional critical path. Must call billing repair operations, never direct ledger edits. |

### Emitted First

| Category | Producer | Purpose | Correctness role |
| --- | --- | --- | --- |
| Committed billing ledger effect | billing-service outbox | Publish immutable committed ledger facts to downstream read models, support tools, and analytics. | Derived from Postgres ledger; does not own money truth. |
| Billing operation outcome | billing-service outbox | Publish replay-stable command/event outcomes for cross-service correlation. | Derived from `operation_outcomes` and inbox state. |
| Reconciliation required | billing-service outbox | Notify operators/workers that a durable reconciliation case was opened. | Derived from `reconciliation_cases`; repair still happens through billing operations. |
| Rejected or conflict event | billing-service outbox | Notify producers/operators that an inbound event was quarantined, conflicted, or rejected. | Operational signal only; stored inbox outcome is authoritative. |

## Topic Design

Topic names are versioned at the topic boundary. Schema-compatible additions may
stay within a topic version only when old consumers can ignore new optional
fields. Semantic changes that alter identity, amount meaning, finality, or
required proof need a new versioned topic.

### Consumed Topics

| Topic | Producer | Billing consumer group | Partition key | Ordering assumption | Retention expectation | Path | Lag effect |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `usage.execution.completed.v1` | `gonka-proxy` | `billing-service.usage-finalization.v1` | `accountScopeKey` | Per-account order is expected, but correctness also relies on Postgres idempotency and terminal uniqueness. | Minimum 14 days hot replay; operational target 30 days. | Critical money path. | Lag delays finalized charges and reserve release; user-visible balance may show reserved funds longer. |
| `usage.execution.failed.v1` | `gonka-proxy` | `billing-service.usage-finalization.v1` | `accountScopeKey` | Per-account order is expected; terminal outcome uniqueness handles duplicate or conflicting failure/write-off events. | Minimum 14 days hot replay; operational target 30 days. | Critical money path. | Lag delays hold release/write-off and can increase stale-reservation count. |
| `payments.evidence.normalized.v1` | `payments-service` | `billing-service.payment-evidence-application.v1` | `accountScopeKey` | Future/conditional only; duplicate evidence correctness would be enforced by `(payment_evidence_id, evidence_version)` and fingerprint constraints after contract revalidation. | Future sizing only. | Not current scope. | No current lag effect because payment evidence ingestion is deferred. |
| `billing.reconciliation.command.v1` | billing admin/reconciliation tooling | `billing-service.reconciliation-events.v1` | `accountScopeKey` when account-scoped, otherwise `reconciliationCaseId` | No global order. Each command must be idempotent by case or operation identity. | Minimum 14 days; operator replay should prefer Postgres case state after topic retention. | Critical only for repair. | Lag delays repair but does not change already committed ledger truth. |

Notes:
- Usage failure, timeout, abort, missing-evidence, and write-off-required
  producer outcomes share `usage.execution.failed.v1` with a constrained
  terminal class. Separate topics are not needed until they require distinct
  retention, producer ownership, or consumer groups.
- Every processable consumed critical event must carry `eventId`,
  `eventSchemaVersion`, `eventFingerprint`, `accountScopeKey`,
  `operationIdentity`, safe correlation IDs, and the minimum evidence references
  needed to process or reconcile the event. Raw payloads are not allowed. If one
  of those required identity fields is malformed or missing, billing records a
  quarantined inbox receipt under the broker coordinates before committing the
  offset.

### Emitted Topics

| Topic | Producer | Consumer groups | Partition key | Ordering assumption | Retention expectation | Path | Lag effect |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `billing.ledger.effect.v1` | billing-service outbox relay | Analytics, support projections, reconciliation observers | `accountScopeKey` | Per-account committed ledger effect order follows outbox order. Consumers must still dedupe by `settlementEffectId` or `ledgerEntryId`. | Minimum 30 days; analytics may copy to its own store. | Derived critical fact. | Consumers may lag without mutating billing correctness. |
| `billing.operation.outcome.v1` | billing-service outbox relay | `gonka-proxy`, support projections, analytics | `operationIdentity` for operation-scoped outcomes, otherwise `accountScopeKey` | Per-operation order only. | Minimum 14 days. | Derived outcome signal. | Lag may delay downstream read model updates but not billing truth. |
| `billing.reconciliation.required.v1` | billing-service outbox relay | Operators, repair workers, alerting | `accountScopeKey` or `reconciliationCaseId` | Per-case order only. | Minimum 30 days or until all open cases are visible elsewhere. | Operational repair signal. | Lag delays notification; Postgres case remains source of truth. |
| `billing.ingestion.rejected.v1` | billing-service outbox relay | Producers, operators, alerting | `eventId` when valid, otherwise `eventReceiptIdentity` | No ordering dependency. | Minimum 30 days. | Operational conflict/quarantine signal. | Lag delays producer/operator feedback only. |

Emitted topics are produced through the local outbox. Billing must not mutate
Postgres and publish directly to Redpanda as two unrelated writes.

## Consumer Group Strategy

### Critical Usage Finalization Group

Group: `billing-service.usage-finalization.v1`

Consumes:
- `usage.execution.completed.v1`
- `usage.execution.failed.v1`

Responsibilities:
- Validate event schema, fingerprint, account scope, and operation identity.
- Record or replay the event through `billing_event_inbox`.
- Apply existing finalize or write-off semantics through the money data model.
- Store the inbox outcome and then commit the Redpanda offset.
- Open reconciliation instead of charging when required reserve, pricing, or
  inference evidence is missing or conflicting.

### Future/Conditional Payment Evidence Application Group

Group: `billing-service.payment-evidence-application.v1`

Consumes:
- `payments.evidence.normalized.v1`

Responsibilities when payment/top-up scope is explicitly reopened:
- Treat `payments-service` as the normalized evidence producer, not as a writer
  of billing money state.
- Deduplicate by `(paymentEvidenceId, evidenceVersion)` and evidence
  fingerprint.
- Apply top-up credit, reversal/refund, duplicate, conflict, or reconciliation
  outcomes through the existing top-up and payment evidence model.
- Store event and operation outcomes before offset commit.

Current-scope rule: this group is not a planning or implementation input for
the sign-up-bonus and usage-only money scope.

### Reconciliation Event Group

Group: `billing-service.reconciliation-events.v1`

Consumes:
- `billing.reconciliation.command.v1`

Responsibilities:
- Accept only operator-safe repair intents that reference existing durable
  billing state.
- Re-drive or repair through idempotent billing operations.
- Never perform direct ledger edits outside the ledger operation model.

### Analytics Consumers

Analytics, BI, search, dashboards, or external observers must use their own
consumer groups against emitted billing topics. They must not:

- share billing-service consumer groups;
- read or mutate `billing_event_inbox`;
- block billing offsets;
- participate in billing locks or idempotency records;
- write billing source-of-truth tables.

Analytics lag can affect derived reports, but it cannot affect customer money
correctness.

## Ordering And Partitioning

### Partition Key Decisions

| Topic class | Selected partition key | Rejected primary key | Reason |
| --- | --- | --- | --- |
| Usage terminal completion/failure | `accountScopeKey` | `usageOperationId` | Account-level money invariants serialize through one balance row. Per-account partitioning reduces avoidable lock contention and keeps same-account terminal events ordered. |
| Payment evidence | `accountScopeKey` | `(paymentEvidenceId, evidenceVersion)` | Future/conditional only. If payment/top-up scope is reopened, credits and reversals would mutate the account balance row; per-account ordering remains more useful than evidence-only spread and evidence dedupe still uses the revalidated versioned evidence selector. |
| Reconciliation command | `accountScopeKey` if account-scoped, otherwise `reconciliationCaseId` | Global key | Repair operations need either account-local serialization or case-local serialization. Global ordering would create needless bottlenecks. |
| Ledger effect emitted events | `accountScopeKey` | `settlementEffectId` | Downstream account ledgers and support views need stable per-account order. Consumers still dedupe by `ledgerEntryId` or `settlementEffectId`. |
| Operation outcome emitted events | `operationIdentity` | `request_id` | Operation identity is replay-stable; `request_id` is correlation only. |
| Rejected/conflict emitted events | valid `eventId`, otherwise `eventReceiptIdentity` | `accountScopeKey` | Operational feedback is event-specific and does not need account ordering. Poison receipts without producer identity must still be traceable. |

### Ordering Rules

- Redpanda partition order is an optimization, not the invariant.
- Billing correctness must survive out-of-order delivery by using:
  - `billing_event_inbox` event-level dedupe and conflict detection;
  - existing business-operation idempotency records;
  - `usage_terminal_outcomes` uniqueness;
  - `payment_evidence` versioned-selector uniqueness and fingerprint rules;
  - account balance row locking.
- Same-account processing should be mostly sequential because the producer keys
  critical money topics by `accountScopeKey`, but different partitions can run
  concurrently.
- Processing within one assigned partition is sequential for critical topics.
  This keeps offset handling simple and avoids committing offset N+1 before N
  has a durable outcome.

### Out-Of-Order Handling

- Terminal usage event before reserve exists: write an inbox
  `waiting_dependency` or `reconcile_required` outcome, open a
  `missing_reserve_for_terminal_event` safe problem classification mapped to an
  `ambiguous_terminal_state` reconciliation case, and do not charge customer
  money.
- Future payment evidence before the top-up or attempt exists: if payment/top-up
  scope is reopened, write a stored `waiting_dependency` or
  `reconcile_required` outcome and open a payment/top-up reconciliation case. Do
  not credit the account. This path is not current scope.
- Duplicate terminal event for the same `usageOperationId`: return the stored
  terminal outcome when the business fingerprint matches; route changed
  fingerprint to conflict/reconciliation.
- Failure/write-off event after a committed finalize: treat as terminal conflict
  unless it is an exact replay of the stored outcome.
- Completion event after a committed write-off: treat as terminal conflict or
  reconciliation, not as a second ledger mutation.

## Inbox And Idempotent Consumption

### `billing_event_inbox`

The inbox is the durable event-consumption source of truth. It records every
critical inbound event before offset commit and stores the replay-safe result of
processing that event.

Required columns:

| Column | Type | Notes |
| --- | --- | --- |
| `event_inbox_id` | `UUID` | Internal primary key. |
| `event_id` | `TEXT` nullable | Producer event identity. Required for processable events; nullable only for quarantined poison receipts where the producer identity is malformed or missing. |
| `event_receipt_identity` | `TEXT` | Durable billing receipt identity. For valid producer IDs use `event:<topic>:<eventId>`; for malformed or missing producer IDs use `offset:<topic>:<partition>:<offset>`. |
| `event_identity_basis` | `TEXT` | `producer_event_id` or `broker_offset_receipt`. |
| `producer_authority` | `TEXT` | `gonka-proxy`, `payments-service`, or billing admin authority. |
| `topic` | `TEXT` | Redpanda topic name. |
| `partition_id` | `INTEGER` | Redpanda partition. |
| `offset` | `BIGINT` | Redpanda offset. |
| `event_schema_version` | `TEXT` | Schema version carried by the envelope. |
| `event_fingerprint` | `TEXT` | Canonical hash of the event envelope and semantic payload for processable events; billing-computed receipt fingerprint for poison events whose producer fingerprint is malformed or missing. Hash only, no raw payload. |
| `operation_kind` | `TEXT` | `usage_finalize`, `usage_write_off`, `topup_evidence`, `payment_reversal`, `reconciliation_command`, or `ignored`. |
| `operation_identity_type` | `TEXT` | `usage_operation_id`, `payment_evidence_selector`, `topup_operation_id`, `reconciliation_case_id`, `poison_event_receipt`, or `none`. |
| `operation_identity` | `TEXT` | String form of the business operation identity. |
| `account_id` | `UUID` nullable | Resolved billing account. Null only before resolution or for accountless rejected events. |
| `account_scope_key` | `TEXT` nullable | Required for processable money events. |
| `usage_operation_id` | `UUID` nullable | Usage lineage when present. |
| `topup_operation_id` | `UUID` nullable | Top-up lineage when present. |
| `payment_attempt_id` | `TEXT` nullable | Payment attempt lineage when present. |
| `payment_evidence_id` | `TEXT` nullable | Payment evidence lineage when present. |
| `evidence_version` | `BIGINT` nullable | Payment evidence version when present. |
| `processing_state` | `TEXT` | `received`, `processing`, `committed`, `duplicate_replay`, `conflict`, `waiting_dependency`, `retry_scheduled`, `quarantined`, `reconcile_required`, `ignored`. |
| `stored_outcome_id` | `UUID` nullable | Existing `operation_outcomes` row when a business operation outcome exists. |
| `reconciliation_case_id` | `UUID` nullable | Reconciliation case opened by processing. |
| `retry_count` | `INTEGER` | Number of failed processing attempts after durable receipt. |
| `last_error_class` | `TEXT` nullable | Safe failure class such as `schema_mismatch`, `db_timeout`, `account_contention_timeout`, `missing_dependency`, `poison_event`. |
| `last_error_safe_code` | `TEXT` nullable | Safe problem code. No raw event payload. |
| `next_attempt_at` | `TIMESTAMPTZ` nullable | Retry eligibility. |
| `received_at` | `TIMESTAMPTZ` | Consumer receipt time. |
| `claim_owner` | `TEXT` nullable | Inbox retry or consumer owner currently processing a durable retry row. |
| `claim_generation` | `BIGINT` | Monotonic claim fence for retry/stuck-row recovery. |
| `claim_deadline_at` | `TIMESTAMPTZ` nullable | Lease deadline after which another inbox retry owner may reclaim. |
| `claimed_at` | `TIMESTAMPTZ` nullable | Processing claim time. |
| `processed_at` | `TIMESTAMPTZ` nullable | Durable terminal processing outcome time. |
| `created_at` | `TIMESTAMPTZ` | Insert time. |
| `updated_at` | `TIMESTAMPTZ` | Last state update. |

Required constraints and indexes:

- `PRIMARY KEY (event_inbox_id)`.
- `UNIQUE (event_receipt_identity)`.
- `UNIQUE (topic, event_id)` where `event_id IS NOT NULL`.
- `UNIQUE (topic, partition_id, offset)`.
- `CHECK (retry_count >= 0)`.
- `CHECK (claim_generation >= 0)`.
- constrained `topic`, `operation_kind`, `operation_identity_type`, and
  `processing_state` text sets.
- `CHECK ((event_identity_basis = 'producer_event_id') = (event_id IS NOT NULL))`.
- `CHECK ((payment_evidence_id IS NULL) = (evidence_version IS NULL))`.
- processable money events require `event_identity_basis = 'producer_event_id'`;
  `broker_offset_receipt` is valid only for `quarantined`, `ignored`, or
  rejected poison receipts.
- `idx_event_inbox_claim` on `(processing_state, next_attempt_at, event_inbox_id)`
  for retry or local repair workers.
- `idx_event_inbox_claim_lease` on
  `(processing_state, next_attempt_at, claim_deadline_at, event_inbox_id)` for
  committed-offset retry ownership and stale-claim recovery.
- `idx_event_inbox_operation` on
  `(operation_identity_type, operation_identity, event_inbox_id)`.
- `idx_event_inbox_payment_evidence` on
  `(payment_evidence_id, evidence_version, event_inbox_id)` where
  `payment_evidence_id IS NOT NULL` and `evidence_version IS NOT NULL`.
- `idx_event_inbox_account_recent` on
  `(account_id, received_at DESC, event_inbox_id DESC)` where
  `account_id IS NOT NULL`.
- `idx_event_inbox_offsets` on
  `(topic, partition_id, offset, event_inbox_id)` for offset readback.

### Replay And Conflict Rules

- Same valid `(topic, event_id)` and same `event_fingerprint` returns the stored
  inbox outcome. If the original processing committed a business outcome, return
  the linked `stored_outcome_id`.
- Poison or rejected events without a valid producer event ID replay by
  `(topic, partition_id, offset)` and `event_receipt_identity`; this permits a
  durable quarantine outcome and offset commit without fabricating a producer
  identity.
- Same `(topic, event_id)` with a changed `event_fingerprint` transitions to or
  reads `conflict`; it must not mutate money.
- Same `usageOperationId` terminal replay is idempotent even if the broker event
  is redelivered with a different offset, as long as the terminal business
  fingerprint matches the stored terminal outcome.
- Same `(paymentEvidenceId, evidenceVersion)` and same evidence fingerprint
  returns the existing evidence/outcome. Same versioned selector with changed
  fingerprint is conflict.
- A different `evidenceVersion` under the same `paymentEvidenceId` is a distinct
  semantic evidence operation. It must evaluate top-up state, prior evidence
  lineage, and reversal/refund/adjustment rules before any money mutation; it
  must not silently replace a prior accepted ledger effect.
- Duplicate Redpanda delivery cannot duplicate ledger effects because ledger
  mutation is guarded by the inbox row, business idempotency record, terminal
  uniqueness, payment evidence versioned-selector uniqueness, and account-row
  transaction.
- Consumer offset commit happens only after the inbox row records a durable
  terminal or retry outcome for the event.

### Mapping To Existing Tables

Inbound events do not bypass the approved money model:

- Usage completion maps to existing finalize semantics:
  `usage_operations`, `usage_holds`, `usage_terminal_outcomes`,
  `ledger_entries`, `account_balances`, `idempotency_records`, and
  `operation_outcomes`.
- Usage failure/timeout/write-off maps to existing write-off or reconciliation
  semantics over the same usage tables.
- Future payment evidence maps to `payment_evidence`, `topup_operations`,
  `payment_attempts`, `ledger_entries`, `account_balances`,
  `idempotency_records`, and `operation_outcomes` only after payment/top-up
  scope is explicitly reopened.
- Ambiguous or invalid processable events map to `reconciliation_cases` plus a
  stored inbox outcome. They do not create silent money edits.

The event-level idempotency key is not the only business dedupe key. Valid event
dedupe uses `(topic, event_id)`; poison receipt dedupe uses
`event_receipt_identity` / `(topic, partition_id, offset)`. Money dedupe uses
existing semantic identities such as `usageOperationId`,
`(paymentEvidenceId, evidenceVersion)`, `topupOperationId`, and
`settlementEffectId`.

## Transaction Model

Processing one event follows this order:

1. Read one event from the assigned Redpanda partition.
2. Validate the envelope shape, allowed topic, supported schema version, safe
   required fields, event ID, account scope, operation identity, and event
   fingerprint. If producer event ID or semantic identity is malformed or
   missing, compute `event_receipt_identity` from `(topic, partition, offset)`
   and route to durable quarantine. Do not log raw payloads.
3. Open one PostgreSQL transaction.
4. Insert or lock the `billing_event_inbox` row by valid `(topic, event_id)` or,
   for poison receipts without valid producer identity, by
   `(topic, partition_id, offset)` / `event_receipt_identity`.
5. If the inbox row exists with the same fingerprint and a terminal outcome,
   return the stored outcome, commit the transaction, and then commit the
   Redpanda offset.
6. If the inbox row exists with a changed fingerprint, store or read the
   conflict outcome, optionally create a reconciliation case and rejected-event
   outbox row, commit the transaction, and then commit the offset.
7. Resolve the account and lock `account_balances` by `account_id` before
   locking mutable operation/evidence rows.
8. Lock or create the business idempotency record for the semantic operation.
   For event-originated operations, the idempotency key is derived from the
   business identity, not only from the broker offset:
   - usage completion: `usage-finalize:<usageOperationId>`;
   - usage failure/write-off: `usage-write-off:<usageOperationId>`;
   - future payment evidence after scope reopen:
     `payment-evidence:<paymentEvidenceId>:v<evidenceVersion>`;
   - reconciliation command: `reconciliation:<reconciliationCaseId>:<commandId>`.
9. Apply the existing reserve/finalize/write-off operation, or future evidence
   operation after payment/top-up reopen, through the
   approved data model:
   - lock `usage_operations` and `usage_holds` for usage terminal events;
   - lock `topup_operations`, `payment_attempts`, and `payment_evidence` for
     future payment evidence events only after scope reopen;
   - append ledger effects only when the domain operation allows it;
   - update `account_balances` in the same transaction;
   - store `operation_outcomes`.
10. Insert outbox rows for emitted billing events derived from committed ledger,
    operation outcome, reconciliation, or rejected/conflict state.
11. Update `billing_event_inbox` with `committed`, `duplicate_replay`,
    `conflict`, `reconcile_required`, `waiting_dependency`, `retry_scheduled`,
    `quarantined`, or `ignored`, including `stored_outcome_id` or
    `reconciliation_case_id` when present.
12. Commit the PostgreSQL transaction.
13. Commit the Redpanda offset after the database commit succeeds.

Crash behavior:

- Crash before DB commit: no durable outcome exists; Redpanda redelivers and
  processing starts again.
- Crash after DB commit and before offset commit: Redpanda redelivers; the
  consumer reads the committed inbox/business outcome, returns it without a new
  money effect, and commits the offset.
- Crash after offset commit: the durable DB outcome already exists.
- Crash in the outbox relay after DB commit: outbox row remains pending and can
  be retried without changing money state.

## Failure Handling

| Failure | Billing behavior | Offset policy |
| --- | --- | --- |
| Duplicate event with same fingerprint | Return stored inbox/business outcome. No new ledger effect. | Commit after durable replay outcome is read. |
| Same valid event ID with changed fingerprint | Mark inbox `conflict`, store safe conflict outcome, emit `billing.ingestion.rejected.v1`, and create reconciliation if money ambiguity exists. | Commit after conflict outcome is durable. |
| Out-of-order terminal usage event | If reserve/hold is missing, store `waiting_dependency` or `reconcile_required`; do not charge. | Commit when durable waiting/reconciliation outcome is written. |
| Missing reserve for terminal usage event | Open `ambiguous_terminal_state` reconciliation tied to `usageOperationId`; no customer charge. | Commit after reconciliation outcome is durable. |
| Stale or mismatched pricing snapshot | Reject or reconcile. Finalize may only use the reserve-bound pricing lineage; no pricing call inside transaction. | Commit after stored failure/reconciliation outcome. |
| Missing `inferenceId` or unqualified inference evidence | For chargeable completion requiring inference evidence, create `missing_inference_evidence` reconciliation. Failure/write-off events may release/write off if their fingerprint proves the non-chargeable terminal state. | Commit after durable outcome. |
| Unsupported event schema version | Mark `quarantined`, store `schema_mismatch`, emit rejected/conflict signal. | Commit after quarantine outcome; do not block partition indefinitely. |
| Poison event with malformed required fields | Mark `quarantined` with safe error class. If `eventId` or semantic identity is malformed or missing, store the inbox row under `event_receipt_identity = offset:<topic>:<partition>:<offset>` with `operation_identity_type = poison_event_receipt`. No raw payload stored. | Commit after quarantine outcome. |
| DB timeout before durable outcome | Roll back transaction, do not commit offset, retry with bounded backoff. | Do not commit. |
| Account row lock timeout | Store `retry_scheduled` when the inbox row can be safely updated with `next_attempt_at`, `retry_count`, and safe failure class; otherwise retry by redelivery. Classify separately as `account_contention_timeout`. | Commit only if retry outcome is durable; otherwise do not commit. |
| Redpanda unavailable to consumer | Consumer stops or pauses. Existing Postgres truth is unaffected. | No new offsets. |
| Redpanda unavailable to outbox relay | Keep outbox rows pending; ledger/outcome truth remains committed. | Not applicable to consumed offsets. |
| Billing consumer lag | Alert and backpressure producers/partitions according to lag thresholds. Stale reservations and delayed credits are reconciled from Postgres. | Continue processing while DB is healthy. |
| DLQ/quarantine redrive | Operator replays the same valid event/fingerprint or records a new corrected event ID that references the original `event_inbox_id` / `event_receipt_identity`. Changed same-ID payload stays conflict. Poison receipts without valid producer ID replay only by broker receipt identity and cannot become a money operation without a corrected event. | Normal inbox rules apply. |

### Committed-Offset Retry Ownership

`retry_scheduled` is allowed only after the consumer has durably stored an inbox
row and enough safe identity to re-enter processing from Postgres. Once the
consumer commits the Redpanda offset for that row, Redpanda redelivery is no
longer the recovery owner.

Owner: billing-service event ingestion owns a local inbox retry worker. The
worker is part of the async event-ingestion runtime, not the HTTP request path,
and it must be planned with the same event-ingestion slice if committed-offset
`retry_scheduled` states are implemented. Deployment packaging can still choose
a separate worker binary or an explicitly managed worker process, but the
durable owner is the inbox retry worker over `billing_event_inbox`.

Lifecycle:
- consumer stores `retry_scheduled`, increments `retry_count`, sets
  `next_attempt_at`, records `last_error_class`, clears any stale claim, commits
  the DB transaction, and then commits the Redpanda offset;
- retry worker claims eligible rows with `FOR UPDATE SKIP LOCKED` where
  `processing_state = 'retry_scheduled'`, `next_attempt_at <= now()`, and no
  unexpired `claim_deadline_at` exists; it sets `claim_owner`,
  increments `claim_generation`, and records a bounded claim deadline;
- the worker re-enters the same idempotent business processor using
  `event_inbox_id`, `event_receipt_identity`, operation identity, and the
  stored fingerprint. It never reconstructs behavior from raw payload;
- success transitions to `committed`, `duplicate_replay`, `waiting_dependency`,
  or `reconcile_required` with the same stored outcome rules as first delivery;
- retryable failure writes a later `next_attempt_at` with bounded backoff;
  exhausted or non-retryable failure transitions to `reconcile_required` or
  `quarantined` and opens or links the relevant reconciliation case;
- graceful shutdown stops claiming new rows and either finishes the in-flight
  transaction or lets the claim deadline expire. Crash after claim but before
  state update is recovered by stale-claim reclamation through
  `claim_generation` and `claim_deadline_at`.

## Reconciliation

Redpanda ingestion feeds reconciliation, but reconciliation remains Postgres
owned.

### Stale Reservations With No Terminal Event

The existing stale reservation scan remains necessary. Redpanda lag can delay a
terminal event, so stale-reservation reconciliation must check:

- active `usage_holds` past `expires_at`;
- whether `billing_event_inbox` has a committed, waiting, retrying, or
  quarantined terminal event for the same `usageOperationId`;
- whether the producer can safely redrive a terminal event;
- whether the safe repair is write-off, release, or manual review.

### Terminal Event Consumed But Processing Failed

If processing fails before a durable outcome, Redpanda redelivery is the retry.
If the inbox row is durable and the offset has been committed, the inbox retry
worker owns recovery from `retry_scheduled`. Repeated retryable failures keep the
row scheduled with bounded backoff; exhausted or non-retryable failures
transition to `reconcile_required` or `quarantined` with a safe failure class. A
reconciliation case links the inbox row and the business operation identity.

### Future/Conditional Payment Evidence Duplicate Or Conflict

This section is not current planning input. If customer payment/top-up is
explicitly reopened, duplicate payment evidence is handled by the versioned
`(payment_evidence_id, evidence_version)` selector and
`evidence_payload_fingerprint` rules:

- same versioned selector and fingerprint returns stored outcome;
- same versioned selector with changed fingerprint becomes evidence conflict;
- a new `evidence_version` under the same `payment_evidence_id` is evaluated as
  a new immutable evidence claim. If it would change an already-posted credit,
  billing must create an explicit reversal/refund/adjustment effect or
  reconciliation/manual-review path, not rewrite the prior ledger row;
- same fingerprint with another evidence selector cannot credit twice and opens
  or reads duplicate-evidence reconciliation.

### Missing Or Ambiguous Inference Evidence

Chargeable completion without required qualified inference evidence does not
create a customer charge. It creates or reads a `missing_inference_evidence`
case by `usageOperationId`. Later evidence repair must re-enter through an
idempotent billing operation, not by directly changing ledger rows.

### DLQ / Quarantine Replay Procedure

The quarantine source is `billing_event_inbox`, with optional emitted
`billing.ingestion.rejected.v1` for operational visibility.

Safe replay rules:
- replay the same event ID only with the same fingerprint;
- replay of a poison receipt without valid producer event ID is keyed by
  `event_receipt_identity` and the original `(topic, partition, offset)`;
- corrected producer payloads must use a new event ID and reference the original
  rejected event ID or, for poison receipts without one, the original
  `event_inbox_id` / `event_receipt_identity`;
- operator replay cannot skip business idempotency or terminal uniqueness;
- any money-changing repair creates normal ledger effects and audit rows.

### Operator-Safe Repair Path

Operators repair through explicit reconciliation commands or admin operations
that reference durable billing IDs:

- `event_inbox_id`
- `eventReceiptIdentity`
- `usageOperationId`
- `paymentEvidenceId`
- `evidenceVersion`
- `topupOperationId`
- `settlementEffectId`
- `reconciliationCaseId`
- `ledgerEntryId`

Support notes and logs must remain privacy-safe.

## Performance Requirements

### Hot Path Target

Design target for one uncontended critical event:

- one Redpanda event read;
- one short PostgreSQL transaction;
- no outbound network calls inside the transaction;
- O(1) lookup by event identity and business operation identity;
- p95 database transaction time under 100 ms in local integration benchmarks;
- p99 under 250 ms under the planned benchmark workload, excluding intentional
  account-row contention.

Planning may adjust numeric thresholds only with benchmark evidence and must not
weaken the correctness model to hit a latency target.

### Expected DB Transaction Shape

Usage terminal event:
- lock/insert inbox;
- lock account balance;
- lock business idempotency;
- lock usage operation and hold;
- insert terminal outcome, ledger effect, operation outcome, audit row, and
  outbox rows as needed;
- update balance, hold, usage operation, idempotency, and inbox.

Future payment evidence event:
- lock/insert inbox;
- lock account balance;
- lock business idempotency;
- lock top-up, payment attempt, and payment evidence rows;
- insert ledger effect, operation outcome, audit row, and outbox rows as needed;
- update balance, top-up/evidence state, idempotency, and inbox.

This transaction shape is dormant future context. It must not be planned for the
current sign-up-bonus and usage-only scope.

### Required Indexes

New inbox/outbox indexes:
- `UNIQUE (event_receipt_identity)` on `billing_event_inbox`.
- `UNIQUE (topic, event_id)` on `billing_event_inbox` where
  `event_id IS NOT NULL`.
- `UNIQUE (topic, partition_id, offset)` on `billing_event_inbox`.
- `idx_event_inbox_claim(processing_state, next_attempt_at, event_inbox_id)`.
- `idx_event_inbox_claim_lease(processing_state, next_attempt_at, claim_deadline_at, event_inbox_id)`.
- `idx_event_inbox_operation(operation_identity_type, operation_identity, event_inbox_id)`.
- `idx_event_inbox_payment_evidence(payment_evidence_id, evidence_version, event_inbox_id)`.
- `idx_event_inbox_account_recent(account_id, received_at DESC, event_inbox_id DESC)`.
- `idx_event_outbox_publish(state, next_attempt_at, outbox_event_id)`.
- `idx_event_outbox_source(source_table, source_id, outbox_event_id)`.

Existing hot-path indexes remain required:
- account lookup by `account_scope_key`;
- balance lock by `account_id`;
- usage operation lookup by `usage_operation_id`;
- future top-up lookup by `topup_operation_id` only after payment/top-up reopen;
- future payment evidence lookup by `(payment_evidence_id, evidence_version)`
  plus lineage readback by `payment_evidence_id` only after payment/top-up
  reopen;
- idempotency lookup by `(account_id, operation_kind, idempotency_key)`;
- reconciliation claim and duplicate-open-case indexes.

### Batching Policy

- Consumers may fetch batches from Redpanda for throughput.
- Critical money events are processed one at a time per assigned partition.
- Each event gets its own database transaction.
- Offset commits advance only after all prior events in the partition have
  durable outcomes.
- Outbox relay may publish batches, but each outbox row keeps independent
  attempt state and publish outcome.

### Safe Consumer Concurrency

- Maximum critical consumer concurrency equals assigned partitions, capped by
  the Postgres pool budget reserved for the consumer process.
- Do not process the same assigned partition concurrently for critical topics.
- The account balance row remains the intentional per-account contention point.
- If hot accounts cause lock timeouts, increase topic partitions, tune producer
  account key distribution, apply backpressure, or rate-limit at the producer.
  Do not replace the account lock with cache or broker ordering as the money
  invariant.

### Backpressure And Lag

Initial alert thresholds for critical topics:

- warning: oldest unprocessed event age above 60 seconds or partition lag above
  1,000 messages for 5 minutes;
- critical: oldest unprocessed event age above 5 minutes or partition lag above
  10,000 messages for 5 minutes;
- future payment evidence critical threshold, if payment/top-up scope is
  reopened: accepted evidence age above 10 minutes before durable billing
  outcome.

Backpressure behavior:
- pause partitions when DB timeout, lock-timeout, or outbox-pending rates cross
  configured thresholds;
- expose consumer readiness separately from HTTP readiness if the worker is a
  separate binary;
- fail closed for new paid execution admission if terminal event lag indicates
  billing can no longer release holds within the accepted stale-reservation
  budget.

## Observability

Metrics use low-cardinality labels only: topic, consumer group, outcome,
failure class, terminal kind, and operation kind. Do not label metrics by raw
event ID, account ID, request ID, inference ID, payment evidence ID, or API key.

Required metrics:
- Redpanda consumer lag by topic, partition, and group.
- Oldest unprocessed event age by topic and group.
- Event processing latency by topic, group, operation kind, and outcome.
- Database transaction latency by operation kind and outcome.
- Account row lock wait and timeout count.
- Idempotent replay count.
- Event fingerprint conflict count.
- Business-operation conflict count.
- Quarantine and rejected-event count by safe failure class.
- DLQ/redrive attempt count and outcome.
- Stale reservation count and age.
- Ledger effect commit count by effect type.
- Retry count by failure class.
- Outbox publish lag, attempt count, and publish failure count.

Required logs:
- safe event envelope identifiers: topic, partition, offset,
  `eventReceiptIdentity`, event ID when valid, event schema version, and safe
  operation identity;
- processing state transition;
- safe failure class/problem code;
- reconciliation case ID when opened;
- stored outcome ID when available;
- trace ID/request ID only as correlation.

Logs, traces, metrics, inbox rows, outbox rows, and audit rows must not contain:
- raw prompts;
- raw completions;
- SSE chunks;
- bearer tokens;
- API keys;
- DSNs;
- payment secrets;
- raw webhook payloads;
- full provider payloads.

Traces:
- span per consumed event;
- child span for database transaction;
- child span for outbox publish attempt;
- attributes limited to low-cardinality topic/group/operation/failure labels and
  safe correlation IDs.

## Test Requirements

Future event-ingestion planning must include at least these proof classes. They
are not current sign-up-bonus and usage-only planning input unless async event
ingestion is explicitly reopened:

- Duplicate Redpanda delivery returns the stored inbox and business outcome
  without duplicate ledger effects.
- Same `(topic, eventId)` with changed fingerprint becomes conflict and does not
  mutate money.
- Same `usageOperationId` terminal replay returns stored outcome.
- Completion and failure/write-off terminal conflict for the same
  `usageOperationId` opens conflict/reconciliation and does not double-mutate.
- Out-of-order terminal usage event before reserve writes waiting/reconciliation
  outcome and does not charge.
- Crash after DB commit before offset commit is simulated by redelivery and
  proves no duplicate effect.
- Poison event quarantine stores a privacy-safe failure class and commits the
  offset after quarantine, including the case where producer `eventId` or
  semantic identity is malformed or missing and the durable identity is
  `eventReceiptIdentity`.
- Unsupported schema version routes to quarantine or rejected outcome without
  partition blockage.
- Committed-offset `retry_scheduled` rows are recovered by the inbox retry
  worker without Redpanda redelivery, including stale-claim reclamation after
  worker crash or shutdown.
- Concurrent consumers processing different partitions for the same account
  serialize on the account balance row and preserve non-negative balance.
- Future payment evidence duplicate and changed-fingerprint conflict follow
  revalidated versioned evidence rules only after payment/top-up scope is
  reopened: same `(paymentEvidenceId, evidenceVersion)` and fingerprint replays,
  same selector with changed fingerprint conflicts, and a new version under the
  same lineage cannot silently rewrite a prior ledger effect.
- Future payment evidence before top-up/payment attempt does not credit and opens
  or reads reconciliation only after payment/top-up scope is reopened.
- DB timeout before durable outcome does not commit offset.
- Outbox relay retry publishes each committed ledger effect at least once while
  consumers can dedupe by ledger/effect identity.
- Consumer lag and stale reservation reconciliation tests prove delayed
  terminal events do not create duplicate charges or stuck invisible holds.
- Benchmark target for event finalization throughput includes uncontended,
  same-account contention, duplicate replay, and future payment evidence
  scenarios only when payment/top-up scope is reopened.

## Data Model Delta

No approved money authority is replaced. This historical event-ingestion
addendum is superseded for current planning. Its inbox/outbox and
payment-evidence versioned schema delta is future/conditional context and must
not be planned until async event ingestion or payment/top-up scope is explicitly
reopened. The active current data-model delta is the `signup_bonus_grants`
design in `design/data-model.md`.

| New table or column | Why needed | Indexes / constraints | Hot path impact | Correctness invariant |
| --- | --- | --- | --- | --- |
| New table `billing_event_inbox` | Durable event receipt, replay, conflict, quarantine, retry, and stored outcome for consumed Redpanda events. | `PRIMARY KEY (event_inbox_id)`, `UNIQUE (event_receipt_identity)`, partial `UNIQUE (topic, event_id)` when event ID is valid, `UNIQUE (topic, partition_id, offset)`, constrained state/kind/topic/identity checks, claim/lease/operation/account indexes. | One insert or locked read per consumed event before business mutation. | Offset is committed only after this row records a durable outcome; duplicate delivery cannot duplicate money effects; poison events without producer identity still have a durable quarantine receipt. |
| Repaired `payment_evidence_lineages` / `payment_evidence` versioned model | Future/conditional context for payments-service's stable `paymentEvidenceId` plus required `evidenceVersion` selector. Must be revalidated when payment/top-up scope is reopened. | `payment_evidence_lineages(payment_evidence_id)` binds one lineage to one top-up/attempt/account; `payment_evidence` has internal row ID and `UNIQUE (payment_evidence_id, evidence_version)`. | Future payment evidence event lookup would use the versioned selector; lineage readback still groups by `payment_evidence_id`. | A new evidence version is a new immutable evidence claim and cannot silently rewrite a prior ledger effect if the future contract still uses this selector. |
| New nullable `source_event_inbox_id` on `idempotency_records` | Links event-originated business idempotency to the consumed event without making event ID the business key. | FK to `billing_event_inbox`; index on `(source_event_inbox_id)` where non-null. | No extra lookup on normal replay path; useful for support and reconciliation. | Business idempotency remains keyed by account/kind/idempotency key; event linkage is traceability, not the only dedupe boundary. |
| New nullable `source_event_inbox_id` on `operation_outcomes` | Returns support-safe outcome lineage for event-originated operations. | FK to `billing_event_inbox`; index on `(source_event_inbox_id)` where non-null. | Written with the stored outcome. | Replays can connect event outcome and operation outcome without reading raw payload. |
| New table `billing_event_outbox` | Atomic linkage from committed billing state to emitted Redpanda events. | `PRIMARY KEY (outbox_event_id)`, `UNIQUE (topic, event_id)`, source row uniqueness where required, publish claim index by `(state, next_attempt_at, outbox_event_id)`. | One insert per emitted event in the same DB transaction as source state. | Billing never relies on a DB write plus direct broker publish dual write for emitted facts. |
| New optional `source_table` / `source_id` lineage in `billing_event_outbox` | Identifies whether the emitted fact came from `ledger_entries`, `operation_outcomes`, `reconciliation_cases`, or `billing_event_inbox`. | `idx_event_outbox_source(source_table, source_id, outbox_event_id)`, constrained `source_table`. | Supports relay and support readback; no money lock dependency. | Emitted event can always be traced back to authoritative Postgres state. |
| Optional new `event_inbox_id` on `reconciliation_cases` plus versioned evidence selector | Links event-caused reconciliation to the exact inbox row and, for payment evidence, the exact `(payment_evidence_id, evidence_version)` claim. | FK to `billing_event_inbox`; partial index on `(event_inbox_id)` where non-null; duplicate-open-case uniqueness remains based on existing reason/lineage keys, with evidence cases keyed by the versioned selector. | Written only for event-caused cases. | Event ambiguity is visible without making event ID the reconciliation dedupe key when a stronger business key exists. |

### `billing_event_outbox`

Required columns:

| Column | Type | Notes |
| --- | --- | --- |
| `outbox_event_id` | `UUID` | Internal primary key. |
| `event_id` | `TEXT` | Billing-produced event identity. |
| `topic` | `TEXT` | Emitted topic. |
| `event_key` | `TEXT` | Redpanda partition key. |
| `event_schema_version` | `TEXT` | Emitted schema version. |
| `event_fingerprint` | `TEXT` | Canonical event fingerprint. |
| `source_table` | `TEXT` | `ledger_entries`, `operation_outcomes`, `reconciliation_cases`, or `billing_event_inbox`. |
| `source_id` | `TEXT` | String form of source row ID. |
| `account_id` | `UUID` nullable | Account context where available. |
| `account_scope_key` | `TEXT` nullable | Emitted key context where available. |
| `state` | `TEXT` | `pending`, `publishing`, `published`, `retry_scheduled`, `failed`. |
| `payload` | `JSONB` | Bounded privacy-safe event payload. |
| `attempt_count` | `INTEGER` | Publish attempts. |
| `last_error_class` | `TEXT` nullable | Safe publish failure class. |
| `next_attempt_at` | `TIMESTAMPTZ` | Publish eligibility. |
| `created_at` | `TIMESTAMPTZ` | Insert time. |
| `published_at` | `TIMESTAMPTZ` nullable | Successful publish time. |
| `updated_at` | `TIMESTAMPTZ` | Last update time. |

Outbox relay rules:
- relay claims rows with `FOR UPDATE SKIP LOCKED`;
- publish to Redpanda using `event_key`;
- mark `published` only after the broker acknowledges the message;
- publish retry must reuse the same `event_id` and `event_fingerprint`;
- downstream consumers dedupe by the source identity, not by trusting a single
  broker delivery.

## Contract Consequences For Later Phases

This design does not create runtime event schemas. It also does not approve
event-contract work for the current sign-up-bonus and usage-only scope. If async
event ingestion is explicitly reopened later, contract design must define the
event envelopes for:

- `usage.execution.completed.v1`
- `usage.execution.failed.v1`
- future `payments.evidence.normalized.v1` only if payment/top-up scope is also
  reopened
- `billing.ledger.effect.v1`
- `billing.operation.outcome.v1`
- `billing.reconciliation.required.v1`
- `billing.ingestion.rejected.v1`

The contracts must preserve the design decisions here:
- `accountScopeKey` is required for processable critical money events.
- `request_id` remains correlation only.
- settlement uses `usageOperationId`, `settlementEffectId`, and qualified
  `inferenceId` for current usage paths. Future payment/top-up settlement may
  also use `paymentEvidenceId`, `evidenceVersion`, and `topupOperationId` only
  after scope reopen and provider-contract revalidation.
- future `payments.evidence.normalized.v1` and any synchronous
  `billing.topup.applyEvidence` repair must include or otherwise encode the
  revalidated version-scoped selector `(paymentEvidenceId, evidenceVersion)`.
- rejected/quarantined event contracts must allow billing to report either a
  valid producer `eventId` or the billing `eventReceiptIdentity` /
  `eventInboxId` for poison receipts whose producer identity was malformed or
  missing.
- event payloads carry fingerprints and safe references, not raw request,
  response, SSE, token, payment, or webhook bodies.

## Future Reopen Questions

These are future reopen questions, not current planning inputs:

1. Exact Redpanda partition counts and retention periods need deployment sizing
   input. The design requires per-account partition keys for critical money
   topics and gives minimum retention expectations.
2. Event contract design must confirm whether `gonka-proxy` can always include
   `accountScopeKey`, `usageOperationId`, terminal fingerprint, pricing snapshot
   lineage, and qualified inference evidence in terminal completion events.
3. Future payment event contract design must confirm the exact envelope names for
   `evidenceVersion`, corrected-event references, and poison-event receipt
   references. Selector semantics must be revalidated before billing requires
   `(paymentEvidenceId, evidenceVersion)` for normalized evidence processing.
4. Runtime packaging must decide whether the event-ingestion runtime is a
   separate binary, which is preferred by `docs/repo-architecture.md` for
   distinct lifecycle workloads, or an explicitly managed worker process. The
   committed-offset retry owner is not open: `retry_scheduled` belongs to the
   billing inbox retry worker over `billing_event_inbox`.
5. Planning for the current sign-up-bonus and usage-only scope must not include
   event-ingestion inbox/outbox or payment-evidence work from this addendum.
   Event ingestion planning requires an explicit reopen.

## Next Phase

Next phase for the active workflow is follow-up technical design review for the
current sign-up-bonus and usage-only repair. This artifact should be inspected
only to verify that Redpanda payment-evidence ingestion is clearly
future/conditional and not current planning input.

Do not create `tasks.md`, code, migrations, generated SQL, runtime adapters, or
event contracts from this historical addendum unless the workflow explicitly
reopens async event ingestion or payment/top-up scope.
