# PGO Profile Lifecycle

## When To Load
Load this when the performance spec is considering profile-guided optimization, production CPU profile collection, `default.pgo`, profile merging, workload-specific binaries, source-skew risk, or rollout validation for PGO.

Keep this at the contract boundary. The spec should decide whether PGO is allowed and what evidence it must satisfy, not tune compiler internals.

## Option Comparisons
- No PGO yet: choose when representative CPU profiles are unavailable or current bottlenecks are not CPU-bound.
- Single production profile: choose when one workload dominates and production sampling is representative enough.
- Merged fleet profile: choose when one binary serves multiple workload classes and a weighted common profile is acceptable.
- Workload-specific builds: choose when workload classes are materially different and operational complexity is justified.
- Benchmark-derived profile: choose only when production profiles are impossible and the benchmark is scenario-level and representative.
- Disable or rollback PGO: choose when canary or benchmark evidence shows regression, build risk, or profile-source mismatch.

## Accepted Examples
Accepted example: a read-heavy API binary may use PGO after CPU profiles are sampled from at least three busy canary or production instances during peak read traffic, each for the same wall duration. The spec requires `benchstat` on targeted benchmarks, canary latency/error guardrails, and a rollback build with `-pgo=off`.

Accepted example: a single worker binary runs both import and reconciliation modes. The spec compares merged weighted profiles versus workload-specific builds and selects merged PGO only if neither mode regresses beyond its threshold.

Accepted example: after a large package move or function rename, the spec treats the old profile as stale and requires a new profile before relying on PGO for the moved hot path.

## Rejected Examples
Rejected example: using a microbenchmark CPU profile as `default.pgo` for an HTTP service binary.

Rejected example: committing a `default.pgo` file without stating which binary, workload, collection window, and freshness rule it represents.

Rejected example: enabling PGO because it "usually helps" while the current bottleneck hypothesis is DB pool wait or lock contention.

Rejected example: using one profile path for multiple main packages that serve different workloads without stating why the profile applies to each.

## Pass/Fail Rules
Pass when:
- PGO is tied to a CPU-bound bottleneck or an explicit release performance objective
- profile source, workload, sampling window, merge rule, and freshness policy are documented
- build behavior names `default.pgo`, `-pgo=auto`, explicit `-pgo=path`, or `-pgo=off` as applicable
- validation includes before/after benchmarks and runtime canary guardrails
- source-skew, refactor, new-code, and workload-mix risks have reopen conditions

Fail when:
- profile representativeness is assumed rather than justified
- PGO is used to avoid fixing workload, DB/cache, or concurrency bottlenecks
- profile lifecycle lacks owner, refresh cadence, or rollback path
- a single profile is applied across binaries or workloads without comparison
- performance claims omit build-time, binary-size, and canary-risk considerations when relevant

## Validation Commands
Use these as proof obligations:

```bash
curl -o pgo-a.pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
curl -o pgo-b.pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
go tool pprof -proto pgo-a.pprof pgo-b.pprof > merged.pprof
go build -pgo=merged.pprof ./cmd/service
go build -pgo=off ./cmd/service
go test -run='^$' -bench='BenchmarkServiceHotPath$' -benchmem -count=20 ./internal/service > pgo-bench.txt
benchstat pgo-bench.txt
```

If the repository uses `default.pgo`, require the exact main package path and update/rollback rule in the plan.

## Exa Source Links
- [Go profile-guided optimization](https://go.dev/doc/pgo)
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [Go testing package benchmarks](https://pkg.go.dev/testing/)
- [benchstat command](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
