# Resilience and Load

## Treat failure as partial and load-bearing

Remote work can succeed while its response is lost, fail slowly, return stale data, accept only part of a batch, or recover while callers continue retrying. Model latency and overload as failures, not only crashes.

For every critical edge define:

| Contract | Decision |
| --- | --- |
| Deadline | End-to-end budget and remaining budget passed downstream |
| Admission | Maximum accepted rate and concurrency; overload response |
| Attempt | Per-attempt timeout below the remaining deadline |
| Retry | One owner, eligible failures, attempt and time budget, backoff and jitter |
| Identity | How an ambiguous attempt is retried or reconciled safely |
| Isolation | Resource pool, tenant, cell, or dependency boundary |
| Degradation | Cheaper result or unavailable feature that preserves core invariants |
| Recovery | Reconnect, replay, drain, repair, restore, or operator action |
| Signal | Latency, errors, saturation, backlog age, drops, divergence, recovery progress |

## Deadline and retry budget

Propagate the caller's deadline through the call graph. Stop work that can no longer produce a useful response. Budget connection setup, attempts, backoff, and response processing explicitly.

Retry only when the operation is safe through idempotency, a stable request identity, or reconciliation and the failure is plausibly transient. Keep retries at one deliberate layer where possible; nested retries multiply load. Bound attempts and total time, use exponential backoff with jitter, and honor server throttling signals. A retry budget is part of capacity planning.

For an ambiguous side effect, preserve the same operation identity and resolve status before creating new intent. Hand provider-specific behavior to `external-api-integration` and broker-specific delivery to `reliable-messaging`.

## Stable overload behavior

Keep admitted work within the useful capacity of the full dependency chain:

- bound queues, concurrency, memory, connections, and per-tenant consumption;
- apply backpressure toward the producer where the protocol permits it;
- reject early and cheaply when admitted work would miss its deadline;
- shed optional or expensive work before core work;
- serve a defined degraded result when it preserves product invariants;
- reserve capacity for recovery, health, control, and critical tenants;
- test capacity after losing a replica, zone, cell, or dependency.

If arrival rate exceeds service rate, backlog grows without bound. Record backlog in time as well as count. Estimate drain time using spare service rate after new work; a queue that drains only when traffic stops has no recovery capacity.

## Isolation patterns

### Bulkhead or cell

Use when one tenant, dependency, workload class, or failure domain must not exhaust another. Define isolation unit, reserved versus shared capacity, routing, spillover policy, utilization cost, and behavior when one partition fills. Shared control planes and databases can defeat apparent isolation.

### Circuit breaker

Use when repeated calls to a persistently failing dependency consume scarce resources and a fast failure or fallback is meaningful. Define classified failures, sampling window, open behavior, half-open probe budget, recovery evidence, and metrics. Reuse a platform breaker when its scope and semantics fit; another breaker layer can create interacting state machines.

### Rate limit and load shedding

Rate limits express a consumption policy; load shedding protects current capacity. Define the key and fairness scope, burst allowance, response contract, retry guidance, priority, and which invariants degraded work must preserve.

### Queue-based leveling

Use when producers may burst beyond consumer rate and delayed completion fits the product contract. Define durable acceptance, maximum backlog age, admission limit, ordering, retry, poison handling, worker scale, drain rate, and user-visible pending or failed states. Hand job execution to `durable-background-jobs` and delivery to `reliable-messaging`.

## Failure-domain audit

Trace common-mode dependencies across apparently redundant instances:

- database, cache, broker, DNS, identity, secrets, configuration, routing, time, and certificate authorities;
- control planes and deployment systems;
- quotas, accounts, regions, networks, and third-party providers;
- shared client pools, thread pools, queues, and retry policies;
- operators, runbooks, and credentials needed for recovery.

Check failover direction and capacity. Automatic routing away from unhealthy capacity can overload the remainder and turn a local failure into a cascade.

## Proof matrix

Test at the smallest safe scope, then at the composed boundary:

- representative load to the declared ceiling and beyond it;
- slow dependency, dropped response, refused connection, and partial result;
- retry storm and throttling response;
- hot tenant or key and unfair consumption;
- queue growth, maximum age, poison work, and recovery drain;
- instance or failure-domain loss at peak load;
- stale, duplicate, and reordered data during recovery;
- dependency restoration, catch-up, and return from degraded mode.

Measure useful success and tail latency together with saturation, rejected work, retry amplification, backlog age, resource isolation, and recovery time. A fault test is incomplete if it stops before the system returns to a known good state.

## Primary sources

- [Amazon Builders' Library: Timeouts, retries, and backoff with jitter](https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/)
- [Amazon Builders' Library: Making retries safe with idempotent APIs](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/)
- [Google SRE: Addressing Cascading Failures](https://sre.google/sre-book/addressing-cascading-failures/)
- [Azure Architecture Center: Bulkhead pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/bulkhead)
- [Azure Architecture Center: Circuit Breaker pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/circuit-breaker)
- [Azure Architecture Center: Queue-Based Load Leveling pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/queue-based-load-leveling)
