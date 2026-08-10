# Runtime Limits And Capacity

## When To Load

Load this when the decision is about the process's own envelope: container
memory, GC behavior, `GOMAXPROCS`, `GOMEMLIMIT`, `GOGC`, connection pool size,
admission concurrency, or how many requests may be in flight at once.

## Behavior Change Thesis

The canonical Go answer to "this container is OOM-killed" or "raise capacity" is
to set `GOMEMLIMIT`, lower `GOGC`, add `automaxprocs`, and size the pool to match
the concurrency limit. In this repository each of those is already done, made
harmful by doing it, or inverts an ordering chosen on purpose — and the number
that actually decides whether the process survives its own concurrency is one no
layer computes at runtime except a startup warning.

## Decision Rubric

- `GOMAXPROCS` is deliberately unset (`cmd/service/internal/bootstrap/runtime_limits.go`).
  Since Go 1.25 the runtime defaults it from the cgroup CPU bandwidth limit and
  re-reads it as that limit changes; setting it through the environment or
  `runtime.GOMAXPROCS` disables **both** behaviors. An `automaxprocs`-style
  dependency is a regression here, not an addition.
- `GOMEMLIMIT` is published at startup as `runtime.memory_limit_ratio` (default
  `0.9`) times the detected cgroup limit, through automemlimit's
  `memlimit.FromCgroup` and `debug.SetMemoryLimit`. It is skipped when the
  platform already set `GOMEMLIMIT`, when there is no cgroup limit, or when the
  ratio rounds to zero — each skip logs `runtime_memory_limit_skipped` with a
  `reason`. Whether the GC has a limit is a log line to read, not a guess.
- `GOGC` is never set by this service. Changing it is a new decision trading CPU
  for memory against a limit the GC already has; state which one is scarce.
- The admission constants are ordered on purpose across `internal/config/http_config.go`
  and `internal/config/postgres_config.go`, each stating the relation from its own side:
  `http.max_in_flight` (256) sits **above** `postgres.max_open_conns` (25) so
  shedding engages after the pool saturates rather than before it is used, and
  `http.max_connections` (4096) sits above both so the informative `503` stays
  the common rejection and the kernel backlog is only the backstop. Raising the
  pool to "match" concurrency moves the queue somewhere with no answer to send.
- Admitted request bodies cost `http.max_in_flight` × `http.max_body_bytes`
  (defaults: 256 MiB). `reportRequestBufferBudget` compares that product against
  a quarter of the GC's limit and warns `runtime_request_buffer_budget_exceeded`.
  Raising `max_body_bytes` for uploads multiplies it, and the first symptom
  otherwise is GC CPU against a limit live data has already passed, then an OOM
  kill — while readiness still reports healthy.

## Reject

- A memory or capacity claim whose evidence is a Go benchmark: `-benchmem` shows
  per-operation churn, not live heap under real traffic. `otelruntime` already
  exports runtime memory and GC metrics, and `otelpgx.RecordStats` already
  exports pool acquire and wait metrics; a claim about either has instruments
  without adding any.

## Validation Shape

`make bench` and `make bench-compare` for the per-operation half. For the
envelope half, name the startup log line (`runtime_memory_limit_applied`,
`runtime_memory_limit_skipped`, `runtime_request_buffer_budget_exceeded`) and
the runtime or pool metric that must move, with the threshold that decides
acceptance. [Benchmarking](../../../../../docs/benchmarking.md) owns proof level,
workload, and completion policy.
