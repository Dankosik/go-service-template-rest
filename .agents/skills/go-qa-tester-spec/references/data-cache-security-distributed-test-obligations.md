# Data, Cache, Security, And Distributed Test Obligations

## Behavior Change Thesis
When loaded for symptom "the proof depends on durable state, cache behavior, tenant-scoped storage, migration, or async/distributed recovery", this file makes the model choose stateful observables and replay/fallback rows instead of likely mistake "use mocks or a successful API response as proof."

## When To Load
Load this when test strategy must cover SQL transactions, migrations, deterministic queries, cache key/fallback/staleness, tenant-safe durable/cache scoping, outbox/inbox, dedup, replay, ordering, compensation, reconciliation, or mixed-version compatibility.

## Decision Rubric
- Transaction proof must observe all-or-nothing durable state, rollback on representative failure, and recognizable error/cancellation behavior when relevant.
- Query proof must control data shape enough to prove ordering, pagination boundaries, filters, cursor validity, and equal sort-key ties.
- Cache proof must distinguish correctness, staleness, isolation, fallback, serialization/TTL, origin protection, and degraded cache behavior.
- Tenant/security proof here is stateful: tenant dimensions in keys, queries, persisted rows, cache entries, and durable side effects. Use API-boundary guidance when the main proof is caller-visible status or payload.
- Distributed proof must include duplicate, replay, ordering, ack-after-durable-state, retry class, poison, compensation/forward recovery, or reconciliation rows when those are part of the approved flow.
- Migration/backfill proof must cover expand/contract compatibility, generated SQL drift, idempotent/resumable backfill, verification gate, and destructive-step block where applicable.
- If the owning data/cache/security/distributed spec has not approved behavior, record a blocker rather than inventing it in QA strategy.

## Imitate
| Surface | Required Rows | Selected Proof | Observable To Copy |
| --- | --- | --- | --- |
| SQL transaction | all statements succeed; mid-transaction failure; context cancellation; retry after rollback | Integration | persisted rows all present or all absent, returned error class, no inconsistent read |
| Pagination/query | empty; first page; last page; invalid cursor; equal sort-key tie; concurrent insert if specified | Integration plus contract if client-visible | stable order, valid cursor, no duplicate/missing row across boundary |
| Cache | hit; miss; stale entry; corrupt entry; Redis timeout; tenant key mismatch; parallel miss | Unit and/or integration | returned value, origin call count, cache write/delete/bypass, fallback signal |
| Distributed flow | first delivery; duplicate delivery; out-of-order event; retryable error; poison message; replay after restart | Integration or process proof | durable state, inbox/outbox row, ack timing, retry counter, DLQ/escalation, reconciliation output |
| Migration/backfill | expand; old app/new schema compatibility; resumable backfill; verification gate; contract step | Migration validation/integration | migration applies cleanly, generated SQL drift resolved, backfill idempotent, destructive step blocked until verified |

## Reject
- "Mock repository proves rollback." The durable state boundary is exactly what needs proof.
- "API returns 200, so cache works." A successful response can hide stale reads, tenant key misses, serialization bugs, or origin stampede.
- "Distributed handler unit test" as the only proof for ack timing, inbox/outbox dedup, replay safety, or reconciliation.
- "Migration applies" without backfill resumability, old/new compatibility, or destructive-step verification when those risks exist.

## Agent Traps
- Do not duplicate API-boundary advice here. If the main proof is HTTP status/body/auth response, load the API reference instead.
- Do not treat a fake cache as proof of Redis TTL, serialization, eviction, or connection-failure behavior.
- Do not skip negative tenant rows when tenant identity affects keys, queries, or persisted side effects.
- Do not assume ordering or exactly-once behavior for distributed flows unless approved.

## Validation Shape
Stateful strategy is ready when it names the durable/cache/message observable, the controlled data shape, the failure or replay trigger, and the repository validation family such as integration, SQL generation/drift, or migration rehearsal.
