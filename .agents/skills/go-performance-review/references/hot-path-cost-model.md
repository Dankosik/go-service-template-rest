# Hot Path Cost Model Review

## When To Load
Load this when a review touches request hot paths, loops, serialization, batching, fan-out, repeated parsing, repeated encoding/decoding, map/slice copying, compression, hashing, regex, template rendering, or algorithmic shape.

Use this to explain why the work scales with input size, request count, fan-out width, or payload bytes. Keep the finding grounded in the changed hot path and avoid speculative micro-tuning.

## Review Smell Patterns
- A changed loop adds nested scans over the same data.
- A per-item path repeats `json.Marshal`, `json.Unmarshal`, `fmt.Sprintf`, regex compilation, time parsing, compression, hashing, or template rendering.
- A request handler materializes the same payload multiple times as `[]byte`, `string`, maps, or structs.
- A diff copies full slices or maps on every request when only a small projection changed.
- Batching is removed or fan-out width grows with input size.
- New sorting or filtering happens after over-fetching a large dataset.
- The PR optimizes a cold helper while the hot path cost is in I/O, serialization, or fan-out.
- The benchmark input is much smaller than the actual production limit, hiding asymptotic growth.

## Evidence Required
- State the scaling dimension: page size, item count, payload bytes, fan-out width, cache misses, or request concurrency.
- Use a benchmark or request-path measurement with at least typical and high-end input sizes.
- Use CPU and allocation profiles when the cost is repeated CPU work or repeated materialization.
- Use query-count or dependency timing evidence when the "loop" is I/O rather than CPU.
- Do not require a benchmark for obvious unbounded work, but ask for the smallest proof when the impact depends on workload size.

## Bad Finding
```text
[medium] [go-performance-review] internal/api/list.go:74
Issue:
`fmt.Sprintf` in a loop is slow.
Impact:
The endpoint may be slower.
Suggested fix:
Use strings.Builder.
Reference:
N/A
```

Why it fails: it treats a generic micro-optimization as a blocker without proving the loop is hot, the input size matters, or formatting dominates cost.

## Good Finding
```text
[high] [go-performance-review] internal/api/list.go:74
Issue:
Axis: Hot Path Cost; the new response builder formats and re-parses the item timestamp inside the per-row loop after the handler already loaded normalized timestamps. This adds CPU and allocation work proportional to page size on the list endpoint's hot path, and the PR benchmark only covers 10 items while the API allows 500.
Impact:
At max page size the handler performs 500 duplicate parse/format operations per request, which can dominate latency and allocation rate under list traffic even if the 10-item benchmark looks neutral.
Suggested fix:
Reuse the normalized timestamp from the loaded model or precompute it once per item at the data boundary. Validate with benchmarks for typical and max page sizes using `-benchmem` and benchstat.
Reference:
Go benchmark methodology and diagnostics guidance for CPU/allocation profiling.
```

## Validation Command Examples
```bash
go test -run '^$' -bench '^BenchmarkListResponse/(size=50|size=500)$' -benchmem -count=10 ./internal/api > new.txt
benchstat old.txt new.txt
go test -run '^$' -bench '^BenchmarkListResponse/size=500$' -benchmem -cpuprofile cpu.out -memprofile mem.out ./internal/api
go tool pprof -top cpu.out
go tool pprof -top -alloc_space mem.out
```

For fan-out paths, add dimensions to sub-benchmark names, such as `shards=8`, `shards=64`, or `misses=100`, so benchstat compares the same workload slice.

## Source Links From Exa
- [Go Diagnostics](https://go.dev/doc/diagnostics)
- [testing package benchmarks](https://pkg.go.dev/testing)
- [Go Benchmark Data Format](https://go.dev/design/14313-benchmark-format)
- [runtime/pprof package](https://pkg.go.dev/runtime/pprof)

