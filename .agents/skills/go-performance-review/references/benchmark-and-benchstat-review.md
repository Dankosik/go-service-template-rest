# Benchmark And Benchstat Review

## Behavior Change Thesis
When loaded for symptom "the diff adds or relies on Go benchmark or `benchstat` evidence," this file makes the model choose a benchmark-methodology finding instead of likely mistake "trust the benchmark table without checking timer boundaries, workload dimensions, allocation metrics, or noisy deltas."

## When To Load
Load this when the review touches `testing.B`, `B.Loop`, `b.N`, `-benchmem`, custom benchmark metrics, `benchstat`, benchmark CI, or PR benchmark claims.

## Decision Rubric
- Verify the benchmark measures the changed cost, not setup, fixture generation, random data generation, DB seeding, or mocked-away dependencies.
- Check timer boundaries. For `b.N`, expensive setup usually needs `b.ResetTimer`; for `B.Loop`, still check that workload setup and dimensions are realistic.
- Require `-benchmem` when the claim mentions allocations, GC, buffer reuse, or `sync.Pool`.
- Prefer old and new results collected with the same benchmark, flags, package, Go version, input size, `GOMAXPROCS`, cache state, and environment.
- Use repeated runs plus `benchstat` when the delta is small, noisy, or used to justify complexity.
- Separate statistical significance from practical significance. A tiny significant win may not justify a harder implementation.
- For the target repo's toolchain, verify `go doc testing.B.Loop` before recommending `B.Loop` in new benchmark code.

## Imitate
```text
[medium] [go-performance-review] internal/parser/parser_test.go:31
Issue:
Axis: Evidence; `BenchmarkParseBatch` builds the 10k-record JSON fixture inside the measured loop and the PR compares one old run to one new run. The claimed 8% parser improvement is therefore measuring fixture generation noise along with the changed parser.
Impact:
The benchmark can hide a real parser regression or falsely approve the new buffering code, especially because the PR also claims lower allocations without `-benchmem`.
Suggested fix:
Move fixture construction before the benchmark loop, use `B.Loop` if the repo toolchain supports it or reset the timer before the `b.N` loop, run old and new with `-benchmem -count=10`, and compare with `benchstat`.
Reference:
N/A
```

Copy the shape: identify the invalid measured cost, tie it to the PR claim, and request a same-workload A/B rerun.

## Reject
```text
Issue:
Benchmarks should use benchstat.
Impact:
The numbers are less good.
Suggested fix:
Run it more.
```

Reject it because it does not name the claim, missing metric, benchmark weakness, or merge risk.

```text
Issue:
Use `B.Loop` everywhere.
```

Reject it when the repo toolchain does not support `B.Loop`, the existing benchmark is correct, or the real problem is unrealistic workload shape.

## Agent Traps
- Flagging lack of `benchstat` on a tiny local benchmark where no performance claim is being used for merge approval.
- Missing that mocks or precomputed fixtures remove the changed I/O, serialization, or allocation cost.
- Treating `~` or wide confidence intervals as proof because the direction looks favorable.
- Ignoring custom metric direction, units, or whether a reported metric is per-op, total, or derived.
- Forgetting result sinks for pure computation benchmarks when optimizer elimination is plausible.

## Validation Shape
```bash
go test -run '^$' -bench '^BenchmarkParseBatch$' -benchmem -count=10 ./internal/parser > old.txt
go test -run '^$' -bench '^BenchmarkParseBatch$' -benchmem -count=10 ./internal/parser > new.txt
benchstat old.txt new.txt
benchstat -filter '.name:ParseBatch' old.txt new.txt
go test -run '^$' -bench '^BenchmarkParseBatch/size=(1k|10k|100k)$' -benchmem -count=10 ./internal/parser > new.txt
```

When the benchmark is parallel, include the intended `-cpu` list and state why it matches production concurrency.
