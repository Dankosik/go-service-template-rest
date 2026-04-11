# Hot Path Cost Model Review

## Behavior Change Thesis
When loaded for symptom "the changed hot path adds loops, copies, serialization, batching changes, fan-out growth, or repeated parse/encode/decode work," this file makes the model choose a scaling-dimension finding instead of likely mistake "block on isolated micro-optimization folklore such as `fmt.Sprintf` is slow."

## When To Load
Load this when the review turns on how work scales with request count, page size, item count, payload bytes, fan-out width, or repeated materialization in a changed hot path.

## Decision Rubric
- Name the scaling dimension first: item count, page size, payload bytes, fan-out width, cache misses, or request concurrency.
- Distinguish CPU/materialization loops from I/O loops. Use `db-cache-and-io-amplification.md` when the loop is primarily round trips.
- Prefer structural fixes such as reuse existing normalized data, batch once, move parse/encode to a boundary, avoid over-fetching, or preserve batching.
- Do not make a finding from a single operation inside a loop unless the path is hot and the loop bound can be material.
- Compare typical and high-end input sizes. Tiny benchmark inputs can hide asymptotic or per-item growth.
- Do not require a benchmark for obvious unbounded work, but ask for the smallest proof when impact depends on real workload size.

## Imitate
```text
[high] [go-performance-review] internal/api/list.go:74
Issue:
Axis: Hot Path Cost; the new response builder formats and re-parses the item timestamp inside the per-row loop after the handler already loaded normalized timestamps. This adds CPU and allocation work proportional to page size on the list endpoint's hot path, and the PR benchmark only covers 10 items while the API allows 500.
Impact:
At max page size the handler performs 500 duplicate parse/format operations per request, which can dominate latency and allocation rate under list traffic even if the 10-item benchmark looks neutral.
Suggested fix:
Reuse the normalized timestamp from the loaded model or precompute it once per item at the data boundary. Validate with benchmarks for typical and max page sizes using `-benchmem` and benchstat.
Reference:
N/A
```

Copy the shape: changed repeated work, scaling bound, why the proof misses that bound, and a structural fix.

## Reject
```text
Issue:
`fmt.Sprintf` in a loop is slow.
Suggested fix:
Use strings.Builder.
```

Reject it because it treats a generic micro-optimization as a blocker without proving the loop is hot, the input size matters, or formatting dominates cost.

```text
Issue:
This helper is optimized, so the endpoint should be faster.
```

Reject it when the request hot path cost is in I/O, serialization, batching, or fan-out outside that helper.

## Agent Traps
- Naming a CPU micro-cost while missing the larger repeated encode/decode, copy, or fan-out multiplier.
- Forgetting that max allowed page size or fan-out width can matter more than the benchmark's default input.
- Suggesting clever syntax rewrites before reducing duplicate work.
- Treating payload amplification and over-fetching as style rather than latency and throughput risk.
- Failing to state when the correct proof is query-count or dependency timing rather than a CPU benchmark.

## Validation Shape
```bash
go test -run '^$' -bench '^BenchmarkListResponse/(size=50|size=500)$' -benchmem -count=10 ./internal/api > new.txt
benchstat old.txt new.txt
go test -run '^$' -bench '^BenchmarkListResponse/size=500$' -benchmem -cpuprofile cpu.out -memprofile mem.out ./internal/api
go tool pprof -top cpu.out
go tool pprof -top -alloc_space mem.out
```

For fan-out paths, add dimensions to sub-benchmark names, such as `shards=8`, `shards=64`, or `misses=100`, so benchstat compares the same workload slice.
