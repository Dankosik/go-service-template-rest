# Reliable messaging operations

Read only the sections matching the requested boundary. Broker-specific references remain authoritative for exact acknowledgement, retention, ordering, and redrive mechanics.

## Ordering, concurrency, and backpressure

Order only the business scope that requires it. Map that scope to a partition key, message group, subject partition, or single-consumer lane and state what may reorder outside it. Guard state changes with an expected version or current-state predicate so late delivery becomes a safe no-op, conflict, or reconciliation signal.

Size consumer concurrency from downstream capacity. Bound fetched but unfinished work with batch size, prefetch, maximum pending acknowledgements, or in-flight limits. Align processing deadlines with acknowledgement, poll, or visibility deadlines; extend ownership only while useful work remains alive.

During ownership transfer or deploy, stop new fetches, finish or abandon bounded in-flight work, commit only completed effects, and drain old consumers before removing schema compatibility. Keep rollback readers and writers until new-version lag and quarantine stay inside the declared window.

## Retry, quarantine, and reconciliation

Classify failures by the action that can change the outcome:

| Failure | Delivery behavior | Owner |
| --- | --- | --- |
| transient transport/dependency | capped backoff with jitter and total attempt/time budget | publisher or consumer |
| rate/capacity pressure | delayed retry plus reduced intake/concurrency | service owner |
| invalid schema/auth/payload | durable quarantine with safe evidence | producer/schema/security owner |
| business conflict/prerequisite | retry only on a named changing signal; otherwise reconcile | domain owner |
| unknown | small bounded retry budget, then quarantine and investigate | owning team |

A poison record retains original logical identity, tenant, schema version, first/last failure, attempt count, source location, and a redacted payload or authorized pointer. Quarantine is not a normal retry queue.

If delayed retry creates another broker message, treat that handoff as a new publish boundary: preserve logical identity, require durable acceptance and routing before settling the original delivery, bound outstanding retry publishes, and treat a lost confirmation as ambiguous.

Reconciliation compares authoritative business state with outbox, broker progress, inbox/effects, and quarantine. Each repair is idempotent and records what changed.

## Redrive and replay

Treat redrive and replay as migrations. Pin source/destination, immutable IDs or ranges, schema and code version, snapshot counts, and original identity/causation. Canary a bounded batch, rate-limit against live capacity, observe business effects, and stop on a predeclared invariant. Prefer a separate cursor or consumer so live progress remains recoverable.

Before production action, require explicit authorization for the exact environment, range, destination, velocity, and stop condition. Fresh readback must prove the requested range and resulting business state, not merely broker command success.

## Security and tenant isolation

Authenticate workload identities and authorize producers, consumers, and operators separately at the narrowest topic, queue, subject, group, and tenant scope available. Encrypt in transit and at rest when classification requires it. Rotate credentials through reversible overlap.

Never let an untrusted payload select an authorized tenant without verifying it against the authenticated principal. Keep secrets and sensitive payloads out of logs, metrics, quarantine dashboards, and replay artifacts; use redacted metadata or authorized pointers.

## Signals, retention, and recovery capacity

Observe lifecycle health rather than broker health alone:

- outbox depth/oldest age, publish acceptance latency, ambiguity, and terminal errors;
- consumer lag/oldest age, unfinished work, processing and acknowledgement latency, redelivery, and ownership expiry;
- dedup conflicts, effect latency/failure, quarantine depth/age, replay rate, and reconciliation drift;
- downstream saturation, worker concurrency, memory, connections, and broker headroom.

Alerts name an owner and safe recovery action. Retention must exceed credible detection plus recovery time. Capacity proof includes catch-up after that outage without starving live traffic.

## Operational proof

When these branches are in scope, test reorder and ownership transfer, backpressure saturation, poison and retry exhaustion, bounded redrive/replay, retention expiry, reconciliation repair, mixed-version drain/rollback, authentication denial, and cross-tenant attempts. Record the expected business state and observable stop signal for each test.
