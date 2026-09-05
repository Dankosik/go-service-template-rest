# Cache Rollout And Proof

Load when migration, mixed versions, rollout, rollback, or outcome proof can
change the decision.

Version key and serializer meaning across rolling deploys or introduce a new
namespace with explicit read/write transition and cleanup. Declare cold-fill
demand, canary/ramp, bypass or no-cache rollback, and retirement of old keys and
invalidators. A rollback stops new use but must tolerate entries and fills
created by the newer version.

Falsify both key directions: equivalent requests reuse one entry, while any
tenant/auth/policy/locale/representation difference cannot retrieve another
variant. Exercise update during fill, negative then create, slow/error fill,
cache timeout/outage, eviction/cold fleet, and rollback when those paths are in
scope. Inspect the actual retrieval path; key-string inequality alone is not
proof.

Measure the same before/after workload and report distributions plus origin
QPS/concurrency/load, hit/miss/fill/stale/error/eviction rates, value age, skew,
and recovery time relevant to the claim. A performance win exists only when the
user target and origin-load goal improve without violating correctness,
freshness, isolation, or degraded behavior.
