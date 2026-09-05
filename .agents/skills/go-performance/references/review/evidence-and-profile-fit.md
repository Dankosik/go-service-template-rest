# Evidence And Profile Fit

## Load When

Load this when a change carries a performance claim and the review question is
whether the supplied proof can observe it: pasted benchmark tables, `benchstat`
output, or a CPU, heap, allocs, goroutine, block, mutex, or trace artifact.

## Decide

- Compare only samples with matching recorded package, pattern, count,
  benchmark time, build tags, Go environment, workload identity, dependency
  image, schema identity, CPU, and `GOMAXPROCS`. Re-capture mismatched sides
  with the same `go test -bench` command before using `benchstat`.
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
  about who can reach that port. The local diagnostic is the matching `go test`
  profile flag and `go tool pprof` or `go tool trace`.
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

## Prove

Name the one artifact that discriminates the disputed claim and the command that
produces it. [Benchmarking](../../../../../docs/benchmarking.md) owns proof level,
workload identity, and completion policy; load the matching leaf for capture
rather than restating a protocol in the finding.
