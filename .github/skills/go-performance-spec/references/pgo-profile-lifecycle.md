# PGO Profile Lifecycle

## Behavior Change Thesis
When loaded for symptom "the proposal involves PGO, default.pgo, profile merging, profile freshness, or source skew," this file makes the model require a representative CPU-profile lifecycle and rollback plan instead of likely mistake enabling PGO because it usually helps or using a stale/microbenchmark profile for a service binary.

## When To Load
Load when the spec is considering profile-guided optimization, production CPU profile collection, `default.pgo`, profile merging, workload-specific binaries, source-skew risk, or rollout validation for PGO.

## Decision Rubric
- Choose no PGO yet when representative CPU profiles are unavailable or the current bottleneck is not CPU-bound.
- Use a single production profile only when one workload dominates and the sampling window is representative.
- Use merged fleet profiles when one binary serves multiple workload classes and a weighted common profile is acceptable.
- Consider workload-specific builds only when workload classes are materially different and operational complexity is justified.
- Allow benchmark-derived profiles only when production profiles are impossible and the benchmark is scenario-level and representative.
- Require a disable/rollback path when canary or benchmark evidence shows regression, build risk, or profile-source mismatch.

## Imitate
- Read-heavy API binary may use PGO after CPU profiles are sampled from at least three busy canary or production instances during peak read traffic, each for the same wall duration. Copy the representative-source gate and `-pgo=off` rollback build.
- Worker binary runs import and reconciliation modes. The spec compares merged weighted profiles against workload-specific builds and selects merged PGO only if neither mode regresses beyond threshold. Copy the workload-mix comparison.
- After a large package move or function rename, the old profile is treated as stale and a new profile is required before relying on PGO for the moved hot path. Copy the source-skew reopen rule.

## Reject
- Microbenchmark CPU profile used as `default.pgo` for an HTTP service binary.
- Committing `default.pgo` without stating binary, workload, collection window, owner, and freshness rule.
- Enabling PGO while the bottleneck hypothesis is DB pool wait or lock contention.
- One profile path applied to multiple main packages with different workloads and no applicability argument.

## Agent Traps
- Using PGO to avoid making workload, DB/cache, or concurrency decisions.
- Treating a profile as evergreen across refactors, workload shifts, or new hot paths.
- Omitting build-time, binary-size, and canary-risk considerations when release behavior matters.
- Forgetting to name whether the build uses `default.pgo`, `-pgo=auto`, explicit `-pgo=path`, or `-pgo=off`.

## Validation Shape
Use proof obligations like these with repository-specific capture and package paths:

```bash
curl -o pgo-a.pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
curl -o pgo-b.pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
go tool pprof -proto pgo-a.pprof pgo-b.pprof > merged.pprof
go build -pgo=merged.pprof ./cmd/service
go build -pgo=off ./cmd/service
go test -run='^$' -bench='BenchmarkServiceHotPath$' -benchmem -count=20 ./internal/service > pgo-bench.txt
benchstat pgo-bench.txt
```

If the repository uses `default.pgo`, require the exact main package path, profile owner, refresh trigger, and rollback rule.
