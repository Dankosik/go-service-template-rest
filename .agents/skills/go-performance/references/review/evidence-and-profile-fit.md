# Evidence And Profile Fit

## When To Load

Load this when a change carries a performance claim and the review question is
whether the supplied proof can observe it: pasted benchmark tables, `benchstat`
output, or a CPU, heap, allocs, goroutine, block, mutex, or trace artifact.

## Behavior Change Thesis

A pasted `go test -bench` table reads as evidence because it has numbers, and
the reflex correction is "run it again under `benchstat`". That misses what
makes results comparable here: the metadata `make bench-compare` checks, which a
raw command never records. The other half is profile semantics — a mutex profile
read as "where goroutines are stuck" points the fix at the waiters instead of
the holder that made them wait.

## Decision Rubric

- Comparability is enforced by `make bench-compare`, which fails when the
  recorded package, pattern, count, benchmark time, build tags, Go environment,
  workload identity, dependency image, schema fingerprint, CPU identity and
  count, or `GOMAXPROCS` differ between sides. A hand-run `go test -bench`
  captures none of it, so the correction is to re-capture through
  `make bench-baseline` → change → `make bench` → `make bench-compare` with a
  `BENCH_WORKLOAD_ID`, not to add a `benchstat` invocation.
- The mutex profile records the stack at the **end** of the critical section —
  the holder whose lock made others wait. The block profile records the
  **blocking site** — the waiter. They answer different questions and suggest
  different fixes.
- An empty block or mutex profile proves nothing when the capture was live:
  those rates are off unless the process enabled them. `go test -blockprofile`
  and `-mutexprofile` enable them for you, so treat the trap as specific to
  profiles taken from a running service.
- A live profile is not a step available by default here.
  `observability.pprof.enabled` is `false` and the handlers ride the diagnostics
  listener, which binds every interface — turning it on is a deployment decision
  about who can reach that port. The reachable diagnostic is
  `make bench-profile BENCH_PROFILE=cpu|memory|block|mutex|trace`.
- Ask for the smallest proof that would change the decision. A benchmark that
  clears a local CPU or allocation claim is not improved by a load test, and a
  microbenchmark never clears a service-level percentile.

## Reject

- Treating statistical significance as sufficient: `benchstat` reports whether a
  delta is distinguishable from noise, not whether it is worth the
  implementation. Judge the delta against the accepted budget, and say so when a
  significant win does not earn added complexity.
- Rerunning until a comparison turns favorable. Fix the count and the comparison
  rule before capture; for small decision-critical deltas alternate baseline and
  candidate batches on one idle testbed rather than running each side once.

## Validation Shape

Name the one artifact that discriminates the disputed claim and the command that
produces it. [Benchmarking](../../../../../docs/benchmarking.md) owns proof level,
workload definition, capture, and completion policy; cite it rather than
restating a protocol in the finding.
