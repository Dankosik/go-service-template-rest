# Reference Selector

Each row names a pressure where this repository, a pinned version, or a
documented API contract overrides the obvious answer. State the expected
behavior change before loading.

Proof level, workload definition, capture, comparison, PGO lifecycle, and
completion policy have no reference here: [Benchmarking](../../../../../docs/benchmarking.md)
owns them. Database attribution belongs to
[`postgres-performance`](../../../../../docs/universal-disciplines/postgres-performance/SKILL.md).
Adding a reference back requires a decision it would change.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| A performance claim rests on a pasted benchmark table, a `benchstat` report, or a profile artifact | [evidence-and-profile-fit.md](evidence-and-profile-fit.md) | Re-capture through the target that records comparability, and read block and mutex profiles as the different questions they answer. |
| Changed code does work per item: loops, per-row queries or calls, fan-out, retries, fallback on miss | [amplification-and-scaling.md](../amplification-and-scaling.md) | Name the multiplier and the maximum the contract allows, instead of flagging an unbounded micro-cost or clearing on success-path proof. |
| A change reuses memory to reduce allocation: `sync.Pool`, shared buffers, retained backing arrays | [allocation-and-pooling.md](allocation-and-pooling.md) | Test that allocation is the bottleneck and that reuse survives a miss, instead of reading fewer allocs/op as a GC win. |
