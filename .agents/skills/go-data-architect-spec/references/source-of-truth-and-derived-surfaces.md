# Source Of Truth And Derived Surfaces

## When To Load
Load this when the task asks which data surface is authoritative, introduces an audit log, outbox table, CDC feed, event stream, materialized view, search index, dashboard table, export, or projection.

Use it to keep write authority, evidence history, and read convenience separate. A projection may be rebuilt, a CDC stream may notify, and an audit log may explain, but none of them becomes the business source of truth unless the spec explicitly chooses that model.

## Decision Examples

### Example 1: Order lifecycle with downstream notifications
Context: `orders` owns order acceptance, fulfillment status, and cancellation. Fulfillment and notifications need events, and support needs a customer-facing read surface.

Selected option: Keep `orders` current state and invariant-bearing transitions in the service-owned OLTP tables. Write an outbox row in the same local transaction for downstream delivery. Treat notification topics, search documents, dashboard tables, and materialized views as derived-only surfaces with lag, replay, rebuild, and correction owners.

Rejected options:
- Make the Kafka topic or CDC consumer the source of truth for order lifecycle.
- Let support dashboards update order status directly.
- Use an audit log as the only authoritative model when the business process needs current-state constraints.

Migration and rollback consequences:
- If the outbox route misbehaves, pause the connector or consumer and replay from the authoritative tables or outbox position.
- If a projection is corrupt, rebuild it from order truth or event history; do not repair it with writes that bypass the owner.
- Contracting old event fields is conditional on all consumers tolerating the new envelope and on replay from the retained source.

### Example 2: Audit trail vs domain event stream
Context: Compliance wants to know who changed payout settings, while other services need to react to payout approval and reversal.

Selected option: Keep an audit table for actor, reason, before/after metadata, and operator traceability. Keep domain events or outbox records for business facts that downstream systems consume. The payout state table or ledger facts remain authoritative for business decisions.

Rejected options:
- Use one generic `events` table for audit, replay, integration, and read model materialization without separate semantics.
- Treat provider webhook payloads as the local lifecycle truth.
- Adopt event sourcing only because auditability is required.

Migration and rollback consequences:
- Audit records are append-only evidence; redaction, retention, and legal hold rules must be explicit before destructive changes.
- If event consumers need a new field, add it compatibly and keep old fields until replay and mixed-version consumers are handled.
- If event sourcing is later approved, plan a separate migration because existing audit rows usually lack complete domain replay semantics.

### Example 3: Customer metrics projection
Context: Product wants a fast "customer health" page that combines orders, refunds, support cases, and usage events.

Selected option: Build a projection or materialized view for the page with a freshness budget. Keep correctness-critical actions on owner data paths and state the rebuild and reconciliation owner.

Rejected options:
- Cross-service direct table reads into private databases.
- Updating customer health as if it were the owner of order, refund, or support state.
- Changing owner schemas primarily to fit dashboard query shape.

Migration and rollback consequences:
- Roll back by disabling the projection reader and falling back to owner APIs or a slower query path.
- Rebuild drifted projection data from source facts; keep projection writes idempotent.
- If the projection becomes acceptance-critical, reopen the data specification because source-of-truth ownership changed.

## Source Links Gathered Through Exa
- PostgreSQL, "Logical Decoding": https://www.postgresql.org/docs/current/logicaldecoding.html
- PostgreSQL, "Logical Replication Publication": https://www.postgresql.org/docs/current/logical-replication-publication.html
- PostgreSQL, "CREATE MATERIALIZED VIEW": https://www.postgresql.org/docs/current/sql-creatematerializedview.html
- PostgreSQL, "REFRESH MATERIALIZED VIEW": https://www.postgresql.org/docs/current/sql-refreshmaterializedview.html
- Debezium, "Outbox Event Router": https://debezium.io/documentation/reference/stable/transformations/outbox-event-router.html

