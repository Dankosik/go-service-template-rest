# Production Contract

Status: **Unresolved — production promotion is blocked until the owning service
accepts every applicable field.** Use `N/A` only with a concrete reason. This
file becomes service-owned with the first production feature and is not a
template-wide source of deployment defaults.

## Service scope

- Business capabilities: Unresolved.
- Authoritative facts and data owners: Unresolved.
- Public and internal interfaces: Unresolved.

## Dependency contract

| Dependency | Criticality | Readiness | Deadline | Concurrency | Retry owner | Identity | Ambiguous outcome | Recovery |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Unresolved | Unresolved | Unresolved | Unresolved | Unresolved | Unresolved | Unresolved | Unresolved | Unresolved |

## Capacity envelope

- Expected and burst arrival rate: Unresolved.
- Service time, payload bounds, and expected concurrency: Unresolved.
- First capacity ceiling and required headroom: Unresolved.
- Database connections, worker concurrency, and queue-age objective: N/A for
  the health-only scaffold; reopen when the service retains those capabilities.
- Surviving capacity after one failure-domain loss: Unresolved.
- Comparable workload evidence: Unresolved.

## Consistency and durability

- Transaction and read guarantees: Unresolved.
- Asynchronous propagation, replay, deduplication, and retention: Unresolved.
- RPO, RTO, backup owner, and restore proof: Unresolved.

## Edge and trust

- TLS termination and trusted proxy topology: Unresolved.
- Gateway retry and fleet rate-limit owners: Unresolved.
- Metrics and `pprof` reachability: Unresolved.
- Egress, identity, and authorization authorities: Unresolved.

## Operation and recovery

- SLI, SLO, and alert queries: Unresolved.
- Runbook and manual intervention paths: Unresolved.
- Rollback authority and mixed-version window: Unresolved.
- Reconciliation and recovery proof: Unresolved.
