# Read Write Topology And Projections

## When To Load
Load this when the task involves read/write separation, read services, projections, CQRS, materialized views, search indexes, exports, dashboards, aggregators, BFFs, or stale-read correctness.

Use it to keep command authority and query convenience separate. A projection, cache, read replica, search index, export table, or materialized view can improve reads, but it must not quietly become the owner of invariant-bearing writes.

## Decision Examples

### Example 1: Category page summary vs checkout correctness
Context: Category pages need product, price, and inventory summaries with 90 seconds of acceptable staleness. Checkout cannot accept stale price or inventory.

Selected option: Use a derived read model, materialized view, API composition layer, or BFF for category pages. Keep checkout on the write owners or a reservation/command path that enforces correctness. State the projection as derived-only with a freshness contract.

Rejected options:
- Direct cross-service joins against `pricing` and `inventory` databases.
- Routing checkout through the category projection.
- Creating a "read service" without a clear owner for projection lag, rebuild, and support.

Evidence that would change the decision:
- Category pages need stronger freshness than the projection can reliably meet.
- Read load is modest and a simpler owner API composition path satisfies latency.
- A separate read product has a team owner, support budget, and rebuild/lag observability.
- Checkout can tolerate reservation semantics or pending status rather than immediate finality.

Failure modes and rollback implications:
- Projection lag can cause user-visible mismatch; disclose freshness and keep final acceptance on owners.
- Corrupt projection state should be rebuilt from write truth or event history.
- If the read path causes incidents, route reads back to owner APIs or a simpler cache; do not mutate through the projection.

### Example 2: Administrative dashboard with many joins
Context: Admins need a dashboard that joins orders, payments, refunds, and customer support state. It is not part of command acceptance.

Selected option: Build a dashboard-specific read model or materialized view with explicit staleness and rebuild rules. Source events or owner APIs should update it; each write owner remains authoritative.

Rejected options:
- Cross-service query joins in the dashboard against private operational databases.
- Event sourcing only because the dashboard is complex.
- Changing owners' write schemas primarily to satisfy dashboard query shape.

Evidence that would change the decision:
- The dashboard starts driving operational decisions that require current, correctness-critical state.
- Query traffic is low enough that API composition avoids projection complexity.
- Historical audit or replay becomes a true domain requirement, making event sourcing worth considering.

Failure modes and rollback implications:
- A projection can miss events, process duplicates, or drift; consumers need idempotency and reconciliation.
- A dashboard read model may expose stale state; label it as such and route critical actions to owners.
- Rollback can disable the projection consumer and rebuild later if write owners remain unaffected.

### Example 3: Long-running export
Context: A customer export scans large records and must not disturb request-path writes.

Selected option: Use a read replica, materialized export projection, or explicit read fence that defines what "complete enough" means. Run it through a worker runtime with backpressure, cancellation, and ownership of export status.

Rejected options:
- Request-path synchronous export.
- Vague exact-snapshot promises without a read fence.
- Moving write ownership to the export store.

Evidence that would change the decision:
- The export is used as a legal ledger or settlement source and needs stronger source-of-truth semantics.
- Data volume is small enough that an owner API can stream safely with deadline and rate limits.
- Freshness or completeness requirements cannot be met by the chosen read topology.

Failure modes and rollback implications:
- Export backlog can become unbounded; define queue limits and operator actions.
- Rebuilds can overload write stores; throttle and schedule rebuild windows.
- Rollback should preserve the authoritative write store and either pause exports or route to a slower safe path.

## Source Links Gathered Through Exa
- Azure Architecture Center, "CQRS pattern": https://learn.microsoft.com/en-us/azure/architecture/patterns/cqrs
- Azure Architecture Center, "Materialized View pattern": https://learn.microsoft.com/en-us/azure/architecture/patterns/materialized-view
- Azure Architecture Center, "Event Sourcing pattern": https://learn.microsoft.com/en-us/azure/architecture/patterns/event-sourcing
- Microservices.io, "Database per service": https://microservices.io/patterns/data/database-per-service.html
- Microservices.io, "Transactional outbox": https://microservices.io/patterns/data/transactional-outbox.html
- AWS Prescriptive Guidance, "Transactional outbox pattern": https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/transactional-outbox.html

