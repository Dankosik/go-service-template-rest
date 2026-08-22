# Benchmarking

Use the narrowest standard tool that measures the accepted claim.

| Claim | Command family |
| --- | --- |
| Go CPU, allocations, contention, or in-process HTTP | `go test -bench` |
| Baseline/candidate comparison | `go tool -modfile=tools/go.mod benchstat` |
| CPU, heap, block, mutex, or scheduler attribution | `go test` profiles and `go tool pprof/trace` |
| Real PostgreSQL cost | integration-tagged `go test -bench` using the existing Testcontainers harness |
| Client-visible HTTP capacity | pinned `grafana/k6` container with `test/performance/http/single-flow.js` |

Before measuring, record the operation, fixture size, concurrency or arrival
model, warm/cold state, environment, and accepted budget. Compare the same
source, toolchain, host, dependency versions, and workload. Production traffic,
dependency state, and GC can still require runtime telemetry.

Benchmarks are explicit evidence, not default CI gates. Do not add a blocking
threshold until a stable comparable testbed and a named response owner exist.

Load one leaf:

- [Go and in-process HTTP](benchmarking/go.md)
- [PostgreSQL](benchmarking/postgres.md)
- [External HTTP](benchmarking/http.md)
