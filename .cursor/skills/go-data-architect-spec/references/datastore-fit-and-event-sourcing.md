# Datastore Fit And Event Sourcing

## Behavior Change Thesis
When loaded for a task where JSONB, document stores, Redis, search, time-series or columnar storage, event sourcing, or "scale" pressure might drive the truth model, this file makes the model choose an access-pattern and recovery fit test instead of likely mistake "adopt the fashionable datastore or event stream because it sounds flexible, scalable, or auditable."

## When To Load
Load this for datastore selection, flexible JSON events, event sourcing, Redis/search as truth, document or key-value stores, time-series or columnar storage, or scale/auditability claims used as storage justification.

## Decision Rubric
- Default service business truth to SQL OLTP unless another engine is justified by access patterns, invariant locality, retention/replay needs, recovery model, and operational readiness.
- Build an access-pattern catalog before approving a datastore: write unit, read predicates, join needs, cardinality, retention horizon, replay needs, hot keys/tenants, consistency needs, and repair path.
- Use `jsonb` for bounded adjunct attributes with weak relational invariants. Keep identity, money, lifecycle state, dedupe keys, tenant keys, and heavily filtered fields relational.
- Treat Redis, search indexes, caches, and materialized views as derived unless the spec explicitly accepts their loss, recovery, and correctness semantics as truth.
- Use columnar, time-series, or analytics stores for scan-heavy derived analysis or observability windows; do not move OLTP correctness there by default.
- Approve event sourcing only when event-native replay, temporal reconstruction, or event-contract semantics justify event evolution, snapshots, projection rebuilds, and operational complexity. Auditability alone is not enough.

## Imitate

### Usage billing plus analytics
Context: Product wants flexible JSON events in Postgres forever. Finance needs monthly billable recomputation after pricing fixes. Legal needs deletion for personal data and two-year billing evidence.

Choose authoritative append-only usage facts with tenant, subject, event identity, billable dimensions, event time, processed time, dedupe key, and retention partitioning. Keep flexible raw payloads only as adjunct evidence with bounded retention. Derive analytics into a columnar or analytics surface if scan workload justifies it.

Copy this because billing replay depends on typed facts and dedupe, while analytics flexibility can be derived.

### Redis holds for scarce seats
Context: A ticketing team wants Redis holds plus eventual Postgres writes because seat drops create hot contention.

Use Redis only as an acceleration or admission-control layer unless the spec explicitly accepts Redis durability, failover, and recovery semantics as the truth owner. Keep sold-seat authority and payment reconciliation in SQL with leases, constraints, or ledger-style allocation.

Copy this because scarce capacity needs a durable invariant owner, even when a cache helps absorb traffic.

### Event sourcing proposal for auditability
Context: A workflow needs operator audit, replay after bugs, downstream notifications, and current status queries.

Choose current-state tables plus transition history, audit rows, and outbox events unless full event-native reconstruction is required. Approve event sourcing only if the domain state is naturally rebuilt from events and the team can own event versioning, snapshots, projections, and repair tooling.

Copy this because audit, replay, and notifications are separable from event-sourced truth.

## Reject
- "Use MongoDB because the event shape changes often." Flexibility does not answer invariants, queries, retention, or reconciliation.
- "Put all activity in JSONB and index fields later." That hides billable dimensions, dedupe keys, tenant filters, and migration cost until they are urgent.
- "Kafka is the database because consumers can replay." Replay source and invariant owner are different decisions.
- "Search is the write model because support finds records there." Search optimizes discovery; it should not own correction or lifecycle truth.
- "Move high-volume events to a columnar store, then compute customer-visible quota there." Analytical scans and customer-visible correctness have different consistency and recovery needs.

## Agent Traps
- Do not recommend a new datastore without naming on-call, backup/restore, migration, observability, and local developer workflow consequences.
- Do not treat volume alone as proof SQL cannot own the source of truth; check write unit, index cost, partitioning, retention, and query shape first.
- Do not call a ledger "event sourcing" unless the current state is defined by replaying versioned domain events.
- Do not turn raw partner or product payloads into canonical state because they are convenient to store.

## Validation Shape
- Datastore-fit proof includes access-pattern catalog, invariant mapping, source-of-truth owner, retention/replay needs, recovery plan, and rejected engine rationale.
- Event-sourcing proof includes event schema ownership, versioning/evolution plan, snapshot policy, projection rebuild path, temporal query needs, and operator repair model.
- Derived-store proof includes rebuild source, staleness budget, failure behavior, and authoritative bypass path for correctness-critical reads.
