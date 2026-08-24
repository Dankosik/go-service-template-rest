# Benchmarking

Use the narrowest standard tool that measures the accepted claim.

| Claim | Command family |
| --- | --- |
| Go CPU, allocations, contention, or in-process HTTP | `make benchmark-capture` |
| Baseline/candidate comparison | `make benchmark-compare` |
| CPU, heap, block, mutex, or scheduler attribution | `go test` profiles and `go tool pprof/trace` |
| Real PostgreSQL cost | `make benchmark-capture` with the integration tag and existing Testcontainers harness |
| Client-visible HTTP capacity | pinned `grafana/k6` container with `test/performance/http/single-flow.js` |

Before measuring, record the operation, fixture size, concurrency or arrival
model, warm/cold state, environment, and accepted budget. Compare the same
source, toolchain, host, dependency versions, and workload. Production traffic,
dependency state, and GC can still require runtime telemetry.

`scripts/dev/benchmark.sh` is the evidence owner. Go and PostgreSQL capture
writes raw samples plus a `.meta` envelope containing candidate, exact command,
workload, budget, response owner, host, toolchain, dependency, and schema
identity. Comparison rejects mismatched envelopes before invoking the pinned
`benchstat`. HTTP capture retains its k6 summary, log, and non-secret run
metadata under `.artifacts/bench/http/`.

Benchmarks are explicit evidence, not default CI gates. Do not add a blocking
threshold until a stable comparable testbed and a named response owner exist.
CI runs only a non-load k6 inspection when the performance harness changes.

Load one leaf:

- [Go and in-process HTTP](benchmarking/go.md)
- [PostgreSQL](benchmarking/postgres.md)
- [External HTTP](benchmarking/http.md)
