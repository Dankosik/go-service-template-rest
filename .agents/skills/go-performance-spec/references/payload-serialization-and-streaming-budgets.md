# Payload Serialization And Streaming Budgets

## When To Load
Load this when the performance spec must decide payload size limits, JSON encoding/decoding budgets, response body size telemetry, streaming versus buffered responses, flush behavior, or large request/response shape.

Keep this contract-oriented. Hand off API representation design when fields, pagination, streaming semantics, or status codes become the primary client contract.

## Option Comparisons
- Smaller representation: choose when unneeded fields, nested resources, or unbounded arrays dominate latency and bytes.
- Pagination or chunking: choose when the API can expose bounded slices without requiring a streaming contract.
- Streaming response: choose when clients can consume incremental data and the contract can define ordering, partial failure, flushing, and timeout behavior.
- Buffered response: choose when all-or-nothing semantics, error handling, or small payloads make streaming unnecessary.
- Precomputed or delayed JSON segment: choose only when correctness and invalidation rules are explicit.
- Alternative encoding: choose only when API compatibility and client support are part of the spec.

## Accepted Examples
Accepted example: a report endpoint rejects unbounded `include=items` expansion and selects cursor pagination with `page_size <= 200`, response body `<= 256 KiB` for the default shape, and separate export operation for larger results.

Accepted example: a streaming JSON-lines export defines row order, first-byte latency target, per-row encoding budget, flush cadence, cancellation behavior, and what happens if encoding fails after partial output.

Accepted example: a decoder-heavy ingest path uses request body size buckets and `encoding/json.Decoder`-based streaming only after the contract defines max document size, unknown-field policy, and validation failure behavior.

## Rejected Examples
Rejected example: switching to streaming to "make it faster" while clients require all-or-nothing JSON array semantics and no partial failure mode exists.

Rejected example: optimizing JSON encoding while response shape still includes optional nested collections by default and has no byte-size ceiling.

Rejected example: relying on `Flush` as a guaranteed client-observed boundary through every proxy and protocol path.

Rejected example: adding a custom binary response for performance without API compatibility, content negotiation, and client migration policy.

## Pass/Fail Rules
Pass when:
- payload budgets include request body size, response body size, item count, and field expansion limits where relevant
- streaming contracts define ordering, flush cadence, partial failure, cancellation, and client compatibility
- local proof measures encoding/decoding under representative payload buckets
- runtime telemetry includes body size histograms or equivalent low-cardinality dimensions when payload size matters
- API handoff is recorded when representation or media type changes are client-visible

Fail when:
- serialization is optimized before bounding payload shape
- streaming behavior is chosen without partial-output semantics
- large payload validation lacks size limits and timeout/deadline expectations
- local benchmarks use tiny payloads while production payloads are large or skewed
- flush or full-duplex behavior is assumed without protocol/intermediary caveats

## Validation Commands
Use these as proof obligations:

```bash
go test -run='^$' -bench='BenchmarkEncodeReport/(page50|page200|export10k)$' -benchmem -count=20 ./internal/reports > encode.txt
benchstat encode.txt
go test -run='^$' -bench='BenchmarkDecodeIngest/(small|large|invalid)$' -benchmem -count=20 ./internal/ingest > decode.txt
benchstat decode.txt
go test -run='^$' -bench='BenchmarkStreamExport/export10k$' -trace trace.out ./internal/reports
go tool trace trace.out
```

For runtime validation, require metrics for request duration, request body size, response body size, timeout/cancellation count, encode/decode failures, and client-visible partial-stream failure count if streaming is used.

## Exa Source Links
- [encoding/json package](https://pkg.go.dev/encoding/json)
- [net/http package](https://pkg.go.dev/net/http)
- [OpenTelemetry HTTP metrics semantic conventions](https://opentelemetry.io/docs/specs/semconv/http/http-metrics/)
- [OpenTelemetry metrics supplementary guidelines](https://opentelemetry.io/docs/specs/otel/metrics/supplementary-guidelines)
- [Go diagnostics](https://go.dev/doc/diagnostics)
