# Runtime Limits And Capacity

## Load When

Load this when the decision is about the process's own envelope: container
memory, GC behavior, `GOMAXPROCS`, `GOMEMLIMIT`, `GOGC`, connection pool size,
admission concurrency, or how many requests may be in flight at once.

## Decide

- `cmd/service/internal/bootstrap/runtime_memory_limit.go` owns memory-limit
  application and skip reasons. Preserve Go's container-aware `GOMAXPROCS`
  default unless the accepted workload requires an override; an environment or
  runtime override disables automatic CPU-limit updates.
- Compare memory policy with the detected container limit and platform settings.
  Read the owner's applied/skipped startup signal before claiming a GC limit;
  changing `GOGC` trades CPU for memory and needs evidence of which is scarce.
- `internal/config/http_config.go` and `internal/config/postgres_config.go` own
  admission, connection and pool capacity. Use their current values and the
  accepted workload instead of copying defaults into this method. Raising the
  pool to match HTTP concurrency can move a queue into the database rather than
  improve throughput; preserve an informative overload rejection path.
- `cmd/service/internal/bootstrap/runtime_request_buffer_budget.go` owns the
  request-buffer estimate and warning. Relate admitted bodies to container
  memory before increasing body size or in-flight concurrency; readiness alone
  does not establish memory headroom.

## Reject

- A memory or capacity claim whose evidence is a Go benchmark: `-benchmem` shows
  per-operation churn, not live heap under real traffic. `otelruntime` already
  exports runtime memory and GC metrics, and `otelpgx.RecordStats` already
  exports pool acquire and wait metrics; a claim about either has instruments
  without adding any.

## Prove

`go test -bench` and `benchstat` for the per-operation half. For the
envelope half, name the startup log line (`runtime_memory_limit_applied`,
`runtime_memory_limit_skipped`, `runtime_request_buffer_budget_exceeded`) and
the runtime or pool metric that must move, with the threshold that decides
acceptance. [Benchmarking](../../../../../docs/benchmarking.md) owns proof level,
workload, and completion policy.
