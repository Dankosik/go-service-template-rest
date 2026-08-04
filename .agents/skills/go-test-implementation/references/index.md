# Reference Selector

Load at most one reference by default; load more only for an independent pressure.

| Pressure | Load | Required effect |
| --- | --- | --- |
| An approved requirement, invariant, review finding, or bug report has to become named tests, or the layer carrying the proof is unsettled. | [proving-layer-and-oracle.md](proving-layer-and-oracle.md) | Assert what the approved behavior states and count the effects it allows — instead of reading the assertion back off the implementation or accepting any error at all. |
| The behavior is a schedule, timeout, backoff, cancellation, shutdown ordering, or goroutine handoff. | [deterministic-time-and-concurrency.md](deterministic-time-and-concurrency.md) | Advance a fake clock inside `synctest.Test` and wait for durable blocking — instead of sleeping past a real deadline and polling for the result. |
| The obligation involves SQL, migrations, transactions, row locks, concurrent claims, tenant isolation, or cache backend behavior. | [postgres-integration-proof.md](postgres-integration-proof.md) | Run what the engine decides against the `pgtest` database-per-test fixture — instead of a fake querier that returns whatever the fake was written to return. |

Mechanical test idiom needs no reference, because gates already decide it:
`errorlint` rejects `==` on wrapped errors, `contextcheck` rejects a dropped
parent context, `thelper` requires `t.Helper()`, `usetesting` requires the `t.*`
resource helpers, and `t.Setenv` panics in a parallel test. `SKILL.md` owns
routing to another skill.

Verify decisive newer Go or dependency behavior against the active toolchain or
current official primary docs rather than a snapshot in these files.
