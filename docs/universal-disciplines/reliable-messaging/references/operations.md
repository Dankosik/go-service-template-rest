# Messaging Operations

Load for ordering/concurrency, backpressure, retry/quarantine, redrive/replay,
drain, retention, security, or operational recovery.

Order only the required business scope and map it to partition/group/subject or
one consumer lane. Guard late state with expected version/current predicate.
Bound fetched unfinished work and consumer concurrency by downstream capacity;
align processing with visibility/ack/poll deadlines. Drain by stopping new
fetches, finishing or abandoning bounded work, and retaining old schema readers
until lag and quarantine close.

Classify failures: transient gets capped jittered retry; capacity pressure
reduces intake and delays retry; invalid schema/auth/payload quarantines with
safe evidence; business prerequisite retries only on a named changing signal;
unknown gets a small budget then investigation. A delayed retry message is a
new publish boundary and settles the original only after durable routing.

Quarantine retains original identity, tenant, schema, source, attempts, and
redacted failure evidence. Redrive/replay pins source/destination/range, code and
schema versions, counts, original identity, velocity, canary, invariant, and
stop signal; use a separate cursor/consumer so live progress remains recoverable.
Reconciliation compares authority with outbox, broker progress, inbox/effects,
and quarantine through idempotent repairs.

Authenticate producers, consumers, and operators separately at narrow topic,
queue, subject, group, and tenant scopes; payload tenant never overrides
authenticated scope. Keep sensitive payloads out of logs, metrics, quarantine,
and replay artifacts.

Observe outbox age and publish ambiguity, lag/oldest work, unfinished/ack
latency, redelivery/expiry, dedup/effect failures, quarantine/replay,
reconciliation drift, downstream saturation, and retention headroom. Recovery
capacity must catch up after the accepted outage without starving live traffic.
