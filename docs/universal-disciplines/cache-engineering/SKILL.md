---
name: cache-engineering
description: "Cache value, key, freshness, fill, invalidation, degradation, and value depth reference reached from go-db-cache."
---

# Cache Engineering

Use only after the active phase identifies a cache-specific decision. Inherit
its authority, artifact, review, proof, output, and completion contract; do not
select design, diagnosis, implementation, review, or production-action mode.

Core invariant: a cache is a bounded copy with explicit authority, key scope,
freshness, fill and invalidation ownership, degraded behavior, and a falsifier.
Reject a cache whose measured value does not justify that complexity.

Load one branch:

- whether caching should exist -> [value.md](references/value.md);
- value, key, freshness, or invalidation contract ->
  [contract.md](references/contract.md);
- concurrent fill, hot key, outage, or layer-specific behavior ->
  [runtime.md](references/runtime.md);
- migration, rollout, or proof -> [rollout-proof.md](references/rollout-proof.md).
