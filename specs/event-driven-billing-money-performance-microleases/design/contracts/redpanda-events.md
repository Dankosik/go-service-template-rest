# Redpanda Event Contract Design

Status: review-ready design context
Trigger: terminal, checkpoint, close, billing fact, rejection, and
reconciliation events.

Runtime authority remains future `api/proto/events/v1/*.proto` files plus
generated DTOs. This file is design-only.

## Event Principles

- Events are transport and replay infrastructure, not reserve or money mutation
  authority.
- Producers write local outbox rows in the same transaction as source state.
- Consumers record durable inbox/idempotency before committing offsets.
- Broker idempotence or transactions do not replace business idempotency.
- Every event has stable event ID, aggregate identity, producer identity,
  contract version, schema version, event fingerprint, occurred/produced time,
  trace correlation, and safe payload.
- No raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs,
  payment secrets, raw provider payloads, raw event payloads, dynamic proof
  URLs, or sensitive request bodies.

## Proxy To Billing Events

### `MicroleaseChildTerminalSubmitted`

Purpose: settle one durable child debit after external execution or failure.

Required payload:

- microlease ID;
- account scope;
- proxy allocator owner ID;
- generation/fence;
- child debit authorization ID;
- child sequence;
- child cap USD atoms;
- terminal kind;
- charged/released/write-off USD atoms;
- request basis fingerprint;
- terminal basis fingerprint;
- pricing snapshot ID/fingerprint/policy version;
- qualified inference evidence ID or safe missing-evidence class;
- safe execution reference;
- terminal deadline and observed terminal time.

Billing effect:

- validate lineage and fingerprint;
- apply one terminal path;
- cap customer charge by child and parent authority;
- write ledger, outcome, child projection, outbox fact, or reconciliation.

### `MicroleaseCheckpointReported`

Purpose: provide allocator progress and lag/backpressure evidence.

Required payload:

- microlease ID;
- account scope;
- owner and generation/fence;
- checkpoint sequence;
- high-water mark;
- allocated child count and cap sum;
- terminal submitted/published/accepted counts;
- unresolved child count and cap sum;
- local remaining capacity;
- checkpoint fingerprint;
- checkpoint reason: progress, cutoff, shutdown, repair.

Billing effect:

- update checkpoint state;
- drive admission controls and reconciliation;
- no release unless close-proof criteria are met.

### `MicroleaseCloseReported`

Purpose: prove no future child debits for a grant and release only proven
unallocated capacity.

Required payload:

- all checkpoint fields;
- close reason: exhausted, cutoff, expiry, shutdown, operator, repair;
- allocator closed timestamp;
- final local row state fingerprint.

Billing effect:

- release `issued - allocated_child_cap_sum` only after proof validation;
- keep unresolved child cap reserved;
- open reconciliation for gaps or conflicts.

## Billing Outbox Events

### `MicroleaseIssued`

Emitted after billing reserves exposure and stores outcome.

Safe fields:

- microlease ID;
- account scope or support-safe account reference where allowed;
- owner/fence;
- issued cap;
- cutoff/expiry;
- result class;
- policy versions;
- pricing snapshot ID/fingerprint.

### `MicroleaseTerminalApplied`

Emitted after child terminal settlement.

Safe fields:

- microlease ID;
- child debit authorization ID;
- terminal kind;
- charged/released/write-off atoms;
- result class;
- ledger entry ID;
- settlement effect ID;
- reconciliation case ID if opened.

### `MicroleaseClosed`

Emitted after close/release/reconcile decision.

Safe fields:

- microlease ID;
- close state;
- released atoms;
- unresolved reserved atoms;
- reconciliation case ID when opened.

### `MicroleaseAdmissionRejected`

Emitted for safe operational projections when issue/replenish is denied.

Safe fields:

- reason class;
- strict/fail-closed class;
- use class;
- lag/stale/backlog bucket;
- no raw request data.

## Topic And Partitioning Direction

Planning must choose exact topic names, but the design requires:

- terminal events partitionable by account or microlease identity to preserve
  useful ordering where possible;
- checkpoint/close events partitionable by microlease identity;
- billing facts partitionable by account or aggregate identity;
- poison/quarantine handling without raw payload dumps;
- retention long enough for hot replay, retry storms, and reconciliation.

Ordering is a performance aid, not correctness authority. Billing still dedupes
and validates every event through inbox/idempotency.

## Consumer Rules

- Validate schema version, producer identity, event fingerprint, account binding,
  owner/fence, and amount ranges before money mutation.
- Insert or lock inbox record before applying effect.
- Same event/business ID plus same fingerprint replays.
- Same event/business ID plus changed fingerprint conflicts.
- Commit offset only after DB commit for applied, duplicate, conflict, or
  quarantine outcome.
- Use bounded retry/backoff and dead-letter/quarantine with support-safe reason.

## Producer Rules

- Proxy terminal and checkpoint events must come from durable local source rows.
- Billing facts must come from billing outbox rows written in the money
  transaction.
- Producer idempotence should be enabled where available, but business
  idempotency remains mandatory.
- Event payloads use USD atoms for money. Decimal display strings are not the
  settlement authority.

## Review-Relevant Non-Goals

- No Redpanda reserve command/outcome path for normal issue/replenish.
- No event-only terminal mutation without billing inbox/idempotency.
- No ClickHouse or analytics projection as admission authority.
- No payload-level raw request/proof dumps for debugging.
