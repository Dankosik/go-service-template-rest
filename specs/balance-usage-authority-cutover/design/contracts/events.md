# Event Contract Design

Status: review-ready
Date: 2026-06-02

## Contract Authority

Runtime event authority is `api/proto/events/v1/*.proto`, with generated DTOs
under `internal/api/events/v1`. This file records required event semantics for
technical design review only.

## Topics

Use the existing topic family unless OpenAPI/proto authoring proves a narrower
name is required:

- `billing.microlease.terminal.v1`
- `billing.microlease.checkpoint.v1`
- `billing.microlease.close.v1`
- `billing.microlease.facts.v1`

The existing config default `redpanda.billing_facts_topic` must stay aligned
with Redpanda safe-topic labels and tests. If implementation keeps
`billing.facts.v1` in any safe-label code, that mismatch must be repaired.

## Inbound Events

`MicroleaseChildTerminalSubmitted`

- Producer: `gonka-proxy`.
- Consumer owner: `cmd/billing-worker` terminal consumer role.
- Business identity: microlease ID plus debit authorization ID or child debit
  ID.
- Required lineage: account scope key, proxy allocator owner, microlease
  generation, child sequence, child cap USD atoms, request basis fingerprint,
  terminal basis fingerprint, terminal kind, charged/released/write-off atoms,
  pricing snapshot identity and fingerprint, safe terminal time, and safe
  metadata.
- Store effect: apply terminal outcome, ledger effects, operation outcome,
  child debit update, reconciliation case if needed, and billing outbox fact.

`MicroleaseCheckpointReported`

- Producer: `gonka-proxy`.
- Consumer owner: `cmd/billing-worker` checkpoint consumer role.
- Business identity: microlease ID plus checkpoint sequence.
- Required lineage: allocated child high-water, allocated child count, allocated
  cap sum, terminal submitted/published/accepted counts, unresolved child
  count/cap, local remaining atoms, owner/fence/generation, fingerprint.
- Store effect: record checkpoint, update microlease checkpoint state, open
  reconciliation when summaries conflict, and update admission/readback state.

`MicroleaseCloseReported`

- Producer: `gonka-proxy`.
- Consumer owner: `cmd/billing-worker` close consumer role.
- Business identity: microlease ID plus close sequence.
- Required lineage: same close proof fields used by the HTTP close route.
- Store effect: release only proven unallocated capacity, keep unresolved child
  cap reserved, or open reconciliation.

## Outbound Events

Billing facts are emitted from `billing_outbox` after committed local state.

Required event classes:

- microlease issued or replenished;
- terminal applied;
- microlease closed or close gap detected;
- admission rejected or gate changed;
- reconciliation case opened/updated/resolved.

Outbound payloads must be support-safe and contain no raw prompts,
completions, SSE chunks, bearer tokens, API keys, DSNs, payment secrets, raw
provider payloads, raw event payloads, dynamic proof URLs, or sensitive request
bodies.

## Idempotency, Ordering, And Offset Discipline

- Event envelope includes event ID, event type, schema/contract version,
  producer identity, event fingerprint, and safe observed time.
- `billing_event_inbox` dedupes by topic/event ID and topic/partition/offset.
- Changed fingerprint for the same business/event identity becomes conflict or
  quarantine, not a second money effect.
- Worker commits Redpanda offset only after the store applies, duplicates, or
  quarantines the event.
- Store operations are idempotent by event/business identity and link to
  operation outcomes where applicable.
- Redpanda ordering is not a money correctness guarantee. Postgres row locks,
  idempotency records, child debit state, and reconciliation cases own
  correctness.

## Retry, Quarantine, And Redrive

- Retryable store, producer, or offset failures do not commit offsets.
- Invalid schema, producer identity, fingerprint mismatch, over-cap terminal
  facts, and unsafe metadata are quarantined with support-safe reason class.
- Redrive must re-enter through the same inbox and app-level idempotency path.
- Reconciliation readbacks expose quarantine and conflict counts by low
  cardinality reason, not raw payload.

## Privacy

Event payloads and inbox/outbox rows must carry only safe lineage:

- account scope key;
- microlease ID;
- child debit/debit authorization ID;
- pricing snapshot identity and fingerprint;
- request and terminal basis fingerprints;
- safe execution or inference evidence IDs;
- bounded reason classes and safe metadata.

They must not carry raw public request bodies, prompt/completion content, SSE
data, auth credentials, payment/provider payloads, or dynamic proof URLs.
