# Source Of Truth And Derived Surfaces

## Behavior Change Thesis
When loaded for a task where audit logs, outbox rows, CDC, events, search, dashboards, exports, or projections might be treated as authoritative, this file makes the model choose one invariant-bearing write authority plus classified derived surfaces instead of likely mistake "the stream/view/log becomes the business source of truth because it is convenient."

## When To Load
Load this for authority ambiguity across current-state tables, history/evidence, integration feeds, and read surfaces.

## Decision Rubric
- Name the authoritative owner for each invariant-bearing entity or process before naming delivery or query surfaces.
- Classify each extra surface as one of: current truth, append-only evidence, integration outbox or CDC feed, read projection, search index, export, or analytics copy.
- For every derived surface, state lag budget, rebuild source, correction owner, and which correctness-critical paths must bypass it.
- Keep audit rows, domain events, and projections distinct. Audit explains actor and change context; events publish business facts; projections answer read queries.
- Treat provider payloads and webhook states as evidence to normalize, not as local lifecycle truth.
- If a derived surface becomes acceptance-critical, reopen the data specification because the truth boundary changed.

## Imitate

### Order lifecycle with downstream notifications
Context: `orders` owns acceptance, fulfillment status, and cancellation. Notifications need events and support needs a fast search view.

Choose service-owned OLTP order tables for current truth and invariant-bearing transitions. Write an outbox row in the same local transaction for downstream delivery. Treat notification topics, search documents, dashboard tables, and materialized views as derived-only surfaces with replay, rebuild, lag, and correction owners.

Copy this because it separates write authority, atomic integration evidence, and query convenience without denying that projections are useful.

### Audit trail vs domain event stream
Context: Compliance wants to know who changed payout settings. Other services need to react to payout approval and reversal.

Use an audit table for actor, reason, before/after metadata, and operator traceability. Use domain events or outbox records for consumed business facts. Keep payout state tables or ledger facts authoritative for business decisions.

Copy this because auditability alone does not imply event sourcing or a generic `events` table for every purpose.

### Customer health projection
Context: Product wants a "customer health" page combining orders, refunds, support cases, and usage facts.

Build a projection or materialized view with a freshness budget. Keep correctness-critical actions on owner data paths. State the rebuild and reconciliation owner.

Copy this because it gives product the read shape while preserving the owner data paths.

## Reject
- "Kafka is the source of truth for order lifecycle because every transition is emitted." This confuses integration evidence with the invariant owner.
- "Support can update the dashboard row to repair order status." This bypasses the write owner and creates split truth.
- "A single `events` table covers audit, replay, integration, and read models." This hides incompatible semantics and retention needs.
- "A CDC consumer owns the current status because it sees every change." CDC observes local truth; it does not own it unless the spec explicitly changes the model.

## Agent Traps
- Do not use "event" to mean audit row, outbox envelope, domain fact, and projection update in the same paragraph.
- Do not recommend direct cross-service table reads for a projection unless a temporary exception and removal path are explicit.
- Do not repair corrupted projections with ad hoc writes when the authoritative source can rebuild them.
- Do not say "event sourcing" when the prompt only asks for auditability or downstream notifications.

## Validation Shape
- Truth map lists each critical entity or fact, owner, authoritative table or fact stream, derived surfaces, and forbidden writers.
- Projection proof covers freshness, rebuild command or source, idempotent correction path, and bypass rules for correctness-critical reads.
- Integration proof covers same-transaction outbox or equivalent linkage and replay from retained source evidence.
