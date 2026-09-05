# Resilience And Load

Load when a remote dependency, retry, overload, failover, queue, autoscaling, or
availability target can change the design.

For each edge fix the end-to-end deadline, per-attempt budget, admission and
concurrency ceiling, single retry owner, stable identity for ambiguity,
isolation boundary, degraded result, recovery path, and operator signals.
Retry only safe identities within attempt/time budgets; nested retries multiply
load. Preserve ambiguous side-effect identity and resolve it before creating new
intent.

Bound queues, memory, pools, connections, and per-tenant work. Reject early when
admitted work would miss its deadline, shed optional work before core work, and
reserve capacity for recovery/control. A queue whose drain rate cannot exceed
new arrivals has no recovery capacity; measure backlog age, not only count.

Bulkheads/cells require an isolation unit, capacity, routing, spillover, and
behavior when full. Circuit breakers require failure classification, sampling,
open behavior, half-open budget, and recovery evidence; multiple breaker layers
can create competing state machines. Rate limits express consumption policy;
load shedding protects current capacity. Autoscaling cannot create downstream
capacity or replace admission control.

Audit common-mode database, cache, broker, DNS, identity, secrets, routing,
time/certificate, quota/account/region, control-plane, pool, and operator
dependencies. Recalculate surviving capacity because failover can turn a local
loss into overload.

Proof covers representative load through the ceiling, slow/partial/lost
responses, retry storm and throttling, hot tenant/key, queue growth and drain,
failure-domain loss, stale/duplicate data during recovery, and return from
degraded mode. Measure useful success with tail latency, saturation, rejected
work, amplification, isolation, backlog age, and recovery time.
