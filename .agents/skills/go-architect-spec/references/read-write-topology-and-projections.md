# Read Write Topology And Projections

## Behavior Change Thesis
When loaded for read/write topology pressure, this file makes the model keep query surfaces derived-only with freshness, rebuild, and bypass rules instead of letting a projection, cache, read service, or dashboard become hidden write authority.

## When To Load
Load when the task involves read/write separation, read services, projections, CQRS, materialized views, search indexes, exports, dashboards, aggregators, BFFs, or stale-read correctness.

## Decision Rubric
- State the one-line authority rule: who owns write truth, what is derived-only, and which correctness-critical path must bypass the projection.
- Use projections, materialized views, search indexes, read replicas, exports, aggregators, or BFFs for query shape and scale, not for command acceptance.
- Give every projection a support owner, freshness contract, rebuild path, drift detection, and failure mode.
- Reject direct cross-service joins against private operational stores as steady state.
- Consider event sourcing only when immutable history/replay, conflict handling, natural domain events, or CQRS/eventual-consistency benefits justify the operational complexity; not because the read model is complex.
- For exports and scans, define a stable read fence or documented consistency boundary instead of promising exact snapshots casually.

## Imitate

### Category Summary Vs Checkout Correctness
Context: category pages can tolerate stale product, price, and inventory summaries. Checkout cannot accept stale price or inventory.

Choose: use a derived read model, materialized view, API composition layer, or BFF for category pages. Keep checkout on write owners or a reservation/command path that enforces correctness.

Copy: this makes the projection useful without promoting it to authority.

### Administrative Dashboard With Many Joins
Context: admins need a dashboard joining orders, payments, refunds, and support state. It is not part of command acceptance.

Choose: build a dashboard-specific read model with explicit staleness and rebuild rules, sourced from owner events or owner APIs. Keep critical actions routed to owners.

Copy: this avoids changing owners' write schemas primarily for dashboard query shape.

### Long-Running Export
Context: a customer export scans large records and must not disturb request-path writes.

Choose: use a read replica, materialized export projection, or explicit read fence. Run execution through a worker with backpressure, cancellation, and export-status ownership.

Copy: this names completeness boundaries and avoids moving write ownership to the export store.

## Reject
- "The projection is usually current, so checkout can use it." Bad because acceptance semantics cannot depend on best-effort freshness.
- "The read service can query private operational tables directly." Bad because schemas and connection pools become shared runtime contracts.
- "Event source the system because the dashboard is hard." Bad unless immutable facts, replay, conflict handling, or CQRS/eventual-consistency trade-offs are true domain requirements.
- "Export as exact snapshot" without a read fence. Bad because it invents stronger consistency than the architecture proves.

## Agent Traps
- Do not hide staleness behind vague words like "near real-time"; use the supplied freshness boundary or mark it as an assumption.
- Do not forget rebuild and drift. A projection without a rebuild path becomes an accidental datastore.
- Do not make a derived read path the only path for operator repair or correctness-critical decisions.
- Do not let read-model ownership erase command ownership; separate support owner from write authority.
- Do not turn event sourcing into a whole-system choice when only one bounded ledger or workflow needs historical reconstruction.
