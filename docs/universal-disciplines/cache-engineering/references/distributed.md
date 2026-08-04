# Distributed Cache Mechanics

This file owns remote shared-cache and cross-process coordination mechanics. The common cache contract remains canonical in `SKILL.md`.

## Justify the remote dependency

A distributed cache pays when shared reuse or coordination removes more origin work than network hops, serialization, remote failures, and operations add. Compare it with per-request/in-process reuse under the same workload. Separate cache data from durable or coordination data when eviction and durability semantics differ.

Record server/product/version, topology, replication/failover model, client library/version, deadlines/retries, memory limit, eviction policy, persistence, and cluster slot rules. Current Redis `latest` documentation contains version-specific behavior; verify the deployed version before adopting a command or guarantee.

## Make item operations atomic enough

Write value, schema/authority revision, generation, and expiry in one atomic server operation where possible. A successful plain overwrite can clear an existing Redis TTL unless the operation explicitly preserves or replaces it; an executable TTL check should detect immortal cache entries.

`MULTI/EXEC` prevents interleaving of its queued commands but has no rollback: runtime errors do not undo successful commands. `WATCH` provides optimistic compare-and-set and can abort on modification, expiry, or eviction. Lua scripts and Redis Functions can implement short atomic state transitions but block other server work for their execution time; bound input and complexity.

Redis 8.4 added string compare-and-set/delete options and `DELEX`; earlier versions need a verified `WATCH` retry or short script. Functions require Redis 7+. Keep these as version gates, not assumed capabilities.

Treat a client timeout as an ambiguous outcome: the server may have applied the command. Make invalidation, generation changes, and retries idempotent or compare-and-set, then read back or reconcile when the result matters.

## Expiry, eviction, and capacity

Redis expiration uses wall-clock timestamps and removes expired keys both on access and through active expiry work. Synchronized expiry can create fill and expiry latency spikes; spread non-semantic deadlines with jitter while preserving the maximum-age contract.

Choose eviction from measured access and value distributions. Redis LRU/LFU policies are approximate, so entry survival is never a correctness invariant. `noeviction` returns errors for memory-growing writes while reads can continue; `volatile-*` behaves similarly when no eligible expiring keys exist. Reserve memory headroom for replication/AOF buffers and temporary command overshoot, not only the stored dataset.

Set measured budgets for serialized bytes, key overhead, value count, network transfer, encode/decode CPU, and large-command latency. Compression is a benchmark decision. Partition/shard keys with the real skew: one hot key or one oversized value can dominate a slot even when aggregate capacity looks healthy.

## Use leases only for duplicate suppression

When a hot key duplicates work both inside each process and across the fleet, process-local single-flight and a cross-process lease remove different work and may be composed. A lease alone can leave same-process callers polling the remote cache; local single-flight alone still permits one fill per process. Keep both only when the measured duplication exists at both scopes.

If a lease is justified, acquire it with a unique owner token and TTL, and release it only with atomic compare-and-delete. A plain delete can remove a successor's lease. The lease protects only its remaining validity window; pause, clock drift, partition, expiry, eviction, or failover can allow overlapping owners.

Therefore a fill lease suppresses load but publication still checks authority generation. If a lock protects authoritative side effects, move that work to a system with the required coordination semantics or add downstream fencing that rejects stale owners. Asynchronous replication and `WAIT` do not make Redis strongly consistent under failover.

## Degrade without amplifying origin load

Use distinct connect, command, shared-fill, and caller deadlines. Bound transient retries with backoff and jitter inside the request budget. During timeout, outage, eviction, resharding, or failover, cap fallback concurrency before the origin; choose allowed stale, shed, or fail-closed behavior from the cached-value contract.

Recovery ramps misses and refreshes. Validate replica read freshness, cluster redirections, slot locality for multi-key atomics/scripts, and cold data after failover or resharding.

Observe Redis and origin together:

- command/connect latency, timeouts, errors, retries, and fallback rate;
- used dataset memory, non-evictable headroom, fragmentation, rejected writes;
- expired/evicted keys and time over the memory limit;
- hot-key/slot skew, network bytes, value sizes, and encode/decode latency;
- origin QPS/concurrency/load and end-to-end p95/p99 during faults and recovery.

## Primary sources

Checked 2026-08-02:

- [Redis `EXPIRE`](https://redis.io/docs/latest/commands/expire/), [`SET`](https://redis.io/docs/latest/commands/set/), and [`TTL`](https://redis.io/docs/latest/commands/ttl/)
- [Redis key eviction](https://redis.io/docs/latest/develop/reference/eviction/) and [latency from expires](https://redis.io/docs/latest/operate/oss_and_stack/management/optimization/latency/#latency-generated-by-expires)
- [Redis transactions](https://redis.io/docs/latest/develop/using-commands/transactions/)
- [Redis Lua scripting](https://redis.io/docs/latest/develop/programmability/eval-intro/) and [Redis Functions](https://redis.io/docs/latest/develop/programmability/functions-intro/)
- [Distributed locks with Redis](https://redis.io/docs/latest/develop/clients/patterns/distributed-locks/)
- [Redis replication](https://redis.io/docs/latest/operate/oss_and_stack/management/replication/) and [Redis Cluster scaling](https://redis.io/docs/latest/operate/oss_and_stack/management/scaling/)
- [Redis client error handling](https://redis.io/docs/latest/develop/clients/error-handling/)
