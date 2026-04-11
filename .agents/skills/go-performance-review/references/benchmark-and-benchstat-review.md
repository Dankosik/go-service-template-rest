# Benchmark And Benchstat Review

## When To Load
Load this when a review touches Go benchmarks, benchmark claims in a PR, `testing.B`, `B.Loop`, `-benchmem`, custom benchmark metrics, `benchstat`, or performance CI evidence.

Use this to review methodology, not to demand benchmark ceremony for every small change.

## Review Smell Patterns
- Only one benchmark sample is shown for old and new code.
- The benchmark omits `-benchmem` while claiming fewer allocations or lower GC pressure.
- Setup, fixture generation, random data generation, or DB seeding runs inside the timed loop.
- A `b.N` benchmark forgets `b.ResetTimer` after expensive setup.
- A benchmark uses tiny toy inputs to justify production hot-path complexity.
- The benchmark removes the changed cost with mocks or precomputed fixtures.
- A pure computation benchmark has no observable result or sink and may be optimized away. `B.Loop` reduces this risk, but does not fix unrealistic workloads.
- `benchstat` shows `~` or a wide confidence interval, but the PR treats the delta as proven.
- A custom metric lacks a clear "higher is better" or "lower is better" interpretation.
- Old and new benchmark runs used different flags, environment, data size, `GOMAXPROCS`, cache state, or Go versions without explanation.

## Evidence Required
- Use the same benchmark pattern, input sizes, package, flags, Go version, and environment for old and new runs.
- Prefer `B.Loop` for new benchmarks when the repository's Go version supports it; otherwise verify the `b.N` loop and timer boundaries.
- Use `-benchmem` for allocation, GC, buffer reuse, and `sync.Pool` claims.
- Prefer at least 10 old/new samples for benchstat when the benchmark is noisy or the delta is small.
- Treat statistical significance and practical significance separately: a tiny but significant delta may not justify complexity.
- Explain the workload dimensions the benchmark varies, such as input bytes, row count, page size, fan-out width, or concurrency.

## Bad Finding
```text
[low] [go-performance-review] internal/parser/parser_test.go:31
Issue:
Benchmarks should use benchstat.
Impact:
The numbers are less good.
Suggested fix:
Run it more.
Reference:
N/A
```

Why it fails: it does not name the claim, the missing metric, the benchmark weakness, or the merge risk.

## Good Finding
```text
[medium] [go-performance-review] internal/parser/parser_test.go:31
Issue:
Axis: Evidence; `BenchmarkParseBatch` builds the 10k-record JSON fixture inside the measured loop and the PR compares one old run to one new run. The claimed 8% parser improvement is therefore measuring fixture generation noise along with the changed parser.
Impact:
The benchmark can hide a real parser regression or falsely approve the new buffering code, especially because the PR also claims lower allocations without `-benchmem`.
Suggested fix:
Move fixture construction before the benchmark loop, use `B.Loop` or reset the timer before the `b.N` loop, run old and new with `-benchmem -count=10`, and compare with `benchstat`.
Reference:
Go `testing.B` benchmark timing rules and benchstat A/B comparison guidance.
```

## Validation Command Examples
```bash
go test -run '^$' -bench '^BenchmarkParseBatch$' -benchmem -count=10 ./internal/parser > old.txt
go test -run '^$' -bench '^BenchmarkParseBatch$' -benchmem -count=10 ./internal/parser > new.txt
benchstat old.txt new.txt
benchstat -filter '.name:ParseBatch' old.txt new.txt
go test -run '^$' -bench '^BenchmarkParseBatch/size=(1k|10k|100k)$' -benchmem -count=10 ./internal/parser > new.txt
```

When the benchmark is parallel, include the intended `-cpu` list and state why it matches production concurrency.

## Source Links From Exa
- [testing package benchmarks](https://pkg.go.dev/testing)
- [More predictable benchmarking with testing.B.Loop](https://go.dev/blog/testing-b-loop)
- [benchstat command](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
- [Go Benchmark Data Format](https://go.dev/design/14313-benchmark-format)

